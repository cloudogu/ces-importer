package importer

import (
	backupEcosystem "github.com/cloudogu/k8s-backup-lib/api/ecosystem"
	componentEcoClient "github.com/cloudogu/k8s-component-lib/client"
	doguLibClient "github.com/cloudogu/k8s-dogu-lib/v2/client"
	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"testing"
)

func TestCreateK8SClientSet(t *testing.T) {
	t.Run("create k8s client set", func(t *testing.T) {
		clientSet, err := CreateK8SClientSet(&rest.Config{}, "test")
		assert.NoError(t, err)

		assert.NotNil(t, clientSet)
		assert.NotNil(t, clientSet.Pvc)
		assert.NotNil(t, clientSet.Secret)
		assert.NotNil(t, clientSet.ConfigMap)
		assert.NotNil(t, clientSet.Dogu)
		assert.NotNil(t, clientSet.Pod)
		assert.NotNil(t, clientSet.Job)
		assert.NotNil(t, clientSet.BackupSchedule)
		assert.NotNil(t, clientSet.Component)
	})

	t.Run("Error: create k8s client set", func(t *testing.T) {
		oldgetK8sClientsSet := getK8sClientsSet
		defer func() {
			getK8sClientsSet = oldgetK8sClientsSet
		}()

		getK8sClientsSet = func(c *rest.Config) (*kubernetes.Clientset, error) {
			return nil, assert.AnError
		}

		_, err := CreateK8SClientSet(&rest.Config{}, "test")
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to create k8s client set")
	})

	t.Run("Error: create ecosystem client", func(t *testing.T) {
		oldgetEcoSystemClient := getEcoSystemClient
		defer func() {
			getEcoSystemClient = oldgetEcoSystemClient
		}()

		getEcoSystemClient = func(c *rest.Config) (*doguLibClient.EcoSystemV2Client, error) {
			return nil, assert.AnError
		}

		_, err := CreateK8SClientSet(&rest.Config{}, "test")
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to create ecosystem client")
	})

	t.Run("Error: create component client", func(t *testing.T) {
		oldgetComponentsClient := getComponentsClient
		defer func() {
			getComponentsClient = oldgetComponentsClient
		}()

		getComponentsClient = func(c *rest.Config) (*componentEcoClient.V1Alpha1Client, error) {
			return nil, assert.AnError
		}

		_, err := CreateK8SClientSet(&rest.Config{}, "test")
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to create component client")
	})

	t.Run("Error: create backup client", func(t *testing.T) {
		oldgetBackupClient := getBackupClient
		defer func() {
			getBackupClient = oldgetBackupClient
		}()

		getBackupClient = func(c *rest.Config) (*backupEcosystem.V1Alpha1Client, error) {
			return nil, assert.AnError
		}

		_, err := CreateK8SClientSet(&rest.Config{}, "test")
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to create ecosystem backup client")
	})
}
