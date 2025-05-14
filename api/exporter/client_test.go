package exporter

import (
	"context"
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

		sut := &client{
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

		sut := &client{
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

		sut := &client{
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
	tests := []struct {
		name       string
		hostName   string
		apiKey     string
		httpClient requestExecuter
		expected   *client
	}{
		{
			name:       "valid inputs",
			hostName:   "example.com",
			apiKey:     "validApiKey",
			httpClient: &http.Client{},
			expected: &client{
				baseUrl:    "https://example.com",
				apiKey:     "validApiKey",
				httpClient: &http.Client{},
			},
		},
		{
			name:       "empty hostName",
			hostName:   "",
			apiKey:     "someApiKey",
			httpClient: &http.Client{},
			expected: &client{
				baseUrl:    "https://",
				apiKey:     "someApiKey",
				httpClient: &http.Client{},
			},
		},
		{
			name:       "empty API key",
			hostName:   "example.com",
			apiKey:     "",
			httpClient: &http.Client{},
			expected: &client{
				baseUrl:    "https://example.com",
				apiKey:     "",
				httpClient: &http.Client{},
			},
		},
		{
			name:       "nil httpClient",
			hostName:   "example.com",
			apiKey:     "validApiKey",
			httpClient: nil,
			expected: &client{
				baseUrl:    "https://example.com",
				apiKey:     "validApiKey",
				httpClient: nil,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// when
			actual := NewClient(tc.hostName, tc.apiKey, tc.httpClient)

			// then
			assert.Equal(t, tc.expected.baseUrl, actual.baseUrl)
			assert.Equal(t, tc.expected.apiKey, actual.apiKey)
			assert.Equal(t, tc.expected.httpClient, actual.httpClient)
		})
	}
}
