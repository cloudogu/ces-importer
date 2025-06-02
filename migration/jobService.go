// Package migration provides functionality for data migration operations in Kubernetes.
// This package includes components for creating, running, and monitoring Kubernetes jobs
// that perform data import and migration tasks.
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

// Static assertion to ensure JobService implements the JobRunner interface
var _ JobRunner = JobService{}

// getStreamerFunc is a function type that retrieves a log streamer for a specific job pod
// It takes a job name and pod log options and returns a streamer for accessing pod logs
type getStreamerFunc func(jobName string, options *corev1.PodLogOptions) (streamer, error)

// getWatcherFunc is a function type that creates a watcher for monitoring job status changes
// It takes a context and resource version and returns a watch interface for receiving events
type getWatcherFunc func(ctx context.Context, resourceVersion, jobName string) (watchAPI.Interface, error)

// JobServiceDependencies contains all the dependencies required to create a JobService
// It includes dependencies for job creation, job client for interacting with Kubernetes jobs,
// and pod client for accessing pod information and logs
type JobServiceDependencies struct {
	JobProviderDependencies
	JobClient jobClient
	PodClient podClient
}

// JobService orchestrates the creation, execution, and monitoring of Kubernetes jobs
// It provides functionality to run jobs, watch their status, and retrieve their logs
type JobService struct {
	jobClient                   // For interacting with Kubernetes jobs API
	jobCreator                  // For creating job specifications
	getWatcher  getWatcherFunc  // Function to create job status watchers
	getStreamer getStreamerFunc // Function to retrieve log streamers
}

// watchAdapter adapts the jobClient.Watch function to implement the watch.Interface
// This allows the use of RetryWatcher with the jobClient
type watchAdapter struct {
	watchFunc func(ctx context.Context, opts metav1.ListOptions) (watchAPI.Interface, error)
}

// WatchWithContext implements the watch.Interface.WatchWithContext method
// It delegates to the wrapped watchFunc, allowing the adapter to be used with RetryWatcher
func (w watchAdapter) WatchWithContext(ctx context.Context, options metav1.ListOptions) (watchAPI.Interface, error) {
	return w.watchFunc(ctx, options)
}

