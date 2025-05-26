package main

import (
	"context"
	"fmt"
	"github.com/cloudogu/ces-importer/api/exporter"
	"github.com/cloudogu/ces-importer/configuration"
	"github.com/cloudogu/ces-importer/logging"
	backupEcosystem "github.com/cloudogu/k8s-backup-operator/pkg/api/ecosystem"
	"k8s.io/client-go/kubernetes"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"log/slog"
	"os"
	ctrl "sigs.k8s.io/controller-runtime"
)

func main() {
	os.Exit(run())
}

func run() int {
	slog.Info("New import job started.")
	ctx := context.Background()

	jobConfig, err := configuration.ReadJobConfig()
	if err != nil {
		slog.Error("failed to read job configuration", "cause", err)
		return 1
	}

	slog.Debug("Successfully read job configuration")

	logInitializer := logging.NewLogInitializer(jobConfig.Logging.Level)
	err = logInitializer.Initialize()
	if err != nil {
		slog.Error("failed to initialize logger", "cause", err)
		return 1
	}

	slog.Debug("Successfully initialized logger")

	exporterService := createAPIService(jobConfig.API)

	slog.Debug("Successfully created service for exporter API")

	k8sClientSet, err := createK8Sclientset(jobConfig.Namespace)
	if err != nil {
		slog.Error("failed to create k8s client set", "cause", err)
		return 1
	}

	slog.Debug("Successfully created k8s client set")

	importJob, err := NewImportExecutor(jobConfig, exporterService, k8sClientSet)
	if err != nil {
		slog.Error("failed to create executer for import", "cause", err)
		return 1
	}

	slog.Info("Import executer created, start data synchronization...")

	err = importJob.Start(ctx)
	if err != nil {
		slog.Error("Import job failed", "cause", err)
		return 1
	}

	slog.Info("Import job finished.")
	return 0

}

func createAPIClient(apiCfg configuration.API) *exporter.Client {
	var options []exporter.HTTPClientOption

	if apiCfg.SkipTLSVerify {
		options = append(options, exporter.WithInsecure())
	}

	return exporter.NewClient(apiCfg.ExporterHost, apiCfg.ExporterHost, options...)
}

func createAPIService(apiCfg configuration.API) *exporter.Service {
	exportClient := createAPIClient(apiCfg)
	exportService := exporter.NewService(exportClient)

	return exportService
}

type k8sClients struct {
	configMap      corev1.ConfigMapInterface
	secret         corev1.SecretInterface
	backupSchedule backupEcosystem.BackupScheduleInterface
}

func createK8Sclientset(namespace string) (k8sClients, error) {
	k8sRestConfig, err := ctrl.GetConfig()
	if err != nil {
		return k8sClients{}, fmt.Errorf("failed to read kube config: %w", err)
	}

	k8sClientSet, err := kubernetes.NewForConfig(k8sRestConfig)
	if err != nil {
		return k8sClients{}, fmt.Errorf("failed to create k8s client set: %w", err)
	}

	k8sCoreClient := k8sClientSet.CoreV1()
	k8sConfigMapClient := k8sCoreClient.ConfigMaps(namespace)
	k8sSecretClient := k8sCoreClient.Secrets(namespace)

	backupClient, err := backupEcosystem.NewForConfig(k8sRestConfig)
	if err != nil {
		return k8sClients{}, fmt.Errorf("failed to create ecosystem backup client: %w", err)
	}

	backupScheduleClient := backupClient.BackupSchedules(namespace)

	return k8sClients{
		configMap:      k8sConfigMapClient,
		secret:         k8sSecretClient,
		backupSchedule: backupScheduleClient,
	}, nil
}
