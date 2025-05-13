package main

import (
	"context"
	"fmt"
	"github.com/cloudogu/ces-importer/api/exporter"
	"github.com/cloudogu/ces-importer/configuration"
	migrationConfig "github.com/cloudogu/ces-importer/migration/config"
	"os/exec"

	backupEcosystem "github.com/cloudogu/k8s-backup-operator/pkg/api/ecosystem"

	"github.com/cloudogu/ces-importer/sync"
	"github.com/cloudogu/k8s-registry-lib/repository"
	"k8s.io/client-go/kubernetes"
	"net/http"
	ctrl "sigs.k8s.io/controller-runtime"
)

type dataSyncer interface {
	SyncData(ctx context.Context) error
}

type configSyncer interface {
	SyncConfig(ctx context.Context) error
}

type ImportExecutor struct {
	configSyncer
	dataSyncer
}

func NewImportExecutor() (*ImportExecutor, error) {
	jobConfig, err := configuration.ReadJobConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to read job configuration: %w", err)
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

	exporterApiClient := exporter.NewClient(jobConfig.API.ExporterApiKey, http.DefaultClient)
	globalConfigRepo := repository.NewGlobalConfigRepository(configMaps)
	doguConfigRepo := repository.NewDoguConfigRepository(configMaps)
	sensitiveDoguConfigRepo := repository.NewSensitiveDoguConfigRepository(secrets)

	backupClient, err := backupEcosystem.NewForConfig(clusterConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create backup schedule client: %w", err)
	}
	backupScheduleClient := backupClient.BackupSchedules(jobConfig.Namespace)

	commandMaker := func(name string, arg ...string) sync.Command {
		return exec.Command(name, arg...)
	}

	exportDoguApiClient := exporter.NewExportDoguClient(exporterApiClient, jobConfig.API.ExporterHost)
	systemInfoApiClient := exporter.NewSystemInfoClient(exporterApiClient, jobConfig.API.ExporterHost)

	ds := sync.NewRsyncSyncer(jobConfig.API.ExporterHost, jobConfig.SSH.User, jobConfig.SSH.PrivateSSHKeyPath, commandMaker, exportDoguApiClient, systemInfoApiClient)

	cs := migrationConfig.NewConfigImporter(jobConfig.ExporterHost, exporterApiClient, globalConfigRepo, doguConfigRepo, sensitiveDoguConfigRepo, backupScheduleClient)

	return &ImportExecutor{
		configSyncer: cs,
		dataSyncer:   ds,
	}, nil
}

func (j ImportExecutor) Start(ctx context.Context) error {
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
