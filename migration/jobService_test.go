package migration

import (
	"bytes"
	"context"
	"github.com/cloudogu/ces-importer/configuration"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"io"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/rest"
	"log/slog"
	"testing"
)

type mockReadCloser struct {
	io.Reader
}

// Close is a no-op close method to fulfill the io.ReadCloser interface
func (m mockReadCloser) Close() error {
	return nil
}

func TestNewJobService(t *testing.T) {
	t.Run("should return new job service", func(t *testing.T) {
		sut, err := NewJobService(JobServiceDependencies{
			JobProviderDependencies: JobProviderDependencies{
				JobContainerConfig: configuration.JobContainer{
					Image: configuration.ContainerImage{
						Registry:   "registry.cloudogu.com",
						Repository: "testRepo",
						Tag:        "0.0.1",
					},
					ImagePullPolicy: "Never",
					Resources: configuration.ResourceRequirements{
						Limits: configuration.ResourceList{
							CPU:    "500m",
							Memory: "256Mi",
						},
						Requests: configuration.ResourceList{
							CPU:    "500m",
							Memory: "256Mi",
						},
					},
				},
				PVCClient: newMockPvcClient(t),
			},
			JobClient: newMockJobClient(t),
			PodClient: newMockPodClient(t),
		})

		assert.NoError(t, err)
		assert.NotNil(t, sut)
		assert.NotNil(t, sut.jobClient)
		assert.NotNil(t, sut.jobCreator)
		assert.NotNil(t, sut.getStreamer)
		assert.NotNil(t, sut.getWatcher)
	})

	t.Run("should return error when job provider cannot be created", func(t *testing.T) {
		sut, err := NewJobService(JobServiceDependencies{})

		assert.Error(t, err)
		assert.ErrorContains(t, err, "failed to create job provider")
		assert.Nil(t, sut)
	})

}

