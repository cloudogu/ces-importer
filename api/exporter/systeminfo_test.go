package exporter

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestNewSystemInfoClient(t *testing.T) {
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
			client := NewSystemInfoService(tt.apiClient)

			if client == nil {
				t.Fatalf("Expected non-nil SystemInfoService, got nil")
			}

			if client.apiClient != tt.apiClient {
				t.Errorf("Expected apiClient: %+v, got: %+v", tt.apiClient, client.apiClient)
			}
		})
	}
}

func TestSystemInfoClient_GetSystemInfo(t *testing.T) {
	exmapleJson := `{
	  "fqdn": "exporter.example.com",
	  "isMultinode": true,
	  "dogus": [
		{
		  "name": "official/redmine",
		  "version": "5.1.2-1",
		  "volume": {
			"sizeInBytes": 10737418240
		  }
		},
		{
		  "name": "official/jenkins",
		  "version": "2.414.3-2",
		  "volume": {
			"sizeInBytes": 21474836480
		  }
		}
	  ],
	  "components": [
		{
		  "name": "nginx",
		  "version": "1.24.0"
		},
		{
		  "name": "postgresql",
		  "version": "15.3"
		}
	  ]
	}`

	t.Run("should get system info successfully", func(t *testing.T) {
		mApiClient := newMockApiClient(t)
		mApiClient.EXPECT().DoGetRequest(testCtx, "/system-info").Return([]byte(exmapleJson), nil)

		emc := &SystemInfoService{
			apiClient: mApiClient,
		}

		sysInfo, err := emc.GetSystemInfo(testCtx)

		require.NoError(t, err)
		assert.Equal(t, "exporter.example.com", sysInfo.FQDN)
		assert.True(t, sysInfo.IsMultinode)
		assert.Len(t, sysInfo.Dogus, 2)
		assert.Equal(t, "official/redmine", sysInfo.Dogus[0].Name)
		assert.Equal(t, "5.1.2-1", sysInfo.Dogus[0].Version)
		assert.Equal(t, int64(10737418240), sysInfo.Dogus[0].Volume.SizeInBytes)
		assert.Equal(t, "official/jenkins", sysInfo.Dogus[1].Name)
		assert.Equal(t, "2.414.3-2", sysInfo.Dogus[1].Version)
		assert.Equal(t, int64(21474836480), sysInfo.Dogus[1].Volume.SizeInBytes)
		assert.Len(t, sysInfo.Components, 2)
		assert.Equal(t, "nginx", sysInfo.Components[0].Name)
		assert.Equal(t, "1.24.0", sysInfo.Components[0].Version)
		assert.Equal(t, "postgresql", sysInfo.Components[1].Name)
		assert.Equal(t, "15.3", sysInfo.Components[1].Version)
	})

	t.Run("should fail get system info for error in request", func(t *testing.T) {
		mApiClient := newMockApiClient(t)
		mApiClient.EXPECT().DoGetRequest(testCtx, "/system-info").Return(nil, assert.AnError)

		emc := &SystemInfoService{
			apiClient: mApiClient,
		}

		_, err := emc.GetSystemInfo(testCtx)

		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to get system info:")
	})

	t.Run("should fail get system info for error while parsing response", func(t *testing.T) {
		mApiClient := newMockApiClient(t)
		mApiClient.EXPECT().DoGetRequest(testCtx, "/system-info").Return([]byte(`this is no json`), nil)

		emc := &SystemInfoService{
			apiClient: mApiClient,
		}

		_, err := emc.GetSystemInfo(testCtx)

		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to parse system info response:")
	})
}
