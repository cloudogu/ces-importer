package main

import (
	"context"
	"fmt"
	"github.com/cloudogu/ces-importer/api/exporter"
	"github.com/cloudogu/ces-importer/api/importer"
	"github.com/cloudogu/ces-importer/configuration"
	"github.com/cloudogu/ces-importer/cron"
	"github.com/cloudogu/ces-importer/logging"
	"github.com/cloudogu/ces-importer/mail"
	"github.com/cloudogu/ces-importer/migration"
	"github.com/cloudogu/ces-importer/systeminfo"
	componentEcoClient "github.com/cloudogu/k8s-component-operator/pkg/api/ecosystem"
	doguLibClient "github.com/cloudogu/k8s-dogu-lib/v2/client"
	"github.com/cloudogu/k8s-registry-lib/repository"
	"k8s.io/client-go/kubernetes"
	batchv1 "k8s.io/client-go/kubernetes/typed/batch/v1"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"log/slog"
	"os"
	"os/signal"
	ctrl "sigs.k8s.io/controller-runtime"
	"sync/atomic"
	"syscall"
	"time"
)

func main() {
	ctx := context.Background()

	cfg, err := configuration.ReadCoordinatorConfig()
	if err != nil {
		panic(fmt.Errorf("failed to read config: %w", err))
	}

	migrator, err := createMigrator(cfg)
	if err != nil {
		panic(fmt.Errorf("failed to create migrator: %w", err))
	}

	// start migration-loops
	cronTask, err := runMigration(ctx, cfg, migrator)
	if err != nil {
		panic(fmt.Errorf("failed to run migration: %w", err))
	}

	// Wait for interrupt signals to gracefully shut down the server with a timeout of 5 seconds.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	<-quit
	slog.Info("Shutdown ces-importer ...")

	done := make(chan struct{})

	if cronTask != nil {
		go func() {
			slog.Info("stopping cron-task ...")
			cronTask.Stop()
			slog.Info("cron-task stopped")
			close(done)
		}()
	}

	select {
	case <-done:
		slog.Info("Shutdown completed")
	case <-time.After(5 * time.Second):
		slog.Info("shutdown-timeout of 5 seconds reached")
	}

	slog.Info("exiting")
}

