package migration

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync/atomic"

	"github.com/cloudogu/ces-importer/cron"
	"github.com/cloudogu/ces-importer/migration/manual"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
)

type migrationRunner interface {
	RunMigration(context.Context) error
}

type ConfigmapClient interface {
	Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error)
	Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error
}

// Run is the main function to run the migration initiating the delta and final migration
func Run(ctx context.Context, finalTimestampStr, regularCron string, changeFQDN bool, runner migrationRunner, cmc ConfigmapClient) error {
	var migrationRunning atomic.Bool

	// start the configmap watcher async
	go func() {
		err := manual.StartManualMigrationConfigmapWatcher(ctx, cmc, "ecosystem", runner, &migrationRunning)
		if err != nil {
			slog.Warn(fmt.Sprintf("Configmap watcher stopped: %v", err))
		}
	}()

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
		doneFinalMigration <- runFinalMigration(ctx, finalTimestamp, runner, &migrationRunning, changeFQDN)
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
		if migrationRunning.Load() {
			slog.Warn("Migration is currently running. Not responding to cron trigger to create a new delta migration.")
			return 0, nil
		}
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

func runFinalMigration(ctx context.Context, finalTimestamp FinalTimestamp, migrator migrationRunner, migrationRunning *atomic.Bool, changeFQDN bool) error {
	finalTimestamp.WaitUntilReady(ctx, func() bool {
		return !migrationRunning.Load()
	})

	slog.Info("Starting final migration")
	finalContext := SetFinalMigration(ctx)

	if changeFQDN {
		slog.Info("Triggering fqdn change")
		finalContext = SetTriggerFQDNChange(finalContext)
	} else {
		slog.Info("No fqdn change triggered")
	}

	return migrator.RunMigration(finalContext)
}
