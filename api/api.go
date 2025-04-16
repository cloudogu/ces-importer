package api

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
)

// http constants
const (
	// apiKeyAuthName contains the name of the header key to authenticate against the exporter API without basic auth.
	apiKeyAuthName = "X-CES-EXPORTER-API-KEY"
)

// exporter endpoints
const (
	// EndpointExportMode contains the endpoint which returns data on the readiness of the exporter system.
	EndpointExportMode = "/export/mode"
)

// ExportMode contains data about the export readiness of the exporter system.
type ExportMode struct {
	// IsActive indicates whether the exporter system is ready to conduct an export (true) or not (false).
	IsActive bool `json:"isActive"`
}

// Dogu contains data on a single installed dogu on the exporter side.
type Dogu struct {
	// Name holds the dogu name including the namespace delimited by a slash ("/").
	Name string `json:"name"`
	// Version holds the dogu version.
	Version string `json:"version"`
	// Volume contains data on the dogu's persistent storage.
	Volume DoguVolume `json:"volume"`
}

// DoguVolume contains data on the dogu's persistent storage.
type DoguVolume struct {
	// SizeInBytes contains the expected dogu volume size.
	//
	// While int32 (~2 Gibi bytes) is too small for comfort, int64 (~9 Exbi bytes) should suffice to accommodate the
	// size of even the largest dogu volume.
	SizeInBytes int64 `json:"sizeInBytes"`
}

type Component struct {
	// Name holds the component name.
	Name string `json:"name"`
	// Version holds the component version.
	Version string `json:"version"`
}

// SystemInfo contains data on vital data on the exporter side.
type SystemInfo struct {
	// FQDN contains the DNS name of the exporter systeḿ.
	FQDN string
	// IsMultinode indicates whether the exporter system is a classic CES or a multinode CES instance.
	IsMultinode bool
	// Dogus contains data on all installed dogus on the exporter side.
	Dogus []Dogu
	// Components contains data on all installed components on the exporter side.
	Components []Component
}

// DoGetRequest creates an HTTP GET request towards the exporter API. Any unexpected HTTP codes (other than 200 OK) or
// errors will be returned as an error. For authentication, request headers will automatically be enriched with the
// provided API key.
func DoGetRequest(ctx context.Context, exporterUrl string, apiKey string, httpClient http.Client) (result []byte, err error) {
	_, err = url.Parse(exporterUrl)
	if err != nil {
		return result, fmt.Errorf("exporter URL %s appears to be invalid (please check ces-importer config values): %w", exporterUrl, err)
	}

	request, err := http.NewRequest(http.MethodGet, exporterUrl, nil)
	if err != nil {
		return result, fmt.Errorf("failed to create request to %s: %w", exporterUrl, err)
	}

	request.WithContext(ctx)
	request.Header.Set(apiKeyAuthName, apiKey)

	response, err := httpClient.Do(request)
	if err != nil {
		return result, fmt.Errorf("request to %s failed with an error: %w", exporterUrl, err)
	}

	defer func() { _ = response.Body.Close() }()
	responseMsg, err := io.ReadAll(response.Body)
	if err != nil {
		return result, fmt.Errorf("failed to read response body for %s", exporterUrl)
	}

	if response.StatusCode != http.StatusOK {
		return result, fmt.Errorf("received unexpected response to %s (wanted %d got %d): %s",
			exporterUrl, http.StatusOK, response.StatusCode, string(responseMsg))
	}

	slog.Log(ctx, slog.LevelDebug, "Successfully called %s with response %#v", responseMsg)
	return result, nil
}
