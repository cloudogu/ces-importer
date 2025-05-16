package configuration

import (
	"context"
	"github.com/cloudogu/ces-importer/api/exporter"
	backupv1 "github.com/cloudogu/k8s-backup-operator/pkg/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func Test_cesBackupScheduleImporter_importBackupSchedules(t *testing.T) {
	testCtx := context.Background()
	t.Run("should import backup schedules", func(t *testing.T) {
		mockBsc := newMockBackupScheduleClient(t)
		mockBsc.EXPECT().Delete(testCtx, "schedule1", metav1.DeleteOptions{}).Return(nil)
		mockBsc.EXPECT().Delete(testCtx, "schedule2", metav1.DeleteOptions{}).Return(nil)

		bs1 := &backupv1.BackupSchedule{ObjectMeta: metav1.ObjectMeta{Name: "schedule1"}, Spec: backupv1.BackupScheduleSpec{Schedule: "0 0 * * *", Provider: veleroBackupProvider}}
		bs2 := &backupv1.BackupSchedule{ObjectMeta: metav1.ObjectMeta{Name: "schedule2"}, Spec: backupv1.BackupScheduleSpec{Schedule: "2 2 * 3 *", Provider: veleroBackupProvider}}

		mockBsc.EXPECT().Create(testCtx, bs1, metav1.CreateOptions{}).Return(bs1, nil)
		mockBsc.EXPECT().Create(testCtx, bs2, metav1.CreateOptions{}).Return(bs2, nil)

		bsi := &cesBackupScheduleImporter{
			backupScheduleClient: mockBsc,
		}

		schedules := []exporter.BackupSchedule{
			{Name: "schedule1", Schedule: "0 0 * * *"},
			{Name: "schedule2", Schedule: "2 2 * 3 *"},
		}

		err := bsi.importBackupSchedules(testCtx, schedules)

		require.NoError(t, err)
	})

	t.Run("should continue to import backup schedules if deletion of previous fails", func(t *testing.T) {
		mockBsc := newMockBackupScheduleClient(t)
		mockBsc.EXPECT().Delete(testCtx, "schedule1", metav1.DeleteOptions{}).Return(assert.AnError)
		mockBsc.EXPECT().Delete(testCtx, "schedule2", metav1.DeleteOptions{}).Return(assert.AnError)

		bs1 := &backupv1.BackupSchedule{ObjectMeta: metav1.ObjectMeta{Name: "schedule1"}, Spec: backupv1.BackupScheduleSpec{Schedule: "0 0 * * *", Provider: veleroBackupProvider}}
		bs2 := &backupv1.BackupSchedule{ObjectMeta: metav1.ObjectMeta{Name: "schedule2"}, Spec: backupv1.BackupScheduleSpec{Schedule: "2 2 * 3 *", Provider: veleroBackupProvider}}

		mockBsc.EXPECT().Create(testCtx, bs1, metav1.CreateOptions{}).Return(bs1, nil)
		mockBsc.EXPECT().Create(testCtx, bs2, metav1.CreateOptions{}).Return(bs2, nil)

		bsi := &cesBackupScheduleImporter{
			backupScheduleClient: mockBsc,
		}

		schedules := []exporter.BackupSchedule{
			{Name: "schedule1", Schedule: "0 0 * * *"},
			{Name: "schedule2", Schedule: "2 2 * 3 *"},
		}

		err := bsi.importBackupSchedules(testCtx, schedules)

		require.NoError(t, err)
	})

	t.Run("should fail to import backup schedules on error while creating", func(t *testing.T) {
		mockBsc := newMockBackupScheduleClient(t)
		mockBsc.EXPECT().Delete(testCtx, "schedule1", metav1.DeleteOptions{}).Return(nil)

		bs1 := &backupv1.BackupSchedule{ObjectMeta: metav1.ObjectMeta{Name: "schedule1"}, Spec: backupv1.BackupScheduleSpec{Schedule: "0 0 * * *", Provider: veleroBackupProvider}}

		mockBsc.EXPECT().Create(testCtx, bs1, metav1.CreateOptions{}).Return(bs1, assert.AnError)

		bsi := &cesBackupScheduleImporter{
			backupScheduleClient: mockBsc,
		}

		schedules := []exporter.BackupSchedule{
			{Name: "schedule1", Schedule: "0 0 * * *"},
			{Name: "schedule2", Schedule: "2 2 * 3 *"},
		}

		err := bsi.importBackupSchedules(testCtx, schedules)

		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to create backup schedule 'schedule1':")
	})
}
