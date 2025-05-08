package exporter

import (
	"context"
	"io"
)

type apiClient interface {
	DoGetRequest(ctx context.Context, exporterUrl string) (result []byte, err error)
	DoPostRequest(ctx context.Context, exporterUrl string, body io.Reader, pathParams []string) (result []byte, err error)
}
