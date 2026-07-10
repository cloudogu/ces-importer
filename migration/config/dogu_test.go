package configuration

import (
	"context"
	"os"
	"testing"

	doguCommons "github.com/cloudogu/ces-commons-lib/dogu"
	ceserrors "github.com/cloudogu/ces-commons-lib/errors"
	"github.com/cloudogu/ces-importer/migration"
	regConfig "github.com/cloudogu/k8s-registry-lib/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func Test_getLocalConfigFileForDogu(t *testing.T) {
	localConfigFile := getLocalConfigFileForDogu("/data", "cas")
	assert.Equal(t, "/data/cas/localConfig/local.yaml", localConfigFile)
}

func Test_importLocalConfig(t *testing.T) {
	t.Run("should not import local config if empty", func(t *testing.T) {

		tempDir := t.TempDir()
		localConfigFile := tempDir + "/local.yaml"

		originalGetLocalConfigFileForDogu := getLocalConfigFileForDogu
		getLocalConfigFileForDogu = func(dataBasePath string, dogu string) string {
			return localConfigFile
		}
		defer func() {
			getLocalConfigFileForDogu = originalGetLocalConfigFileForDogu
		}()

		err := importLocalConfig("", "cas", []migration.KeyValue{})
		require.NoError(t, err)

		_, err = os.ReadFile(localConfigFile)
		require.Error(t, err)
		assert.ErrorIs(t, err, os.ErrNotExist)
	})

	t.Run("should import local config", func(t *testing.T) {

		tempDir := t.TempDir()
		localConfigFile := tempDir + "/local.yaml"

		originalGetLocalConfigFileForDogu := getLocalConfigFileForDogu
		getLocalConfigFileForDogu = func(dataBasePath string, dogu string) string {
			return localConfigFile
		}
		defer func() {
			getLocalConfigFileForDogu = originalGetLocalConfigFileForDogu
		}()

		cfg := []migration.KeyValue{
			{"key1", "value1"},
			{"sub/key/foo", "bar"},
		}

		err := importLocalConfig("", "cas", cfg)
		require.NoError(t, err)

		file, err := os.ReadFile(localConfigFile)
		require.NoError(t, err)

		require.Equal(t, "key1: value1\nsub:\n    key:\n        foo: bar\n", string(file))
	})

	t.Run("should import local config for file that already exists", func(t *testing.T) {

		tempDir := t.TempDir()
		localConfigFile := tempDir + "/local.yaml"
		_, err := os.Create(localConfigFile)
		require.NoError(t, err)

		originalGetLocalConfigFileForDogu := getLocalConfigFileForDogu
		getLocalConfigFileForDogu = func(dataBasePath string, dogu string) string {
			return localConfigFile
		}
		defer func() {
			getLocalConfigFileForDogu = originalGetLocalConfigFileForDogu
		}()

		cfg := []migration.KeyValue{
			{"key1", "value1"},
			{"sub/key/foo", "bar"},
		}

		err = importLocalConfig("", "cas", cfg)
		require.NoError(t, err)

		file, err := os.ReadFile(localConfigFile)
		require.NoError(t, err)

		require.Equal(t, "key1: value1\nsub:\n    key:\n        foo: bar\n", string(file))
	})

	t.Run("should import local config and truncate existing file", func(t *testing.T) {

		tempDir := t.TempDir()
		localConfigFile := tempDir + "/local.yaml"
		_, err := os.Create(localConfigFile)
		require.NoError(t, err)
		err = os.WriteFile(localConfigFile, []byte("some previous content"), 0644)
		require.NoError(t, err)

		originalGetLocalConfigFileForDogu := getLocalConfigFileForDogu
		getLocalConfigFileForDogu = func(dataBasePath string, dogu string) string {
			return localConfigFile
		}
		defer func() {
			getLocalConfigFileForDogu = originalGetLocalConfigFileForDogu
		}()

		cfg := []migration.KeyValue{
			{"key1", "value1"},
			{"sub/key/foo", "bar"},
		}

		err = importLocalConfig("", "cas", cfg)
		require.NoError(t, err)

		file, err := os.ReadFile(localConfigFile)
		require.NoError(t, err)

		require.Equal(t, "key1: value1\nsub:\n    key:\n        foo: bar\n", string(file))
	})

	t.Run("should fail to import local config on error opening file", func(t *testing.T) {

		localConfigFile := "not-exists/local.yaml"

		originalGetLocalConfigFileForDogu := getLocalConfigFileForDogu
		getLocalConfigFileForDogu = func(dataBasePath string, dogu string) string {
			return localConfigFile
		}
		defer func() {
			getLocalConfigFileForDogu = originalGetLocalConfigFileForDogu
		}()

		cfg := []migration.KeyValue{
			{"key1", "value1"},
			{"sub/key/foo", "bar"},
		}

		err := importLocalConfig("", "cas", cfg)
		require.Error(t, err)
		assert.ErrorIs(t, err, os.ErrNotExist)
		assert.ErrorContains(t, err, "failed to open local config file at 'not-exists/local.yaml': open not-exists/local.yaml: no such file or directory")
	})
}

