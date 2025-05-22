package main

import (
	"context"
	"fmt"
	"github.com/cloudogu/ces-importer/api/exporter"
	"github.com/cloudogu/ces-importer/configuration"
	"github.com/cloudogu/ces-importer/logging"
	migrationConfig "github.com/cloudogu/ces-importer/migration/config"
	"github.com/cloudogu/ces-importer/sync"
	backupEcosystem "github.com/cloudogu/k8s-backup-operator/pkg/api/ecosystem"
	"github.com/cloudogu/k8s-registry-lib/repository"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
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

func NewImportExecuter() (*ImportExecuter, error) {
	jobConfig, err := configuration.ReadJobConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to read job configuration: %w", err)
	}

	logInitializer := logging.NewLogInitializer(jobConfig.Logging.Level)
	err = logInitializer.Initialize()
	if err != nil {
		panic(err)
	}

	clusterConfig, err := ctrl.GetConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get k8s cluster config: %w", err)
	}

	k8sClient, err := kubernetes.NewForConfig(clusterConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create k8s client: %w", err)
	}

	configMaps := k8sClient.CoreV1().ConfigMaps(jobConfig.Namespace)
	secrets := k8sClient.CoreV1().Secrets(jobConfig.Namespace)

	exporterApiClient := createAPIClient(jobConfig.API)
	exporterService := exporter.NewService(exporterApiClient)

	globalConfigRepo := repository.NewGlobalConfigRepository(configMaps)
	doguConfigRepo := repository.NewDoguConfigRepository(configMaps)
	sensitiveDoguConfigRepo := repository.NewSensitiveDoguConfigRepository(secrets)

	backupClient, err := backupEcosystem.NewForConfig(clusterConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create backup schedule client: %w", err)
	}
	backupScheduleClient := backupClient.BackupSchedules(jobConfig.Namespace)

	exportDoguApiClient := exporter.NewExportDoguClient(exporterApiClient)
	systemInfoApiClient := exporter.NewSystemInfoClient(exporterApiClient)

	ds := sync.NewRsyncSyncer(jobConfig.API.ExporterHost, jobConfig.SSH.User, jobConfig.SSH.PrivateSSHKeyPath, exportDoguApiClient, systemInfoApiClient, jobConfig.Exclude, jobConfig.DoguVolumeBasePath)

	cs := migrationConfig.NewConfigImporter(jobConfig.DoguVolumeBasePath, exporterService.ConfigService, globalConfigRepo, doguConfigRepo, sensitiveDoguConfigRepo, backupScheduleClient)

	return &ImportExecuter{
		configSyncer: cs,
		dataSyncer:   ds,
	}, nil
}

func createAPIClient(apiCfg configuration.API) *exporter.Client {
	var options []exporter.HTTPClientOption

	if apiCfg.SkipTLSVerify {
		options = append(options, exporter.WithInsecure())
	}

	return exporter.NewClient(apiCfg.ExporterHost, apiCfg.ExporterHost, options...)
}

func (j ImportExecuter) Start(ctx context.Context) error {
	err := j.configSyncer.SyncConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to sync configuration: %w", err)
	}

	err = j.dataSyncer.SyncData(ctx)
	if err != nil {
		return fmt.Errorf("failed to sync data: %w", err)
	}

	return nil
}
