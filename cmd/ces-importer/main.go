package main

import (
	"context"
	"fmt"
	"github.com/cloudogu/ces-importer/api/exporter"
	"github.com/cloudogu/ces-importer/api/importer"
	"github.com/cloudogu/ces-importer/configuration"
	"github.com/cloudogu/ces-importer/cron"
	"github.com/cloudogu/ces-importer/logging"
	"github.com/cloudogu/ces-importer/migration"
	"github.com/cloudogu/ces-importer/systeminfo"
	componentEcoClient "github.com/cloudogu/k8s-component-operator/pkg/api/ecosystem"
	ecoSystemV2 "github.com/cloudogu/k8s-dogu-operator/v3/api/ecoSystem"
	"k8s.io/client-go/kubernetes"
	"log/slog"
	"net/http"
	ctrl "sigs.k8s.io/controller-runtime"
)

func main() {
	ctx := context.Background()

	cfg, err := configuration.ReadConfigFromEnv()
	if err != nil {
		panic(fmt.Errorf("failed to read config: %w", err))
	}

	err = logging.Initialize(cfg)
	if err != nil {
		panic(err)
	}

	exporterApiClient := exporter.NewClient(cfg.ExporterApiKey, http.DefaultClient)
	exportModeClient := exporter.NewExportModeClient(exporterApiClient, cfg.ExporterHost)
	exportModeValidator := migration.NewExportModeValidatorApiClient(exportModeClient)

	systemInfoApiClient := exporter.NewSystemInfoClient(exporterApiClient, cfg.ExporterHost)

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

	doguStartStopper := importer.NewDoguClient(doguClient)

	ecosystemComponentClient, err := componentEcoClient.NewForConfig(k8sRestConfig)
	if err != nil {
		panic(fmt.Errorf("failed to create component client: %w", err))
	}
	componentClient := ecosystemComponentClient.Components(cfg.ImporterNamespace)

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
		MailSender:             nil,
		LogWriter:              nil,
		JobRunner:              nil,
		DoguStopper:            doguStartStopper,
		DoguStarter:            doguStartStopper,
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
