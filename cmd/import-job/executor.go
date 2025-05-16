package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/cloudogu/ces-importer/api/exporter"
	"github.com/cloudogu/ces-importer/configuration"
	"github.com/cloudogu/ces-importer/logging"
	migrationConfig "github.com/cloudogu/ces-importer/migration/config"
	backupEcosystem "github.com/cloudogu/k8s-backup-operator/pkg/api/ecosystem"
	"log/slog"
	"os"
	"os/exec"

	"github.com/cloudogu/ces-importer/sync"
	"github.com/cloudogu/k8s-registry-lib/repository"
	"k8s.io/client-go/kubernetes"
	"net/http"
	ctrl "sigs.k8s.io/controller-runtime"
)

type dataSyncer interface {
	SyncData(ctx context.Context, config configuration.Job) error
}

type configSyncer interface {
	SyncConfig(ctx context.Context) error
}

type ImportExecutor struct {
	configSyncer
	dataSyncer
}

// createInsecureHTTPClient creates an HTTP client that accepts self-signed certificates
func createInsecureHTTPClient() *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	if transport.TLSClientConfig == nil {
		transport.TLSClientConfig = &tls.Config{}
	}
	transport.TLSClientConfig.InsecureSkipVerify = true
	return &http.Client{Transport: transport}
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

	exporterApiClient := exporter.NewClient(jobConfig.ExporterHost, jobConfig.ExporterApiKey, createInsecureHTTPClient())
	exporterService := exporter.NewService(exporterApiClient)

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
	systemInfoApiClient := exporter.NewSystemInfoClient(exporterApiClient)

	ds := sync.NewRsyncSyncer(jobConfig.API.ExporterHost, jobConfig.SSH.User, jobConfig.SSH.PrivateSSHKeyPath, commandMaker, exportDoguApiClient, systemInfoApiClient)

	cs := migrationConfig.NewConfigImporter(exporterService.ConfigApiClient, globalConfigRepo, doguConfigRepo, sensitiveDoguConfigRepo, backupScheduleClient)

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

	config, err := configuration.ReadJobConfig()
	if err != nil {
		return fmt.Errorf("failed to read job configuration: %w", err)
	}
	err = j.dataSyncer.SyncData(ctx, config)
	if err != nil {
		return fmt.Errorf("failed to sync data: %w", err)
	}

	return nil
}
