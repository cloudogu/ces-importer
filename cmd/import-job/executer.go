package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/cloudogu/ces-importer/api/exporter"
	"github.com/cloudogu/ces-importer/api/importer"
	"github.com/cloudogu/ces-importer/configuration"
	"github.com/cloudogu/ces-importer/migration"
	migrationConfig "github.com/cloudogu/ces-importer/migration/config"
	migrationFQDN "github.com/cloudogu/ces-importer/migration/fqdn"
	"github.com/cloudogu/ces-importer/migration/sync"
	"github.com/cloudogu/k8s-registry-lib/repository"
	"log/slog"
)

type dataSyncer interface {
	SyncData(ctx context.Context) error
}

type configSyncer interface {
	SyncConfig(ctx context.Context) error
}

type fqdnChanger interface {
	ChangeFQDN(ctx context.Context) error
}

type ImportExecuter struct {
	configSyncer
	dataSyncer
	fqdnChanger
}

func NewImportExecuter(cfg configuration.Job, apiService *exporter.Service, k8sClientSet importer.K8sClients) *ImportExecuter {
	globalConfigRepo := repository.NewGlobalConfigRepository(k8sClientSet.ConfigMap)
	doguConfigRepo := repository.NewDoguConfigRepository(k8sClientSet.ConfigMap)
	sensitiveDoguConfigRepo := repository.NewSensitiveDoguConfigRepository(k8sClientSet.Secret)

	ds := sync.NewRsyncSyncer(cfg.API.ExporterHost, cfg.SSH.User, cfg.SSH.PrivateSSHKeyPath, apiService.ExportDoguService, apiService.SystemInfoService, cfg.Exclude, cfg.DoguVolumeBasePath, cfg.ExcludedDogus, cfg.Verbose)
	excludedConfig := migrationConfig.ExcludedConfig{ExcludedGlobalConfigKeys: cfg.AdditionalExcludedConfigurationKeys, ExcludedDoguConfigKeys: cfg.ExcludedDoguConfigurations}
	cs := migrationConfig.NewConfigImporter(cfg.DoguVolumeBasePath, apiService.ConfigService, globalConfigRepo, doguConfigRepo, sensitiveDoguConfigRepo, k8sClientSet.BackupSchedule, excludedConfig)
	fc := migrationFQDN.NewService(apiService.ConfigService, globalConfigRepo, k8sClientSet.ConfigMap, k8sClientSet.Secret)

	return &ImportExecuter{
		configSyncer: cs,
		dataSyncer:   ds,
		fqdnChanger:  fc,
	}
}

func (j ImportExecuter) Start(ctx context.Context) error {
	slog.Info("Starting data and configuration sync.")
	var syncError error

	if err := j.dataSyncer.SyncData(ctx); err != nil {
		syncError = errors.Join(syncError, err)
	} else {
		slog.Info("Dogu data has been synced successfully.")
	}

	if err := j.configSyncer.SyncConfig(ctx); err != nil {
		syncError = errors.Join(syncError, err)
	} else {
		slog.Info("Configuration has been synced successfully.")
	}

	if syncError != nil {
		return syncError
	}

	if !migration.TriggerFQDNChange(ctx) {
		slog.Info("No FQDN change triggered.")
		return nil
	}

	if err := j.ChangeFQDN(ctx); err != nil {
		return fmt.Errorf("failed to change fqdn: %w", err)
	}

	slog.Info("FQDN has been changed.")

	return nil
}
