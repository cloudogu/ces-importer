package configuration

import (
	"context"
	"fmt"
	"github.com/cloudogu/ces-commons-lib/dogu"
	"github.com/cloudogu/ces-importer/configuration"
	"github.com/cloudogu/ces-importer/migration"
	regConfig "github.com/cloudogu/k8s-registry-lib/config"
	"path"
	"strings"
)

type configGetter interface {
	GetConfig(ctx context.Context) (*migration.Configuration, error)
}

type globalConfigRepo interface {
	Get(ctx context.Context) (regConfig.GlobalConfig, error)
	Create(ctx context.Context, globalConfig regConfig.GlobalConfig) (regConfig.GlobalConfig, error)
	SaveOrMerge(ctx context.Context, globalConfig regConfig.GlobalConfig) (regConfig.GlobalConfig, error)
	Delete(ctx context.Context) error
}

type doguConfigRepo interface {
	Create(ctx context.Context, doguConfig regConfig.DoguConfig) (regConfig.DoguConfig, error)
	Get(ctx context.Context, name dogu.SimpleName) (regConfig.DoguConfig, error)
	SaveOrMerge(ctx context.Context, doguConfig regConfig.DoguConfig) (regConfig.DoguConfig, error)
	Delete(ctx context.Context, name dogu.SimpleName) error
}

type globalConfigImporter interface {
	importGlobalConfig(ctx context.Context, config migration.GlobalConfig) error
}

type doguConfigImporter interface {
	importDoguConfigs(ctx context.Context, config []migration.DoguConfig) error
}

type backupScheduleImporter interface {
	importBackupSchedules(ctx context.Context, config []migration.BackupSchedule) error
}

type ConfigImporter struct {
	getter                 configGetter
	globalConfigImporter   globalConfigImporter
	doguConfigImporter     doguConfigImporter
	backupScheduleImporter backupScheduleImporter
}

type ConfigRepos struct {
	globalConfigRepo        globalConfigRepo
	doguConfigRepo          doguConfigRepo
	sensitiveDoguConfigRepo doguConfigRepo
}

func NewConfigRepos(globalConfigRepo globalConfigRepo, doguConfigRepo doguConfigRepo, sensitiveDoguConfigRepo doguConfigRepo) *ConfigRepos {
	return &ConfigRepos{
		globalConfigRepo:        globalConfigRepo,
		doguConfigRepo:          doguConfigRepo,
		sensitiveDoguConfigRepo: sensitiveDoguConfigRepo,
	}
}

func NewConfigImporter(dataBasePath string, configGetter configGetter, repos ConfigRepos, backupScheduleClient backupScheduleClient, excludedGlobalConfigKeys []string, excludedDoguConfiguration []configuration.DoguConfigurationKeys) *ConfigImporter {
	excludedConfigKeysByDogu := make(map[string][]string)
	for _, configs := range excludedDoguConfiguration {
		excludedConfigKeysByDogu[configs.DoguName] = append(excludedConfigKeysByDogu[configs.DoguName], configs.Keys...)
	}

	gci := &cesGlobalConfigImporter{repos.globalConfigRepo, excludedGlobalConfigKeys}
	dci := &cesDoguConfigImporter{dataBasePath, repos.doguConfigRepo, repos.sensitiveDoguConfigRepo, excludedConfigKeysByDogu}
	bsi := &cesBackupScheduleImporter{backupScheduleClient: backupScheduleClient}

	return &ConfigImporter{
		getter:                 configGetter,
		globalConfigImporter:   gci,
		doguConfigImporter:     dci,
		backupScheduleImporter: bsi,
	}
}

func (ci *ConfigImporter) SyncConfig(ctx context.Context) error {
	config, err := ci.getter.GetConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to get configuration from exporter: %w", err)
	}

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
	// sanitize key
	key = strings.TrimPrefix(key, "/")

	for _, pattern := range keyPatterns {
		matched, err := path.Match(pattern, key)
		if err == nil && matched {
			return true
		}
	}
	return false
}
