package main

import (
	"context"
	"testing"

	"github.com/cloudogu/ces-importer/api/exporter"
	"github.com/cloudogu/ces-importer/configuration"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/utils/ptr"
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
	const scaleDown int32 = 0
	t.Run("should stop dogu", func(t *testing.T) {
		// given
		clientSetMock := fake.NewClientset()
		inputDeploy := appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "jenkins",
				Namespace: "ecosystem",
			},
			Spec: appsv1.DeploymentSpec{Replicas: ptr.To(scaleDown)},
			Status: appsv1.DeploymentStatus{
				Replicas:      scaleDown,
				ReadyReplicas: 1,
			},
		}

		_, err := clientSetMock.AppsV1().Deployments("ecosystem").Create(testCtx, &inputDeploy, metav1.CreateOptions{})
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
		err = deactivateImporterDogus(testCtx, inputInfo, config, clientSetMock)

		// then
		require.NoError(t, err)
	})
}

func Test_activateImporterDogus(t *testing.T) {
	const scaleUp int32 = 1
	t.Run("should stop dogu", func(t *testing.T) {
		// given
		clientSetMock := fake.NewClientset()
		inputDeploy := appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "jenkins",
				Namespace: "ecosystem",
			},
			Spec: appsv1.DeploymentSpec{Replicas: ptr.To(scaleUp)},
			Status: appsv1.DeploymentStatus{
				Replicas:      scaleUp,
				ReadyReplicas: 0,
			},
		}

		_, err := clientSetMock.AppsV1().Deployments("ecosystem").Create(testCtx, &inputDeploy, metav1.CreateOptions{})
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
		err = activateImporterDogus(testCtx, inputInfo, config, clientSetMock)

		// then
		require.NoError(t, err)
	})
}
