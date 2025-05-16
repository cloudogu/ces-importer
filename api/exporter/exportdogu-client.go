package exporter

import (
	"context"
	"encoding/json"
	"fmt"
	path2 "path"
)

const exportModeEndpoint = "/export/dogu"

type ExportDoguClient struct {
	apiClient apiClient
}

func NewExportDoguClient(apiClient apiClient) *ExportDoguClient {
	return &ExportDoguClient{
		apiClient: apiClient,
	}
}

func (emc *ExportDoguClient) GetExportDogu(ctx context.Context) (export *DoguExport, err error) {
	result, err := emc.apiClient.DoGetRequest(ctx, exportModeEndpoint)
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
	path := path2.Join(exportModeEndpoint, doguName)
	result, err := emc.apiClient.DoPostRequest(ctx, path, nil)
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
