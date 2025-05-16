package configuration

import (
	"context"
	regConfig "github.com/cloudogu/k8s-registry-lib/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestConfigImporter_importGlobalCertificates(t *testing.T) {
	testCtx := context.Background()

	t.Run("should import global certificates successfully", func(t *testing.T) {
		// given
		emptyConfig := regConfig.CreateGlobalConfig(map[regConfig.Key]regConfig.Value{})
		mockConfigRepo := newMockGlobalConfigRepo(t)
		mockConfigRepo.EXPECT().Get(testCtx).Return(emptyConfig, nil)

		expectedConfig := regConfig.CreateGlobalConfig(map[regConfig.Key]regConfig.Value{})
		newCfg, err := expectedConfig.Set("certificate/server.crt", "some crt")
		require.NoError(t, err)
		newCfg, err = newCfg.Set("certificate/server.key", "some key")
		require.NoError(t, err)
		newCfg, err = newCfg.Set("certificate/type", "selfsigned")
		require.NoError(t, err)
		expectedConfig = regConfig.GlobalConfig{Config: newCfg}
		mockConfigRepo.EXPECT().SaveOrMerge(testCtx, expectedConfig).Return(expectedConfig, nil)

		testConfig := []keyValue{
			{"certificate/server.crt", "some crt"},
			{"certificate/server.key", "some key"},
			{"certificate/type", "selfsigned"},
			{"fqdn", "test.ces.importer"},
			// this key should not be migrated
			{"ignoredkey", "some value"},
		}

		importer := &cesGlobalConfigImporter{
			globalConfigRepo: mockConfigRepo,
		}

		// when
		err = importer.importGlobalCertificates(testCtx, testConfig)

		// then
		require.NoError(t, err)
	})
}

func TestConfigImporter_importGlobalFQDN(t *testing.T) {
	testCtx := context.Background()

	t.Run("should import global fqdn successfully", func(t *testing.T) {
		// given
		emptyConfig := regConfig.CreateGlobalConfig(map[regConfig.Key]regConfig.Value{})
		mockConfigRepo := newMockGlobalConfigRepo(t)
		mockConfigRepo.EXPECT().Get(testCtx).Return(emptyConfig, nil)

		expectedConfig := regConfig.CreateGlobalConfig(map[regConfig.Key]regConfig.Value{})
		newCfg, err := expectedConfig.Set("fqdn", "test.ces.importer")
		require.NoError(t, err)
		expectedConfig = regConfig.GlobalConfig{Config: newCfg}
		mockConfigRepo.EXPECT().SaveOrMerge(testCtx, expectedConfig).Return(expectedConfig, nil)

		testConfig := []keyValue{
			{"certificate/server.crt", "some crt"},
			{"certificate/server.key", "some key"},
			{"certificate/type", "selfsigned"},
			{"fqdn", "test.ces.importer"},
			// this key should not be migrated
			{"ignoredkey", "some value"},
		}

		importer := &cesGlobalConfigImporter{
			globalConfigRepo: mockConfigRepo,
		}

		// when
		err = importer.importGlobalFQDN(testCtx, testConfig)

		// then
		require.NoError(t, err)
	})
}

