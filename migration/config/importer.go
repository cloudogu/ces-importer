package configuration

import (
	"context"
	"fmt"
	"github.com/cloudogu/k8s-registry-lib/repository"
	"path"
)

type configGetter interface {
	GetConfig(ctx context.Context) (*configuration, error)
}

type ConfigImporter struct {
	getter                  configGetter
	globalConfigRepo        *repository.GlobalConfigRepository
	doguConfigRepo          *repository.DoguConfigRepository
	sensitiveDoguConfigRepo *repository.DoguConfigRepository
}

func (ci *ConfigImporter) importConfiguration(ctx context.Context) error {
	config, err := ci.getter.GetConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to get configuration from exporter: %w", err)
	}

	mergeNginxExternalsConfigIntoGlobalConfig(config)

	if err := ci.importGlobalConfig(ctx, config.GlobalConfig); err != nil {
		return fmt.Errorf("failed to import global configuration: %w", err)
	}

	if err := ci.importDoguConfig(ctx, config.DoguConfigs); err != nil {
		return fmt.Errorf("failed to import dogu configuration: %w", err)
	}

	if err := ci.importBackupSchedules(ctx, config.BackupSchedules); err != nil {
		return fmt.Errorf("failed to import backup schedules: %w", err)
	}

	return nil
}

func (ci *ConfigImporter) importBackupSchedules(ctx context.Context, config []backupSchedule) error {

	return nil
}

func matchesAnyKeyByPattern(key string, keyPatterns []string) bool {
	for _, pattern := range keyPatterns {
		matched, err := path.Match(pattern, key)
		if err == nil && matched {
			return true
		}
	}
	return false
}
