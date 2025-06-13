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

	exportAPIService := exporter.NewServiceFromConfig(
		exporter.APIHost(cfg.ExporterHost),
		exporter.APIKey(cfg.ExporterApiKey),
		exporter.SkipTLSVerification(cfg.SkipTLSVerify),
	)

	exportAPIService.MaintenanceModeService.SetMessage(cfg.Migration.MaintenanceModeMessage.Title, cfg.Migration.MaintenanceModeMessage.Text)

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

	exportModeValidator := migration.NewExportModeValidatorApiClient(exportAPIService.ExportModeService)

	systemInfoProvider, err := systeminfo.NewSystemInfoProvider(k8sClientSet.Component, k8sClientSet.Dogu, exportAPIService.SystemInfoService)
	if err != nil {
		return nil, fmt.Errorf("failed to create systemInfo provider: %w", err)
	}

	systemInfoValidator := systeminfo.NewValidator(cfg.General.ExcludedDogus)

	doguVolumeResizer := systeminfo.NewDoguVolumeResizer(k8sClientSet.Dogu, k8sClientSet.Pvc, cfg.General.ExcludedDogus)

	globalConfig := repository.NewGlobalConfigRepository(k8sClientSet.ConfigMap)

	mailSender := mail.CreateSender(
		cfg.Smtp,
		cfg.ExporterHost,
		[]string{logging.PathAppLogFile},
		globalConfig,
	)

	deps := migration.MigratorDependencies{
		ExportModeValidator:    exportModeValidator,
		SystemInfoProvider:     systemInfoProvider,
		SystemInfoValidator:    systemInfoValidator,
		DoguVolumeResizer:      doguVolumeResizer,
		MaintenanceModeHandler: exportAPIService.MaintenanceModeService,
		JobRunner:              jobService,
		DoguStopper:            k8sClientSet.DoguControl,
		DoguStarter:            k8sClientSet.DoguControl,
		LogInitializer:         logInitializer,
		MailSender:             mailSender,
	}

	return migration.NewMigrator(deps), nil
}
