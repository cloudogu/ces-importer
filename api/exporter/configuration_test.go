package exporter

import (
	"context"
	"errors"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGetConfig(t *testing.T) {
	tests := []struct {
		name           string
		mockResponse   []byte
		mockError      error
		expectedConfig *Configuration
		expectedError  bool
	}{
		{
			name:         "successful config fetch",
			mockResponse: []byte(`{"global": [{"key": "key","value": "value"}],"dogus": [{"name": "test-dogu","normal": [{"key": "key","value": "value"}]}]}`),
			expectedConfig: &Configuration{
				GlobalConfig: []KeyValue{
					{Key: "key", Value: "value"},
				},
				DoguConfigs: []DoguConfig{
					{Name: "test-dogu",
						NormalConfig: []KeyValue{
							{Key: "key", Value: "value"},
						},
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

			getter := ConfigApiClient{
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

func TestNewExporterConfigGetter(t *testing.T) {
	tests := []struct {
		name           string
		exporterHost   string
		apiClient      apiClient
		expectedGetter *ConfigApiClient
	}{
		{
			name:         "valid parameters",
			exporterHost: "test-host",
			apiClient:    newMockApiClient(t),
			expectedGetter: &ConfigApiClient{
				apiClient: newMockApiClient(t),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			getter := NewConfigApiClient(tt.apiClient)
			assert.Equal(t, tt.expectedGetter.apiClient, getter.apiClient)
		})
	}
}
