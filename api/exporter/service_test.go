package exporter

import (
	"net/http"
	"testing"

	"github.com/cloudogu/ces-importer/configuration"
	"github.com/stretchr/testify/assert"
)

func TestNewService(t *testing.T) {
	apiClientMock := newMockApiClient(t)

	sut := NewService(apiClientMock)

	assert.NotNil(t, sut)
	assert.NotNil(t, sut.MaintenanceModeService)
	assert.NotNil(t, sut.ConfigService)
}

func TestNewServiceFromConfig(t *testing.T) {
	t.Run("should create service from config", func(t *testing.T) {
		cfg := configuration.API{
			ExporterHost:   "exporter",
			ExporterApiKey: "apiKey",
			SkipTLSVerify:  false,
		}

		svc := NewServiceFromConfig(cfg)

		assert.Equal(t, "https://exporter/ces-exporter", svc.MaintenanceModeService.apiClient.(*Client).baseUrl)
		assert.Equal(t, "apiKey", svc.MaintenanceModeService.apiClient.(*Client).apiKey)
	})

	t.Run("should create service from config and skip verify TLS", func(t *testing.T) {
		cfg := configuration.API{
			ExporterHost:   "exporter",
			ExporterApiKey: "apiKey",
			SkipTLSVerify:  true,
		}

		svc := NewServiceFromConfig(cfg)

		assert.Equal(t, "https://exporter/ces-exporter", svc.MaintenanceModeService.apiClient.(*Client).baseUrl)
		assert.Equal(t, "apiKey", svc.MaintenanceModeService.apiClient.(*Client).apiKey)
		assert.True(t, svc.MaintenanceModeService.apiClient.(*Client).httpClient.(*http.Client).Transport.(*http.Transport).TLSClientConfig.InsecureSkipVerify)
	})

}
