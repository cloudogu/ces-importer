package configuration

import (
	"context"
	"fmt"
	regConfig "github.com/cloudogu/k8s-registry-lib/config"
	"log/slog"
)

func (gci *cesGlobalConfigImporter) importGlobalConfigByKeys(ctx context.Context, config globalConfig, keys []string) error {
	gc, err := gci.globalConfigRepo.Get(ctx)
	if err != nil {
		return fmt.Errorf("failed to get global config: %w", err)
	}

	// import config from exporter
	for _, kv := range config {
		if matchesAnyKeyByPattern(kv.Key, keys) {
			slog.Debug("Setting global config", "key", kv.Key, "value", kv.Value)
			newGlobalConfig, err := gc.Set(regConfig.Key(kv.Key), regConfig.Value(kv.Value))
			if err != nil {
				return fmt.Errorf("failed to set config key %s: %w", kv.Key, err)
			}

			gc = regConfig.GlobalConfig{Config: newGlobalConfig}
		}
	}

	_, err = gci.globalConfigRepo.SaveOrMerge(ctx, gc)
	if err != nil {
		return fmt.Errorf("failed to save new global config: %w", err)
	}

	return nil
}

func (gci *cesGlobalConfigImporter) importGlobalCertificates(ctx context.Context, config globalConfig) error {
	slog.Info("Importing global certificates...")

	var globalConfigKeysCerticate = []string{
		"certificate/*",
	}

	err := gci.importGlobalConfigByKeys(ctx, config, globalConfigKeysCerticate)

	slog.Info("...Successfully imported global certificates.")
	return err
}

func (gci *cesGlobalConfigImporter) importGlobalFQDN(ctx context.Context, config globalConfig) error {
	slog.Info("Importing fqdn...")

	var globalConfigKeysFQDN = []string{
		"fqdn",
	}

	err := gci.importGlobalConfigByKeys(ctx, config, globalConfigKeysFQDN)

	slog.Info("...Successfully imported fqdn.")
	return err
}
