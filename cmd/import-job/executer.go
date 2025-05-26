package main

import (
	"context"
	"fmt"
	"github.com/cloudogu/ces-importer/api/exporter"
	"github.com/cloudogu/ces-importer/configuration"
	migrationConfig "github.com/cloudogu/ces-importer/migration/config"
	"github.com/cloudogu/ces-importer/sync"
	"github.com/cloudogu/k8s-registry-lib/repository"
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

func NewImportExecutor(cfg configuration.Job, apiService *exporter.Service, k8sClientSet k8sClients) (*ImportExecutor, error) {
	globalConfigRepo := repository.NewGlobalConfigRepository(k8sClientSet.configMap)
	doguConfigRepo := repository.NewDoguConfigRepository(k8sClientSet.configMap)
	sensitiveDoguConfigRepo := repository.NewSensitiveDoguConfigRepository(k8sClientSet.secret)

	ds := sync.NewRsyncSyncer(jobConfig.API.ExporterHost, jobConfig.SSH.User, jobConfig.SSH.PrivateSSHKeyPath, exportDoguApiClient, systemInfoApiClient, jobConfig.Exclude, jobConfig.DoguVolumeBasePath)

	cs := migrationConfig.NewConfigImporter(jobConfig.DoguVolumeBasePath, exporterService.ConfigService, globalConfigRepo, doguConfigRepo, sensitiveDoguConfigRepo, backupScheduleClient)

	return &ImportExecuter{
		configSyncer: cs,
		dataSyncer:   ds,
	}, nil
}

func (j ImportExecutor) Start(ctx context.Context) error {
	err := j.dataSyncer.SyncData(ctx)
	if err != nil {
		return fmt.Errorf("failed to sync data: %w", err)
	}

	err = j.configSyncer.SyncConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to sync configuration: %w", err)
	}

	return nil
}
