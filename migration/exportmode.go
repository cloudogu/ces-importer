package migration

import (
	"context"
	"fmt"
)

type exportModeClient interface {
	GetExportMode(ctx context.Context) (isActive bool, err error)
}

type ExportModeValidatorApiClient struct {
	apiClient exportModeClient
}

func NewExportModeValidatorApiClient(apiClient exportModeClient) *ExportModeValidatorApiClient {
	return &ExportModeValidatorApiClient{
		apiClient: apiClient,
	}
}

func (e *ExportModeValidatorApiClient) Validate(ctx context.Context) error {
	isActive, err := e.apiClient.GetExportMode(ctx)
	if err != nil {
		return fmt.Errorf("failed to validate export mode: %w", err)
	}

	if !isActive {
		return fmt.Errorf("export mode is not active")
	}

	return nil
}
