package configuration

import (
	"context"
	"testing"
	"time"

	"github.com/cloudogu/ces-importer/migration"
	backupv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
)

var notFoundError = apierrors.NewNotFound(
	schema.GroupResource{
		Group:    "",
		Resource: "configmaps",
	},
	"notfound",
)

func Test_cesBackupScheduleImporter_importBackupSchedules(t *testing.T) {
	testCtx := context.Background()
	t.Run("should import backup schedules", func(t *testing.T) {
		mockWatcherSchedule1 := watch.NewFake()
		mockWatcherSchedule2 := watch.NewFake()

		go func() {
			mockWatcherSchedule1.Delete(&backupv1.BackupSchedule{ObjectMeta: metav1.ObjectMeta{Name: "schedule1"}})
			mockWatcherSchedule2.Delete(&backupv1.BackupSchedule{ObjectMeta: metav1.ObjectMeta{Name: "schedule2"}})
		}()

		mockBsc := newMockBackupScheduleClient(t)
		mockBsc.EXPECT().Watch(testCtx, metav1.SingleObject(metav1.ObjectMeta{Name: "schedule1"})).Return(mockWatcherSchedule1, nil)
		mockBsc.EXPECT().Watch(testCtx, metav1.SingleObject(metav1.ObjectMeta{Name: "schedule2"})).Return(mockWatcherSchedule2, nil)
		mockBsc.EXPECT().Delete(mock.Anything, "schedule1", metav1.DeleteOptions{}).Return(nil)
		mockBsc.EXPECT().Delete(mock.Anything, "schedule2", metav1.DeleteOptions{}).Return(nil)
		mockBsc.EXPECT().Get(mock.Anything, "schedule1", metav1.GetOptions{}).Return(nil, notFoundError)
		mockBsc.EXPECT().Get(mock.Anything, "schedule2", metav1.GetOptions{}).Return(nil, notFoundError)

		bs1 := &backupv1.BackupSchedule{ObjectMeta: metav1.ObjectMeta{Name: "schedule1"}, Spec: backupv1.BackupScheduleSpec{Schedule: "0 0 * * *", Provider: veleroBackupProvider}}
		bs2 := &backupv1.BackupSchedule{ObjectMeta: metav1.ObjectMeta{Name: "schedule2"}, Spec: backupv1.BackupScheduleSpec{Schedule: "2 2 * 3 *", Provider: veleroBackupProvider}}

		mockBsc.EXPECT().Create(testCtx, bs1, metav1.CreateOptions{}).Return(bs1, nil)
		mockBsc.EXPECT().Create(testCtx, bs2, metav1.CreateOptions{}).Return(bs2, nil)

		bsi := &cesBackupScheduleImporter{
			backupScheduleClient: mockBsc,
		}

		schedules := []migration.BackupSchedule{
			{Name: "schedule1", Schedule: "0 0 * * *"},
			{Name: "schedule2", Schedule: "2 2 * 3 *"},
		}

		err := bsi.importBackupSchedules(testCtx, schedules)

		require.NoError(t, err)
	})

	t.Run("should continue to import backup schedules if deletion of previous fails", func(t *testing.T) {
		mockWatcherSchedule1 := watch.NewFake()
		mockWatcherSchedule2 := watch.NewFake()

		go func() {
			mockWatcherSchedule2.Delete(&backupv1.BackupSchedule{ObjectMeta: metav1.ObjectMeta{Name: "schedule2"}})
		}()

		mockBsc := newMockBackupScheduleClient(t)
		mockBsc.EXPECT().Watch(testCtx, metav1.SingleObject(metav1.ObjectMeta{Name: "schedule1"})).Return(mockWatcherSchedule1, nil)
		mockBsc.EXPECT().Watch(testCtx, metav1.SingleObject(metav1.ObjectMeta{Name: "schedule2"})).Return(mockWatcherSchedule2, nil)
		mockBsc.EXPECT().Delete(mock.Anything, "schedule1", metav1.DeleteOptions{}).Return(assert.AnError)
		mockBsc.EXPECT().Delete(mock.Anything, "schedule2", metav1.DeleteOptions{}).Return(nil)
		mockBsc.EXPECT().Get(mock.Anything, "schedule2", metav1.GetOptions{}).Return(nil, notFoundError)

		bs1 := &backupv1.BackupSchedule{ObjectMeta: metav1.ObjectMeta{Name: "schedule1"}, Spec: backupv1.BackupScheduleSpec{Schedule: "0 0 * * *", Provider: veleroBackupProvider}}
		bs2 := &backupv1.BackupSchedule{ObjectMeta: metav1.ObjectMeta{Name: "schedule2"}, Spec: backupv1.BackupScheduleSpec{Schedule: "2 2 * 3 *", Provider: veleroBackupProvider}}

		mockBsc.EXPECT().Create(testCtx, bs2, metav1.CreateOptions{}).Return(bs2, nil)

		bsi := &cesBackupScheduleImporter{
			backupScheduleClient: mockBsc,
		}

		schedules := []migration.BackupSchedule{
			{Name: "schedule1", Schedule: "0 0 * * *"},
			{Name: "schedule2", Schedule: "2 2 * 3 *"},
		}

		err := bsi.importBackupSchedules(testCtx, schedules)

		assert.NoError(t, err)
		mockBsc.AssertNotCalled(t, "Create", testCtx, bs1, metav1.CreateOptions{})
	})

	t.Run("should fail to import backup schedules on error while creating", func(t *testing.T) {
		mockWatcherSchedule1 := watch.NewFake()

		go func() {
			mockWatcherSchedule1.Delete(&backupv1.BackupSchedule{ObjectMeta: metav1.ObjectMeta{Name: "schedule1"}})
		}()

		mockBsc := newMockBackupScheduleClient(t)
		mockBsc.EXPECT().Watch(testCtx, metav1.SingleObject(metav1.ObjectMeta{Name: "schedule1"})).Return(mockWatcherSchedule1, nil)
		mockBsc.EXPECT().Delete(mock.Anything, "schedule1", metav1.DeleteOptions{}).Return(nil)
		mockBsc.EXPECT().Get(mock.Anything, "schedule1", metav1.GetOptions{}).Return(nil, notFoundError)

		bs1 := &backupv1.BackupSchedule{ObjectMeta: metav1.ObjectMeta{Name: "schedule1"}, Spec: backupv1.BackupScheduleSpec{Schedule: "0 0 * * *", Provider: veleroBackupProvider}}
		bs2 := &backupv1.BackupSchedule{ObjectMeta: metav1.ObjectMeta{Name: "schedule2"}, Spec: backupv1.BackupScheduleSpec{Schedule: "2 2 * 3 *", Provider: veleroBackupProvider}}

		mockBsc.EXPECT().Create(testCtx, bs1, metav1.CreateOptions{}).Return(nil, assert.AnError)

		bsi := &cesBackupScheduleImporter{
			backupScheduleClient: mockBsc,
		}

		schedules := []migration.BackupSchedule{
			{Name: "schedule1", Schedule: "0 0 * * *"},
			{Name: "schedule2", Schedule: "2 2 * 3 *"},
		}

		err := bsi.importBackupSchedules(testCtx, schedules)
		require.Error(t, err)

		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to create backup schedule")

		mockBsc.AssertNotCalled(t, "Create", testCtx, bs2, metav1.CreateOptions{})
	})

	t.Run("watcher returns Not found error", func(t *testing.T) {
		mockWatcherSchedule2 := watch.NewFake()

		go func() {
			mockWatcherSchedule2.Delete(&backupv1.BackupSchedule{ObjectMeta: metav1.ObjectMeta{Name: "schedule2"}})
		}()

		mockBsc := newMockBackupScheduleClient(t)
		mockBsc.EXPECT().Watch(testCtx, metav1.SingleObject(metav1.ObjectMeta{Name: "schedule1"})).Return(nil, errors.NewNotFound(schema.GroupResource{}, "schedule1"))
		mockBsc.EXPECT().Watch(testCtx, metav1.SingleObject(metav1.ObjectMeta{Name: "schedule2"})).Return(mockWatcherSchedule2, nil)
		mockBsc.EXPECT().Delete(mock.Anything, "schedule2", metav1.DeleteOptions{}).Return(nil)
		mockBsc.EXPECT().Get(mock.Anything, "schedule2", metav1.GetOptions{}).Return(nil, notFoundError)

		bs1 := &backupv1.BackupSchedule{ObjectMeta: metav1.ObjectMeta{Name: "schedule1"}, Spec: backupv1.BackupScheduleSpec{Schedule: "0 0 * * *", Provider: veleroBackupProvider}}
		bs2 := &backupv1.BackupSchedule{ObjectMeta: metav1.ObjectMeta{Name: "schedule2"}, Spec: backupv1.BackupScheduleSpec{Schedule: "2 2 * 3 *", Provider: veleroBackupProvider}}

		mockBsc.EXPECT().Create(testCtx, bs1, metav1.CreateOptions{}).Return(bs1, nil)
		mockBsc.EXPECT().Create(testCtx, bs2, metav1.CreateOptions{}).Return(bs2, nil)

		bsi := &cesBackupScheduleImporter{
			backupScheduleClient: mockBsc,
		}

		schedules := []migration.BackupSchedule{
			{Name: "schedule1", Schedule: "0 0 * * *"},
			{Name: "schedule2", Schedule: "2 2 * 3 *"},
		}

		err := bsi.importBackupSchedules(testCtx, schedules)
		assert.NoError(t, err)
		mockBsc.AssertNotCalled(t, "Delete", mock.Anything, "schedule1", metav1.DeleteOptions{})
	})

	t.Run("watcher returns error", func(t *testing.T) {
		mockWatcherSchedule2 := watch.NewFake()

		go func() {
			mockWatcherSchedule2.Delete(&backupv1.BackupSchedule{ObjectMeta: metav1.ObjectMeta{Name: "schedule2"}})
		}()

		mockBsc := newMockBackupScheduleClient(t)
		mockBsc.EXPECT().Watch(testCtx, metav1.SingleObject(metav1.ObjectMeta{Name: "schedule1"})).Return(nil, assert.AnError)
		mockBsc.EXPECT().Watch(testCtx, metav1.SingleObject(metav1.ObjectMeta{Name: "schedule2"})).Return(mockWatcherSchedule2, nil)
		mockBsc.EXPECT().Delete(mock.Anything, "schedule2", metav1.DeleteOptions{}).Return(nil)
		mockBsc.EXPECT().Get(mock.Anything, "schedule2", metav1.GetOptions{}).Return(nil, notFoundError)

		bs1 := &backupv1.BackupSchedule{ObjectMeta: metav1.ObjectMeta{Name: "schedule1"}, Spec: backupv1.BackupScheduleSpec{Schedule: "0 0 * * *", Provider: veleroBackupProvider}}
		bs2 := &backupv1.BackupSchedule{ObjectMeta: metav1.ObjectMeta{Name: "schedule2"}, Spec: backupv1.BackupScheduleSpec{Schedule: "2 2 * 3 *", Provider: veleroBackupProvider}}

		mockBsc.EXPECT().Create(testCtx, bs2, metav1.CreateOptions{}).Return(bs2, nil)

		bsi := &cesBackupScheduleImporter{
			backupScheduleClient: mockBsc,
		}

		schedules := []migration.BackupSchedule{
			{Name: "schedule1", Schedule: "0 0 * * *"},
			{Name: "schedule2", Schedule: "2 2 * 3 *"},
		}

		err := bsi.importBackupSchedules(testCtx, schedules)
		assert.NoError(t, err)
		mockBsc.AssertNotCalled(t, "Delete", mock.Anything, "schedule1", metav1.DeleteOptions{})
		mockBsc.AssertNotCalled(t, "Create", mock.Anything, bs1, metav1.CreateOptions{})
	})

	t.Run("client returns Not found error on delete", func(t *testing.T) {
		mockWatcherSchedule1 := watch.NewFake()
		mockWatcherSchedule2 := watch.NewFake()

		go func() {
			mockWatcherSchedule2.Delete(&backupv1.BackupSchedule{ObjectMeta: metav1.ObjectMeta{Name: "schedule2"}})
		}()

		mockBsc := newMockBackupScheduleClient(t)
		mockBsc.EXPECT().Watch(testCtx, metav1.SingleObject(metav1.ObjectMeta{Name: "schedule1"})).Return(mockWatcherSchedule1, nil)
		mockBsc.EXPECT().Watch(testCtx, metav1.SingleObject(metav1.ObjectMeta{Name: "schedule2"})).Return(mockWatcherSchedule2, nil)
		mockBsc.EXPECT().Delete(mock.Anything, "schedule1", metav1.DeleteOptions{}).Return(errors.NewNotFound(schema.GroupResource{}, "schedule1"))
		mockBsc.EXPECT().Delete(mock.Anything, "schedule2", metav1.DeleteOptions{}).Return(nil)
		mockBsc.EXPECT().Get(mock.Anything, "schedule2", metav1.GetOptions{}).Return(nil, notFoundError)

		bs1 := &backupv1.BackupSchedule{ObjectMeta: metav1.ObjectMeta{Name: "schedule1"}, Spec: backupv1.BackupScheduleSpec{Schedule: "0 0 * * *", Provider: veleroBackupProvider}}
		bs2 := &backupv1.BackupSchedule{ObjectMeta: metav1.ObjectMeta{Name: "schedule2"}, Spec: backupv1.BackupScheduleSpec{Schedule: "2 2 * 3 *", Provider: veleroBackupProvider}}

		mockBsc.EXPECT().Create(testCtx, bs1, metav1.CreateOptions{}).Return(bs1, nil)
		mockBsc.EXPECT().Create(testCtx, bs2, metav1.CreateOptions{}).Return(bs2, nil)

		bsi := &cesBackupScheduleImporter{
			backupScheduleClient: mockBsc,
		}

		schedules := []migration.BackupSchedule{
			{Name: "schedule1", Schedule: "0 0 * * *"},
			{Name: "schedule2", Schedule: "2 2 * 3 *"},
		}

		err := bsi.importBackupSchedules(testCtx, schedules)
		assert.NoError(t, err)
	})

	t.Run("watcher timeout while waiting for delete event", func(t *testing.T) {
		oldTimoeut := watchTimeout
		defer func() {
			watchTimeout = oldTimoeut
		}()

		watchTimeout = 0 * time.Second

		mockWatcherSchedule1 := watch.NewFake()

		mockBsc := newMockBackupScheduleClient(t)
		mockBsc.EXPECT().Watch(testCtx, metav1.SingleObject(metav1.ObjectMeta{Name: "schedule1"})).Return(mockWatcherSchedule1, nil)
		mockBsc.EXPECT().Delete(mock.Anything, "schedule1", metav1.DeleteOptions{}).Return(nil)
		mockBsc.EXPECT().Get(mock.Anything, "schedule1", metav1.GetOptions{}).Return(nil, notFoundError)

		bs1 := &backupv1.BackupSchedule{ObjectMeta: metav1.ObjectMeta{Name: "schedule1"}, Spec: backupv1.BackupScheduleSpec{Schedule: "0 0 * * *", Provider: veleroBackupProvider}}

		bsi := &cesBackupScheduleImporter{
			backupScheduleClient: mockBsc,
		}

		schedules := []migration.BackupSchedule{
			{Name: "schedule1", Schedule: "0 0 * * *"},
		}

		err := bsi.importBackupSchedules(testCtx, schedules)
		assert.NoError(t, err)

		mockBsc.AssertNotCalled(t, "Create", testCtx, bs1, metav1.CreateOptions{})
	})

	t.Run("watcher receives wrong event object or wrong backup schedule name", func(t *testing.T) {
		mockWatcherSchedule1 := watch.NewFake()
		mockWatcherSchedule2 := watch.NewFake()

		go func() {
			mockWatcherSchedule1.Delete(&v1.ConfigMap{})
			mockWatcherSchedule1.Delete(&backupv1.BackupSchedule{ObjectMeta: metav1.ObjectMeta{Name: "invalid"}})
			mockWatcherSchedule1.Delete(&backupv1.BackupSchedule{ObjectMeta: metav1.ObjectMeta{Name: "schedule1"}})
			mockWatcherSchedule2.Delete(&backupv1.BackupSchedule{ObjectMeta: metav1.ObjectMeta{Name: "schedule2"}})
		}()

		mockBsc := newMockBackupScheduleClient(t)
		mockBsc.EXPECT().Watch(testCtx, metav1.SingleObject(metav1.ObjectMeta{Name: "schedule1"})).Return(mockWatcherSchedule1, nil)
		mockBsc.EXPECT().Watch(testCtx, metav1.SingleObject(metav1.ObjectMeta{Name: "schedule2"})).Return(mockWatcherSchedule2, nil)
		mockBsc.EXPECT().Delete(mock.Anything, "schedule1", metav1.DeleteOptions{}).Return(nil)
		mockBsc.EXPECT().Delete(mock.Anything, "schedule2", metav1.DeleteOptions{}).Return(nil)
		mockBsc.EXPECT().Get(mock.Anything, "schedule1", metav1.GetOptions{}).Return(nil, notFoundError)
		mockBsc.EXPECT().Get(mock.Anything, "schedule2", metav1.GetOptions{}).Return(nil, notFoundError)

		bs1 := &backupv1.BackupSchedule{ObjectMeta: metav1.ObjectMeta{Name: "schedule1"}, Spec: backupv1.BackupScheduleSpec{Schedule: "0 0 * * *", Provider: veleroBackupProvider}}
		bs2 := &backupv1.BackupSchedule{ObjectMeta: metav1.ObjectMeta{Name: "schedule2"}, Spec: backupv1.BackupScheduleSpec{Schedule: "2 2 * 3 *", Provider: veleroBackupProvider}}

		mockBsc.EXPECT().Create(testCtx, bs1, metav1.CreateOptions{}).Return(bs1, nil)
		mockBsc.EXPECT().Create(testCtx, bs2, metav1.CreateOptions{}).Return(bs2, nil)

		bsi := &cesBackupScheduleImporter{
			backupScheduleClient: mockBsc,
		}

		schedules := []migration.BackupSchedule{
			{Name: "schedule1", Schedule: "0 0 * * *"},
			{Name: "schedule2", Schedule: "2 2 * 3 *"},
		}

		err := bsi.importBackupSchedules(testCtx, schedules)

		require.NoError(t, err)
	})
}
