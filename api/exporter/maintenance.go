package exporter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/cloudogu/ces-importer/migration"
)

const (
	// EndpointMaintenanceMode contains the endpoint which returns data which describe the current
	EndpointMaintenanceMode = "/maintenance/mode"
)

var _ migration.MaintenanceModeHandler = MaintenanceModeService{}

// MaintenanceMode contains data of the current maintenance state
type MaintenanceMode struct {
	Activate bool    `json:"activate"`
	Message  Message `json:"message"`
}

// Message contains the title and text of the maintenance message
type Message struct {
	Title string `json:"title"`
	Text  string `json:"text"`
}

// MaintenanceModeStatus returns just the current state of the maintenance mode on exporter side
type MaintenanceModeStatus struct {
	IsActive bool `json:"isActive"`
}

type MaintenanceModeService struct {
	apiClient
	serviceURL string
}

// NewMaintenanceModeService creates a new maintenance service for the given exporter API client.
func NewMaintenanceModeService(baseURL string, client apiClient) *MaintenanceModeService {
	return &MaintenanceModeService{
		serviceURL: baseURL + EndpointMaintenanceMode,
		apiClient:  client,
	}
}

// GetMaintenanceModeStatus returns the current maintenance mode status of the exporter system.
func (s MaintenanceModeService) GetMaintenanceModeStatus(ctx context.Context) (bool, error) {
	result, err := s.DoGetRequest(ctx, s.serviceURL)
	if err != nil {
		return false, fmt.Errorf("failed to get maintenance mode status: %w", err)
	}

	response, err := decodeMaintenanceModeStatus(result)
	if err != nil {
		return false, fmt.Errorf("failed to decode maintenance mode status: %w", err)
	}

	return response.IsActive, nil
}

// Enable enables the maintenance mode of the exporter system with the given title and message.
func (s MaintenanceModeService) Enable(ctx context.Context, title, message string) error {
	return s.setMaintenanceMode(ctx, MaintenanceMode{Activate: true, Message: Message{Title: title, Text: message}})
}

// Disable disables the maintenance mode of the exporter system.
func (s MaintenanceModeService) Disable(ctx context.Context) error {
	return s.setMaintenanceMode(ctx, MaintenanceMode{Activate: false})
}

func (s MaintenanceModeService) setMaintenanceMode(ctx context.Context, mode MaintenanceMode) error {
	var buf bytes.Buffer

	if err := json.NewEncoder(&buf).Encode(mode); err != nil {
		return fmt.Errorf("failed to encode maintenance mode: %w", err)
	}

	result, err := s.DoPostRequest(ctx, s.serviceURL, &buf, nil)
	if err != nil {
		return fmt.Errorf("failed to set maintenance mode: %w", err)
	}

	response, err := decodeMaintenanceModeStatus(result)
	if err != nil {
		return fmt.Errorf("failed to decode maintenance mode status: %w", err)
	}

	if response.IsActive != mode.Activate {
		return fmt.Errorf("received unexpected mode status in response: %t", response.IsActive)
	}

	return nil
}

func decodeMaintenanceModeStatus(result []byte) (MaintenanceModeStatus, error) {
	var status MaintenanceModeStatus
	if err := json.Unmarshal(result, &status); err != nil {
		return MaintenanceModeStatus{}, fmt.Errorf("failed to decode maintenance mode status: %w", err)
	}

	return status, nil
}
