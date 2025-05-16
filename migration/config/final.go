package configuration

import (
	"context"
	"fmt"
	regConfig "github.com/cloudogu/k8s-registry-lib/config"
	"log/slog"
	"strings"
)

type BackupType int

const (
	Backup BackupType = iota
	Cleanup
	Restore
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

func (gci *cesGlobalConfigImporter) backupGlobalConfigByKeys(ctx context.Context, keys []string, backupType BackupType) (e error) {
	backupPrefix := "ces-import-backup/"
	gc, err := gci.globalConfigRepo.Get(ctx)
	if err != nil {
		return fmt.Errorf("failed to get global config: %w", err)
	}
	for key, value := range gc.GetAll() {
		switch backupType {
		case Backup:
			if matchesAnyKeyByPattern(key.String(), keys) {
				slog.Debug("Backup global config", "key", key, "value", value)
				newGlobalConfig, err := gc.Set(regConfig.Key(backupPrefix+key.String()), value)
				if err != nil {
					return fmt.Errorf("failed to set config key %s: %w", key, err)
				}
				gc = regConfig.GlobalConfig{Config: newGlobalConfig}
			}
		case Cleanup:
			if strings.HasPrefix(key.String(), backupPrefix) && matchesAnyKeyByPattern(key.String()[len(backupPrefix):], keys) {
				newGlobalConfig := gc.Delete(key)
				gc = regConfig.GlobalConfig{Config: newGlobalConfig}
			}
		case Restore:
			if strings.HasPrefix(key.String(), backupPrefix) && matchesAnyKeyByPattern(key.String()[len(backupPrefix):], keys) {
				newGlobalConfig, err := gc.Set(regConfig.Key(key.String()[len(backupPrefix):]), value)
				if err != nil {
					return fmt.Errorf("failed to set config key %s: %w", key, err)
				}
				gc = regConfig.GlobalConfig{Config: newGlobalConfig}
				defer func() {
					e = gci.backupGlobalConfigByKeys(ctx, keys, Cleanup)
				}()
			}
		default:
			return fmt.Errorf("Invalid BackupType: %v: %w", backupType, err)
		}

	}
	_, err = gci.globalConfigRepo.SaveOrMerge(ctx, gc)
	if err != nil {
		return fmt.Errorf("failed to save backup of global config: %w", err)
	}

	return nil
}