func TestJobService_Run(t *testing.T) {
	t.Run("Job Completed Successfully with logs", func(t *testing.T) {
		ctx := context.TODO()

		expLogs := "test-Log"

		job := &batchv1.Job{
			ObjectMeta: metav1.ObjectMeta{
				Name:            "test-job",
				ResourceVersion: "1",
			},
		}

		jobCreatorMock := newMockJobCreator(t)
		jobCreatorMock.EXPECT().createImportJob(ctx).Return(job, nil)

		jobClientMock := newMockJobClient(t)
		jobClientMock.EXPECT().Create(ctx, job, metav1.CreateOptions{}).Return(job, nil)

		watcherMock := watch.NewFake()

		go func() {
			watcherMock.Add(job)

			succeededJob := job.DeepCopy()
			succeededJob.Status.Succeeded = 1
			watcherMock.Modify(succeededJob)
		}()

		getWatcherMock := newMockGetWatcherFunc(t)
		getWatcherMock.EXPECT().Execute(ctx, job.Name).Return(watcherMock, nil)

		readCloserMock := mockReadCloser{Reader: bytes.NewBufferString(expLogs)}

		streamerMock := newMockStreamer(t)
		streamerMock.EXPECT().Stream(ctx).Return(readCloserMock, nil)

		getStreamerMock := newMockGetStreamerFunc(t)
		getStreamerMock.EXPECT().Execute(job.Name, &corev1.PodLogOptions{}).Return(streamerMock, nil)

		sut := &JobService{
			jobClient:   jobClientMock,
			jobCreator:  jobCreatorMock,
			getWatcher:  getWatcherMock.Execute,
			getStreamer: getStreamerMock.Execute,
		}

		logReader, err := sut.Run(ctx)

		assert.NoError(t, err)
		assert.NotNil(t, logReader)

		logs, err := io.ReadAll(logReader)
		assert.NoError(t, err)
		assert.Equal(t, expLogs, string(logs))
	})

	t.Run("fail to create job - no logs available", func(t *testing.T) {
		ctx := context.TODO()

		jobCreatorMock := newMockJobCreator(t)
		jobCreatorMock.EXPECT().createImportJob(ctx).Return(nil, assert.AnError)

		sut := &JobService{
			jobClient:   newMockJobClient(t),
			jobCreator:  jobCreatorMock,
			getWatcher:  newMockGetWatcherFunc(t).Execute,
			getStreamer: newMockGetStreamerFunc(t).Execute,
		}

		logReader, err := sut.Run(ctx)

		assert.ErrorIs(t, err, assert.AnError)
		assert.Nil(t, logReader)

	})

	t.Run("fail to save job - no logs available", func(t *testing.T) {
		ctx := context.TODO()

		job := &batchv1.Job{
			ObjectMeta: metav1.ObjectMeta{
				Name:            "test-job",
				ResourceVersion: "1",
			},
		}

		jobCreatorMock := newMockJobCreator(t)
		jobCreatorMock.EXPECT().createImportJob(ctx).Return(job, nil)

		jobClientMock := newMockJobClient(t)
		jobClientMock.EXPECT().Create(ctx, job, metav1.CreateOptions{}).Return(nil, assert.AnError)

		sut := &JobService{
			jobClient:   jobClientMock,
			jobCreator:  jobCreatorMock,
			getWatcher:  newMockGetWatcherFunc(t).Execute,
			getStreamer: newMockGetStreamerFunc(t).Execute,
		}

		logReader, err := sut.Run(ctx)

		assert.ErrorIs(t, err, assert.AnError)
		assert.Nil(t, logReader)
	})

	t.Run("fail to get watcher", func(t *testing.T) {
		ctx := context.TODO()

		expLogs := "test-Log"

		job := &batchv1.Job{
			ObjectMeta: metav1.ObjectMeta{
				Name:            "test-job",
				ResourceVersion: "1",
			},
		}

		jobCreatorMock := newMockJobCreator(t)
		jobCreatorMock.EXPECT().createImportJob(ctx).Return(job, nil)

		jobClientMock := newMockJobClient(t)
		jobClientMock.EXPECT().Create(ctx, job, metav1.CreateOptions{}).Return(job, nil)

		getWatcherMock := newMockGetWatcherFunc(t)
		getWatcherMock.EXPECT().Execute(ctx, job.Name).Return(nil, assert.AnError)

		readCloserMock := mockReadCloser{Reader: bytes.NewBufferString(expLogs)}

		streamerMock := newMockStreamer(t)
		streamerMock.EXPECT().Stream(ctx).Return(readCloserMock, nil)

		getStreamerMock := newMockGetStreamerFunc(t)
		getStreamerMock.EXPECT().Execute(job.Name, &corev1.PodLogOptions{}).Return(streamerMock, nil)

		sut := &JobService{
			jobClient:   jobClientMock,
			jobCreator:  jobCreatorMock,
			getWatcher:  getWatcherMock.Execute,
			getStreamer: getStreamerMock.Execute,
		}

		logReader, err := sut.Run(ctx)

		assert.ErrorIs(t, err, assert.AnError)
		assert.NotNil(t, logReader)

		logs, err := io.ReadAll(logReader)
		assert.NoError(t, err)
		assert.Equal(t, expLogs, string(logs))
	})

	t.Run("fail because of watch errors", func(t *testing.T) {
		ctx := context.TODO()

		job := &batchv1.Job{
			ObjectMeta: metav1.ObjectMeta{
				Name:            "test-job",
				ResourceVersion: "1",
			},
		}

		jobCreatorMock := newMockJobCreator(t)
		jobCreatorMock.EXPECT().createImportJob(ctx).Return(job, nil)

		jobClientMock := newMockJobClient(t)
		jobClientMock.EXPECT().Create(ctx, job, metav1.CreateOptions{}).Return(job, nil)

		tests := []struct {
			name   string
			event  watch.Event
			expErr string
		}{
			{
				name: "received error event from type Status",
				event: watch.Event{
					Type: watch.Error,
					Object: &metav1.Status{
						Message: "error during test",
					},
				},
				expErr: "error during test",
			},
			{
				name: "received error event with unexpected type",
				event: watch.Event{
					Type:   watch.Error,
					Object: &batchv1.Job{},
				},
				expErr: "failed to cast event object to status",
			},
			{
				name: "received unexpected event type",
				event: watch.Event{
					Type:   watch.Modified,
					Object: &corev1.Pod{},
				},
				expErr: "received unexpected event type during watch of job",
			},
			{
				name: "received job failed event",
				event: watch.Event{
					Type: watch.Modified,
					Object: &batchv1.Job{
						ObjectMeta: metav1.ObjectMeta{
							Name:            "test-job",
							ResourceVersion: "1",
						},
						Status: batchv1.JobStatus{
							Failed: 1,
						},
					},
				},
				expErr: "job test-job failed",
			},
			{
				name: "received job deleted event",
				event: watch.Event{
					Type: watch.Deleted,
					Object: &batchv1.Job{
						ObjectMeta: metav1.ObjectMeta{
							Name:            "test-job",
							ResourceVersion: "1",
						},
						Status: batchv1.JobStatus{
							Failed: 0,
						},
					},
				},
				expErr: "job has been deleted during migration",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				watcherMock := watch.NewFake()

				getWatcherMock := newMockGetWatcherFunc(t)
				getWatcherMock.EXPECT().Execute(ctx, job.Name).Return(watcherMock, nil)

				expLogs := "test-Log"

				readCloserMock := mockReadCloser{Reader: bytes.NewBufferString(expLogs)}

				streamerMock := newMockStreamer(t)
				streamerMock.EXPECT().Stream(ctx).Return(readCloserMock, nil)

				getStreamerMock := newMockGetStreamerFunc(t)
				getStreamerMock.EXPECT().Execute(job.Name, &corev1.PodLogOptions{}).Return(streamerMock, nil)

				sut := &JobService{
					jobClient:   jobClientMock,
					jobCreator:  jobCreatorMock,
					getWatcher:  getWatcherMock.Execute,
					getStreamer: getStreamerMock.Execute,
				}

				go func() {
					watcherMock.Action(tt.event.Type, tt.event.Object)
				}()

				logReader, err := sut.Run(ctx)

				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expErr)
				assert.NotNil(t, logReader)

				logs, err := io.ReadAll(logReader)
				assert.NoError(t, err)
				assert.Equal(t, expLogs, string(logs))
			})
		}
	})

	t.Run("fail to get logs", func(t *testing.T) {
		ctx := context.TODO()

		job := &batchv1.Job{
			ObjectMeta: metav1.ObjectMeta{
				Name:            "test-job",
				ResourceVersion: "1",
			},
		}

		var buf bytes.Buffer

		logger := slog.New(slog.NewJSONHandler(&buf, nil))
		slog.SetDefault(logger)

		t.Run("fail to get streamer", func(t *testing.T) {
			jobCreatorMock := newMockJobCreator(t)
			jobCreatorMock.EXPECT().createImportJob(ctx).Return(job, nil)

			jobClientMock := newMockJobClient(t)
			jobClientMock.EXPECT().Create(ctx, job, metav1.CreateOptions{}).Return(job, nil)

			watcherMock := watch.NewFake()

			go func() {
				succeededJob := job.DeepCopy()
				succeededJob.Status.Succeeded = 1
				watcherMock.Add(succeededJob)
			}()

			getWatcherMock := newMockGetWatcherFunc(t)
			getWatcherMock.EXPECT().Execute(ctx, job.Name).Return(watcherMock, nil)

			getStreamerMock := newMockGetStreamerFunc(t)
			getStreamerMock.EXPECT().Execute(job.Name, &corev1.PodLogOptions{}).Return(nil, assert.AnError)

			sut := &JobService{
				jobClient:   jobClientMock,
				jobCreator:  jobCreatorMock,
				getWatcher:  getWatcherMock.Execute,
				getStreamer: getStreamerMock.Execute,
			}

			logReader, err := sut.Run(ctx)

			assert.NoError(t, err)
			assert.Nil(t, logReader)

			assert.Contains(t, buf.String(), "failed to create request for logs for job")
		})

		t.Run("fail to stream logs", func(t *testing.T) {
			jobCreatorMock := newMockJobCreator(t)
			jobCreatorMock.EXPECT().createImportJob(ctx).Return(job, nil)

			jobClientMock := newMockJobClient(t)
			jobClientMock.EXPECT().Create(ctx, job, metav1.CreateOptions{}).Return(job, nil)

			watcherMock := watch.NewFake()

			go func() {
				succeededJob := job.DeepCopy()
				succeededJob.Status.Succeeded = 1
				watcherMock.Add(succeededJob)
			}()

			getWatcherMock := newMockGetWatcherFunc(t)
			getWatcherMock.EXPECT().Execute(ctx, job.Name).Return(watcherMock, nil)

			streamerMock := newMockStreamer(t)
			streamerMock.EXPECT().Stream(ctx).Return(nil, assert.AnError)

			getStreamerMock := newMockGetStreamerFunc(t)
			getStreamerMock.EXPECT().Execute(job.Name, &corev1.PodLogOptions{}).Return(streamerMock, nil)

			sut := &JobService{
				jobClient:   jobClientMock,
				jobCreator:  jobCreatorMock,
				getWatcher:  getWatcherMock.Execute,
				getStreamer: getStreamerMock.Execute,
			}

			logReader, err := sut.Run(ctx)

			assert.NoError(t, err)
			assert.Nil(t, logReader)

			assert.Contains(t, buf.String(), "failed to stream logs for job")
		})
	})
}

