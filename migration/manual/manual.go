package manual

import (
	"context"
	"fmt"
	"log/slog"
	"sync/atomic"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
)

type migrationRunner interface {
	RunMigration(context.Context) error
}

type watcher interface {
	watch.Interface
}

type configmapClient interface {
	Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error)
	Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error
}

var labelSelector = "k8s.cloudogu.com/start-migration"

// StartManualMigrationConfigmapWatcher starts a watcher for ConfigMaps with the label "k8s.cloudogu.com/start-migration".
// When a ConfigMap is added, the migration is started.
func StartManualMigrationConfigmapWatcher(ctx context.Context, client configmapClient, namespace string, migrator migrationRunner, migrationRunning *atomic.Bool) error {
	slog.Info("Starting manual migration starter ConfigMap watcher.", "labelSelector", labelSelector)

	for {
		select {
		case <-ctx.Done():
			slog.Info("manual migration starter ConfigMap watcher stopped due to context cancellation")
			return ctx.Err()
		default:
		}

		watcher, err := client.Watch(ctx, metav1.ListOptions{
			LabelSelector: labelSelector,
		})
		if err != nil {
			return fmt.Errorf("failed to create manual migration starter ConfigMap watcher: %w", err)
		}

		err = handleWatchEvents(ctx, watcher, migrator, migrationRunning, client)
		if err != nil {
			slog.Warn("Watch connection closed, reconnecting...")
			time.Sleep(1 * time.Second)
			continue
		}

		return nil
	}
}

// handleWatchEvents handles watch events from the watcher.
func handleWatchEvents(ctx context.Context, watcher watcher, migrator migrationRunner, migrationRunning *atomic.Bool, client configmapClient) error {
	defer watcher.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case event, ok := <-watcher.ResultChan():
			if !ok {
				return fmt.Errorf("watch channel closed")
			}

			switch event.Type {
			case watch.Added:
				startManualMigration(ctx, event, client, migrationRunning, migrator)
			}
		}
	}
}

// startManualMigration starts the migration when a ConfigMap with the label "k8s.cloudogu.com/start-migration" is added.
func startManualMigration(ctx context.Context, event watch.Event, client configmapClient, migrationRunning *atomic.Bool, migrator migrationRunner) {
	slog.Info("manual migration starter configmap with start-migration label added", "object", event.Object)
	configMap, ok := event.Object.(*corev1.ConfigMap)
	if !ok {
		slog.Error("Failed to cast event object to ConfigMap")
		return
	}

	deleteErr := client.Delete(ctx, configMap.Name, metav1.DeleteOptions{})
	if deleteErr != nil {
		slog.Error("Failed to delete manual migration starter configmap", "name", configMap.Name, "error", deleteErr)
	} else {
		slog.Info("Successfully deleted manual migration starter configmap", "name", configMap.Name)
	}

	if migrationRunning.Load() {
		slog.Info("Migration is already running. Skipping manual migration start.")
		return
	}
	err := migrator.RunMigration(ctx)
	if err != nil {
		slog.Error(fmt.Sprintf("Failed to run manual migration: %v", err))
	}
	return
}