func runMigration(ctx context.Context, cfg configuration.Coordinator, migrator *migration.Migrator) (*cron.Task, error) {
	var migrationRunning atomic.Bool

	finalTimestamp, err := migration.ParseFinalTimestamp(cfg.FinalTimestamp)
	if err != nil {
		slog.Warn("failed to parse final timestamp, use zero value", "cause", err)
		finalTimestamp = migration.FinalTimestamp{}
	}

	cronLooper, err := cron.New(ctx, cfg.Migration.RegularCron, func(ctx context.Context) (int, error) {
		// set migration to running to prevent simultaneous migrations
		migrationRunning.Store(true)
		defer migrationRunning.Store(false)

		if finalTimestamp.Expired() {
			slog.Warn("Final migration timestamp is expired. Skipping delta migration.")
			return 0, nil
		}

		// run delta migration
		slog.Info("Start delta migration", "startTime", migration.Now().String())

		if dErr := migrator.RunMigration(ctx); dErr != nil {
			return 1, fmt.Errorf("failed to run delta migration: %w", dErr)
		}

		slog.Info("Delta migration succeeded", "endTime", migration.Now().String())

		return 0, nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create cron looper for expression %q: %w", cfg.Migration.RegularCron, err)
	}

	slog.Info("Starting main delta migration loop")
	go cronLooper.Run()

	if finalTimestamp.IsZero() {
		slog.Info("No final migration timestamp configured. Final migration will NOT run.")
		return cronLooper, nil
	}

	slog.Info(fmt.Sprintf("Starting final migration loop at %q", finalTimestamp.String()))

	go func() {
		finalTimestamp.WaitUntilReady(ctx, func() bool {
			return !migrationRunning.Load()
		})

		if fErr := runFinalMigrationLoop(ctx, migrator); fErr != nil {
			slog.Error("failed to run final migration: ", "error", fErr.Error())
		}

		slog.Info("Final migration succeeded")
	}()

	return cronLooper, nil
}

func runFinalMigrationLoop(ctx context.Context, migrator *migration.Migrator) error {
	slog.Info("Starting final migration")

	ctx = migration.SetFinalMigration(ctx)
	return migrator.RunMigration(ctx)
}

func createMigrator(cfg configuration.Coordinator) (*migration.Migrator, error) {
	logInitializer := logging.NewLogInitializer(cfg.Logging.Level)
	err := logInitializer.InitializeWithLogFile()
	if err != nil {
		return nil, fmt.Errorf("failed to initilize log: %w", err)
	}

	logWriter := logging.NewWriter(logging.PathJobLogFile)

	exportAPIService := createAPIService(cfg.API)

	k8sClientSet, err := createK8Sclientset(cfg.Namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to create clients for kubernetes: %v", err)
	}

	jobService, err := migration.NewJobService(migration.JobServiceDependencies{
		JobProviderDependencies: migration.JobProviderDependencies{
			JobContainerConfig: cfg.JobContainer,
			SSHConfig:          cfg.SSH,
			APIKey:             cfg.API.ExporterApiKey,
			DoguVolumeBasePath: cfg.JobConfig.DoguVolumeBasePath,
			PVCClient:          migration.NewPVCGetter(k8sClientSet.pvcClient),
		},
		JobClient: k8sClientSet.jobClient,
		PodClient: k8sClientSet.podClient,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create a new job service: %v", err)
	}

	// Validate Secrets
	if vErr := cfg.ValidateSecrets(context.Background(), k8sClientSet.secret); vErr != nil {
		return nil, fmt.Errorf("found invalid secrets in configuration: %w", vErr)
	}

	exporterApiClient := createAPIClient(cfg.API)
	exportModeClient := exporter.NewExportModeClient(exporterApiClient)
	exportModeValidator := migration.NewExportModeValidatorApiClient(exportModeClient)

	systemInfoApiClient := exporter.NewSystemInfoClient(exporterApiClient)

	doguStartStopper := importer.NewDoguClient(k8sClientSet.doguClient)

	systemInfoProvider, err := systeminfo.NewSystemInfoProvider(k8sClientSet.componentClient, k8sClientSet.doguClient, systemInfoApiClient)
	if err != nil {
		return nil, fmt.Errorf("failed to create systemInfo provider: %w", err)
	}

	systemInfoValidator, err := systeminfo.NewValidator(systemInfoProvider, k8sClientSet.doguClient, k8sClientSet.pvcClient)
	if err != nil {
		return nil, fmt.Errorf("failed to create systeminfo validator: %w", err)
	}

	globalConfig := repository.NewGlobalConfigRepository(k8sClientSet.configMap)

	mailSender := mail.CreateSender(
		cfg.Smtp,
		cfg.ExporterHost,
		[]string{logging.PathAppLogFile, logging.PathJobLogFile},
		globalConfig,
	)

	deps := migration.MigratorDependencies{
		ExportModeValidator: exportModeValidator,
		SystemInfoValidator: systemInfoValidator,
		MaintenanceModeHandler: &maintenanceModeHandler{
			service: exportAPIService.MaintenanceModeService,
			title:   cfg.Migration.MaintenanceModeMessage.Title,
			message: cfg.Migration.MaintenanceModeMessage.Text,
		},
		JobRunner:      jobService,
		DoguStopper:    doguStartStopper,
		DoguStarter:    doguStartStopper,
		LogWriter:      logWriter,
		LogInitializer: logInitializer,
		MailSender:     mailSender,
	}

	return migration.NewMigrator(deps), nil
}

func createAPIService(apiCfg configuration.API) *exporter.Service {
	exportClient := createAPIClient(apiCfg)
	exportService := exporter.NewService(exportClient)

	return exportService
}

func createAPIClient(apiCfg configuration.API) *exporter.Client {
	var options []exporter.HTTPClientOption

	if apiCfg.SkipTLSVerify {
		options = append(options, exporter.WithInsecure())
	}

	return exporter.NewClient(apiCfg.ExporterHost, apiCfg.ExporterApiKey, options...)
}

type k8sClients struct {
	pvcClient       corev1.PersistentVolumeClaimInterface
	podClient       corev1.PodInterface
	jobClient       batchv1.JobInterface
	configMap       corev1.ConfigMapInterface
	secret          corev1.SecretInterface
	doguClient      doguLibClient.DoguInterface
	componentClient componentEcoClient.ComponentInterface
}

func createK8Sclientset(namespace string) (k8sClients, error) {
	k8sRestConfig, err := ctrl.GetConfig()
	if err != nil {
		return k8sClients{}, fmt.Errorf("failed to read kube config: %w", err)
	}

	k8sClientSet, err := kubernetes.NewForConfig(k8sRestConfig)
	if err != nil {
		return k8sClients{}, fmt.Errorf("failed to create k8s client set: %w", err)
	}

	k8sCoreClient := k8sClientSet.CoreV1()
	k8sPVCClient := k8sCoreClient.PersistentVolumeClaims(namespace)
	k8sPodClient := k8sCoreClient.Pods(namespace)
	k8sConfigMapClient := k8sCoreClient.ConfigMaps(namespace)
	k8sSecretClient := k8sCoreClient.Secrets(namespace)

	k8sJobClient := k8sClientSet.BatchV1().Jobs(namespace)

	ecoSystemClient, err := doguLibClient.NewForConfig(k8sRestConfig)
	if err != nil {
		return k8sClients{}, fmt.Errorf("failed to create ecosystem client: %w", err)
	}

	k8sDoguClient := ecoSystemClient.Dogus(namespace)

	v1Alpha1Client, err := componentEcoClient.NewForConfig(k8sRestConfig)
	if err != nil {
		return k8sClients{}, fmt.Errorf("failed to create component client: %w", err)
	}

	k8sComponentClient := v1Alpha1Client.Components(namespace)

	return k8sClients{
		pvcClient:       k8sPVCClient,
		podClient:       k8sPodClient,
		jobClient:       k8sJobClient,
		configMap:       k8sConfigMapClient,
		secret:          k8sSecretClient,
		doguClient:      k8sDoguClient,
		componentClient: k8sComponentClient,
	}, nil
}

type maintenanceModeHandler struct {
	service *exporter.MaintenanceModeService
	title   string
	message string
}

func (m *maintenanceModeHandler) Enable(ctx context.Context) error {
	return m.service.Enable(ctx, m.title, m.message)
}

func (m *maintenanceModeHandler) Disable(ctx context.Context) error {
	return m.service.Disable(ctx)
}
