package main

import (
	"context"
	"fmt"
	"github.com/cloudogu/ces-importer/configuration"
	migrationConfig "github.com/cloudogu/ces-importer/migration/config"
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

type ImportExecuter struct {
	configSyncer
	dataSyncer
}

func NewImportExecuter(cfg configuration.Job, apiService apiService, k8sClientSet k8sClients) (*ImportExecuter, error) {
	globalConfigRepo := repository.NewGlobalConfigRepository(k8sClientSet.configMap)
	doguConfigRepo := repository.NewDoguConfigRepository(k8sClientSet.configMap)
	sensitiveDoguConfigRepo := repository.NewSensitiveDoguConfigRepository(k8sClientSet.secret)

	ds := sync.NewRsyncSyncer(cfg.API.ExporterHost, cfg.SSH.User, cfg.SSH.PrivateSSHKeyPath, apiService.dogu, apiService.system, cfg.Exclude, cfg.DoguVolumeBasePath)

	cs := migrationConfig.NewConfigImporter(cfg.DoguVolumeBasePath, apiService.config, globalConfigRepo, doguConfigRepo, sensitiveDoguConfigRepo, k8sClientSet.backupSchedule)

	return &ImportExecuter{
		configSyncer: cs,
		dataSyncer:   ds,
	}, nil
}

func (j ImportExecuter) Start(ctx context.Context) error {
	slog.Info("Starting data and configuration sync.")

	err := j.dataSyncer.SyncData(ctx)
	if err != nil {
		return fmt.Errorf("failed to sync data: %w", err)
	}

	slog.Info("Dogu data has been synced.")

	err = j.configSyncer.SyncConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to sync configuration: %w", err)
	}

	slog.Info("Configuration has been synced.")

	return nil
}
