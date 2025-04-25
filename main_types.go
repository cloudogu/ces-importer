package main

import (
	"context"

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
