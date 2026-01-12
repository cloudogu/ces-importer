package manual

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/cloudogu/ces-importer/api/importer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
)

func TestStartManualMigrationConfigmapWatcher(t *testing.T) {
	t.Run("should stop on context cancellation", func(t *testing.T) {
		// given
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // cancel immediately

		mockConfigMapClient := newMockConfigmapClient(t)

		mockMigrator := newMockMigrationRunner(t)
		migrationRunning := &atomic.Bool{}

		// when
		err := StartManualMigrationConfigmapWatcher(ctx, mockConfigMapClient, "test-namespace", mockMigrator, migrationRunning)

		// then
		assert.ErrorIs(t, err, context.Canceled)
	})

	t.Run("should return error when watcher creation fails", func(t *testing.T) {
		// given
		ctx := context.Background()
		mockConfigMapClient := newMockConfigmapClient(t)
		mockConfigMapClient.EXPECT().Watch(ctx, metav1.ListOptions{
			LabelSelector: labelSelector,
		}).Return(nil, fmt.Errorf("watch creation failed"))

		mockMigrator := newMockMigrationRunner(t)
		migrationRunning := &atomic.Bool{}

		// when
		err := StartManualMigrationConfigmapWatcher(ctx, mockConfigMapClient, "test-namespace", mockMigrator, migrationRunning)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create manual migration starter ConfigMap watcher")
	})

	t.Run("should handle watch events successfully and reconnect", func(t *testing.T) {
		// given
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		mockWatcher1 := newMockWatcher(t)
		resultChan1 := make(chan watch.Event, 1)
		mockWatcher1.EXPECT().ResultChan().Return(resultChan1)
		mockWatcher1.EXPECT().Stop().Return()

		mockWatcher2 := newMockWatcher(t)
		resultChan2 := make(chan watch.Event)
		mockWatcher2.EXPECT().ResultChan().Return(resultChan2)
		mockWatcher2.EXPECT().Stop().Return()

		mockConfigMapClient := newMockConfigmapClient(t)
		mockConfigMapClient.EXPECT().Watch(ctx, metav1.ListOptions{
			LabelSelector: labelSelector,
		}).Return(mockWatcher1, nil).Once()

		mockConfigMapClient.EXPECT().Watch(ctx, metav1.ListOptions{
			LabelSelector: labelSelector,
		}).Return(mockWatcher2, nil).Once()

		mockConfigMapClient.EXPECT().Delete(ctx, "test-configmap", metav1.DeleteOptions{}).Return(nil)

		mockMigrator := newMockMigrationRunner(t)
		mockMigrator.EXPECT().RunMigration(ctx).Return(nil)
		migrationRunning := &atomic.Bool{}

		// send event and close first channel, then cancel context after reconnect
		go func() {
			time.Sleep(100 * time.Millisecond)
			resultChan1 <- watch.Event{
				Type: watch.Added,
				Object: &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-configmap",
					},
				},
			}
			close(resultChan1)

			// Wait for reconnection and then cancel context
			time.Sleep(1500 * time.Millisecond) // wait longer than the 1 second sleep in manual.go
			cancel()
		}()

		// when
		err := StartManualMigrationConfigmapWatcher(ctx, mockConfigMapClient, "test-namespace", mockMigrator, migrationRunning)

		// then
		assert.ErrorIs(t, err, context.Canceled)
	})
}

func TestHandleWatchEvents(t *testing.T) {
	t.Run("should stop on context cancellation", func(t *testing.T) {
		// given
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // cancel immediately

		mockWatcher := newMockWatcher(t)
		mockWatcher.EXPECT().Stop().Return()
		mockWatcher.EXPECT().ResultChan().Return(make(chan watch.Event))

		mockMigrator := newMockMigrationRunner(t)
		migrationRunning := &atomic.Bool{}
		clientset := importer.K8sClients{}

		// when
		err := handleWatchEvents(ctx, mockWatcher, mockMigrator, migrationRunning, clientset.ConfigMap)

		// then
		assert.ErrorIs(t, err, context.Canceled)
	})

	t.Run("should return error when watch channel closes", func(t *testing.T) {
		// given
		ctx := context.Background()
		mockWatcher := newMockWatcher(t)
		resultChan := make(chan watch.Event)
		close(resultChan) // close channel immediately
		mockWatcher.EXPECT().ResultChan().Return(resultChan)
		mockWatcher.EXPECT().Stop().Return()

		mockMigrator := newMockMigrationRunner(t)
		migrationRunning := &atomic.Bool{}
		clientset := importer.K8sClients{}

		// when
		err := handleWatchEvents(ctx, mockWatcher, mockMigrator, migrationRunning, clientset.ConfigMap)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "watch channel closed")
	})

	t.Run("should handle Added event successfully", func(t *testing.T) {
		// given
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		mockWatcher := newMockWatcher(t)
		resultChan := make(chan watch.Event, 1)
		mockWatcher.EXPECT().ResultChan().Return(resultChan)
		mockWatcher.EXPECT().Stop().Return()

		mockConfigMapClient := newMockConfigmapClient(t)
		mockConfigMapClient.EXPECT().Delete(ctx, "test-configmap", metav1.DeleteOptions{}).Return(nil)

		mockMigrator := newMockMigrationRunner(t)
		mockMigrator.EXPECT().RunMigration(ctx).Return(nil)
		migrationRunning := &atomic.Bool{}

		// send event and close channel
		go func() {
			time.Sleep(100 * time.Millisecond)
			resultChan <- watch.Event{
				Type: watch.Added,
				Object: &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-configmap",
					},
				},
			}
			close(resultChan)
		}()

		// when
		err := handleWatchEvents(ctx, mockWatcher, mockMigrator, migrationRunning, mockConfigMapClient)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "watch channel closed")
	})

	t.Run("should ignore non-Added events", func(t *testing.T) {
		// given
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		mockWatcher := newMockWatcher(t)
		resultChan := make(chan watch.Event, 1)
		mockWatcher.EXPECT().ResultChan().Return(resultChan)
		mockWatcher.EXPECT().Stop().Return()

		mockMigrator := newMockMigrationRunner(t)
		migrationRunning := &atomic.Bool{}
		clientset := importer.K8sClients{}

		// send modified event and close channel
		go func() {
			time.Sleep(100 * time.Millisecond)
			resultChan <- watch.Event{
				Type: watch.Modified,
				Object: &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-configmap",
					},
				},
			}
			close(resultChan)
		}()

		// when
		err := handleWatchEvents(ctx, mockWatcher, mockMigrator, migrationRunning, clientset.ConfigMap)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "watch channel closed")
	})
}

