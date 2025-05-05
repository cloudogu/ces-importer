package configuration

import (
	"fmt"
	"os"
)

const (
	exporterHostEnv   = "EXPORTER_HOST"
	exporterApiKeyEnv = "EXPORTER_API_KEY"
)

// API contains the configuration data for the exporter API.
type API struct {
	// ExporterHost configures the FQDN under which the exporter will be available for CES data export. The importer
	// will contact the exporter API which returns all required data like data paths etc.
	// The exporter API endpoint is fixed and will be routed on exporter side. This value is required.
	ExporterHost string
	// ExporterApiKey contains the API key to authenticate against the source system's exporter system info endpoint.
	// This value is required.
	ExporterApiKey string
}

func ReadAPIConfiguration() (API, error) {
	confExporterHost := os.Getenv(exporterHostEnv)
	if confExporterHost == "" {
		return API{}, fmt.Errorf(errorFormat, exporterHostEnv)
	}

	confExporterApiKey := os.Getenv(exporterApiKeyEnv)
	if confExporterApiKey == "" {
		return API{}, fmt.Errorf(errorFormat, exporterApiKeyEnv)
	}

	return API{
		ExporterHost:   confExporterHost,
		ExporterApiKey: confExporterApiKey,
	}, nil
}