func Test_importDoguConfigWithRepo(t *testing.T) {
	testCtx := context.Background()

	t.Run("should import dogu config with repo", func(t *testing.T) {
		doguName := doguCommons.SimpleName("cas")

		newDoguCfg := regConfig.CreateDoguConfig(doguName, map[regConfig.Key]regConfig.Value{})

		mockRepo := newMockDoguConfigRepo(t)
		mockRepo.EXPECT().Get(testCtx, doguName).Return(newDoguCfg, nil)

		expectedDoguCfg := regConfig.CreateDoguConfig(doguName, map[regConfig.Key]regConfig.Value{
			"key1":        "value1",
			"sub/key/foo": "bar",
		})
		mockRepo.EXPECT().Update(testCtx, mock.Anything).RunAndReturn(func(ctx context.Context, config regConfig.DoguConfig) (regConfig.DoguConfig, error) {
			assert.Equal(t, testCtx, ctx)

			diff := expectedDoguCfg.Diff(config.Config)
			assert.Len(t, diff, 0)

			return expectedDoguCfg, nil
		})

		cfg := []migration.KeyValue{
			{"key1", "value1"},
			{"sub/key/foo", "bar"},
		}

		err := importDoguConfigWithRepo(testCtx, "cas", cfg, mockRepo, []string{})

		require.NoError(t, err)
	})

	t.Run("should fail import dogu config with repo on getting previous config", func(t *testing.T) {
		doguName := doguCommons.SimpleName("cas")

		newDoguCfg := regConfig.CreateDoguConfig(doguName, map[regConfig.Key]regConfig.Value{})

		mockRepo := newMockDoguConfigRepo(t)
		mockRepo.EXPECT().Get(testCtx, doguName).Return(newDoguCfg, assert.AnError)

		cfg := []migration.KeyValue{
			{"key1", "value1"},
			{"sub/key/foo", "bar"},
		}

		err := importDoguConfigWithRepo(testCtx, "cas", cfg, mockRepo, []string{})

		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to get original dogu config:")
	})

	t.Run("should fail import dogu config with repo on create new config", func(t *testing.T) {
		doguName := doguCommons.SimpleName("cas")

		newDoguCfg := regConfig.CreateDoguConfig(doguName, map[regConfig.Key]regConfig.Value{})

		mockRepo := newMockDoguConfigRepo(t)
		mockRepo.EXPECT().Get(testCtx, doguName).Return(newDoguCfg, ceserrors.NewNotFoundError(assert.AnError))

		mockRepo.EXPECT().Create(testCtx, newDoguCfg).Return(newDoguCfg, assert.AnError)

		cfg := []migration.KeyValue{
			{"key1", "value1"},
			{"sub/key/foo", "bar"},
		}

		err := importDoguConfigWithRepo(testCtx, "cas", cfg, mockRepo, []string{})

		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to create new dogu config:")
	})

	t.Run("should fail import dogu config with repo on save new config", func(t *testing.T) {
		doguName := doguCommons.SimpleName("cas")

		mockRepo := newMockDoguConfigRepo(t)
		newDoguCfg := regConfig.CreateDoguConfig(doguName, map[regConfig.Key]regConfig.Value{})
		mockRepo.EXPECT().Get(testCtx, doguName).Return(newDoguCfg, nil)

		expectedDoguCfg := regConfig.CreateDoguConfig(doguName, map[regConfig.Key]regConfig.Value{
			"key1":        "value1",
			"sub/key/foo": "bar",
		})
		mockRepo.EXPECT().Update(testCtx, mock.Anything).RunAndReturn(func(ctx context.Context, config regConfig.DoguConfig) (regConfig.DoguConfig, error) {
			assert.Equal(t, testCtx, ctx)

			diff := expectedDoguCfg.Diff(config.Config)
			assert.Len(t, diff, 0)

			return expectedDoguCfg, assert.AnError
		})

		cfg := []migration.KeyValue{
			{"key1", "value1"},
			{"sub/key/foo", "bar"},
		}

		err := importDoguConfigWithRepo(testCtx, "cas", cfg, mockRepo, []string{})

		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to save dogu config:")
	})
}

