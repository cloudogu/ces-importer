package exporter

import (
	"context"
	"fmt"
)

// pathHealth contains the endpoint which returns the current health status of the exporting system
const pathHealth = "/health"

type HealthService struct {
	apiClient apiClient
}

func NewHealthService(apiClient apiClient) *HealthService {
	return &HealthService{
		apiClient: apiClient,
	}
}

func (emc *HealthService) GetIsHealthy(ctx context.Context) (bool, error) {
	_, err := emc.apiClient.DoGetRequest(ctx, pathHealth)
	if err != nil {
		return false, fmt.Errorf("failed to get exporter health status: %w", err)
	}

	return true, nil
}
