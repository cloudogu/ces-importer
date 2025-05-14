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
	ecoSystemV2 "github.com/cloudogu/k8s-dogu-operator/v3/api/ecoSystem"
	"io"
	"k8s.io/client-go/kubernetes"
	"log/slog"
	"net/http"
	"net/smtp"
	"os"
	ctrl "sigs.k8s.io/controller-runtime"
)

func main() {
	ctx := context.Background()

	cfg, err := configuration.ReadConfigFromEnv()
	if err != nil {
		panic(fmt.Errorf("failed to read config: %w", err))
	}

	logInitializer := logging.NewLogInitializer(
		func(name string, flag int, perm os.FileMode) (logging.File, error) {
			return os.OpenFile(name, flag, perm)
		},
		io.MultiWriter,
		cfg,
	)
	err = logInitializer.Initialize()
	if err != nil {
		panic(err)
	}

	mailSender := mail.CreateSender(
		cfg.MailConfig,
		smtp.SendMail,
		os.ReadFile,
		[]string{logging.PathAppLogFile, logging.PathJobLogFile},
	)

	logWriter := logging.NewWriter(
		logging.PathJobLogFile,
		io.Copy,
		func(name string, flag int, perm os.FileMode) (logging.File, error) {
			return os.OpenFile(name, flag, perm)
		},
	)

	exporterApiClient := exporter.NewClient(cfg.ExporterHost, cfg.ExporterApiKey, http.DefaultClient)
	exportModeClient := exporter.NewExportModeClient(exporterApiClient)
	exportModeValidator := migration.NewExportModeValidatorApiClient(exportModeClient)

	systemInfoApiClient := exporter.NewSystemInfoClient(exporterApiClient)

	k8sRestConfig, err := ctrl.GetConfig()
	if err != nil {
		panic(fmt.Errorf("failed to read kube config: %w", err))
	}

	kubernetesClient, err := kubernetes.NewForConfig(k8sRestConfig)
	if err != nil {
		panic(fmt.Errorf("failed to create kube-client: %w", err))
	}
	pvcClient := kubernetesClient.CoreV1().PersistentVolumeClaims(cfg.ImporterNamespace)

	ecosystemDoguClient, err := ecoSystemV2.NewForConfig(k8sRestConfig)
	if err != nil {
		panic(fmt.Errorf("failed to create dogu client: %w", err))
	}
	doguClient := ecosystemDoguClient.Dogus(cfg.ImporterNamespace)

	ecosystemComponentClient, err := componentEcoClient.NewForConfig(k8sRestConfig)
	if err != nil {
		panic(fmt.Errorf("failed to create component client: %w", err))
	}
	componentClient := ecosystemComponentClient.Components(cfg.ImporterNamespace)

	doguStartStopper := importer.NewDoguClient(doguClient)

	systemInfoProvider, err := systeminfo.NewSystemInfoProvider(componentClient, doguClient, systemInfoApiClient)
	if err != nil {
		panic(fmt.Errorf("failed to create systemInfo provider: %w", err))
	}

	systemInfoValidator, err := systeminfo.NewValidator(systemInfoProvider, doguClient, pvcClient)
	if err != nil {
		panic(fmt.Errorf("failed to create systeminfo validator: %w", err))
	}

	deps := migration.MigratorDependencies{
		ExportModeValidator:    exportModeValidator,
		SystemInfoValidator:    systemInfoValidator,
		MaintenanceModeHandler: nil,
		JobRunner:              nil,
		DoguStopper:            doguStartStopper,
		DoguStarter:            doguStartStopper,
		LogWriter:              logWriter,
		LogInitializer:         logInitializer,
		MailSender:             mailSender,
	}
	migrator := migration.NewMigrator(deps)

	cronLooper, err := cron.New(ctx, cfg.MigrationRegularCron, func(ctx context.Context) (int, error) {
		err = migrator.RunMigration(ctx)
		if err != nil {
			return 1, err
		}

		return 0, nil
	})
	if err != nil {
		panic(fmt.Errorf("failed to create cron looper for expression %q: %w", cfg.MigrationRegularCron, err))
	}

	slog.Info("Starting main loop")
	cronLooper.Run()
}
