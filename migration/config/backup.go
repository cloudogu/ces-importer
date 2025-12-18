package configuration

import (
	"context"
	"fmt"
	"github.com/cloudogu/ces-importer/migration"
	backupv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"log/slog"
	"time"
)

const veleroBackupProvider = "velero"

var (
	watchTimeout = 10 * time.Second
)

type backupScheduleClient interface {
	Create(ctx context.Context, backupSchedule *backupv1.BackupSchedule, opts metav1.CreateOptions) (*backupv1.BackupSchedule, error)
	Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error)
	Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error
}

type cesBackupScheduleImporter struct {
	backupScheduleClient backupScheduleClient
}

func (bsi *cesBackupScheduleImporter) importBackupSchedules(ctx context.Context, config []migration.BackupSchedule) error {
	slog.Info("Importing backup schedules...")
	for _, schedule := range config {
		if err := bsi.delete(ctx, schedule.Name); err != nil {
			slog.Warn("failed to delete backup schedule", "name", schedule.Name, "schedule", schedule.Schedule, "error", err)
			continue
		}

		bs := &backupv1.BackupSchedule{
			ObjectMeta: metav1.ObjectMeta{
				Name: schedule.Name,
			},
			Spec: backupv1.BackupScheduleSpec{
				Schedule: schedule.Schedule,
				Provider: veleroBackupProvider,
			},
		}
		_, err := bsi.backupScheduleClient.Create(ctx, bs, metav1.CreateOptions{})
		if err != nil {
			return fmt.Errorf("failed to create backup schedule '%s': %w", schedule.Name, err)
		}

		slog.Debug("imported backup schedule", "name", schedule.Name, "schedule", schedule.Schedule)
	}

	slog.Info("...Successfully imported backup schedules.")
	return nil
}

func (bsi *cesBackupScheduleImporter) delete(ctx context.Context, scheduleName string) error {
	watcher, err := bsi.backupScheduleClient.Watch(ctx, metav1.SingleObject(metav1.ObjectMeta{Name: scheduleName}))
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}

		return fmt.Errorf("failed to watch backup schedule resource '%s': %w", scheduleName, err)
	}

	defer watcher.Stop()

	watchCtx, cancel := context.WithTimeout(ctx, watchTimeout)
	defer cancel()

	if err = bsi.backupScheduleClient.Delete(watchCtx, scheduleName, metav1.DeleteOptions{}); err != nil {
		if errors.IsNotFound(err) {
			return nil
		}

		return fmt.Errorf("failed to delete backup schedule resoruce'%s': %w", scheduleName, err)
	}

	slog.Debug("marked backup schedule as deleted, wait for deletion of resource", "name", scheduleName)

	for {
		select {
		case <-watchCtx.Done():
			return fmt.Errorf("timeout while waiting for backup schedule '%s' to be deleted", scheduleName)
		case event := <-watcher.ResultChan():
			schedule, ok := event.Object.(*backupv1.BackupSchedule)
			if !ok {
				slog.Warn("failed to cast backup schedule object to backup schedule", "object", event.Object)
				continue
			}

			if schedule.GetName() != scheduleName {
				slog.Warn("received unexpected backup schedule", "name", schedule.GetName(), "expected", scheduleName)
				continue
			}

			if event.Type == watch.Deleted {
				return nil
			}
		}
	}
}
