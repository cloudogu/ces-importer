package configuration

import (
	"context"
	"testing"

	"github.com/cloudogu/ces-importer/migration"
	regConfig "github.com/cloudogu/k8s-registry-lib/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// createSimpleKeyValueConfig builds the common key1/key2 exporter config and the expected
// resulting global config shared by several subtests below.
func createSimpleKeyValueConfig(t *testing.T) ([]migration.KeyValue, regConfig.GlobalConfig) {
	t.Helper()

	testConfig := []migration.KeyValue{
		{"key1", "value1"},
		{"key2", "value2"},
	}

	expectedConfig := regConfig.CreateGlobalConfig(map[regConfig.Key]regConfig.Value{})
	newCfg, err := expectedConfig.Set("key1", "value1")
	require.NoError(t, err)
	newCfg, err = newCfg.Set("key2", "value2")
	require.NoError(t, err)
	expectedConfig = regConfig.GlobalConfig{Config: newCfg}

	return testConfig, expectedConfig
}

func TestConfigImporter_importGlobalConfig(t *testing.T) {
	testCtx := context.Background()

	t.Run("should import global config successfully", func(t *testing.T) {
		// given
		mockConfigRepo := newMockGlobalConfigRepo(t)
		mockConfigRepo.EXPECT().Get(testCtx).Return(regConfig.GlobalConfig{}, nil)

		testConfig, expectedConfig := createSimpleKeyValueConfig(t)
		mockConfigRepo.EXPECT().Update(testCtx, expectedConfig).Return(expectedConfig, nil)

		importer := &cesGlobalConfigImporter{
			globalConfigRepo:     mockConfigRepo,
			additionalKeysToKeep: []string{},
		}

		// when
		err := importer.importGlobalConfig(testCtx, testConfig)

		// then
		require.NoError(t, err)
	})

	t.Run("should keep previous proxy/certificate/k8s config", func(t *testing.T) {
		// given
		mockConfigRepo := newMockGlobalConfigRepo(t)

		previousConfig := regConfig.CreateGlobalConfig(map[regConfig.Key]regConfig.Value{
			"certificate/type": "selfsigned",
			"certificate/key":  "certKey",
			"k8s/foo1":         "bar1",
			"proxy/foo2":       "bar2",
			"something/else":   "foobar",
		})
		mockConfigRepo.EXPECT().Get(testCtx).Return(previousConfig, nil)

		expectedConfig := regConfig.CreateGlobalConfig(map[regConfig.Key]regConfig.Value{
			"certificate/type": "selfsigned",
			"certificate/key":  "certKey",
			"k8s/foo1":         "bar1",
			"proxy/foo2":       "bar2",
			"key1":             "value1",
			"key2":             "value2",
		})

		mockConfigRepo.EXPECT().Update(testCtx, mock.Anything).RunAndReturn(func(ctx context.Context, config regConfig.GlobalConfig) (regConfig.GlobalConfig, error) {
			assert.Equal(t, testCtx, ctx)

			diff := expectedConfig.Diff(config.Config)
			assert.Len(t, diff, 0)

			return expectedConfig, nil
		})

		testConfig := []migration.KeyValue{
			{"key1", "value1"},
			{"key2", "value2"},
		}

		importer := &cesGlobalConfigImporter{
			globalConfigRepo:     mockConfigRepo,
			additionalKeysToKeep: []string{},
		}

		// when
		err := importer.importGlobalConfig(testCtx, testConfig)

		// then
		require.NoError(t, err)
	})

	t.Run("should ignore imported proxy/certificate/k8s config", func(t *testing.T) {
		// given
		mockConfigRepo := newMockGlobalConfigRepo(t)

		previousConfig := regConfig.CreateGlobalConfig(map[regConfig.Key]regConfig.Value{})
		mockConfigRepo.EXPECT().Get(testCtx).Return(previousConfig, nil)

		expectedConfig := regConfig.CreateGlobalConfig(map[regConfig.Key]regConfig.Value{
			"key1": "value1",
			"key2": "value2",
		})

		mockConfigRepo.EXPECT().Update(testCtx, mock.Anything).RunAndReturn(func(ctx context.Context, config regConfig.GlobalConfig) (regConfig.GlobalConfig, error) {
			assert.Equal(t, testCtx, ctx)

			diff := expectedConfig.Diff(config.Config)
			assert.Len(t, diff, 0)

			return expectedConfig, nil
		})

		testConfig := []migration.KeyValue{
			{"key1", "value1"},
			{"key2", "value2"},
			{"certificate/type", "selfsigned"},
			{"certificate/key", "certKey"},
			{"k8s/foo1", "bar1"},
			{"proxy/foo2", "bar2"},
		}

		importer := &cesGlobalConfigImporter{
			globalConfigRepo:     mockConfigRepo,
			additionalKeysToKeep: []string{},
		}

		// when
		err := importer.importGlobalConfig(testCtx, testConfig)

		// then
		require.NoError(t, err)
	})

	t.Run("should not overwrite excluded config keys", func(t *testing.T) {
		// given
		mockConfigRepo := newMockGlobalConfigRepo(t)

		importerConfig := regConfig.CreateGlobalConfig(map[regConfig.Key]regConfig.Value{
			"path/to/ignoredKey": "importer-value",
			"/default_dogu":      "importer-value",
			"path/to/key1":       "importer-value",
			"key2":               "importer-value",
		})

		mockConfigRepo.EXPECT().Get(testCtx).Return(importerConfig, nil)

		emptyConfig := regConfig.CreateGlobalConfig(map[regConfig.Key]regConfig.Value{})

		var mergedConfig regConfig.GlobalConfig
		mockConfigRepo.EXPECT().Update(testCtx, mock.Anything).RunAndReturn(func(ctx context.Context, newConfig regConfig.GlobalConfig) (regConfig.GlobalConfig, error) {
			mergedConfig = newConfig

			return emptyConfig, nil
		})

		expectedConfig := regConfig.CreateGlobalConfig(map[regConfig.Key]regConfig.Value{
			"path/to/key1":       "exporter-value",
			"key2":               "exporter-value",
			"path/to/ignoredKey": "importer-value",
			"default_dogu":       "importer-value",
		})

		exporterConfig := []migration.KeyValue{
			{"path/to/key1", "exporter-value"},
			{"key2", "exporter-value"},
			{"/path/to/ignoredKey", "exporter-value"},
			{"default_dogu", "exporter-value"},
		}

		importer := &cesGlobalConfigImporter{
			globalConfigRepo:     mockConfigRepo,
			additionalKeysToKeep: []string{"path/to/ignoredKey", "default_dogu"},
		}

		// when
		err := importer.importGlobalConfig(testCtx, exporterConfig)

		// then
		require.NoError(t, err)
		diff := expectedConfig.Diff(mergedConfig.Config)
		assert.Equal(t, len(diff), 0)
	})

	t.Run("should fail to import global config on get previous config", func(t *testing.T) {
		// given
		mockConfigRepo := newMockGlobalConfigRepo(t)
		mockConfigRepo.EXPECT().Get(testCtx).Return(regConfig.GlobalConfig{}, assert.AnError)

		testConfig := []migration.KeyValue{
			{"key1", "value1"},
			{"key2", "value2"},
		}

		importer := &cesGlobalConfigImporter{
			globalConfigRepo:     mockConfigRepo,
			additionalKeysToKeep: []string{},
		}

		// when
		err := importer.importGlobalConfig(testCtx, testConfig)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to get global config:")
	})

	t.Run("should fail to import global config on create new config", func(t *testing.T) {
		// given
		mockConfigRepo := newMockGlobalConfigRepo(t)
		mockConfigRepo.EXPECT().Get(testCtx).Return(regConfig.GlobalConfig{}, apierrors.NewNotFound(
			schema.GroupResource{
				Group:    "",
				Resource: "configmaps",
			},
			"notfound",
		))

		emptyConfig := regConfig.CreateGlobalConfig(map[regConfig.Key]regConfig.Value{})
		mockConfigRepo.EXPECT().Create(testCtx, emptyConfig).Return(emptyConfig, assert.AnError)

		testConfig := []migration.KeyValue{
			{"key1", "value1"},
			{"key2", "value2"},
		}

		importer := &cesGlobalConfigImporter{
			globalConfigRepo:     mockConfigRepo,
			additionalKeysToKeep: []string{},
		}

		// when
		err := importer.importGlobalConfig(testCtx, testConfig)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to create global config:")
	})

	t.Run("should succeed to create and import global config not found error", func(t *testing.T) {
		// given
		mockConfigRepo := newMockGlobalConfigRepo(t)
		mockConfigRepo.EXPECT().Get(testCtx).Return(regConfig.GlobalConfig{}, apierrors.NewNotFound(
			schema.GroupResource{
				Group:    "",
				Resource: "configmaps",
			},
			"notfound",
		))

		emptyConfig := regConfig.CreateGlobalConfig(map[regConfig.Key]regConfig.Value{})
		mockConfigRepo.EXPECT().Create(testCtx, emptyConfig).Return(emptyConfig, nil)

		testConfig, expectedConfig := createSimpleKeyValueConfig(t)
		mockConfigRepo.EXPECT().Update(testCtx, expectedConfig).Return(expectedConfig, nil)

		importer := &cesGlobalConfigImporter{
			globalConfigRepo:     mockConfigRepo,
			additionalKeysToKeep: []string{},
		}

		// when
		err := importer.importGlobalConfig(testCtx, testConfig)

		// then
		require.NoError(t, err)
	})

	t.Run("should fail to import global config on update new config", func(t *testing.T) {
		// given
		mockConfigRepo := newMockGlobalConfigRepo(t)
		mockConfigRepo.EXPECT().Get(testCtx).Return(regConfig.GlobalConfig{}, nil)

		testConfig, expectedConfig := createSimpleKeyValueConfig(t)
		mockConfigRepo.EXPECT().Update(testCtx, expectedConfig).Return(expectedConfig, assert.AnError)

		importer := &cesGlobalConfigImporter{
			globalConfigRepo:     mockConfigRepo,
			additionalKeysToKeep: []string{},
		}

		// when
		err := importer.importGlobalConfig(testCtx, testConfig)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to save new global config:")
	})
}
