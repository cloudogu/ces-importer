package configuration

import (
	"context"
	regConfig "github.com/cloudogu/k8s-registry-lib/config"
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