func Test_createGetStreamerFunc(t *testing.T) {
	t.Run("should create get streamer function successfully", func(t *testing.T) {
		podName := "test-pod"

		podClientMock := newMockPodClient(t)
		podClientMock.EXPECT().List(mock.Anything, mock.Anything).Return(&corev1.PodList{
			Items: []corev1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: podName,
					},
				},
			},
		}, nil)

		podClientMock.EXPECT().GetLogs(podName, mock.Anything).Return(&rest.Request{})

		getStreamer := createGetStreamerFunc(podClientMock)
		assert.NotNil(t, getStreamer)

		logReader, err := getStreamer(podName, &corev1.PodLogOptions{})
		assert.NoError(t, err)
		assert.NotNil(t, logReader)
	})

	t.Run("fail to list pods", func(t *testing.T) {
		podClientMock := newMockPodClient(t)
		podClientMock.EXPECT().List(mock.Anything, mock.Anything).Return(nil, assert.AnError)

		getStreamer := createGetStreamerFunc(podClientMock)
		assert.NotNil(t, getStreamer)

		logReader, err := getStreamer("test-pod", &corev1.PodLogOptions{})
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to list pods for job")
		assert.Nil(t, logReader)
	})

	t.Run("pods list is empty", func(t *testing.T) {
		podName := "test-pod"

		podClientMock := newMockPodClient(t)
		podClientMock.EXPECT().List(mock.Anything, mock.Anything).Return(&corev1.PodList{Items: nil}, nil)

		getStreamer := createGetStreamerFunc(podClientMock)
		assert.NotNil(t, getStreamer)

		logReader, err := getStreamer(podName, &corev1.PodLogOptions{})
		assert.Error(t, err)
		assert.ErrorContains(t, err, "no pods found for job")
		assert.Nil(t, logReader)
	})
}

