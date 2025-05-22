package main

import (
	"context"
	"fmt"
	"github.com/cloudogu/ces-importer/api/exporter"
	"github.com/cloudogu/ces-importer/configuration"
	"github.com/cloudogu/ces-importer/logging"
	migration "github.com/cloudogu/ces-importer/migration"
	migrationConfig "github.com/cloudogu/ces-importer/migration/config"
	"log/slog"

	backupEcosystem "github.com/cloudogu/k8s-backup-operator/pkg/api/ecosystem"

	"github.com/cloudogu/ces-importer/sync"
	"github.com/cloudogu/k8s-registry-lib/repository"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
)

type dataSyncer interface {
	SyncData(ctx context.Context) error
}

type configSyncer interface {
	SyncConfig(ctx context.Context) error
	SyncCertificates(ctx context.Context) error
	ChangeFQDN(ctx context.Context) error
	Backup(ctx context.Context, backupType migrationConfig.BackupType) error
}

type ImportExecutor struct {
	configSyncer
	dataSyncer
	coordinator configuration.Coordinator
}

func NewImportExecutor() (*ImportExecutor, error) {
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

	_ = sync.NewRsyncSyncer(jobConfig.API.ExporterHost, jobConfig.SSH.User, jobConfig.SSH.PrivateSSHKeyPath)

	cs := migrationConfig.NewConfigImporter(jobConfig.DoguVolumeBasePath, exporterService.ConfigService, globalConfigRepo, doguConfigRepo, sensitiveDoguConfigRepo, backupScheduleClient)

	co, err := configuration.ReadCoordinatorConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get coordinator configuration: %w", err)
	}

	return &ImportExecutor{
		configSyncer: cs,
		coordinator:  co,
	}, nil
}

func createAPIClient(apiCfg configuration.API) *exporter.Client {
	var options []exporter.HTTPClientOption

	if apiCfg.SkipTLSVerify {
		options = append(options, exporter.WithInsecure())
	}

	return exporter.NewClient(apiCfg.ExporterHost, apiCfg.ExporterHost, options...)
}

func (j ImportExecutor) Start(ctx context.Context) (e error) {
	err := j.configSyncer.SyncConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to sync configuration: %w", err)
	}

	//err = j.dataSyncer.SyncData(ctx)
	//if err != nil {
	//	return fmt.Errorf("failed to sync data: %w", err)
	//}

	if j.coordinator.Migration.ChangeFQDN && migration.IsFinalMigration(ctx) {
		slog.Info("Change FQDN in final Migration")
		err = j.configSyncer.Backup(ctx, migrationConfig.Backup)
		// The defer function is only started, if we are in a final migration
		// therefor it do not check for final migration again
		defer func() {
			slog.Info("Run Cleanup after final Migration")
			if e != nil {
				slog.Info("There is an error during fqdn or certificate sync -> Restore old values and cleanup backup")
				err = j.configSyncer.Backup(ctx, migrationConfig.Restore)
				if err != nil {
					slog.Error("failed to restore backup of fqdn and certificates in final migration: %w because of %w", err, e)
					e = fmt.Errorf("failed to restore backup of fqdn and certificates in final migration: %w because of %w", err, e)
				}
			} else {
				slog.Info("There is no error during fqdn or certificate sync -> just cleanup backup")
				err = j.configSyncer.Backup(ctx, migrationConfig.Cleanup)
				if err != nil {
					slog.Error("failed to cleanup backup of fqdn and certificates in final migration: %w because of %w", err, e)
					e = fmt.Errorf("failed to cleanup backup of fqdn and certificates in final migration: %w because of %w", err, e)
				}
			}
		}()
		if err != nil {
			slog.Error("failed to backup fqdn and certificates in final migration: %w", err)
			return fmt.Errorf("failed to backup fqdn and certificates in final migration: %w", err)
		}
		err = j.configSyncer.SyncCertificates(ctx)
		if err != nil {
			slog.Error("failed to sync certificates in final migration: %w", err)
			return fmt.Errorf("failed to sync certificates in final migration: %w", err)
		}
		err = j.configSyncer.ChangeFQDN(ctx)
		if err != nil {
			slog.Error("failed to change fqdn in final migration: %w", err)
			return fmt.Errorf("failed to change fqdn in final migration: %w", err)
		}
	}

	return nil
}
