package exporter

import (
	"context"
	"encoding/json"
	"fmt"
)

type SystemInfoClient struct {
	apiClient *client
	endpoint  string
}

func NewSystemInfoClient(apiClient *client, exporterHost string) *SystemInfoClient {
	return &SystemInfoClient{
		apiClient: apiClient,
		endpoint:  fmt.Sprintf("https://%s/%s", exporterHost, endpointSystemInfo),
	}
}

func (emc *SystemInfoClient) GetSystemInfo(ctx context.Context) (systemInfo *SystemInfo, err error) {
	result, err := emc.apiClient.DoGetRequest(ctx, emc.endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to get system info: %w", err)
	}

	err = json.Unmarshal(result, systemInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to parse system info response: %q: %w", result, err)
	}

	return systemInfo, nil
}
