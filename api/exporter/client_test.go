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

		sut := NewClient(testApiKey, httpClient)

		// when
		actualBytes, err := sut.DoGetRequest(testCtx, server.URL)

		// then
		require.NoError(t, err)
		assert.Equal(t, testResponse, string(actualBytes))
	})
	t.Run("should error on invalid URL", func(t *testing.T) {
		// given

		sut := NewClient(testApiKey, httpClient)

		// when
		_, err := sut.DoGetRequest(testCtx, "pc:/h\x12")

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to create request to pc:/h\x12: parse")
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

		sut := NewClient(testApiKey, httpClient)

		// when
		_, err := sut.DoGetRequest(testCtx, server.URL)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "received unexpected response to")
		assert.ErrorContains(t, err, "oh noez, something bad happened")
	})
}

func Test_client_DoPostRequest(t *testing.T) {
	httpClient := &http.Client{}

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

		sut := NewClient(testApiKey, httpClient)

		// when
		actualBytes, err := sut.DoPostRequest(testCtx, server.URL, nil, []string{})

		// then
		require.NoError(t, err)
		assert.Equal(t, testResponse, string(actualBytes))
	})
	t.Run("should error on invalid URL", func(t *testing.T) {
		// given

		sut := NewClient(testApiKey, httpClient)

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

		sut := NewClient(testApiKey, httpClient)

		// when
		_, err := sut.DoPostRequest(testCtx, server.URL, nil, []string{})

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "received unexpected response to")
		assert.ErrorContains(t, err, "oh noez, something bad happened")
	})
}
