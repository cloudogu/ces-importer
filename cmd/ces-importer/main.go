package main

import (
	"context"
	"fmt"
	"github.com/cloudogu/ces-importer/configuration"
	"github.com/cloudogu/ces-importer/cron"
	"github.com/cloudogu/ces-importer/logging"
	"github.com/cloudogu/ces-importer/migration"
	"io"
	"log/slog"
	"os"
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

	deps := migration.MigratorDependencies{
		LogWriter: logging.NewWriter(
			logging.PathJobLogFile,
			os.Remove,
			func(name string) (logging.File, error) {
				return os.Create(name)
			},
			io.Copy,
		),
		LogInitializer: logInitializer,
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
