package exporter

import (
	"context"
	"encoding/json"
	"fmt"
)

// pathSystemInfo contains the endpoint which returns data which describe the exporter system, f. i. installed dogus etc.
const pathSystemInfo = "/system-info"

type SystemInfoService struct {
	apiClient apiClient
}

func NewSystemInfoService(apiClient apiClient) *SystemInfoService {
	return &SystemInfoService{
		apiClient: apiClient,
	}
}

func (emc *SystemInfoService) GetSystemInfo(ctx context.Context) (systemInfo *SystemInfo, err error) {
	result, err := emc.apiClient.DoGetRequest(ctx, pathSystemInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to get system info: %w", err)
	}

	systemInfo = &SystemInfo{}
	err = json.Unmarshal(result, systemInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to parse system info response: %q: %w", result, err)
	}

	return systemInfo, nil
}
