package exporter

import (
	"context"
	"encoding/json"
	"fmt"
)

type ExportDoguClient struct {
	apiClient apiClient
	endpoint  string
}

func NewExportDoguClient(apiClient apiClient, exporterHost string) *ExportDoguClient {
	return &ExportDoguClient{
		apiClient: apiClient,
		endpoint:  fmt.Sprintf("https://%s%s", exporterHost, endpointExportDogu),
	}
}

func (emc *ExportDoguClient) GetExportDogu(ctx context.Context) (export *DoguExport, err error) {
	result, err := emc.apiClient.DoGetRequest(ctx, emc.endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to check whether export mode is ready: %w", err)
	}

	var doguExport DoguExport
	err = json.Unmarshal(result, &doguExport)
	if err != nil {
		return nil, fmt.Errorf("failed to parse export mode response: %q: %w", result, err)
	}

	return &doguExport, nil
}

func (emc *ExportDoguClient) SetExportDogu(ctx context.Context, doguName string) (export *DoguExport, err error) {
	result, err := emc.apiClient.DoPostRequest(ctx, emc.endpoint, nil, []string{doguName})
	if err != nil {
		return nil, fmt.Errorf("failed to check whether export mode is ready: %w", err)
	}

	var doguExport DoguExport
	err = json.Unmarshal(result, &doguExport)
	if err != nil {
		return nil, fmt.Errorf("failed to parse export mode response: %q: %w", result, err)
	}

	return &doguExport, nil
}
