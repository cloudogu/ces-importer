package main

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cloudogu/ces-importer/api/exporter"
)

var testCtx = context.Background()

const testFqdn = "server.fqdn"

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
		actual, err := fetchExporterSystemInfo(testCtx, testFqdn, exportApiClient)

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
			FQDN:        testFqdn,
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
		_, err := fetchExporterSystemInfo(testCtx, testFqdn, exportApiClient)

		// then
		require.Error(t, err)
	})
}

func Test_deactivateImporterDogus(t *testing.T) {
	t.Run("should stop dogu", func(t *testing.T) {
		// given
		jenkinsDogu := exporter.Dogu{
			Name:    "official/jenkins",
			Version: "2.492.3-4",
			Volume:  exporter.DoguVolume{SizeInBytes: 1234},
		}

		expectedDogus := []exporter.Dogu{jenkinsDogu}
		expectedComps := []exporter.Component{{
			Name:    "k8s/k8s-dogu-operator",
			Version: "3.5.0",
		}}

		stopper := NewMockdoguStopper(t)
		stopper.EXPECT().StopDogu(testCtx, jenkinsDogu).Return(nil)

		inputInfo := &exporter.SystemInfo{
			FQDN:        "server.fqdn",
			IsMultinode: false,
			Dogus:       expectedDogus,
			Components:  expectedComps,
		}

		// when
		err := deactivateImporterDogus(testCtx, inputInfo, stopper)

		// then
		require.NoError(t, err)
	})
	t.Run("should return with error", func(t *testing.T) {
		// given
		jenkinsDogu := exporter.Dogu{
			Name:    "official/jenkins",
			Version: "2.492.3-4",
			Volume:  exporter.DoguVolume{SizeInBytes: 1234},
		}

		expectedDogus := []exporter.Dogu{jenkinsDogu}
		expectedComps := []exporter.Component{{
			Name:    "k8s/k8s-dogu-operator",
			Version: "3.5.0",
		}}

		stopper := NewMockdoguStopper(t)
		stopper.EXPECT().StopDogu(testCtx, jenkinsDogu).Return(assert.AnError)

		inputInfo := &exporter.SystemInfo{
			FQDN:        "server.fqdn",
			IsMultinode: false,
			Dogus:       expectedDogus,
			Components:  expectedComps,
		}

		// when
		err := deactivateImporterDogus(testCtx, inputInfo, stopper)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to deactivate dogu official/jenkins in the importer")
	})
}

func Test_activateImporterDogus(t *testing.T) {
	t.Run("should stop dogu", func(t *testing.T) {
		// given
		jenkinsDogu := exporter.Dogu{
			Name:    "official/jenkins",
			Version: "2.492.3-4",
			Volume:  exporter.DoguVolume{SizeInBytes: 1234},
		}

		expectedDogus := []exporter.Dogu{jenkinsDogu}
		expectedComps := []exporter.Component{{
			Name:    "k8s/k8s-dogu-operator",
			Version: "3.5.0",
		}}

		starter := NewMockdoguStarter(t)
		starter.EXPECT().StartDogu(testCtx, jenkinsDogu).Return(nil)

		inputInfo := &exporter.SystemInfo{
			FQDN:        "server.fqdn",
			IsMultinode: false,
			Dogus:       expectedDogus,
			Components:  expectedComps,
		}

		// when
		err := activateImporterDogus(testCtx, inputInfo, starter)

		// then
		require.NoError(t, err)
	})
	t.Run("starting dogu should fail with error", func(t *testing.T) {
		// given
		jenkinsDogu := exporter.Dogu{
			Name:    "official/jenkins",
			Version: "2.492.3-4",
			Volume:  exporter.DoguVolume{SizeInBytes: 1234},
		}

		expectedDogus := []exporter.Dogu{jenkinsDogu}
		expectedComps := []exporter.Component{{
			Name:    "k8s/k8s-dogu-operator",
			Version: "3.5.0",
		}}

		starter := NewMockdoguStarter(t)
		starter.EXPECT().StartDogu(testCtx, jenkinsDogu).Return(assert.AnError)

		inputInfo := &exporter.SystemInfo{
			FQDN:        "server.fqdn",
			IsMultinode: false,
			Dogus:       expectedDogus,
			Components:  expectedComps,
		}

		config := configuration.Configuration{
			ExporterHost:              testFqdn,
			ExporterSSHUser:           "root",
			ExporterApiKey:            "my-key",
			ImporterPrivateSSHKeyPath: "/something",
			ImporterNamespace:         "ecosystem",
			LogLevel:                  "INFO",
			MigrationRegularCron:      "0,30 * * * * *",
			MigrationFinalTimestamp:   "2025-something",
		}
		// when
		err := activateImporterDogus(testCtx, inputInfo, starter)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to activate dogu official/jenkins in the importer")
	})
}

func Test_createMainLoop(t *testing.T) {
	t.Run("should run the function successfully", func(t *testing.T) {
		// given

		// when
		err = activateImporterDogus(testCtx, inputInfo, config, clientSetMock)

		// then
		require.NoError(t, err)
	})
}
