package configuration

import (
	"context"
	"fmt"
	"log/slog"

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
	globalConfigRepo globalConfigRepo
}

func (gci *cesGlobalConfigImporter) importGlobalConfig(ctx context.Context, config migration.GlobalConfig) error {
	slog.Info("Importing global config...")

	previousGlobalConfig, err := gci.globalConfigRepo.Get(ctx)
	if err != nil {
		return fmt.Errorf("failed to get global config: %w", err)
	}

	configToKeep := make(map[regConfig.Key]regConfig.Value)
	for key, value := range previousGlobalConfig.GetAll() {
		if matchesAnyKeyByPattern(key.String(), globalConfigKeysToKeep) {
			configToKeep[key] = value
		}
	}

	err = gci.globalConfigRepo.Delete(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete global config: %w", err)
	}

	gc, err := gci.globalConfigRepo.Create(ctx, regConfig.CreateGlobalConfig(map[regConfig.Key]regConfig.Value{}))
	if err != nil {
		return fmt.Errorf("failed to create global config: %w", err)
	}

	for kkey, kval := range configToKeep {
		slog.Debug("Setting previous global config-entry", "key", kkey, "value", kval)
		newGlobalConfig, err := gc.Set(kkey, kval)
		if err != nil {
			return fmt.Errorf("failed to set previous config key %s: %w", kkey, err)
		}

		gc = regConfig.GlobalConfig{Config: newGlobalConfig}
	}

	// import config from exporter
	for _, kv := range config {
		if matchesAnyKeyByPattern(kv.Key, globalConfigKeysToKeep) {
			slog.Debug("Ignoring global config-key", "key", kv.Key)
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
