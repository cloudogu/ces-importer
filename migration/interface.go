package migration

import (
	"context"
	"io"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	restclient "k8s.io/client-go/rest"
)

type pvcClient interface {
	GetDoguVolumes(ctx context.Context) ([]doguPVC, error)
}

type jobClient interface {
	Create(ctx context.Context, job *batchv1.Job, opts metav1.CreateOptions) (*batchv1.Job, error)
	Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error)
	List(ctx context.Context, opts metav1.ListOptions) (*batchv1.JobList, error)
}

type jobCreator interface {
	createImportJob(ctx context.Context) (*batchv1.Job, error)
}

type streamer interface {
	Stream(ctx context.Context) (io.ReadCloser, error)
}

type podLister interface {
	List(ctx context.Context, opts metav1.ListOptions) (*corev1.PodList, error)
}

type podLogGetter interface {
	GetLogs(name string, opts *corev1.PodLogOptions) *restclient.Request
}

type podClient interface {
	podLister
	podLogGetter
}
