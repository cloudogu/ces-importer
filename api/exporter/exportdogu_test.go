package exporter

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestNewExportDoguClient(t *testing.T) {
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
			client := NewExportDoguService(tt.apiClient)

			if client == nil {
				t.Fatalf("Expected non-nil ExportModeService, got nil")
			}

			if client.apiClient != tt.apiClient {
				t.Errorf("Expected apiClient: %+v, got: %+v", tt.apiClient, client.apiClient)
			}
		})
	}
}

func TestExportDoguClient_GetExportDogu(t *testing.T) {
	t.Run("should get export dogu successfully", func(t *testing.T) {
		mApiClient := newMockApiClient(t)
		exportDogu := DoguExport{
			Dogu:         "test",
			VolumePath:   "a/b",
			ExporterPort: 3,
		}
		exportDoguBytes, err := json.Marshal(exportDogu)
		if err != nil {
			panic(err)
		}
		mApiClient.EXPECT().DoGetRequest(testCtx, "/export/dogu").Return(exportDoguBytes, nil)

		emc := &ExportDoguService{
			apiClient: mApiClient,
		}

		result, err := emc.GetExportDogu(testCtx)

		require.NoError(t, err)
		assert.Equal(t, exportDogu, *result)
	})

	t.Run("should fail to get export dogu because of an error in the request", func(t *testing.T) {
		mApiClient := newMockApiClient(t)
		mApiClient.EXPECT().DoGetRequest(testCtx, "/export/dogu").Return(nil, assert.AnError)

		emc := &ExportDoguService{
			apiClient: mApiClient,
		}

		_, err := emc.GetExportDogu(testCtx)

		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to get export dogu")
	})

	t.Run("should fail get export mode with an error while parsing response", func(t *testing.T) {
		mApiClient := newMockApiClient(t)
		mApiClient.EXPECT().DoGetRequest(testCtx, "/export/dogu").Return([]byte(`this is no json`), nil)

		emc := &ExportDoguService{
			apiClient: mApiClient,
		}

		_, err := emc.GetExportDogu(testCtx)

		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to parse export dogu response:")
	})
}

func TestExportDoguClient_SetExportDogu(t *testing.T) {
	t.Run("should set export dogu successfully", func(t *testing.T) {
		mApiClient := newMockApiClient(t)
		exportDogu := DoguExport{
			Dogu:         "test",
			VolumePath:   "a/b",
			ExporterPort: 3,
		}
		exportDoguBytes, err := json.Marshal(exportDogu)
		if err != nil {
			panic(err)
		}
		mApiClient.EXPECT().DoPostRequest(testCtx, "/export/dogu/test", nil).Return(exportDoguBytes, nil)

		emc := &ExportDoguService{
			apiClient: mApiClient,
		}

		result, err := emc.SetExportDogu(testCtx, exportDogu.Dogu)

		require.NoError(t, err)
		assert.Equal(t, exportDogu, *result)
	})

	t.Run("should fail to set export dogu because of an error in the request", func(t *testing.T) {
		mApiClient := newMockApiClient(t)
		mApiClient.EXPECT().DoPostRequest(testCtx, "/export/dogu/test", nil).Return(nil, assert.AnError)

		emc := &ExportDoguService{
			apiClient: mApiClient,
		}

		_, err := emc.SetExportDogu(testCtx, "test")

		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to set export dogu")
	})

	t.Run("should fail set export mode with an error while parsing response", func(t *testing.T) {
		mApiClient := newMockApiClient(t)
		mApiClient.EXPECT().DoPostRequest(testCtx, "/export/dogu/test", nil).Return([]byte(`this is no json`), nil)

		emc := &ExportDoguService{
			apiClient: mApiClient,
		}

		_, err := emc.SetExportDogu(testCtx, "test")

		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to parse export dogu response:")
	})
}
