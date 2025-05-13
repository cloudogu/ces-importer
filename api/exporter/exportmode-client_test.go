package exporter

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestNewExportModeClient(t *testing.T) {
	tests := []struct {
		name             string
		apiClient        *client
		exporterHost     string
		expectedError    bool
		expectedEndpoint string
	}{
		{
			name:             "Valid inputs",
			apiClient:        &client{},
			exporterHost:     "example.com",
			expectedError:    false,
			expectedEndpoint: "https://example.com/export/mode",
		},
		{
			name:             "Empty exporterHost",
			apiClient:        &client{},
			exporterHost:     "",
			expectedError:    false,
			expectedEndpoint: "https:///export/mode",
		},
		{
			name:             "Nil apiClient",
			apiClient:        nil,
			exporterHost:     "example.com",
			expectedError:    false,
			expectedEndpoint: "https://example.com/export/mode",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewExportModeClient(tt.apiClient, tt.exporterHost)

			if client == nil {
				t.Fatalf("Expected non-nil ExportModeClient, got nil")
			}

			if client.apiClient != tt.apiClient {
				t.Errorf("Expected apiClient: %+v, got: %+v", tt.apiClient, client.apiClient)
			}

			if client.endpoint != tt.expectedEndpoint {
				t.Errorf("Expected endpoint: %s, got: %s", tt.expectedEndpoint, client.endpoint)
			}
		})
	}
}

func TestExportModeClient_GetExportMode(t *testing.T) {
	t.Run("should get export mode successfully", func(t *testing.T) {
		mApiClient := newMockApiClient(t)
		mApiClient.EXPECT().DoGetRequest(testCtx, "https://example.com/export/mode").Return([]byte(`{"isActive": true}`), nil)

		emc := &ExportModeClient{
			apiClient: mApiClient,
			endpoint:  "https://example.com/export/mode",
		}

		isActive, err := emc.GetExportMode(testCtx)

		require.NoError(t, err)
		assert.True(t, isActive)
	})

	t.Run("should fail get export mode for error in request", func(t *testing.T) {
		mApiClient := newMockApiClient(t)
		mApiClient.EXPECT().DoGetRequest(testCtx, "https://example.com/export/mode").Return(nil, assert.AnError)

		emc := &ExportModeClient{
			apiClient: mApiClient,
			endpoint:  "https://example.com/export/mode",
		}

		_, err := emc.GetExportMode(testCtx)

		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to check whether export mode is ready:")
	})

	t.Run("should fail get export mode for error while parsing response", func(t *testing.T) {
		mApiClient := newMockApiClient(t)
		mApiClient.EXPECT().DoGetRequest(testCtx, "https://example.com/export/mode").Return([]byte(`this is no json`), nil)

		emc := &ExportModeClient{
			apiClient: mApiClient,
			endpoint:  "https://example.com/export/mode",
		}

		_, err := emc.GetExportMode(testCtx)

		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to parse export mode response:")
	})
}
