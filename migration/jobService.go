package migration

import (
	"context"
	"fmt"
	"io"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	watchAPI "k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/watch"
	"log/slog"
)

var _ JobRunner = JobService{}

type getStreamerFunc func(jobName string, options *corev1.PodLogOptions) (streamer, error)

type getWatcherFunc func(resourceVersion string) (watchAPI.Interface, error)

type JobServiceDependencies struct {
	jobProviderDependencies
	JobClient jobClient
	PodClient podClient
}

type JobService struct {
	jobClient
	jobCreator
	getWatcher  getWatcherFunc
	getStreamer getStreamerFunc
}

func NewJobService(deps JobServiceDependencies) (*JobService, error) {
	provider, err := newJobProvider(deps.jobProviderDependencies)
	if err != nil {
		return nil, fmt.Errorf("failed to create job provider: %w", err)
	}

	return &JobService{
		jobClient:   deps.JobClient,
		jobCreator:  provider,
		getWatcher:  createGetWatcherFunc(deps.JobClient),
		getStreamer: createGetStreamerFunc(deps.PodClient),
	}, nil
}

func createGetStreamerFunc(podClient podClient) getStreamerFunc {
	return func(jobName string, options *corev1.PodLogOptions) (streamer, error) {
		jobLabelSelector := &metav1.LabelSelector{
			MatchLabels: map[string]string{
				batchv1.JobNameLabel: jobName,
			},
		}

		pods, err := podClient.List(context.Background(), metav1.ListOptions{LabelSelector: metav1.FormatLabelSelector(jobLabelSelector)})
		if err != nil {
			return nil, fmt.Errorf("failed to list pods for job %s: %w", jobName, err)
		}

		if len(pods.Items) == 0 {
			return nil, fmt.Errorf("no pods found for job %s", jobName)
		}

		podName := pods.Items[0].GetName()

		return podClient.GetLogs(podName, options), nil
	}
}

func createGetWatcherFunc(jobClient jobClient) getWatcherFunc {
	return func(resourceVersion string) (watchAPI.Interface, error) {
		return watch.NewRetryWatcher(resourceVersion, jobClient)
	}
}

func (j JobService) Run(ctx context.Context) (jobLogs io.ReadCloser, err error) {
	job, err := j.createImportJob(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create import job: %w", err)
	}

	jobResource, err := j.jobClient.Create(ctx, job, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to create import job resource: %w", err)
	}

	defer func() {
		logs, logErr := j.getLogs(jobResource.GetName())
		if logErr != nil {
			slog.Error("Failed to get logs for job", "name", jobResource.GetName(), "error", logErr)
			return
		}

		jobLogs = logs
	}()

	watcher, err := j.getWatcher(jobResource.GetResourceVersion())
	if err != nil {
		return nil, fmt.Errorf("failed to create watcher for job %s: %w", jobResource.GetName(), err)
	}

	defer watcher.Stop()

	var errWatch error

	for event := range watcher.ResultChan() {
		if event.Type == watchAPI.Error {
			errWatch = handleWatchError(event)
			break
		}

		jobChange, ok := event.Object.(*batchv1.Job)
		if !ok {
			errWatch = fmt.Errorf("received unexpected event type during watch of job: %T", event.Object)
			break
		}

		if jobChange.Status.Succeeded > 0 {
			slog.Info("Job completed successfully", "name", jobChange.GetName())
			break
		}

		if jobChange.Status.Failed > 0 {
			errWatch = fmt.Errorf("job %s failed", jobChange.GetName())
			break
		}
	}

	if errWatch != nil {
		return nil, fmt.Errorf("received error while watching job: %w", errWatch)
	}

	return nil, nil
}

func (j JobService) getLogs(jobName string) (io.ReadCloser, error) {
	logStreamer, err := j.getStreamer(jobName, &corev1.PodLogOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to create request for logs for job %s: %w", jobName, err)
	}

	logs, err := logStreamer.Stream(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to stream logs for job %s: %w", jobName, err)
	}

	return logs, nil
}

func handleWatchError(event watchAPI.Event) (err error) {
	status, ok := event.Object.(*metav1.Status)
	if !ok {
		return fmt.Errorf("failed to cast event object to status: %T", event.Object)
	}

	return fmt.Errorf(status.Message)
}
