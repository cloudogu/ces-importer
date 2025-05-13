package main

import (
	"context"
	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/cloudogu/ces-importer/api/exporter"
)

type exporterApiClient interface {
	// DoGetRequest allows issuing HTTP requests towards the exporter API. The result will be a byte slice that must
	// be parsed by the caller respectively.
	DoGetRequest(ctx context.Context, url string) ([]byte, error)
}

// doguStopper provides functions to stop a running dogu.
type doguStopper interface {
	// StopDogu stopps the given dogu in the importer system. An error is expected if the dogu is in a non-healthy
	// condition except the dogu is already stopped.
	StopDogu(ctx context.Context, dogu exporter.Dogu) error
}

// doguStarter provides functions to start a stopped dogu.
type doguStarter interface {
	// StartDogu starts the given dogu in the importer system. An error is expected if the dogu is in a non-healthy
	// condition except when the dogu is stopped.
	StartDogu(ctx context.Context, dogu exporter.Dogu) error
}

type doguVolumeSyncer interface {
	// SyncDogu starts copying the volume data of a single dogu as provided by systemInfo.
	SyncDogu(ctx context.Context, port, source, destination string) error
}

type jobProvider interface {
	// CreateImportJob creates a kubernetes job for the synchronization of the data.
	CreateImportJob(ctx context.Context) (*batchv1.Job, error)
}

type jobClient interface {
	Create(ctx context.Context, job *batchv1.Job, opts metav1.CreateOptions) (*batchv1.Job, error)
}