func Test_createGetWatcherFunc(t *testing.T) {
	t.Run("should create get watcher function successfully", func(t *testing.T) {
		jobClientMock := newMockJobClient(t)
		jobList := &batchv1.JobList{
			TypeMeta: metav1.TypeMeta{
				Kind:       "",
				APIVersion: "1",
			},
			ListMeta: metav1.ListMeta{
				ResourceVersion:    "1",
				Continue:           "",
				RemainingItemCount: nil,
			},
			Items: []batchv1.Job{{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
				Spec:   batchv1.JobSpec{},
				Status: batchv1.JobStatus{},
			}},
		}
		jobClientMock.EXPECT().List(mock.Anything, mock.Anything).Return(jobList, nil)
		getWatcher := createGetWatcherFunc(jobClientMock)

		watcher, err := getWatcher(context.TODO(), "test")

		defer watcher.Stop()

		assert.NoError(t, err)
		assert.NotNil(t, watcher)
	})

	t.Run("fail to get watcher", func(t *testing.T) {
		jobClientMock := newMockJobClient(t)
		jobClientMock.EXPECT().List(mock.Anything, mock.Anything).Return(&batchv1.JobList{}, nil)
		getWatcher := createGetWatcherFunc(jobClientMock)

		watcher, err := getWatcher(context.TODO(), "")
		assert.Error(t, err)
		assert.Nil(t, watcher)
	})
}

func TestWatchAdapter_WatchWithContext(t *testing.T) {
	adapter := watchAdapter{watchFunc: func(ctx context.Context, options metav1.ListOptions) (watch.Interface, error) {
		return watch.NewFake(), nil
	}}

	watcher, err := adapter.WatchWithContext(context.TODO(), metav1.ListOptions{})
	defer watcher.Stop()

	assert.NoError(t, err)
	assert.NotNil(t, watcher)
}
