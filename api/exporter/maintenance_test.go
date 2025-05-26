package exporter

import (
	"context"
	"errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
)

func Test_newMaintenanceModeService(t *testing.T) {
	// given
	apiClientMock := newMockApiClient(t)

	// when
	service := NewMaintenanceModeService(apiClientMock)

	// then
	assert.NotNil(t, service)
	assert.Equal(t, apiClientMock, service.apiClient)
}

func Test_maintenanceModeService_GetStatus(t *testing.T) {
	tests := []struct {
		name           string
		responseBody   []byte
		responseErr    error
		expectedStatus bool
		expectedErr    bool
		expectedErrMsg string
	}{
		{
			name:           "should return maintenance mode status when request succeeds",
			responseBody:   []byte(`{"isActive": true}`),
			responseErr:    nil,
			expectedStatus: true,
			expectedErr:    false,
		},
		{
			name:           "should return error when request fails",
			responseBody:   nil,
			responseErr:    errors.New("request failed"),
			expectedErr:    true,
			expectedErrMsg: "failed to get maintenance mode status: request failed",
		},
		{
			name:           "should return error when response cannot be decoded",
			responseBody:   []byte(`invalid json`),
			responseErr:    nil,
			expectedErr:    true,
			expectedErrMsg: "failed to decode maintenance mode status",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given
			apiClientMock := newMockApiClient(t)

			ctx := context.Background()
			apiClientMock.EXPECT().
				DoGetRequest(ctx, endpointMaintenanceMode).
				Return(tt.responseBody, tt.responseErr)

			service := MaintenanceModeService{
				apiClient: apiClientMock,
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

func Test_maintenanceModeService_Enable(t *testing.T) {
	const title = "Maintenance"
	const message = "System is under maintenance"

	tests := []struct {
		name           string
		responseBody   []byte
		responseErr    error
		expectedErr    bool
		expectedErrMsg string
	}{
		{
			name:         "should enable maintenance mode when request succeeds",
			responseBody: []byte(`{"isActive": true}`),
			responseErr:  nil,
			expectedErr:  false,
		},
		{
			name:           "should return error when response status does not match request status",
			responseBody:   []byte(`{"isActive": false}`),
			responseErr:    nil,
			expectedErr:    true,
			expectedErrMsg: "received unexpected mode status in response",
		},
		{
			name:           "should return error when request fails",
			responseBody:   nil,
			responseErr:    errors.New("request failed"),
			expectedErr:    true,
			expectedErrMsg: "failed to set maintenance mode: request failed",
		},
		{
			name:           "should return error when response cannot be decoded",
			responseBody:   []byte(`invalid json`),
			responseErr:    nil,
			expectedErr:    true,
			expectedErrMsg: "failed to decode maintenance mode status",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given
			apiClientMock := newMockApiClient(t)

			ctx := context.Background()
			// Verify that the correct maintenance mode is sent
			apiClientMock.EXPECT().
				DoPostRequest(ctx, endpointMaintenanceMode, mock.Anything).
				Return(tt.responseBody, tt.responseErr)

			service := MaintenanceModeService{
				apiClient: apiClientMock,
			}

			// when
			err := service.Enable(ctx, title, message)

			// then
			if tt.expectedErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErrMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func Test_maintenanceModeService_Disable(t *testing.T) {
	tests := []struct {
		name           string
		responseBody   []byte
		responseErr    error
		expectedErr    bool
		expectedErrMsg string
	}{
		{
			name:         "should disable maintenance mode when request succeeds",
			responseBody: []byte(`{"isActive": false}`),
			responseErr:  nil,
			expectedErr:  false,
		},
		{
			name:           "should return error when response status does not match request status",
			responseBody:   []byte(`{"isActive": true}`),
			responseErr:    nil,
			expectedErr:    true,
			expectedErrMsg: "received unexpected mode status in response",
		},
		{
			name:           "should return error when request fails",
			responseBody:   nil,
			responseErr:    errors.New("request failed"),
			expectedErr:    true,
			expectedErrMsg: "failed to set maintenance mode: request failed",
		},
		{
			name:           "should return error when response cannot be decoded",
			responseBody:   []byte(`invalid json`),
			responseErr:    nil,
			expectedErr:    true,
			expectedErrMsg: "failed to decode maintenance mode status",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given
			apiClientMock := newMockApiClient(t)

			ctx := context.Background()
			// Verify that the correct maintenance mode is sent
			apiClientMock.EXPECT().
				DoPostRequest(ctx, endpointMaintenanceMode, mock.Anything).
				Return(tt.responseBody, tt.responseErr)

			service := MaintenanceModeService{
				apiClient: apiClientMock,
			}

			// when
			err := service.Disable(ctx)

			// then
			if tt.expectedErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErrMsg)
			} else {
				assert.NoError(t, err)
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
