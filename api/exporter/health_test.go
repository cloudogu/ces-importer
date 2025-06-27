package exporter

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestNewHealthClient(t *testing.T) {
	tests := []struct {
		name          string
		apiClient     *Client
		expectedError bool
	}{
		{
			name:          "Valid inputs",
			apiClient:     &Client{},
			expectedError: false,
		},
		{
			name:          "Nil apiClient",
			apiClient:     nil,
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewHealthService(tt.apiClient)

			if client == nil {
				t.Fatalf("Expected non-nil SystemInfoService, got nil")
			}

			if client.apiClient != tt.apiClient {
				t.Errorf("Expected apiClient: %+v, got: %+v", tt.apiClient, client.apiClient)
			}
		})
	}
}

func TestHealthClient_GetIsHealthy(t *testing.T) {

	t.Run("should get system info successfully", func(t *testing.T) {
		mApiClient := newMockApiClient(t)
		mApiClient.EXPECT().DoGetRequest(testCtx, "/health").Return([]byte(""), nil)

		emc := &HealthService{
			apiClient: mApiClient,
		}

		isHealthy, err := emc.GetIsHealthy(testCtx)

		require.NoError(t, err)
		require.True(t, isHealthy)
	})

	t.Run("should fail get system info for error in request", func(t *testing.T) {
		mApiClient := newMockApiClient(t)
		mApiClient.EXPECT().DoGetRequest(testCtx, "/health").Return(nil, assert.AnError)

		emc := &HealthService{
			apiClient: mApiClient,
		}

		_, err := emc.GetIsHealthy(testCtx)

		require.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to get exporter health status")
	})

}