func TestConfigImporter_backupGlobalKeys(t *testing.T) {
	testCtx := context.Background()

	t.Run("should backup global successfully", func(t *testing.T) {
		// given
		emptyConfig := regConfig.CreateGlobalConfig(map[regConfig.Key]regConfig.Value{})
		newCfg, err := emptyConfig.Set("fqdn", "test.ces.importer")
		emptyConfig = regConfig.GlobalConfig{Config: newCfg}
		newCfg, err = emptyConfig.Set("certificate/server.crt", "some crt")
		emptyConfig = regConfig.GlobalConfig{Config: newCfg}
		newCfg, err = emptyConfig.Set("certificate/server.key", "some key")
		emptyConfig = regConfig.GlobalConfig{Config: newCfg}
		newCfg, err = emptyConfig.Set("certificate/type", "some type")
		emptyConfig = regConfig.GlobalConfig{Config: newCfg}

		mockConfigRepo := newMockGlobalConfigRepo(t)
		mockConfigRepo.EXPECT().Get(testCtx).Return(emptyConfig, nil)

		expectedConfig := regConfig.CreateGlobalConfig(map[regConfig.Key]regConfig.Value{})
		newCfg, err = expectedConfig.Set("fqdn", "test.ces.importer")
		expectedConfig = regConfig.GlobalConfig{Config: newCfg}
		newCfg, err = expectedConfig.Set("certificate/server.crt", "some crt")
		expectedConfig = regConfig.GlobalConfig{Config: newCfg}
		newCfg, err = expectedConfig.Set("certificate/server.key", "some key")
		expectedConfig = regConfig.GlobalConfig{Config: newCfg}
		newCfg, err = expectedConfig.Set("certificate/type", "some type")
		expectedConfig = regConfig.GlobalConfig{Config: newCfg}

		newCfg, err = expectedConfig.Set("ces-import-backup/fqdn", "test.ces.importer")
		expectedConfig = regConfig.GlobalConfig{Config: newCfg}
		newCfg, err = expectedConfig.Set("ces-import-backup/certificate/server.key", "some key")
		expectedConfig = regConfig.GlobalConfig{Config: newCfg}
		newCfg, err = expectedConfig.Set("ces-import-backup/certificate/type", "some type")
		expectedConfig = regConfig.GlobalConfig{Config: newCfg}
		newCfg, err = expectedConfig.Set("ces-import-backup/certificate/server.crt", "some crt")
		expectedConfig = regConfig.GlobalConfig{Config: newCfg}
		require.NoError(t, err)
		mockConfigRepo.EXPECT().SaveOrMerge(testCtx, mock.Anything).RunAndReturn(func(ctx context.Context, config regConfig.GlobalConfig) (regConfig.GlobalConfig, error) {
			assert.Equal(t, testCtx, ctx)

			diff := expectedConfig.Diff(config.Config)
			assert.Len(t, diff, 0)

			return expectedConfig, nil
		})

		importer := &cesGlobalConfigImporter{
			globalConfigRepo: mockConfigRepo,
		}

		keys := []string{
			"fqdn",
			"certificate/server.crt",
			"certificate/server.key",
			"certificate/type",
		}

		// when
		err = importer.backupGlobalConfigByKeys(testCtx, keys, Backup)
		// then
		require.NoError(t, err)
	})

	t.Run("should cleanup global successfully", func(t *testing.T) {
		// given
		emptyConfig := regConfig.CreateGlobalConfig(map[regConfig.Key]regConfig.Value{})
		newCfg, err := emptyConfig.Set("fqdn", "new.test.ces.importer")
		emptyConfig = regConfig.GlobalConfig{Config: newCfg}
		newCfg, err = emptyConfig.Set("ces-import-backup/fqdn", "old.test.ces.importer")
		emptyConfig = regConfig.GlobalConfig{Config: newCfg}

		mockConfigRepo := newMockGlobalConfigRepo(t)
		mockConfigRepo.EXPECT().Get(testCtx).Return(emptyConfig, nil)

		expectedConfig := regConfig.CreateGlobalConfig(map[regConfig.Key]regConfig.Value{})
		newCfg, err = expectedConfig.Set("fqdn", "new.test.ces.importer")
		expectedConfig = regConfig.GlobalConfig{Config: newCfg}

		require.NoError(t, err)
		mockConfigRepo.EXPECT().SaveOrMerge(testCtx, mock.Anything).RunAndReturn(func(ctx context.Context, config regConfig.GlobalConfig) (regConfig.GlobalConfig, error) {
			assert.Equal(t, testCtx, ctx)

			diff := expectedConfig.Diff(config.Config)
			assert.Len(t, diff, 0)

			return expectedConfig, nil
		})

		importer := &cesGlobalConfigImporter{
			globalConfigRepo: mockConfigRepo,
		}

		keys := []string{
			"fqdn",
		}

		// when
		err = importer.backupGlobalConfigByKeys(testCtx, keys, Cleanup)
		// then
		require.NoError(t, err)
	})

	t.Run("should cleanup global successfully", func(t *testing.T) {
		// given
		emptyConfig := regConfig.CreateGlobalConfig(map[regConfig.Key]regConfig.Value{})
		newCfg, err := emptyConfig.Set("fqdn", "new.test.ces.importer")
		emptyConfig = regConfig.GlobalConfig{Config: newCfg}
		newCfg, err = emptyConfig.Set("ces-import-backup/fqdn", "old.test.ces.importer")
		emptyConfig = regConfig.GlobalConfig{Config: newCfg}

		mockConfigRepo := newMockGlobalConfigRepo(t)
		mockConfigRepo.EXPECT().Get(testCtx).Return(emptyConfig, nil)

		expectedConfig := regConfig.CreateGlobalConfig(map[regConfig.Key]regConfig.Value{})
		newCfg, err = expectedConfig.Set("fqdn", "old.test.ces.importer")
		expectedConfig = regConfig.GlobalConfig{Config: newCfg}

		require.NoError(t, err)

		callorder := 1

		mockConfigRepo.EXPECT().SaveOrMerge(testCtx, mock.Anything).RunAndReturn(func(ctx context.Context, config regConfig.GlobalConfig) (regConfig.GlobalConfig, error) {
			assert.Equal(t, testCtx, ctx)

			diff := expectedConfig.Diff(config.Config)
			// SaveAndMerge is called twice since the cleanup is defered in the Restore call.
			// Therefor on the firstcall there is a diff of one entry but after the second call the list is empty due to the cleanup
			assert.Len(t, diff, callorder)
			callorder--
			return expectedConfig, nil
		})

		importer := &cesGlobalConfigImporter{
			globalConfigRepo: mockConfigRepo,
		}

		keys := []string{
			"fqdn",
		}

		// when
		err = importer.backupGlobalConfigByKeys(testCtx, keys, Restore)
		// then
		require.NoError(t, err)
		// because SaveAndMerge should be called twice
		assert.Equal(t, -1, callorder)
	})

}
