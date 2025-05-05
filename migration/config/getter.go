package configuration

import (
	"context"
	"encoding/json"
	"fmt"
)

type exporterApiClient interface {
	// DoGetRequest allows issuing HTTP requests towards the exporter API. The result will be a byte slice that must
	// be parsed by the caller respectively.
	DoGetRequest(ctx context.Context, url string) ([]byte, error)
}

func newExporterConfigGetter(exporterHost string, apiClient exporterApiClient) *exporterConfigGetter {
	return &exporterConfigGetter{
		exporterHost: exporterHost,
		apiClient:    apiClient,
	}
}

type exporterConfigGetter struct {
	exporterHost string
	apiClient    exporterApiClient
}

func (e exporterConfigGetter) GetConfig(ctx context.Context) (*configuration, error) {
	url := fmt.Sprintf("https://%s/configuration", e.exporterHost)

	var config configuration
	res, err := e.apiClient.DoGetRequest(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("error getting configuration from exporter: %w", err)
	}

	err = json.Unmarshal(res, &config)
	if err != nil {
		return nil, fmt.Errorf("could not unmarshal configuration from exporter: %w", err)
	}

	return &config, nil
}