func TestStartManualMigration(t *testing.T) {
	t.Run("should start migration successfully", func(t *testing.T) {
		// given
		ctx := context.Background()
		configMap := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-configmap",
			},
		}
		event := watch.Event{
			Type:   watch.Added,
			Object: configMap,
		}

		mockConfigMapClient := newMockConfigmapClient(t)
		mockConfigMapClient.EXPECT().Delete(ctx, "test-configmap", metav1.DeleteOptions{}).Return(nil)

		mockMigrator := newMockMigrationRunner(t)
		mockMigrator.EXPECT().RunMigration(ctx).Return(nil)

		migrationRunning := &atomic.Bool{}

		// when
		startManualMigration(ctx, event, mockConfigMapClient, migrationRunning, mockMigrator)

		// then
		// assertions are handled by mock expectations
	})

	t.Run("should skip migration when already running", func(t *testing.T) {
		// given
		ctx := context.Background()
		configMap := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-configmap",
			},
		}
		event := watch.Event{
			Type:   watch.Added,
			Object: configMap,
		}

		mockConfigMapClient := newMockConfigmapClient(t)
		mockConfigMapClient.EXPECT().Delete(ctx, "test-configmap", metav1.DeleteOptions{}).Return(nil)

		mockMigrator := newMockMigrationRunner(t)
		// RunMigration should NOT be called

		migrationRunning := &atomic.Bool{}
		migrationRunning.Store(true) // migration already running

		// when
		startManualMigration(ctx, event, mockConfigMapClient, migrationRunning, mockMigrator)

		// then
		// assertions are handled by mock expectations (RunMigration not called)
	})

	t.Run("should handle configmap deletion error gracefully", func(t *testing.T) {
		// given
		ctx := context.Background()
		configMap := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-configmap",
			},
		}
		event := watch.Event{
			Type:   watch.Added,
			Object: configMap,
		}

		mockConfigMapClient := newMockConfigmapClient(t)
		mockConfigMapClient.EXPECT().Delete(ctx, "test-configmap", metav1.DeleteOptions{}).Return(fmt.Errorf("deletion failed"))

		mockMigrator := newMockMigrationRunner(t)
		mockMigrator.EXPECT().RunMigration(ctx).Return(nil)

		migrationRunning := &atomic.Bool{}

		// when
		startManualMigration(ctx, event, mockConfigMapClient, migrationRunning, mockMigrator)

		// then
		// should continue despite deletion error
	})

	t.Run("should handle migration error gracefully", func(t *testing.T) {
		// given
		ctx := context.Background()
		configMap := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-configmap",
			},
		}
		event := watch.Event{
			Type:   watch.Added,
			Object: configMap,
		}

		mockConfigMapClient := newMockConfigmapClient(t)
		mockConfigMapClient.EXPECT().Delete(ctx, "test-configmap", metav1.DeleteOptions{}).Return(nil)

		mockMigrator := newMockMigrationRunner(t)
		mockMigrator.EXPECT().RunMigration(ctx).Return(assert.AnError)
		migrationRunning := &atomic.Bool{}

		// when
		startManualMigration(ctx, event, mockConfigMapClient, migrationRunning, mockMigrator)

		// then
		// should handle error gracefully
	})

	t.Run("should handle invalid event object", func(t *testing.T) {
		// given
		ctx := context.Background()
		event := watch.Event{
			Type:   watch.Added,
			Object: &corev1.Pod{}, // wrong type
		}

		mockMigrator := newMockMigrationRunner(t)
		clients := importer.K8sClients{}
		migrationRunning := &atomic.Bool{}

		// when
		startManualMigration(ctx, event, clients.ConfigMap, migrationRunning, mockMigrator)

		// then
		// should return early without error
	})
}