func TestConfigImporter_importDoguConfig(t *testing.T) {
	testCtx := context.Background()

	t.Run("should import dogu config", func(t *testing.T) {
		doguName := doguCommons.SimpleName("cas")

		mockNormalRepo := newMockDoguConfigRepo(t)
		newDoguCfg := regConfig.CreateDoguConfig(doguName, map[regConfig.Key]regConfig.Value{})
		mockNormalRepo.EXPECT().Get(testCtx, doguName).Return(newDoguCfg, nil)

		expectedNormalDoguCfg := regConfig.CreateDoguConfig(doguName, map[regConfig.Key]regConfig.Value{
			"key1":        "value1",
			"sub/key/foo": "bar",
		})
		mockNormalRepo.EXPECT().Update(testCtx, mock.Anything).RunAndReturn(func(ctx context.Context, config regConfig.DoguConfig) (regConfig.DoguConfig, error) {
			assert.Equal(t, testCtx, ctx)

			diff := expectedNormalDoguCfg.Diff(config.Config)
			assert.Len(t, diff, 0)

			return expectedNormalDoguCfg, nil
		})

		mockSensitiveRepo := newMockDoguConfigRepo(t)
		newDoguCfg = regConfig.CreateDoguConfig(doguName, map[regConfig.Key]regConfig.Value{})
		mockSensitiveRepo.EXPECT().Get(testCtx, doguName).Return(newDoguCfg, nil)
		expectedSensitiveDoguCfg := regConfig.CreateDoguConfig(doguName, map[regConfig.Key]regConfig.Value{
			"sensitive": "geheim",
		})
		mockSensitiveRepo.EXPECT().Update(testCtx, mock.Anything).RunAndReturn(func(ctx context.Context, config regConfig.DoguConfig) (regConfig.DoguConfig, error) {
			assert.Equal(t, testCtx, ctx)

			diff := expectedSensitiveDoguCfg.Diff(config.Config)
			assert.Len(t, diff, 0)

			return expectedSensitiveDoguCfg, nil
		})

		tempDir := t.TempDir()
		localConfigFile := tempDir + "/local.yaml"

		originalGetLocalConfigFileForDogu := getLocalConfigFileForDogu
		getLocalConfigFileForDogu = func(dataBasePath string, dogu string) string {
			return localConfigFile
		}
		defer func() {
			getLocalConfigFileForDogu = originalGetLocalConfigFileForDogu
		}()

		cfg := migration.DoguConfig{
			Name: "cas",
			NormalConfig: []migration.KeyValue{
				{"key1", "value1"},
				{"sub/key/foo", "bar"},
			},
			SensitiveConfig: []migration.KeyValue{
				{"sensitive", "geheim"},
			},
			LocalConfig: []migration.KeyValue{
				{"local", "lokal"},
			},
		}

		ci := &cesDoguConfigImporter{
			doguConfigRepo:          mockNormalRepo,
			sensitiveDoguConfigRepo: mockSensitiveRepo,
		}

		err := ci.importDoguConfig(testCtx, cfg)

		require.NoError(t, err)

		file, err := os.ReadFile(localConfigFile)
		require.NoError(t, err)

		require.Equal(t, "local: lokal\n", string(file))
	})

	t.Run("should fail to import dogu config on error in normal config", func(t *testing.T) {
		doguName := doguCommons.SimpleName("cas")

		mockNormalRepo := newMockDoguConfigRepo(t)
		newDoguCfg := regConfig.CreateDoguConfig(doguName, map[regConfig.Key]regConfig.Value{})
		mockNormalRepo.EXPECT().Get(testCtx, doguName).Return(newDoguCfg, assert.AnError)

		cfg := migration.DoguConfig{
			Name: "cas",
			NormalConfig: []migration.KeyValue{
				{"key1", "value1"},
				{"sub/key/foo", "bar"},
			},
			SensitiveConfig: []migration.KeyValue{
				{"sensitive", "geheim"},
			},
			LocalConfig: []migration.KeyValue{
				{"local", "lokal"},
			},
		}

		ci := &cesDoguConfigImporter{
			doguConfigRepo: mockNormalRepo,
		}

		err := ci.importDoguConfig(testCtx, cfg)

		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to import dogu config for dogu 'cas': failed to get original dogu config:")
	})

	t.Run("should fail to import dogu config on error in sensitive config", func(t *testing.T) {
		doguName := doguCommons.SimpleName("cas")

		mockNormalRepo := newMockDoguConfigRepo(t)
		newDoguCfg := regConfig.CreateDoguConfig(doguName, map[regConfig.Key]regConfig.Value{})
		mockNormalRepo.EXPECT().Get(testCtx, doguName).Return(newDoguCfg, nil)
		expectedNormalDoguCfg := regConfig.CreateDoguConfig(doguName, map[regConfig.Key]regConfig.Value{
			"key1":        "value1",
			"sub/key/foo": "bar",
		})
		mockNormalRepo.EXPECT().Update(testCtx, mock.Anything).RunAndReturn(func(ctx context.Context, config regConfig.DoguConfig) (regConfig.DoguConfig, error) {
			assert.Equal(t, testCtx, ctx)

			diff := expectedNormalDoguCfg.Diff(config.Config)
			assert.Len(t, diff, 0)

			return expectedNormalDoguCfg, nil
		})

		mockSensitiveRepo := newMockDoguConfigRepo(t)
		newDoguCfg = regConfig.CreateDoguConfig(doguName, map[regConfig.Key]regConfig.Value{})
		mockSensitiveRepo.EXPECT().Get(testCtx, doguName).Return(newDoguCfg, assert.AnError)

		cfg := migration.DoguConfig{
			Name: "cas",
			NormalConfig: []migration.KeyValue{
				{"key1", "value1"},
				{"sub/key/foo", "bar"},
			},
			SensitiveConfig: []migration.KeyValue{
				{"sensitive", "geheim"},
			},
			LocalConfig: []migration.KeyValue{
				{"local", "lokal"},
			},
		}

		ci := &cesDoguConfigImporter{
			doguConfigRepo:          mockNormalRepo,
			sensitiveDoguConfigRepo: mockSensitiveRepo,
		}

		err := ci.importDoguConfig(testCtx, cfg)

		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to import sensitive dogu config for dogu 'cas': failed to get original dogu config:")
	})

	t.Run("should not fail to import dogu config if localConfig does not exist", func(t *testing.T) {
		doguName := doguCommons.SimpleName("cas")

		mockNormalRepo := newMockDoguConfigRepo(t)
		newDoguCfg := regConfig.CreateDoguConfig(doguName, map[regConfig.Key]regConfig.Value{})
		mockNormalRepo.EXPECT().Get(testCtx, doguName).Return(newDoguCfg, nil)
		expectedNormalDoguCfg := regConfig.CreateDoguConfig(doguName, map[regConfig.Key]regConfig.Value{
			"key1":        "value1",
			"sub/key/foo": "bar",
		})
		mockNormalRepo.EXPECT().Update(testCtx, mock.Anything).RunAndReturn(func(ctx context.Context, config regConfig.DoguConfig) (regConfig.DoguConfig, error) {
			assert.Equal(t, testCtx, ctx)

			diff := expectedNormalDoguCfg.Diff(config.Config)
			assert.Len(t, diff, 0)

			return expectedNormalDoguCfg, nil
		})

		mockSensitiveRepo := newMockDoguConfigRepo(t)
		newDoguCfg = regConfig.CreateDoguConfig(doguName, map[regConfig.Key]regConfig.Value{})
		mockSensitiveRepo.EXPECT().Get(testCtx, doguName).Return(newDoguCfg, nil)
		expectedSensitiveDoguCfg := regConfig.CreateDoguConfig(doguName, map[regConfig.Key]regConfig.Value{
			"sensitive": "geheim",
		})
		mockSensitiveRepo.EXPECT().Update(testCtx, mock.Anything).RunAndReturn(func(ctx context.Context, config regConfig.DoguConfig) (regConfig.DoguConfig, error) {
			assert.Equal(t, testCtx, ctx)

			diff := expectedSensitiveDoguCfg.Diff(config.Config)
			assert.Len(t, diff, 0)

			return expectedSensitiveDoguCfg, nil
		})

		localConfigFile := "not_exists/local.yaml"

		originalGetLocalConfigFileForDogu := getLocalConfigFileForDogu
		getLocalConfigFileForDogu = func(dataBasePath string, dogu string) string {
			return localConfigFile
		}
		defer func() {
			getLocalConfigFileForDogu = originalGetLocalConfigFileForDogu
		}()

		cfg := migration.DoguConfig{
			Name: "cas",
			NormalConfig: []migration.KeyValue{
				{"key1", "value1"},
				{"sub/key/foo", "bar"},
			},
			SensitiveConfig: []migration.KeyValue{
				{"sensitive", "geheim"},
			},
			LocalConfig: []migration.KeyValue{
				{"local", "lokal"},
			},
		}

		ci := &cesDoguConfigImporter{
			doguConfigRepo:          mockNormalRepo,
			sensitiveDoguConfigRepo: mockSensitiveRepo,
		}

		err := ci.importDoguConfig(testCtx, cfg)

		require.NoError(t, err)
	})
}

