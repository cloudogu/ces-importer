package exporter

import (
	"context"
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testCtx = context.Background()

const testApiKey = "testApiKey"
const testResponse = `This was a triumph. I'm making a note here "huge success".'`

func Test_client_DoGetRequest(t *testing.T) {
	httpClient := &http.Client{}

	t.Run("should successfully execute GET request", func(t *testing.T) {
		// given
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer r.Body.Close()

			assert.Equal(t, "GET", r.Method)
			assert.Equal(t, testApiKey, r.Header.Get(apiKeyAuthName))

			w.WriteHeader(http.StatusOK)
			_, err := w.Write([]byte(testResponse))
			require.NoError(t, err)
		}))

		sut := &Client{
			baseUrl:    server.URL,
			apiKey:     testApiKey,
			httpClient: httpClient,
		}

		// when
		actualBytes, err := sut.DoGetRequest(testCtx, "")

		// then
		require.NoError(t, err)
		assert.Equal(t, testResponse, string(actualBytes))
	})
	t.Run("should error on invalid URL", func(t *testing.T) {
		// given

		sut := &Client{
			baseUrl:    "pc:/h\u0012",
			apiKey:     testApiKey,
			httpClient: httpClient,
		}

		// when
		_, err := sut.DoGetRequest(testCtx, "")

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to create request url:")
	})
	t.Run("should error non-OK status code", func(t *testing.T) {
		// given
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer r.Body.Close()

			assert.Equal(t, "GET", r.Method)
			assert.Equal(t, testApiKey, r.Header.Get(apiKeyAuthName))

			w.WriteHeader(http.StatusNotFound)
			_, err := w.Write([]byte("oh noez, something bad happened"))
			require.NoError(t, err)
		}))

		sut := &Client{
			baseUrl:    server.URL,
			apiKey:     testApiKey,
			httpClient: httpClient,
		}

		// when
		_, err := sut.DoGetRequest(testCtx, server.URL)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "received unexpected response to")
		assert.ErrorContains(t, err, "oh noez, something bad happened")
	})
}

func Test_NewClient(t *testing.T) {
	customClient := &http.Client{
		Timeout: 5,
	}

	type expClient struct {
		baseUrl    string
		apiKey     string
		httpClient *http.Client
	}

	tests := []struct {
		name              string
		hostName          string
		apiKey            string
		httpClientOptions []HTTPClientOption
		expected          expClient
	}{
		{
			name:     "valid inputs",
			hostName: "example.com",
			apiKey:   "validApiKey",
			expected: expClient{
				baseUrl:    "https://example.com/ces-exporter",
				apiKey:     "validApiKey",
				httpClient: http.DefaultClient,
			},
		},

		{
			name:     "empty hostName",
			hostName: "",
			apiKey:   "someApiKey",
			expected: expClient{
				baseUrl:    "https:///ces-exporter",
				apiKey:     "someApiKey",
				httpClient: http.DefaultClient,
			},
		},
		{
			name:     "empty API key",
			hostName: "example.com",
			apiKey:   "",
			expected: expClient{
				baseUrl:    "https://example.com/ces-exporter",
				apiKey:     "",
				httpClient: http.DefaultClient,
			},
		},
		{
			name:              "Default Client with insecure option",
			httpClientOptions: []HTTPClientOption{WithInsecure()},
			expected: expClient{
				baseUrl: "https:///ces-exporter",
				apiKey:  "",
				httpClient: func() *http.Client {
					c := &http.Client{}
					c.Transport = &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
					return c
				}(),
			},
		},

		{
			name:              "Use custom Client",
			httpClientOptions: []HTTPClientOption{WithCustomHTTPClient(customClient)},
			expected: expClient{
				baseUrl:    "https:///ces-exporter",
				apiKey:     "",
				httpClient: customClient,
			},
		},

		{
			name:              "Use custom Client with insecure Option",
			httpClientOptions: []HTTPClientOption{WithCustomHTTPClient(customClient), WithInsecure()},
			expected: expClient{
				baseUrl: "https:///ces-exporter",
				apiKey:  "",
				httpClient: func() *http.Client {
					c := &http.Client{Timeout: 5}
					c.Transport = &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
					return c
				}(),
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// when
			actual := NewClient(tc.hostName, tc.apiKey, tc.httpClientOptions...)

			// then
			assert.Equal(t, tc.expected.baseUrl, actual.baseUrl)
			assert.Equal(t, tc.expected.apiKey, actual.apiKey)

			actualHttpClient, ok := actual.httpClient.(*http.Client)
			require.True(t, ok)

			assert.Equal(t, tc.expected.httpClient, actualHttpClient)
		})
	}
}

func Test_client_DoPostRequest(t *testing.T) {
	t.Run("should successfully execute POST request", func(t *testing.T) {
		// given
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer r.Body.Close()

			assert.Equal(t, "POST", r.Method)
			assert.Equal(t, testApiKey, r.Header.Get(apiKeyAuthName))

			w.WriteHeader(http.StatusOK)
			_, err := w.Write([]byte(testResponse))
			require.NoError(t, err)
		}))

		sut := NewClient("test", testApiKey)

		// when
		actualBytes, err := sut.DoPostRequest(testCtx, server.URL, nil, []string{})

		// then
		require.NoError(t, err)
		assert.Equal(t, testResponse, string(actualBytes))
	})
	t.Run("should error on invalid URL", func(t *testing.T) {
		// given

		sut := NewClient("test", testApiKey)

		// when
		_, err := sut.DoPostRequest(testCtx, "pc:/h\x12", nil, []string{})

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to create request to pc:/h\x12: parse")
	})
	t.Run("should error non-OK status code", func(t *testing.T) {
		// given
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer r.Body.Close()

			assert.Equal(t, "POST", r.Method)
			assert.Equal(t, testApiKey, r.Header.Get(apiKeyAuthName))

			w.WriteHeader(http.StatusNotFound)
			_, err := w.Write([]byte("oh noez, something bad happened"))
			require.NoError(t, err)
		}))

		sut := NewClient("test", testApiKey)

		// when
		_, err := sut.DoPostRequest(testCtx, server.URL, nil, []string{})

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "received unexpected response to")
		assert.ErrorContains(t, err, "oh noez, something bad happened")
	})
}
