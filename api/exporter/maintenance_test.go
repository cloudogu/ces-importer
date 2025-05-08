package exporter

import (
	"context"
	"errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
)

func Test_newMaintenanceService(t *testing.T) {
	// given
	apiClientMock := newMockApiClient(t)
	baseURL := "http://example.com/api"
	expectedServiceURL := "http://example.com/api/maintenance/mode"

	// when
	service := newMaintenanceService(baseURL, apiClientMock)

	// then
	assert.NotNil(t, service)
	assert.Equal(t, expectedServiceURL, service.serviceURL)
	assert.Equal(t, apiClientMock, service.apiClient)
}

func Test_maintenanceService_GetMaintenanceModeStatus(t *testing.T) {
	tests := []struct {
		name           string
		responseBody   []byte
		responseErr    error
		expectedStatus MaintenanceModeStatus
		expectedErr    bool
		expectedErrMsg string
	}{
		{
			name:           "should return maintenance mode status when request succeeds",
			responseBody:   []byte(`{"isActive": true}`),
			responseErr:    nil,
			expectedStatus: MaintenanceModeStatus{IsActive: true},
			expectedErr:    false,
		},
		{
			name:           "should return error when request fails",
			responseBody:   nil,
			responseErr:    errors.New("request failed"),
			expectedStatus: MaintenanceModeStatus{},
			expectedErr:    true,
			expectedErrMsg: "failed to get maintenance mode status: request failed",
		},
		{
			name:           "should return error when response cannot be decoded",
			responseBody:   []byte(`invalid json`),
			responseErr:    nil,
			expectedStatus: MaintenanceModeStatus{},
			expectedErr:    true,
			expectedErrMsg: "failed to decode maintenance mode status",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given
			apiClientMock := newMockApiClient(t)
			serviceURL := "http://example.com/api/maintenance/mode"

			ctx := context.Background()
			apiClientMock.EXPECT().
				DoGetRequest(ctx, serviceURL).
				Return(tt.responseBody, tt.responseErr)

			service := maintenanceService{
				apiClient:  apiClientMock,
				serviceURL: serviceURL,
			}

			// when
			status, err := service.GetMaintenanceModeStatus(ctx)

			// then
			if tt.expectedErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErrMsg)
				assert.Equal(t, tt.expectedStatus, status)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedStatus, status)
			}
		})
	}
}

func Test_maintenanceService_EnableMaintenanceMode(t *testing.T) {
	tests := []struct {
		name           string
		title          string
		message        string
		responseBody   []byte
		responseErr    error
		expectedStatus MaintenanceModeStatus
		expectedErr    bool
		expectedErrMsg string
	}{
		{
			name:           "should enable maintenance mode when request succeeds",
			title:          "Maintenance",
			message:        "System is under maintenance",
			responseBody:   []byte(`{"isActive": true}`),
			responseErr:    nil,
			expectedStatus: MaintenanceModeStatus{IsActive: true},
			expectedErr:    false,
		},
		{
			name:           "should return error when request fails",
			title:          "Maintenance",
			message:        "System is under maintenance",
			responseBody:   nil,
			responseErr:    errors.New("request failed"),
			expectedStatus: MaintenanceModeStatus{},
			expectedErr:    true,
			expectedErrMsg: "failed to set maintenance mode: request failed",
		},
		{
			name:           "should return error when response cannot be decoded",
			title:          "Maintenance",
			message:        "System is under maintenance",
			responseBody:   []byte(`invalid json`),
			responseErr:    nil,
			expectedStatus: MaintenanceModeStatus{},
			expectedErr:    true,
			expectedErrMsg: "failed to decode maintenance mode status",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given
			apiClientMock := newMockApiClient(t)
			serviceURL := "http://example.com/api/maintenance/mode"

			ctx := context.Background()
			// Verify that the correct maintenance mode is sent
			apiClientMock.EXPECT().
				DoPostRequest(ctx, serviceURL, mock.Anything, mock.Anything).
				Return(tt.responseBody, tt.responseErr)

			service := maintenanceService{
				apiClient:  apiClientMock,
				serviceURL: serviceURL,
			}

			// when
			status, err := service.EnableMaintenanceMode(ctx, tt.title, tt.message)

			// then
			if tt.expectedErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErrMsg)
				assert.Equal(t, tt.expectedStatus, status)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedStatus, status)
			}
		})
	}
}

func Test_maintenanceService_DisableMaintenanceMode(t *testing.T) {
	tests := []struct {
		name           string
		responseBody   []byte
		responseErr    error
		expectedStatus MaintenanceModeStatus
		expectedErr    bool
		expectedErrMsg string
	}{
		{
			name:           "should disable maintenance mode when request succeeds",
			responseBody:   []byte(`{"isActive": false}`),
			responseErr:    nil,
			expectedStatus: MaintenanceModeStatus{IsActive: false},
			expectedErr:    false,
		},
		{
			name:           "should return error when request fails",
			responseBody:   nil,
			responseErr:    errors.New("request failed"),
			expectedStatus: MaintenanceModeStatus{},
			expectedErr:    true,
			expectedErrMsg: "failed to set maintenance mode: request failed",
		},
		{
			name:           "should return error when response cannot be decoded",
			responseBody:   []byte(`invalid json`),
			responseErr:    nil,
			expectedStatus: MaintenanceModeStatus{},
			expectedErr:    true,
			expectedErrMsg: "failed to decode maintenance mode status",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given
			apiClientMock := newMockApiClient(t)
			serviceURL := "http://example.com/api/maintenance/mode"

			ctx := context.Background()
			// Verify that the correct maintenance mode is sent
			apiClientMock.EXPECT().
				DoPostRequest(ctx, serviceURL, mock.Anything, mock.Anything).
				Return(tt.responseBody, tt.responseErr)

			service := maintenanceService{
				apiClient:  apiClientMock,
				serviceURL: serviceURL,
			}

			// when
			status, err := service.DisableMaintenanceMode(ctx)

			// then
			if tt.expectedErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErrMsg)
				assert.Equal(t, tt.expectedStatus, status)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedStatus, status)
			}
		})
	}
}

func Test_decodeMaintenanceModeStatus(t *testing.T) {
	tests := []struct {
		name           string
		input          []byte
		expectedStatus MaintenanceModeStatus
		expectedErr    bool
		expectedErrMsg string
	}{
		{
			name:           "should decode valid JSON with active status",
			input:          []byte(`{"isActive": true}`),
			expectedStatus: MaintenanceModeStatus{IsActive: true},
			expectedErr:    false,
		},
		{
			name:           "should decode valid JSON with inactive status",
			input:          []byte(`{"isActive": false}`),
			expectedStatus: MaintenanceModeStatus{IsActive: false},
			expectedErr:    false,
		},
		{
			name:           "should return error for invalid JSON",
			input:          []byte(`invalid json`),
			expectedStatus: MaintenanceModeStatus{},
			expectedErr:    true,
			expectedErrMsg: "failed to decode maintenance mode status",
		},
		{
			name:           "should handle JSON with invalid control characters in string",
			input:          []byte(`{"isActive": true, "invalidString": "control chars: \u0000\u0001\u0002"}`),
			expectedStatus: MaintenanceModeStatus{IsActive: true},
			expectedErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// when
			status, err := decodeMaintenanceModeStatus(tt.input)

			// then
			if tt.expectedErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErrMsg)
				assert.Equal(t, tt.expectedStatus, status)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedStatus, status)
			}
		})
	}
}