func TestConfigImporter_importDoguConfigs(t *testing.T) {
	testCtx := context.Background()

	t.Run("should import dogu configs", func(t *testing.T) {
		doguNameCas := doguCommons.SimpleName("cas")

		mockNormalRepo := newMockDoguConfigRepo(t)
		expectedNormalDoguCfg := regConfig.CreateDoguConfig(doguNameCas, map[regConfig.Key]regConfig.Value{
			"key1":        "value1",
			"sub/key/foo": "bar",
		})

		newDoguCfg := regConfig.CreateDoguConfig(doguNameCas, map[regConfig.Key]regConfig.Value{})
		mockNormalRepo.EXPECT().Get(testCtx, doguNameCas).Return(newDoguCfg, nil)
		mockNormalRepo.EXPECT().Update(testCtx, mock.Anything).RunAndReturn(func(ctx context.Context, config regConfig.DoguConfig) (regConfig.DoguConfig, error) {
			assert.Equal(t, testCtx, ctx)

			diff := expectedNormalDoguCfg.Diff(config.Config)
			assert.Len(t, diff, 0)

			return expectedNormalDoguCfg, nil
		})

		mockSensitiveRepo := newMockDoguConfigRepo(t)
		expectedSensitiveDoguCfg := regConfig.CreateDoguConfig(doguNameCas, map[regConfig.Key]regConfig.Value{
			"sensitive": "geheim",
		})

		newDoguCfg = regConfig.CreateDoguConfig(doguNameCas, map[regConfig.Key]regConfig.Value{})
		mockSensitiveRepo.EXPECT().Get(testCtx, doguNameCas).Return(newDoguCfg, nil)
		mockSensitiveRepo.EXPECT().Update(testCtx, mock.Anything).RunAndReturn(func(ctx context.Context, config regConfig.DoguConfig) (regConfig.DoguConfig, error) {
			assert.Equal(t, testCtx, ctx)

			diff := expectedSensitiveDoguCfg.Diff(config.Config)
			assert.Len(t, diff, 0)

			return expectedSensitiveDoguCfg, nil
		})

		tempDir := t.TempDir()
		localConfigFile := tempDir + "/local.yaml"

		originalGetLocalConfigFileForDogu := getLocalConfigFileForDogu
		getLocalConfigFileForDogu = func(dataBasePath string, dogu string) string {
			return localConfigFile
		}
		defer func() {
			getLocalConfigFileForDogu = originalGetLocalConfigFileForDogu
		}()

		cfg := []migration.DoguConfig{
			{
				Name: "cas",
				NormalConfig: []migration.KeyValue{
					{"key1", "value1"},
					{"sub/key/foo", "bar"},
				},
				SensitiveConfig: []migration.KeyValue{
					{"sensitive", "geheim"},
				},
				LocalConfig: []migration.KeyValue{
					{"local", "lokal"},
				},
			},
			{
				Name: "nginx",
				NormalConfig: []migration.KeyValue{
					{"key1", "value1"},
					{"sub/key/foo", "bar"},
				},
				SensitiveConfig: []migration.KeyValue{
					{"sensitive", "geheim"},
				},
				LocalConfig: []migration.KeyValue{
					{"local", "lokal"},
				},
			},
		}

		ci := &cesDoguConfigImporter{
			doguConfigRepo:          mockNormalRepo,
			sensitiveDoguConfigRepo: mockSensitiveRepo,
		}

		err := ci.importDoguConfigs(testCtx, cfg)

		require.NoError(t, err)

		file, err := os.ReadFile(localConfigFile)
		require.NoError(t, err)

		require.Equal(t, "local: lokal\n", string(file))
	})

	t.Run("should import dogu configs except for excluded configuration keys", func(t *testing.T) {
		doguNameCas := doguCommons.SimpleName("cas")

		mockNormalRepo := newMockDoguConfigRepo(t)
		originalConfigs := map[regConfig.Key]regConfig.Value{
			"key1":             "importer-value",
			"sub/key/foo":      "importer-value",
			"excludedkey1":     "importer-value",
			"excluded/key/foo": "importer-value",
		}
		mockNormalRepo.EXPECT().Get(testCtx, doguNameCas).Return(regConfig.CreateDoguConfig("cas", originalConfigs), nil)
		var mergedConfig regConfig.DoguConfig
		mockNormalRepo.EXPECT().Update(testCtx, mock.Anything).RunAndReturn(func(ctx context.Context, config regConfig.DoguConfig) (regConfig.DoguConfig, error) {
			mergedConfig = config

			return mergedConfig, nil
		})

		mockSensitiveRepo := newMockDoguConfigRepo(t)
		originalSensitiveConfigs := map[regConfig.Key]regConfig.Value{
			"sensitivetoexclude": "importer-value",
			"sensitive":          "importer-value",
		}
		mockSensitiveRepo.EXPECT().Get(testCtx, doguNameCas).Return(regConfig.CreateDoguConfig("cas", originalSensitiveConfigs), nil)
		var mergedSensitiveConfig regConfig.DoguConfig
		mockSensitiveRepo.EXPECT().Update(testCtx, mock.Anything).RunAndReturn(func(ctx context.Context, config regConfig.DoguConfig) (regConfig.DoguConfig, error) {
			mergedSensitiveConfig = config

			return mergedSensitiveConfig, nil
		})

		cfg := []migration.DoguConfig{
			{
				Name: "cas",
				NormalConfig: []migration.KeyValue{
					{"key1", "exporter-value"},
					{"sub/key/foo", "exporter-value"},
					{"excludedkey1", "exporter-value"},
					{"excluded/key/foo", "exporter-value"},
					{"excluded/not-set", "exporter-value"},
				},
				SensitiveConfig: []migration.KeyValue{
					{"sensitive", "exporter-value"},
					{"sensitivetoexclude", "exporter-value"},
				},
			},
		}

		expectedNormalDoguCfg := regConfig.CreateDoguConfig(doguNameCas, map[regConfig.Key]regConfig.Value{
			"key1":             "exporter-value",
			"sub/key/foo":      "exporter-value",
			"excludedkey1":     "importer-value",
			"excluded/key/foo": "importer-value",
		})
		expectedSensitiveDoguCfg := regConfig.CreateDoguConfig(doguNameCas, map[regConfig.Key]regConfig.Value{
			"sensitive":          "exporter-value",
			"sensitivetoexclude": "importer-value",
		})

		excludedConfigs := map[string][]string{
			"cas": {"excludedkey1", "excluded/key/foo", "sensitivetoexclude", "excluded/not-set"},
		}
		ci := &cesDoguConfigImporter{
			doguConfigRepo:          mockNormalRepo,
			sensitiveDoguConfigRepo: mockSensitiveRepo,
			excludedDoguConfigKeys:  excludedConfigs,
		}

		err := ci.importDoguConfigs(testCtx, cfg)

		require.NoError(t, err)
		assert.Equal(t, expectedNormalDoguCfg.Config.GetAll(), mergedConfig.Config.GetAll())
		assert.Equal(t, expectedSensitiveDoguCfg.Config.GetAll(), mergedSensitiveConfig.Config.GetAll())
	})

	t.Run("should fail import dogu configs", func(t *testing.T) {
		doguNameCas := doguCommons.SimpleName("cas")

		mockNormalRepo := newMockDoguConfigRepo(t)
		newDoguCfg := regConfig.CreateDoguConfig(doguNameCas, map[regConfig.Key]regConfig.Value{})
		mockNormalRepo.EXPECT().Get(testCtx, doguNameCas).Return(newDoguCfg, assert.AnError)

		cfg := []migration.DoguConfig{
			{
				Name: "cas",
				NormalConfig: []migration.KeyValue{
					{"key1", "value1"},
					{"sub/key/foo", "bar"},
				},
				SensitiveConfig: []migration.KeyValue{
					{"sensitive", "geheim"},
				},
				LocalConfig: []migration.KeyValue{
					{"local", "lokal"},
				},
			},
		}

		ci := &cesDoguConfigImporter{
			doguConfigRepo: mockNormalRepo,
		}

		err := ci.importDoguConfigs(testCtx, cfg)

		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to import dogu config for dogu 'cas':")
	})
}
