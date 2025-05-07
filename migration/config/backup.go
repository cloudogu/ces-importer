package configuration

import (
	"context"
	"fmt"
	backupv1 "github.com/cloudogu/k8s-backup-operator/pkg/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"log/slog"
)

const veleroBackupProvider = "velero"

type backupScheduleClient interface {
	Create(ctx context.Context, backupSchedule *backupv1.BackupSchedule, opts metav1.CreateOptions) (*backupv1.BackupSchedule, error)
	Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error
}

type cesBackupScheduleImporter struct {
	backupScheduleClient backupScheduleClient
}

func (bsi *cesBackupScheduleImporter) importBackupSchedules(ctx context.Context, config []backupSchedule) error {
	slog.Info("Importing backup schedules...")
	for _, schedule := range config {
		if err := bsi.backupScheduleClient.Delete(ctx, schedule.Name, metav1.DeleteOptions{}); err != nil {
			slog.Warn("failed to delete existing backup schedule", "name", schedule.Name, "err", err)
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
