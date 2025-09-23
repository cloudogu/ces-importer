package exporter

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
)

const exporterBasePath = "/ces-exporter"
const customCAPath = "/etc/custom-certs/exporter/exporter.crt"

type Client struct {
	baseUrl    string
	apiKey     string
	httpClient requestExecuter
}

type HTTPClientOption func(*http.Client)

// WithCustomHTTPClient sets a custom HTTP Client to be used when executing requests.
func WithCustomHTTPClient(httpClient *http.Client) HTTPClientOption {
	return func(client *http.Client) {
		slog.Debug("Use custom HTTP Client for API requests")

		*client = *httpClient
	}
}

// WithInsecure configures the HTTP Client to skip TLS certificate verification, enabling insecure connections.
// A valid Client needs to be provided for this option. Either by using the default Client or by creating a custom
// Client with the WithCustomHTTPClient option.
func WithInsecure() HTTPClientOption {
	return func(client *http.Client) {
		slog.Debug("Skip TLS certificate verification for API requests")

		transportConfig := getTransportConfig(client)

		if transportConfig.TLSClientConfig == nil {
			transportConfig.TLSClientConfig = &tls.Config{}
		}

		transportConfig.TLSClientConfig.InsecureSkipVerify = true

		client.Transport = transportConfig
	}
}

// WithCustomCAs configures the HTTP Client to add custom CAs to the root CA.
// Custom CAs are supplied in the ces-importer-custom-cas configmap
func WithCustomCAs() HTTPClientOption {
	return func(client *http.Client) {
		caCert, err := os.ReadFile(customCAPath)
		if err != nil {
			slog.Info(fmt.Sprintf("Skipping custom CAs as none were provided in %s", customCAPath))
			return
		}

		// use system cert pool if it exists
		rootCAs, err := x509.SystemCertPool()
		if err != nil {
			rootCAs = x509.NewCertPool()
		}
		if ok := rootCAs.AppendCertsFromPEM(caCert); !ok {
			slog.Warn("Could not add custom CAs. They might already be included.")
			return
		}

		tlsConfig := &tls.Config{
			RootCAs: rootCAs,
		}

		transportConfig := getTransportConfig(client)
		transportConfig.TLSClientConfig = tlsConfig
		client.Transport = transportConfig
		slog.Info("Added custom TLS CAs to exporter API requests.")
	}
}

// getTransportConfig gets the HTTP Clients transport config or a new one, if the client does not have one
func getTransportConfig(client *http.Client) *http.Transport {
	var transportConfig *http.Transport

	if client.Transport == nil {
		transportConfig = &http.Transport{}
	} else {
		clientTransportConfig, ok := client.Transport.(*http.Transport)
		if !ok {
			transportConfig = &http.Transport{}
		} else {
			transportConfig = clientTransportConfig
		}
	}
	return transportConfig
}

// NewClient creates a Client for easy API access with the given HTTP Client. This allows for generically modifying the
// HTTP Client f. i. adding proxy settings.
func NewClient(hostName string, apiKey string, options ...HTTPClientOption) *Client {
	httpClient := &http.Client{}

	for _, option := range options {
		option(httpClient)
	}

	return &Client{
		baseUrl:    fmt.Sprintf("https://%s%s", hostName, exporterBasePath),
		apiKey:     apiKey,
		httpClient: httpClient,
	}
}

// DoGetRequest creates an HTTP GET request towards the exporter API. Any unexpected HTTP codes (other than 200 OK) or
// errors will be returned as an error. For authentication, request headers will automatically be enriched with the
// provided API key.
func (c *Client) DoGetRequest(ctx context.Context, path string) (result []byte, err error) {
	requestUrl, err := url.JoinPath(c.baseUrl, path)
	if err != nil {
		return result, fmt.Errorf("failed to create request url: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, requestUrl, nil)
	if err != nil {
		return result, fmt.Errorf("failed to create request to %s: %w", requestUrl, err)
	}

	request.Header.Set(apiKeyAuthName, c.apiKey)

	response, err := c.httpClient.Do(request)
	if err != nil {
		return result, fmt.Errorf("request to %s failed with an error: %w", requestUrl, err)
	}

	defer func() { _ = response.Body.Close() }()
	responseMsg, err := io.ReadAll(response.Body)
	if err != nil {
		return result, fmt.Errorf("failed to read response body for %s", requestUrl)
	}

	if response.StatusCode != http.StatusOK {
		slog.Debug(fmt.Sprintf("received unexpected response to %s (wanted %d got %d): %s", requestUrl, http.StatusOK, response.StatusCode, string(responseMsg)))

		return result, fmt.Errorf("received unexpected response to %s (wanted %d got %d)",
			requestUrl, http.StatusOK, response.StatusCode)
	}

	slog.Debug(fmt.Sprintf("Successfully called %s with response %s", requestUrl, string(responseMsg)))
	return responseMsg, nil
}

// DoPostRequest creates an HTTP POST request towards the exporter API. Path params will be appended to the given url.
// Any unexpected HTTP codes (other than 200 OK) or errors will be returned as an error. For authentication, request
// headers will automatically be enriched with the provided API key.
func (c *Client) DoPostRequest(ctx context.Context, path string, body io.Reader) (result []byte, err error) {
	requestUrl, err := url.JoinPath(c.baseUrl, path)
	if err != nil {
		return result, fmt.Errorf("failed to join url path %s: %w", path, err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, requestUrl, body)
	if err != nil {
		return result, fmt.Errorf("failed to create request to %s: %w", requestUrl, err)
	}

	request.Header.Set(apiKeyAuthName, c.apiKey)

	response, err := c.httpClient.Do(request)
	if err != nil {
		return result, fmt.Errorf("request to %s failed with an error: %w", requestUrl, err)
	}

	defer func() { _ = response.Body.Close() }()
	responseMsg, err := io.ReadAll(response.Body)
	if err != nil {
		return result, fmt.Errorf("failed to read response body for %s", requestUrl)
	}

	if response.StatusCode != http.StatusOK {
		slog.Debug(fmt.Sprintf("received unexpected response to %s (wanted %d got %d): %s", requestUrl, http.StatusOK, response.StatusCode, string(responseMsg)))

		return result, fmt.Errorf("received unexpected response to %s (wanted %d got %d)",
			requestUrl, http.StatusOK, response.StatusCode)
	}

	slog.Debug(fmt.Sprintf("Successfully called %s with response %s", requestUrl, string(responseMsg)))
	return responseMsg, nil
}
