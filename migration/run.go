package migration

import (
	"context"
	"errors"
	"fmt"
	"github.com/cloudogu/ces-importer/cron"
	"log/slog"
	"sync/atomic"
)

type migrationRunner interface {
	RunMigration(context.Context) error
}

// Run is the main function to run the migration initiating the delta and final migration
func Run(ctx context.Context, finalTimestampStr, regularCron string, runner migrationRunner) error {
	var migrationRunning atomic.Bool

	finalTimestamp, err := ParseFinalTimestamp(finalTimestampStr)
	if err != nil {
		if errors.Is(err, ErrExpiredTimestamp) {
			slog.Error("no migration (delta/final) will run", "cause", err)
			return nil
		}

		slog.Warn("failed to parse final timestamp, fallback to zero value", "cause", err)
		finalTimestamp = FinalTimestamp{}
	}

	cronLooper, err := cron.New(ctx, regularCron, runDeltaMigration(finalTimestamp, runner, &migrationRunning))
	if err != nil {
		return fmt.Errorf("failed to create cron looper for expression %q: %w", regularCron, err)
	}

	slog.Info("Starting main delta migration loop")
	go cronLooper.Run()
	defer func() {
		cronLooper.Stop()
		slog.Info("stopped delta migration loop")
	}()

	if finalTimestamp.IsZero() {
		slog.Info("No valid final migration timestamp configured. Final migration will NOT run.")
		// Wait for context to be done
		<-ctx.Done()

		slog.Info("Received shutdown signal, stopping infinite delta migration loop.")

		return nil
	}

	slog.Info("Scheduled final migration", "startTime", finalTimestamp.String())

	doneFinalMigration := make(chan error)

	go func() {
		defer close(doneFinalMigration)
		doneFinalMigration <- runFinalMigration(ctx, finalTimestamp, runner, &migrationRunning)
	}()

	select {
	case <-ctx.Done():
		return fmt.Errorf("received shutdown signal before final migration has been completed: %w", ctx.Err())
	case err = <-doneFinalMigration:
		if err != nil {
			return fmt.Errorf("failed to run final migration: %w", err)
		}
	}

	slog.Info("Successfully finished final migration")

	return nil
}

func runDeltaMigration(finalTimestamp FinalTimestamp, runner migrationRunner, migrationRunning *atomic.Bool) cron.JobFunc {
	return func(ctx context.Context) (int, error) {
		// set migration to running to prevent simultaneous migrations
		migrationRunning.Store(true)
		defer migrationRunning.Store(false)

		if !finalTimestamp.IsZero() && finalTimestamp.Expired() {
			slog.Warn("Final migration timestamp is expired. Skipping delta migration.")
			return 0, nil
		}

		// run delta migration
		slog.Info("Start delta migration", "startTime", Now().String())

		if dErr := runner.RunMigration(ctx); dErr != nil {
			return 1, fmt.Errorf("failed to run delta migration: %w", dErr)
		}

		slog.Info("Delta migration succeeded", "endTime", Now().String())

		return 0, nil
	}
}

func runFinalMigration(ctx context.Context, finalTimestamp FinalTimestamp, migrator migrationRunner, migrationRunning *atomic.Bool) error {
	finalTimestamp.WaitUntilReady(ctx, func() bool {
		return !migrationRunning.Load()
	})

	slog.Info("Starting final migration")
	finalContext := SetFinalMigration(ctx)

	return migrator.RunMigration(finalContext)
}
