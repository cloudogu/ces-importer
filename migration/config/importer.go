package configuration

import (
	"context"
	"fmt"
	"github.com/cloudogu/k8s-registry-lib/repository"
)

type configGetter interface {
	GetConfig(ctx context.Context) (*configuration, error)
}

type ConfigImporter struct {
	getter           configGetter
	globalConfigRepo *repository.GlobalConfigRepository
}

func (ci *ConfigImporter) importConfiguration(ctx context.Context) error {
	config, err := ci.getter.GetConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to get configuration from exporter: %w", err)
	}

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

func (ci *ConfigImporter) importDoguConfig(ctx context.Context, config []doguConfig) error {

	return nil
}

func (ci *ConfigImporter) importBackupSchedules(ctx context.Context, config []backupSchedule) error {

	return nil
}
