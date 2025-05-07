package configuration

import (
	"context"
	"fmt"
	"github.com/cloudogu/ces-commons-lib/dogu"
	regConfig "github.com/cloudogu/k8s-registry-lib/config"
	"path"
)

type configGetter interface {
	GetConfig(ctx context.Context) (*configuration, error)
}

type globalConfigRepo interface {
	Get(ctx context.Context) (regConfig.GlobalConfig, error)
	Create(ctx context.Context, globalConfig regConfig.GlobalConfig) (regConfig.GlobalConfig, error)
	SaveOrMerge(ctx context.Context, globalConfig regConfig.GlobalConfig) (regConfig.GlobalConfig, error)
	Delete(ctx context.Context) error
}

type doguConfigRepo interface {
	Create(ctx context.Context, doguConfig regConfig.DoguConfig) (regConfig.DoguConfig, error)
	SaveOrMerge(ctx context.Context, doguConfig regConfig.DoguConfig) (regConfig.DoguConfig, error)
	Delete(ctx context.Context, name dogu.SimpleName) error
}

type globalConfigImporter interface {
	importGlobalConfig(ctx context.Context, config globalConfig) error
}

type doguConfigImporter interface {
	importDoguConfigs(ctx context.Context, config []doguConfig) error
}

type backupScheduleImporter interface {
	importBackupSchedules(ctx context.Context, config []backupSchedule) error
}

type ConfigImporter struct {
	getter                 configGetter
	globalConfigImporter   globalConfigImporter
	doguConfigImporter     doguConfigImporter
	backupScheduleImporter backupScheduleImporter
}

func NewConfigImporter(exporterHost string, apiClient exporterApiClient, globalConfigRepo globalConfigRepo, doguConfigRepo doguConfigRepo, sensitiveDoguConfigRepo doguConfigRepo, backupScheduleClient backupScheduleClient) *ConfigImporter {
	getter := newExporterConfigGetter(exporterHost, apiClient)
	gci := &cesGlobalConfigImporter{globalConfigRepo}
	dci := &cesDoguConfigImporter{doguConfigRepo, sensitiveDoguConfigRepo}
	bsi := &cesBackupScheduleImporter{backupScheduleClient: backupScheduleClient}

	return &ConfigImporter{
		getter:                 getter,
		globalConfigImporter:   gci,
		doguConfigImporter:     dci,
		backupScheduleImporter: bsi,
	}
}

func (ci *ConfigImporter) ImportConfiguration(ctx context.Context) error {
	config, err := ci.getter.GetConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to get configuration from exporter: %w", err)
	}

	mergeNginxExternalsConfigIntoGlobalConfig(config)

	if err := ci.globalConfigImporter.importGlobalConfig(ctx, config.GlobalConfig); err != nil {
		return fmt.Errorf("failed to import global configuration: %w", err)
	}

	if err := ci.doguConfigImporter.importDoguConfigs(ctx, config.DoguConfigs); err != nil {
		return fmt.Errorf("failed to import dogu configuration: %w", err)
	}

	if err := ci.backupScheduleImporter.importBackupSchedules(ctx, config.BackupSchedules); err != nil {
		return fmt.Errorf("failed to import backup schedules: %w", err)
	}

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
