package exporter

import (
	"context"
	"errors"
	"github.com/cloudogu/ces-importer/migration"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGetConfig(t *testing.T) {
	tests := []struct {
		name           string
		mockResponse   []byte
		mockError      error
		expectedConfig *migration.Configuration
		expectedError  bool
	}{
		{
			name:         "successful config fetch",
			mockResponse: []byte(`{"global": [{"key": "key","value": "value"}],"dogus": [{"name": "test-dogu","normal": [{"key": "key","value": "value"}]}], "backupSchedules": [{"name": "bs1","schedule": "0 0 1 * *"}]}`),
			expectedConfig: &migration.Configuration{
				GlobalConfig: []migration.KeyValue{
					{Key: "key", Value: "value"},
				},
				DoguConfigs: []migration.DoguConfig{
					{Name: "test-dogu",
						NormalConfig: []migration.KeyValue{
							{Key: "key", Value: "value"},
						},
					},
				},
				BackupSchedules: []migration.BackupSchedule{
					{
						Name:     "bs1",
						Schedule: "0 0 1 * *",
					},
				},
			},
			expectedError: false,
		},
		{
			name:           "error in API client",
			mockError:      errors.New("network error"),
			expectedConfig: nil,
			expectedError:  true,
		},
		{
			name:           "invalid JSON response",
			mockResponse:   []byte(`invalid-json`),
			expectedConfig: nil,
			expectedError:  true,
		},
	}

	for _, tt := range tests {
		ctx := context.Background()
		t.Run(tt.name, func(t *testing.T) {
			mockClient := newMockApiClient(t)
			mockClient.EXPECT().DoGetRequest(ctx, "/configuration").Return(tt.mockResponse, tt.mockError)

			getter := ConfigService{
				apiClient: mockClient,
			}

			config, err := getter.GetConfig(ctx)

			if (err != nil) != tt.expectedError {
				t.Fatalf("expected error: %v, got: %v", tt.expectedError, err)
			}

			if !tt.expectedError {
				assert.Equal(t, tt.expectedConfig, config)
			}
		})
	}
}

func TestNewConfigService(t *testing.T) {
	tests := []struct {
		name           string
		exporterHost   string
		apiClient      apiClient
		expectedGetter *ConfigService
	}{
		{
			name:         "valid parameters",
			exporterHost: "test-host",
			apiClient:    newMockApiClient(t),
			expectedGetter: &ConfigService{
				apiClient: newMockApiClient(t),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			getter := NewConfigService(tt.apiClient)
			assert.Equal(t, tt.expectedGetter.apiClient, getter.apiClient)
		})
	}
}
