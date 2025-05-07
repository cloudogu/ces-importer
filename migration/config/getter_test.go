package configuration

import (
	"context"
	"errors"
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGetConfig(t *testing.T) {
	tests := []struct {
		name           string
		exporterHost   string
		mockResponse   []byte
		mockError      error
		expectedConfig *configuration
		expectedError  bool
	}{
		{
			name:         "successful config fetch",
			exporterHost: "test-host",
			mockResponse: []byte(`{"global": [{"key": "key","value": "value"}],"dogus": [{"name": "test-dogu","normal": [{"key": "key","value": "value"}]}]}`),
			expectedConfig: &configuration{
				GlobalConfig: []keyValue{
					{Key: "key", Value: "value"},
				},
				DoguConfigs: []doguConfig{
					{Name: "test-dogu",
						NormalConfig: []keyValue{
							{Key: "key", Value: "value"},
						},
					},
				},
			},
			expectedError: false,
		},
		{
			name:           "error in API client",
			exporterHost:   "test-host",
			mockError:      errors.New("network error"),
			expectedConfig: nil,
			expectedError:  true,
		},
		{
			name:           "invalid JSON response",
			exporterHost:   "test-host",
			mockResponse:   []byte(`invalid-json`),
			expectedConfig: nil,
			expectedError:  true,
		},
	}

	for _, tt := range tests {
		ctx := context.Background()
		t.Run(tt.name, func(t *testing.T) {
			mockClient := newMockExporterApiClient(t)
			mockClient.EXPECT().DoGetRequest(ctx, fmt.Sprintf("https://%s/configuration", tt.exporterHost)).Return(tt.mockResponse, tt.mockError)

			getter := exporterConfigGetter{
				exporterHost: tt.exporterHost,
				apiClient:    mockClient,
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
		apiClient      exporterApiClient
		expectedGetter *exporterConfigGetter
	}{
		{
			name:         "valid parameters",
			exporterHost: "test-host",
			apiClient:    newMockExporterApiClient(t),
			expectedGetter: &exporterConfigGetter{
				exporterHost: "test-host",
				apiClient:    newMockExporterApiClient(t),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			getter := newExporterConfigGetter(tt.exporterHost, tt.apiClient)
			assert.Equal(t, tt.expectedGetter.exporterHost, getter.exporterHost)
			assert.Equal(t, tt.expectedGetter.apiClient, getter.apiClient)
		})
	}
}
