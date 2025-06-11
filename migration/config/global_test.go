package configuration

import (
	"context"
	"github.com/cloudogu/ces-importer/migration"
	regConfig "github.com/cloudogu/k8s-registry-lib/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestConfigImporter_importGlobalConfig(t *testing.T) {
	testCtx := context.Background()

	t.Run("should import global config successfully", func(t *testing.T) {
		// given
		mockConfigRepo := newMockGlobalConfigRepo(t)
		mockConfigRepo.EXPECT().Get(testCtx).Return(regConfig.GlobalConfig{}, nil)
		mockConfigRepo.EXPECT().Delete(testCtx).Return(nil)

		emptyConfig := regConfig.CreateGlobalConfig(map[regConfig.Key]regConfig.Value{})
		mockConfigRepo.EXPECT().Create(testCtx, emptyConfig).Return(emptyConfig, nil)

		expectedConfig := regConfig.CreateGlobalConfig(map[regConfig.Key]regConfig.Value{})
		newCfg, err := expectedConfig.Set("key1", "value1")
		require.NoError(t, err)
		newCfg, err = newCfg.Set("key2", "value2")
		require.NoError(t, err)

		expectedConfig = regConfig.GlobalConfig{Config: newCfg}
		mockConfigRepo.EXPECT().SaveOrMerge(testCtx, expectedConfig).Return(expectedConfig, nil)

		testConfig := []migration.KeyValue{
			{"key1", "value1"},
			{"key2", "value2"},
		}

		importer := &cesGlobalConfigImporter{
			globalConfigRepo: mockConfigRepo,
		}

		// when
		err = importer.importGlobalConfig(testCtx, testConfig)

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
		mockConfigRepo.EXPECT().Delete(testCtx).Return(nil)

		emptyConfig := regConfig.CreateGlobalConfig(map[regConfig.Key]regConfig.Value{})
		mockConfigRepo.EXPECT().Create(testCtx, emptyConfig).Return(emptyConfig, nil)

		expectedConfig := regConfig.CreateGlobalConfig(map[regConfig.Key]regConfig.Value{
			"certificate/type": "selfsigned",
			"certificate/key":  "certKey",
			"k8s/foo1":         "bar1",
			"proxy/foo2":       "bar2",
			"key1":             "value1",
			"key2":             "value2",
		})

		mockConfigRepo.EXPECT().SaveOrMerge(testCtx, mock.Anything).RunAndReturn(func(ctx context.Context, config regConfig.GlobalConfig) (regConfig.GlobalConfig, error) {
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
			globalConfigRepo: mockConfigRepo,
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
		mockConfigRepo.EXPECT().Delete(testCtx).Return(nil)

		emptyConfig := regConfig.CreateGlobalConfig(map[regConfig.Key]regConfig.Value{})
		mockConfigRepo.EXPECT().Create(testCtx, emptyConfig).Return(emptyConfig, nil)

		expectedConfig := regConfig.CreateGlobalConfig(map[regConfig.Key]regConfig.Value{
			"key1": "value1",
			"key2": "value2",
		})

		mockConfigRepo.EXPECT().SaveOrMerge(testCtx, mock.Anything).RunAndReturn(func(ctx context.Context, config regConfig.GlobalConfig) (regConfig.GlobalConfig, error) {
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
			globalConfigRepo: mockConfigRepo,
		}

		// when
		err := importer.importGlobalConfig(testCtx, testConfig)

		// then
		require.NoError(t, err)
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
			globalConfigRepo: mockConfigRepo,
		}

		// when
		err := importer.importGlobalConfig(testCtx, testConfig)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to get global config:")
	})

	t.Run("should fail to import global config on delete previous config", func(t *testing.T) {
		// given
		mockConfigRepo := newMockGlobalConfigRepo(t)
		mockConfigRepo.EXPECT().Get(testCtx).Return(regConfig.GlobalConfig{}, nil)
		mockConfigRepo.EXPECT().Delete(testCtx).Return(assert.AnError)

		testConfig := []migration.KeyValue{
			{"key1", "value1"},
			{"key2", "value2"},
		}

		importer := &cesGlobalConfigImporter{
			globalConfigRepo: mockConfigRepo,
		}

		// when
		err := importer.importGlobalConfig(testCtx, testConfig)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to delete global config:")
	})

	t.Run("should fail to import global config on create new config", func(t *testing.T) {
		// given
		mockConfigRepo := newMockGlobalConfigRepo(t)
		mockConfigRepo.EXPECT().Get(testCtx).Return(regConfig.GlobalConfig{}, nil)
		mockConfigRepo.EXPECT().Delete(testCtx).Return(nil)

		emptyConfig := regConfig.CreateGlobalConfig(map[regConfig.Key]regConfig.Value{})
		mockConfigRepo.EXPECT().Create(testCtx, emptyConfig).Return(emptyConfig, assert.AnError)

		testConfig := []migration.KeyValue{
			{"key1", "value1"},
			{"key2", "value2"},
		}

		importer := &cesGlobalConfigImporter{
			globalConfigRepo: mockConfigRepo,
		}

		// when
		err := importer.importGlobalConfig(testCtx, testConfig)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to create global config:")
	})

	t.Run("should fail to import global config on saveOrMerge new config", func(t *testing.T) {
		// given
		mockConfigRepo := newMockGlobalConfigRepo(t)
		mockConfigRepo.EXPECT().Get(testCtx).Return(regConfig.GlobalConfig{}, nil)
		mockConfigRepo.EXPECT().Delete(testCtx).Return(nil)

		emptyConfig := regConfig.CreateGlobalConfig(map[regConfig.Key]regConfig.Value{})
		mockConfigRepo.EXPECT().Create(testCtx, emptyConfig).Return(emptyConfig, nil)

		expectedConfig := regConfig.CreateGlobalConfig(map[regConfig.Key]regConfig.Value{})
		newCfg, err := expectedConfig.Set("key1", "value1")
		require.NoError(t, err)
		newCfg, err = newCfg.Set("key2", "value2")
		require.NoError(t, err)

		expectedConfig = regConfig.GlobalConfig{Config: newCfg}
		mockConfigRepo.EXPECT().SaveOrMerge(testCtx, expectedConfig).Return(expectedConfig, assert.AnError)

		testConfig := []migration.KeyValue{
			{"key1", "value1"},
			{"key2", "value2"},
		}

		importer := &cesGlobalConfigImporter{
			globalConfigRepo: mockConfigRepo,
		}

		// when
		err = importer.importGlobalConfig(testCtx, testConfig)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to save new global config:")
	})
}