// NewJobService creates a new JobService with the provided dependencies
// It initializes the job provider and sets up the necessary functions for watching jobs and streaming logs
// Returns an error if the job provider cannot be created
func NewJobService(deps JobServiceDependencies) (*JobService, error) {
	provider, err := newJobProvider(deps.JobProviderDependencies)
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

// createGetStreamerFunc creates a function that can retrieve log streamers for job pods
// It returns a getStreamerFunc that:
// 1. Finds pods associated with a job using label selectors
// 2. Gets the first pod's name (assuming one pod per job)
// 3. Returns a log streamer for that pod
func createGetStreamerFunc(podClient podClient) getStreamerFunc {
	return func(jobName string, options *corev1.PodLogOptions) (streamer, error) {
		// List all pods with the job label
		pods, err := podClient.List(context.Background(), metav1.ListOptions{LabelSelector: buildJobLabelSelector(jobName)})
		if err != nil {
			return nil, fmt.Errorf("failed to list pods for job %s: %w", jobName, err)
		}

		slog.Debug("Listed pods associated with job", "name", jobName, "length", len(pods.Items))

		// Ensure at least one pod was found
		if len(pods.Items) == 0 {
			return nil, fmt.Errorf("no pods found for job %s", jobName)
		}

		// Get the name of the first pod (jobs typically have one pod)
		podName := pods.Items[0].GetName()

		slog.Debug("Found pod for job", "name", jobName, "pod name", podName)

		// Return a log streamer for the pod
		return podClient.GetLogs(podName, options), nil
	}
}

// createGetWatcherFunc creates a function that can create watchers for monitoring job status
// It returns a getWatcherFunc that creates a RetryWatcher, which automatically reconnects
// if the watch connection is lost
func createGetWatcherFunc(jobClient jobClient) getWatcherFunc {
	return func(ctx context.Context, resourceVersion, jobName string) (watchAPI.Interface, error) {
		// Create an adapter to make jobClient.Watch compatible with RetryWatcher
		wrapper := watchAdapter{
			watchFunc: func(ctx context.Context, opts metav1.ListOptions) (watchAPI.Interface, error) {
				opts.LabelSelector = buildJobLabelSelector(jobName)
				return jobClient.Watch(ctx, opts)
			},
		}

		// Create a RetryWatcher that will automatically reconnect if the watch connection is lost
		return watch.NewRetryWatcherWithContext(ctx, resourceVersion, wrapper)
	}
}

// Create a label selector to find resources belonging to the specified job
func buildJobLabelSelector(jobName string) string {
	jobLabelSelector := &metav1.LabelSelector{
		MatchLabels: map[string]string{
			batchv1.JobNameLabel: jobName,
		},
	}
	return metav1.FormatLabelSelector(jobLabelSelector)
}

// Run creates and executes a Kubernetes job, watches for its completion, and returns its logs
// It performs the following steps:
// 1. Creates a job specification using the job creator
// 2. Submits the job to the Kubernetes API
// 3. Sets up a watcher to monitor the job's status
// 4. Waits for the job to complete or fail
// 5. Retrieves and returns the job's logs
//
// The method returns an io.ReadCloser containing the job logs if successful,
// or an error if any step in the process fails
func (j JobService) Run(ctx context.Context) (jobLogs io.ReadCloser, err error) {
	// Create the job specification
	job, err := j.createImportJob(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create import job: %w", err)
	}

	slog.Info("Created import job", "name", job.GetName())

	// Submit the job to Kubernetes
	jobResource, err := j.jobClient.Create(ctx, job, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to create import job resource: %w", err)
	}

	slog.Info("Submitted import job", "name", jobResource.GetName(), "resource version", jobResource.GetResourceVersion())

	// Set up deferred function to retrieve logs when the method returns
	// This ensures we attempt to get logs regardless of whether the job succeeds or fails
	defer func() {
		logs, logErr := j.getLogs(ctx, jobResource.GetName())
		if logErr != nil {
			slog.Error("Failed to get logs for job", "name", jobResource.GetName(), "error", logErr)
			return
		}

		slog.Info("Retrieved logs for job", "name", jobResource.GetName())

		jobLogs = logs
	}()

	// Create a watcher to monitor the job's status
	watcher, err := j.getWatcher(ctx, jobResource.GetResourceVersion(), jobResource.GetName())
	if err != nil {
		return nil, fmt.Errorf("failed to create watcher for job %s: %w", jobResource.GetName(), err)
	}

	slog.Debug("Got watcher for job", "name", jobResource.GetName())

	// Ensure the watcher is stopped when we're done
	defer watcher.Stop()

	slog.Debug("Starting to wait for job to complete or fail")

	// Process events from the watcher until the job completes, fails, or an error occurs
	errWatch := watchEvents(watcher.ResultChan(), jobResource.GetName())
	if errWatch != nil {
		return nil, fmt.Errorf("received error while watching job: %w", errWatch)
	}

	// Return nil values because the actual logs are set by the deferred function
	return nil, nil
}

// watchEvents processes events from a watcher until the job completes, fails, or an error occurs.
// It logs the event type and returns an error if the job fails, or an error occurs during processing
func watchEvents(resultChan <-chan watchAPI.Event, jobName string) (errWatch error) {
	for event := range resultChan {
		slog.Debug("Received event from watcher for import job", "type", event.Type)

		// Handle watch errors
		if event.Type == watchAPI.Error {
			errWatch = handleWatchError(event)
			break
		}

		// Ensure the event object is a Job
		jobChange, ok := event.Object.(*batchv1.Job)
		if !ok {
			errWatch = fmt.Errorf("received unexpected event type during watch of job: %T", event.Object)
			break
		}

		// ignore events from other jobs
		// this is just a failsafe and should never happen
		if !(jobChange.GetName() == jobName) {
			break
		}

		// Check if the job has succeeded
		if jobChange.Status.Succeeded > 0 {
			slog.Info("Job completed successfully", "name", jobChange.GetName())
			break
		}

		// Check if the job has failed
		if jobChange.Status.Failed > 0 {
			errWatch = fmt.Errorf("job %s failed", jobChange.GetName())
			break
		}
	}

	return errWatch
}

// getLogs retrieves the logs from a job's pod
// It uses the getStreamer function to find the pod associated with the job
// and create a log streamer, then returns the log stream
func (j JobService) getLogs(ctx context.Context, jobName string) (io.ReadCloser, error) {
	slog.Debug("Starting to get logs for job", "name", jobName)

	// Get a log streamer for the job's pod
	logStreamer, err := j.getStreamer(jobName, &corev1.PodLogOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to create request for logs for job %s: %w", jobName, err)
	}

	slog.Debug("Got log streamer for job", "name", jobName)

	// Start streaming the logs
	logs, err := logStreamer.Stream(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to stream logs for job %s: %w", jobName, err)
	}

	return logs, nil
}

// handleWatchError extracts error information from a watch error event
// It attempts to cast the event object to a Status type to get the error message
// Returns a formatted error containing the status message
func handleWatchError(event watchAPI.Event) (err error) {
	// Try to cast the event object to a Status type
	status, ok := event.Object.(*metav1.Status)
	if !ok {
		return fmt.Errorf("failed to cast event object to status: %T", event.Object)
	}

	// Return the status message as an error
	return fmt.Errorf("%s", status.Message)
}
