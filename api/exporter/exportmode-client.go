package exporter

import (
	"context"
	"encoding/json"
	"fmt"
)

type ExportModeClient struct {
	apiClient apiClient
	endpoint  string
}

func NewExportModeClient(apiClient apiClient, exporterHost string) *ExportModeClient {
	return &ExportModeClient{
		apiClient: apiClient,
		endpoint:  fmt.Sprintf("https://%s%s", exporterHost, endpointExportMode),
	}
}

func (emc *ExportModeClient) GetExportMode(ctx context.Context) (isActive bool, err error) {
	result, err := emc.apiClient.DoGetRequest(ctx, emc.endpoint)
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
