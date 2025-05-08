package exporter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
)

const (
	// EndpointMaintenanceMode contains the endpoint which returns data which describe the current
	EndpointMaintenanceMode = "/maintenance/mode"
)

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

type maintenanceService struct {
	apiClient
	serviceURL string
}

// newMaintenanceService creates a new maintenance service for the given exporter API client.
func newMaintenanceService(baseURL string, client apiClient) *maintenanceService {
	return &maintenanceService{
		serviceURL: baseURL + EndpointMaintenanceMode,
		apiClient:  client,
	}
}

// GetMaintenanceModeStatus returns the current maintenance mode status of the exporter system.
func (s maintenanceService) GetMaintenanceModeStatus(ctx context.Context) (MaintenanceModeStatus, error) {
	result, err := s.DoGetRequest(ctx, s.serviceURL)
	if err != nil {
		return MaintenanceModeStatus{}, fmt.Errorf("failed to get maintenance mode status: %w", err)
	}

	return decodeMaintenanceModeStatus(result)
}

// EnableMaintenanceMode enables the maintenance mode of the exporter system with the given title and message.
func (s maintenanceService) EnableMaintenanceMode(ctx context.Context, title, message string) (MaintenanceModeStatus, error) {
	return s.setMaintenanceMode(ctx, MaintenanceMode{Activate: true, Message: Message{Title: title, Text: message}})
}

// DisableMaintenanceMode disables the maintenance mode of the exporter system.
func (s maintenanceService) DisableMaintenanceMode(ctx context.Context) (MaintenanceModeStatus, error) {
	return s.setMaintenanceMode(ctx, MaintenanceMode{Activate: false})
}

func (s maintenanceService) setMaintenanceMode(ctx context.Context, mode MaintenanceMode) (MaintenanceModeStatus, error) {
	var buf bytes.Buffer

	if err := json.NewEncoder(&buf).Encode(mode); err != nil {
		return MaintenanceModeStatus{}, fmt.Errorf("failed to encode maintenance mode: %w", err)
	}

	result, err := s.DoPostRequest(ctx, s.serviceURL, &buf, nil)
	if err != nil {
		return MaintenanceModeStatus{}, fmt.Errorf("failed to set maintenance mode: %w", err)
	}

	return decodeMaintenanceModeStatus(result)
}

func decodeMaintenanceModeStatus(result []byte) (MaintenanceModeStatus, error) {
	var status MaintenanceModeStatus
	if err := json.Unmarshal(result, &status); err != nil {
		return MaintenanceModeStatus{}, fmt.Errorf("failed to decode maintenance mode status: %w", err)
	}

	return status, nil
}
