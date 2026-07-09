package configuration

import (
	"context"
	"fmt"
	"log/slog"

	"slices"
	"strings"

	"github.com/cloudogu/ces-importer/migration"
	regConfig "github.com/cloudogu/k8s-registry-lib/config"
)

var globalConfigKeysToKeep = []string{
	"certificate/*",
	"k8s/*",
	"proxy/*",
	"fqdn",
	"alternativeFQDNs",
	"maintenance",
}

type cesGlobalConfigImporter struct {
	globalConfigRepo     globalConfigRepo
	additionalKeysToKeep []string
}

func (gci *cesGlobalConfigImporter) importGlobalConfig(ctx context.Context, config migration.GlobalConfig) error {
	slog.Info("Importing global config...")

	previousGlobalConfig, err := gci.globalConfigRepo.Get(ctx)
	if err != nil {
		return fmt.Errorf("failed to get global config: %w", err)
	}

	err = gci.globalConfigRepo.Delete(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete global config: %w", err)
	}

	err = migration.WaitForDeletion(func() error {
		_, timeoutError := gci.globalConfigRepo.Get(ctx)
		return timeoutError
	})
	if err != nil {
		return fmt.Errorf("failed to delete previous global config after timeout: %w", err)
	}

	gc, err := gci.globalConfigRepo.Create(ctx, regConfig.CreateGlobalConfig(map[regConfig.Key]regConfig.Value{}))
	if err != nil {
		return fmt.Errorf("failed to create global config: %w", err)
	}

	gc, err = gci.addConfigToKeepToRegistry(previousGlobalConfig, gc)
	if err != nil {
		return err
	}

	// import config from exporter
	for _, kv := range config {
		if matchesAnyKeyByPattern(kv.Key, globalConfigKeysToKeep) {
			slog.Debug("Ignoring global config-key from global exclude list", "key", kv.Key)
			continue
		}
		if slices.Contains(gci.additionalKeysToKeep, strings.TrimPrefix(kv.Key, "/")) {
			slog.Debug("Ignoring global config-key from additional exclude list", "key", kv.Key)
			continue
		}

		slog.Debug("Setting global config", "key", kv.Key, "value", kv.Value)
		newGlobalConfig, err := gc.Set(regConfig.Key(kv.Key), regConfig.Value(kv.Value))
		if err != nil {
			return fmt.Errorf("failed to set config key %s: %w", kv.Key, err)
		}

		gc = regConfig.GlobalConfig{Config: newGlobalConfig}
	}

	_, err = gci.globalConfigRepo.SaveOrMerge(ctx, gc)
	if err != nil {
		return fmt.Errorf("failed to save new global config: %w", err)
	}

	slog.Info("...Successfully imported global config.")
	return nil
}

// addConfigToKeepToRegistry adds the config entries from the previous global config to the new global config for the
// keys that should not be overwritten by the exporting system configuration
func (gci *cesGlobalConfigImporter) addConfigToKeepToRegistry(previousGlobalConfig regConfig.GlobalConfig, gc regConfig.GlobalConfig) (regConfig.GlobalConfig, error) {
	configToKeep := make(map[regConfig.Key]regConfig.Value)
	for key, value := range previousGlobalConfig.GetAll() {
		if matchesAnyKeyByPattern(key.String(), globalConfigKeysToKeep) || slices.Contains(gci.additionalKeysToKeep, strings.TrimPrefix(key.String(), "/")) {
			configToKeep[key] = value
			slog.Debug("Setting previous global config-entry", "key", key.String(), "value", value.String())
			newGlobalConfig, err := gc.Set(key, value)
			if err != nil {
				return regConfig.GlobalConfig{}, fmt.Errorf("failed to set previous config key %s: %w", key.String(), err)
			}

			gc = regConfig.GlobalConfig{Config: newGlobalConfig}
		}
	}
	return gc, nil
}
