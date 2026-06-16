package configuration

import (
	"context"
	"errors"
	"fmt"
	"time"

	"log/slog"
	"os"
	"path"
	"slices"
	"strings"

	doguCommons "github.com/cloudogu/ces-commons-lib/dogu"
	"github.com/cloudogu/ces-importer/migration"
	regConfig "github.com/cloudogu/k8s-registry-lib/config"
)

const (
	localConfigPathTemplate           = "%s/localConfig/local.yaml"
	nginxDoguName                     = "nginx"
	deleteConfigMapTimeoutSeconds     = 10
	deleteConfigMapPollIntervalMillis = 200
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

// importDoguConfigs imports the configurations from the supplied array of  migration.DoguConfig objects
// into the appropiate dogu configuration repositories for the individual dogus
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

// importDoguConfig imports a single dogu configuration from a migration.DoguConfig object into the
// appropriate dogu configuration repositories for sensitive, normal, and local configuration
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

// importDoguConfigWithRepo imports a dogu configuration into a doguConfigRepo by deleting the original repo
// and creating and then filling a new one. Dogu configurations specifically excluded are set to the original values
// or skipped if they didn't exist before the import
func importDoguConfigWithRepo(ctx context.Context, dogu string, exporterDoguConfig []migration.KeyValue, repo doguConfigRepo, excludedKeys []string) error {
	doguName := doguCommons.SimpleName(dogu)

	var originalValues regConfig.DoguConfig
	var err error
	if len(excludedKeys) > 0 {
		// this is only needed if we need to recreate the old keys became some keys were excluded
		originalValues, err = repo.Get(ctx, doguName)
		if err != nil {
			return fmt.Errorf("failed to get original dogu config: %w", err)
		}
	}

	err = repo.Delete(ctx, doguName)
	if err != nil {
		return fmt.Errorf("failed to delete original dogu config: %w", err)
	}

	timeout := time.After(deleteConfigMapTimeoutSeconds * time.Second)

	for {
		_, terr := repo.Get(ctx, doguName)
		if terr != nil {
			// Objekt existiert nicht mehr
			break
		}

		select {
		case <-timeout:
			return fmt.Errorf("timeout waiting for dogu config deletion")
		case <-time.After(deleteConfigMapPollIntervalMillis * time.Millisecond):
		}
	}

	registryDoguConfig, err := repo.Create(ctx, regConfig.CreateDoguConfig(doguName, map[regConfig.Key]regConfig.Value{}))
	if err != nil {
		return fmt.Errorf("failed to create new dogu config: %w", err)
	}

	configValuesToSet := make(map[regConfig.Key]regConfig.Value)

	for _, kv := range exporterDoguConfig {
		keyToSet := regConfig.Key(kv.Key)
		if slices.Contains(excludedKeys, strings.TrimPrefix(keyToSet.String(), "/")) {
			setOriginalValueForKey(originalValues, keyToSet, configValuesToSet)
			continue
		}
		configValuesToSet[keyToSet] = regConfig.Value(kv.Value)
	}

	for keyToSet, valueToSet := range configValuesToSet {
		registryDoguConfig, err = setValueInRegistry(doguName, registryDoguConfig, keyToSet, valueToSet)
		if err != nil {
			return fmt.Errorf("failed to set key %s: %w\n", keyToSet.String(), err)
		}
	}

	_, err = repo.SaveOrMerge(ctx, registryDoguConfig)
	if err != nil {
		return fmt.Errorf("failed to save dogu config: %w", err)
	}

	return nil
}

// setOriginalValueForKey sets the value of a key to the original value from before the import if the key is
// marked as being excluded from importing. A warning is logged if the key was not set before the import and in
// that case, the value will not be set in the dogu configuration
func setOriginalValueForKey(originalValues regConfig.DoguConfig, keyToSet regConfig.Key, configValuesToSet map[regConfig.Key]regConfig.Value) {
	original, exists := originalValues.Get(keyToSet)
	if exists {
		slog.Debug("not importing config-key from exclude list, setting to old value", "doguName", originalValues.DoguName, "key", keyToSet.String())
		configValuesToSet[keyToSet] = original
	} else {
		slog.Warn("config-key was excluded from import and not set in original config, it will not be created by import", "doguName", originalValues.DoguName, "key", keyToSet.String())
	}
}

// setValueInRegistry sets a key-value pair in the dogu configuration registry
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
