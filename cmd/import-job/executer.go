package main

import (
	"context"
	"fmt"
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

func NewImportExecuter(cfg configuration.Job, apiService apiService, k8sClientSet k8sClients) *ImportExecuter {
	globalConfigRepo := repository.NewGlobalConfigRepository(k8sClientSet.configMap)
	doguConfigRepo := repository.NewDoguConfigRepository(k8sClientSet.configMap)
	sensitiveDoguConfigRepo := repository.NewSensitiveDoguConfigRepository(k8sClientSet.secret)

	ds := sync.NewRsyncSyncer(cfg.API.ExporterHost, cfg.SSH.User, cfg.SSH.PrivateSSHKeyPath, apiService.dogu, apiService.system, cfg.Exclude, cfg.DoguVolumeBasePath)
	cs := migrationConfig.NewConfigImporter(cfg.DoguVolumeBasePath, apiService.config, globalConfigRepo, doguConfigRepo, sensitiveDoguConfigRepo, k8sClientSet.backupSchedule)
	fc := migrationFQDN.NewService(apiService.config, globalConfigRepo, k8sClientSet.configMap, k8sClientSet.secret)

	return &ImportExecuter{
		configSyncer: cs,
		dataSyncer:   ds,
		fqdnChanger:  fc,
	}
}

func (j ImportExecuter) Start(ctx context.Context) error {
	slog.Info("Starting data and configuration sync.")

	if err := j.dataSyncer.SyncData(ctx); err != nil {
		return fmt.Errorf("failed to sync data: %w", err)
	}

	slog.Info("Dogu data has been synced.")

	if err := j.configSyncer.SyncConfig(ctx); err != nil {
		return fmt.Errorf("failed to sync configuration: %w", err)
	}

	slog.Info("Configuration has been synced.")

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
