package main

import (
	"context"
	"fmt"
	"github.com/cloudogu/ces-importer/configuration"
	"github.com/cloudogu/ces-importer/cron"
	"github.com/cloudogu/ces-importer/logging"
	"github.com/cloudogu/ces-importer/migration"
	"log/slog"
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

	deps := migration.MigratorDependencies{}
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
