package exporter

import (
	"context"
	"encoding/json"
	"fmt"
)

// pathExportMode contains the endpoint which returns data on the readiness of the exporter system.
const pathExportMode = "/export/mode"

type ExportModeService struct {
	apiClient apiClient
}

func NewExportModeService(apiClient apiClient) *ExportModeService {
	return &ExportModeService{
		apiClient: apiClient,
	}
}

func (emc *ExportModeService) GetExportMode(ctx context.Context) (isActive bool, err error) {
	result, err := emc.apiClient.DoGetRequest(ctx, pathExportMode)
	if err != nil {
		return false, fmt.Errorf("failed to check whether export mode is ready: %w", err)
	}

	var exportMode ExportMode
	err = json.Unmarshal(result, &exportMode)
	if err != nil {
		return false, fmt.Errorf("failed to parse export mode response: %q: %w", result, err)
	}

	return exportMode.IsActive, nil
}
