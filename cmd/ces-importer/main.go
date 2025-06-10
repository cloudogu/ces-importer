package main

import (
	"context"
	"fmt"
	"github.com/cloudogu/ces-importer/api/exporter"
	"github.com/cloudogu/ces-importer/api/importer"
	"github.com/cloudogu/ces-importer/configuration"
	"github.com/cloudogu/ces-importer/logging"
	"github.com/cloudogu/ces-importer/mail"
	"github.com/cloudogu/ces-importer/migration"
	"github.com/cloudogu/ces-importer/systeminfo"
	"github.com/cloudogu/k8s-registry-lib/repository"
	"k8s.io/client-go/rest"
	"log/slog"
	"os"
	"os/signal"
	ctrl "sigs.k8s.io/controller-runtime"
	"syscall"
	"time"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())

	clusterConfig, err := ctrl.GetConfig()
	if err != nil {
		panic(fmt.Errorf("failed to read kube config: %w", err))
	}

	cfg, err := configuration.ReadCoordinatorConfig()
	if err != nil {
		panic(fmt.Errorf("failed to read config: %w", err))
	}

	migrator, err := createMigrator(clusterConfig, cfg)
	if err != nil {
		panic(fmt.Errorf("failed to create migrator: %w", err))
	}

	// start migration-loops
	migrationDone := make(chan struct{})
	defer close(migrationDone)

	go func() {
		if mErr := migration.Run(ctx, cfg.FinalTimestamp, cfg.RegularCron, migrator); mErr != nil {
			slog.Error("failed to run migration", "cause", mErr.Error())
		}

		migrationDone <- struct{}{}
	}()

	// Wait for interrupt signals to gracefully shut down the server with a timeout of 5 seconds.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	<-quit
	slog.Info("Shutdown ces-importer ...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	// Cancel main context
	cancel()

	select {
	case <-shutdownCtx.Done():
		slog.Info("graceful shutdown-timeout of 5 seconds reached, forcing exit")
	case <-migrationDone:
		slog.Info("Migration has been stopped.")
	}

	slog.Info("exiting")
}

func createMigrator(k8sRestConfig *rest.Config, cfg configuration.Coordinator) (*migration.Migrator, error) {
	logInitializer := logging.NewLogInitializer(cfg.Logging.Level)
	err := logInitializer.InitializeWithLogFile()
	if err != nil {
		return nil, fmt.Errorf("failed to initilize log: %w", err)
	}

	logWriter := logging.NewWriter(logging.PathJobLogFile)

	exportAPIService := createAPIService(cfg.API)

	k8sClientSet, err := importer.CreateK8SClientSet(k8sRestConfig, cfg.General.Namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to create clients for kubernetes: %v", err)
	}

	jobService, err := migration.NewJobService(migration.JobServiceDependencies{
		JobProviderDependencies: migration.JobProviderDependencies{
			JobContainerConfig: cfg.JobContainer,
			SSHConfig:          cfg.SSH,
			APIKey:             cfg.API.ExporterApiKey,
			DoguVolumeBasePath: cfg.JobConfig.DoguVolumeBasePath,
			PVCClient:          migration.NewPVCGetter(k8sClientSet.Pvc),
		},
		JobClient: k8sClientSet.Job,
		PodClient: k8sClientSet.Pod,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create a new job service: %v", err)
	}

	// Validate Secrets
	if vErr := cfg.ValidateSecrets(context.Background(), k8sClientSet.Secret); vErr != nil {
		return nil, fmt.Errorf("found invalid secrets in configuration: %w", vErr)
	}

	exporterApiClient := createAPIClient(cfg.API)
	exportModeClient := exporter.NewExportModeClient(exporterApiClient)
	exportModeValidator := migration.NewExportModeValidatorApiClient(exportModeClient)

	systemInfoApiClient := exporter.NewSystemInfoClient(exporterApiClient)

	doguStartStopper := importer.NewDoguClient(k8sClientSet.Dogu)

	systemInfoProvider, err := systeminfo.NewSystemInfoProvider(k8sClientSet.Component, k8sClientSet.Dogu, systemInfoApiClient)
	if err != nil {
		return nil, fmt.Errorf("failed to create systemInfo provider: %w", err)
	}

	systemInfoValidator := systeminfo.NewValidator(cfg.General.ExcludedDogus)

	doguVolumeResizer := systeminfo.NewDoguVolumeResizer(k8sClientSet.Dogu, k8sClientSet.Pvc, cfg.General.ExcludedDogus)

	globalConfig := repository.NewGlobalConfigRepository(k8sClientSet.ConfigMap)

	mailSender := mail.CreateSender(
		cfg.Smtp,
		cfg.ExporterHost,
		[]string{logging.PathAppLogFile, logging.PathJobLogFile},
		globalConfig,
	)

	deps := migration.MigratorDependencies{
		ExportModeValidator: exportModeValidator,
		SystemInfoProvider:  systemInfoProvider,
		SystemInfoValidator: systemInfoValidator,
		DoguVolumeResizer:   doguVolumeResizer,
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
