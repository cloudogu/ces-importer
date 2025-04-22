package main

import (
	"context"
	"testing"

	"github.com/cloudogu/ces-importer/api/exporter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testCtx = context.Background()

func Test_isApiExportReady(t *testing.T) {
	t.Run("should be ready", func(t *testing.T) {
		// given
		exportApiClient := NewMockexporterApiClient(t)
		exportApiClient.EXPECT().DoGetRequest(testCtx, "https://server.fqdn/export/mode").Return([]byte(`{"isActive": true}`), nil)

		// when
		ready, err := isApiExportReady(testCtx, "server.fqdn", exportApiClient)

		// then
		require.NoError(t, err)
		assert.True(t, ready)
	})
	t.Run("should not be ready", func(t *testing.T) {
		// given
		exportApiClient := NewMockexporterApiClient(t)
		exportApiClient.EXPECT().DoGetRequest(testCtx, "https://server.fqdn/export/mode").Return([]byte(`{"isActive": false}`), nil)

		// when
		ready, err := isApiExportReady(testCtx, "server.fqdn", exportApiClient)

		// then
		require.NoError(t, err)
		assert.False(t, ready)
	})
	t.Run("should return error for upstream error", func(t *testing.T) {
		// given
		exportApiClient := NewMockexporterApiClient(t)
		exportApiClient.EXPECT().DoGetRequest(testCtx, "https://server.fqdn/export/mode").Return(nil, assert.AnError)

		// when
		_, err := isApiExportReady(testCtx, "server.fqdn", exportApiClient)

		// then
		require.Error(t, err)
	})
}

func Test_fetchExporterSystemInfo(t *testing.T) {
	t.Run("should return system infos", func(t *testing.T) {
		// given
		exportApiClient := NewMockexporterApiClient(t)
		responseJson := `{"fqdn":"server.fqdn","isMultinode":false,"dogus":[{"name":"official/jenkins","version":"2.492.3-4","volume":{"sizeInBytes":1234}}],"components":[{"name":"k8s/k8s-dogu-operator","version":"3.5.0"}]}`
		exportApiClient.EXPECT().DoGetRequest(testCtx, "https://server.fqdn/system-info").Return([]byte(responseJson), nil)

		// when
		actual, err := fetchExporterSystemInfo(testCtx, "server.fqdn", exportApiClient)

		// then
		require.NoError(t, err)
		expectedDogus := []exporter.Dogu{{
			Name:    "official/jenkins",
			Version: "2.492.3-4",
			Volume:  exporter.DoguVolume{SizeInBytes: 1234},
		}}
		expectedComps := []exporter.Component{{
			Name:    "k8s/k8s-dogu-operator",
			Version: "3.5.0",
		}}

		expected := &exporter.SystemInfo{
			FQDN:        "server.fqdn",
			IsMultinode: false,
			Dogus:       expectedDogus,
			Components:  expectedComps,
		}
		assert.Equal(t, expected, actual)
	})
	t.Run("should return error for upstream error", func(t *testing.T) {
		// given
		exportApiClient := NewMockexporterApiClient(t)
		exportApiClient.EXPECT().DoGetRequest(testCtx, "https://server.fqdn/system-info").Return(nil, assert.AnError)

		// when
		_, err := fetchExporterSystemInfo(testCtx, "server.fqdn", exportApiClient)

		// then
		require.Error(t, err)
	})
}
