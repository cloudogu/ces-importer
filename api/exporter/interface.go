package exporter

import (
	"context"
	"io"
	"net/http"
)

type requestExecuter interface {
	// Do executes the given HTTP request.
	Do(req *http.Request) (*http.Response, error)
}

type apiClient interface {
	// DoGetRequest creates an HTTP GET request towards the exporter API. Any unexpected HTTP codes (other than 200 OK) or
	// errors will be returned as an error. For authentication, request headers will automatically be enriched with the
	// provided API key.
	DoGetRequest(ctx context.Context, path string) (result []byte, err error)
	DoPostRequest(ctx context.Context, path string, body io.Reader) (result []byte, err error)
}
