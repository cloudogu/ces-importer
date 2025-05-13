package main

import (
	"context"
	"fmt"
	"github.com/cloudogu/ces-importer/api/exporter"
	"github.com/cloudogu/ces-importer/configuration"
	"github.com/cloudogu/ces-importer/cron"
	"github.com/cloudogu/ces-importer/logging"
	"github.com/cloudogu/ces-importer/migration"
	"log/slog"
	"net/http"
)

func main() {
	ctx := context.Background()

	cfg, err := configuration.ReadCoordinatorConfig()
	if err != nil {
		panic(fmt.Errorf("failed to read config: %w", err))
	}

	err = logging.Initialize(cfg)
	if err != nil {
		panic(err)
	}

	exportAPIService := createAPIService(cfg.API)

	deps := migration.MigratorDependencies{
		MaintenanceModeHandler: exportAPIService.MaintenanceModeService,
	}

	migrator := migration.NewMigrator(deps)

	cronLooper, err := cron.New(ctx, cfg.Migration.RegularCron, func(ctx context.Context) (int, error) {
		err = migrator.RunMigration(ctx)
		if err != nil {
			return 1, err
		}

		return 0, nil
	})
	if err != nil {
		panic(fmt.Errorf("failed to create cron looper for expression %q: %w", cfg.Migration.RegularCron, err))
	}

	slog.Info("Starting main loop")
	cronLooper.Run()
}

func createAPIService(apiCfg configuration.API) *exporter.Service {
	httpClient := http.DefaultClient
	exportClient := exporter.NewClient(apiCfg.ExporterHost, httpClient)
	exportService := exporter.NewService(apiCfg.ExporterHost, exportClient)

	return exportService
}
