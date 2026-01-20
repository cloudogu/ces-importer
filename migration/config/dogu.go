package configuration

import (
	"context"
	"errors"
	"fmt"
	doguCommons "github.com/cloudogu/ces-commons-lib/dogu"
	"github.com/cloudogu/ces-importer/migration"
	regConfig "github.com/cloudogu/k8s-registry-lib/config"
	"log/slog"
	"os"
	"path"
	"slices"
)

const (
	localConfigPathTemplate = "%s/localConfig/local.yaml"
	nginxDoguName           = "nginx"
)

var getLocalConfigFileForDogu = func(dataBasePath string, dogu string) string {
	doguPath := fmt.Sprintf(localConfigPathTemplate, dogu)

	return path.Join(dataBasePath, doguPath)
}

type cesDoguConfigImporter struct {
	dataBasePath            string
	doguConfigRepo          doguConfigRepo
	sensitiveDoguConfigRepo doguConfigRepo
	excludedDoguConfigKeys  map[string][]string
}

func (dci *cesDoguConfigImporter) importDoguConfigs(ctx context.Context, config []migration.DoguConfig) error {
	slog.Info("Importing dogu config...")

	for _, dc := range config {
		if dc.Name == nginxDoguName {
			continue
		} else {
			if err := dci.importDoguConfig(ctx, dc); err != nil {
				return fmt.Errorf("failed to import dogu config for dogu '%s': %w", dc.Name, err)
			}
		}
	}

	slog.Info("...Successfully imported dogu config.")
	return nil
}

func (dci *cesDoguConfigImporter) importDoguConfig(ctx context.Context, dc migration.DoguConfig) error {
	doguName := dc.Name

	excludedKeys := dci.excludedDoguConfigKeys[doguName]
	if err := importDoguConfigWithRepo(ctx, doguName, dc.NormalConfig, dci.doguConfigRepo, excludedKeys); err != nil {
		return fmt.Errorf("failed to import dogu config for dogu '%s': %w", doguName, err)
	}

	if err := importDoguConfigWithRepo(ctx, doguName, dc.SensitiveConfig, dci.sensitiveDoguConfigRepo, excludedKeys); err != nil {
		return fmt.Errorf("failed to import sensitive dogu config for dogu '%s': %w", doguName, err)
	}

	if err := importLocalConfig(dci.dataBasePath, doguName, dc.LocalConfig); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			slog.Debug("no local config found for dogu", "dogu", doguName)
			return nil
		}

		return fmt.Errorf("failed to import local config for dogu '%s': %w", doguName, err)
	}

	return nil
}

func importDoguConfigWithRepo(ctx context.Context, dogu string, exporterDoguConfig []migration.KeyValue, repo doguConfigRepo, excludedKeys []string) error {
	doguName := doguCommons.SimpleName(dogu)
	err := repo.Delete(ctx, doguName)
	if err != nil {
		return fmt.Errorf("failed to delete original dogu config: %w", err)
	}

	registryDoguConfig, err := repo.Create(ctx, regConfig.CreateDoguConfig(doguName, map[regConfig.Key]regConfig.Value{}))
	if err != nil {
		return fmt.Errorf("failed to create new dogu config: %w", err)
	}

	configValuesToSet := make(map[regConfig.Key]regConfig.Value)

	if len(excludedKeys) > 0 {
		configValuesToSet, err = getExcludedKeyValues(ctx, repo, doguName, excludedKeys, configValuesToSet)
		if err != nil {
			return err
		}
	}

	for _, kv := range exporterDoguConfig {
		key := kv.Key
		if slices.Contains(excludedKeys, key) {
			continue
		}
		configValuesToSet[regConfig.Key(kv.Key)] = regConfig.Value(kv.Value)
	}

	for keyToSet, valueToSet := range configValuesToSet {
		registryDoguConfig, err = setValueInRegistry(doguName, registryDoguConfig, keyToSet, valueToSet)
		if err != nil {
			return err
		}
	}

	_, err = repo.SaveOrMerge(ctx, registryDoguConfig)
	if err != nil {
		return fmt.Errorf("failed to save dogu config: %w", err)
	}

	return nil
}

func getExcludedKeyValues(ctx context.Context, repo doguConfigRepo, doguName doguCommons.SimpleName, excludedKeys []string, configValuesToSet map[regConfig.Key]regConfig.Value) (configValues map[regConfig.Key]regConfig.Value, err error) {
	originalValues, err := repo.Get(ctx, doguName)
	if err != nil {
		return configValuesToSet, fmt.Errorf("failed to get original dogu config: %w", err)
	}
	for _, excludedKey := range excludedKeys {
		slog.Debug("not importing config-key from exclude list, setting to old value", "key", excludedKey)
		original, exists := originalValues.Get(regConfig.Key(excludedKey))
		if exists {
			configValuesToSet[regConfig.Key(excludedKey)] = original
		} else {
			slog.Warn("config-key was excluded from import and not set in original config, it will not be created by import", "key", excludedKey)
			continue
		}
	}
	return configValuesToSet, nil
}

func setValueInRegistry(doguName doguCommons.SimpleName, registryDoguConfig regConfig.DoguConfig, key regConfig.Key, value regConfig.Value) (regConfig.DoguConfig, error) {
	regKey := regConfig.Key(key)
	slog.Debug("Setting dogu config", "key", key, "dogu", doguName)
	newDoguConfig, err := registryDoguConfig.Set(regKey, value)
	if err != nil {
		return regConfig.DoguConfig{}, fmt.Errorf("failed to set key %s: %w\n", key, err)
	}
	registryDoguConfig = regConfig.DoguConfig{
		DoguName: doguName,
		Config:   newDoguConfig,
	}
	return registryDoguConfig, nil
}

func importLocalConfig(dataBasePath string, dogu string, dc []migration.KeyValue) error {
	if len(dc) == 0 {
		return nil
	}

	localConfigFile := getLocalConfigFileForDogu(dataBasePath, dogu)

	file, err := os.OpenFile(localConfigFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to open local config file at '%s': %w", localConfigFile, err)
	}
	defer func(file *os.File) {
		if cErr := file.Close(); cErr != nil {
			slog.Warn("Failed to close local config file", "err", cErr)
		}
	}(file)

	cfgData := regConfig.Entries{}
	for _, kv := range dc {
		cfgData[regConfig.Key(kv.Key)] = regConfig.Value(kv.Value)
	}

	converter := &regConfig.YamlConverter{}
	if err := converter.Write(file, cfgData); err != nil {
		return fmt.Errorf("failed to write local config file to '%s': %w", localConfigFile, err)
	}

	return nil
}
