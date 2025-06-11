package exporter

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
)

const exportModeEndpoint = "/export/dogu"

type ExportDoguService struct {
	apiClient apiClient
}

func NewExportDoguService(apiClient apiClient) *ExportDoguService {
	return &ExportDoguService{
		apiClient: apiClient,
	}
}

func (emc *ExportDoguService) GetExportDogu(ctx context.Context) (export *DoguExport, err error) {
	result, err := emc.apiClient.DoGetRequest(ctx, exportModeEndpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to get export dogu: %w", err)
	}

	var doguExport DoguExport
	err = json.Unmarshal(result, &doguExport)
	if err != nil {
		return nil, fmt.Errorf("failed to parse export dogu response: %q: %w", result, err)
	}

	return &doguExport, nil
}

func (emc *ExportDoguService) SetExportDogu(ctx context.Context, doguName string) (export *DoguExport, err error) {
	apiPath := path.Join(exportModeEndpoint, doguName)
	result, err := emc.apiClient.DoPostRequest(ctx, apiPath, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to set export dogu: %w", err)
	}

	var doguExport DoguExport
	err = json.Unmarshal(result, &doguExport)
	if err != nil {
		return nil, fmt.Errorf("failed to parse export dogu response: %q: %w", result, err)
	}

	return &doguExport, nil
}
