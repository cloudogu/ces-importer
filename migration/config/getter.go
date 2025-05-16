package configuration

import (
	"context"
	"encoding/json"
	"fmt"
)

const pathConfiguration = "/configuration"

type exporterApiClient interface {
	// DoGetRequest allows issuing HTTP requests towards the exporter API. The result will be a byte slice that must
	// be parsed by the caller respectively.
	DoGetRequest(ctx context.Context, path string) ([]byte, error)
}

func newExporterConfigGetter(apiClient exporterApiClient) *exporterConfigGetter {
	return &exporterConfigGetter{
		apiClient: apiClient,
	}
}

type exporterConfigGetter struct {
	apiClient exporterApiClient
}

func (e exporterConfigGetter) GetConfig(ctx context.Context) (*configuration, error) {
	var config configuration
	res, err := e.apiClient.DoGetRequest(ctx, pathConfiguration)
	if err != nil {
		return nil, fmt.Errorf("error getting configuration from exporter: %w", err)
	}

	err = json.Unmarshal(res, &config)
	if err != nil {
		return nil, fmt.Errorf("could not unmarshal configuration from exporter: %w", err)
	}

	return &config, nil
}
