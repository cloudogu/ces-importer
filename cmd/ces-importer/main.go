package main

import (
	"context"
	"fmt"
	"github.com/cloudogu/ces-importer/api/exporter"
	"github.com/cloudogu/ces-importer/configuration"
	"github.com/cloudogu/ces-importer/cron"
	"github.com/cloudogu/ces-importer/logging"
	"github.com/cloudogu/ces-importer/migration"
	"k8s.io/client-go/kubernetes"
	batchv1 "k8s.io/client-go/kubernetes/typed/batch/v1"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"log/slog"
	"net/http"
	ctrl "sigs.k8s.io/controller-runtime"
)

func main() {
	ctx := context.Background()

	cfg, err := configuration.ReadCoordinatorConfig()
	if err != nil {
		panic(fmt.Errorf("failed to read config: %w", err))
	}

	err = logging.Initialize(cfg)
	if err != nil {
		panic(err)
	}

	exportAPIService := createAPIService(cfg.API)

	k8sClientSet, err := createK8Sclientset(cfg.Namespace)

	service, err := migration.NewJobService(migration.JobServiceDependencies{
		JobProviderDependencies: migration.JobProviderDependencies{
			JobContainerConfig: cfg.JobContainer,
			SSHConfig:          cfg.SSH,
			APIKey:             cfg.API.ExporterApiKey,
			DoguVolumeBasePath: cfg.JobConfig.DoguVolumeBasePath,
			PVCClient:          migration.NewPVCGetter(k8sClientSet.pvcClient),
		},
		JobClient: k8sClientSet.jobClient,
		PodClient: k8sClientSet.podClient,
	})
	if err != nil {
		return
	}

	deps := migration.MigratorDependencies{
		MaintenanceModeHandler: exportAPIService.MaintenanceModeService,
		JobRunner:              service,
	}

	migrator := migration.NewMigrator(deps)

	cronLooper, err := cron.New(ctx, cfg.Migration.RegularCron, func(ctx context.Context) (int, error) {
		err = migrator.RunMigration(ctx)
		if err != nil {
			return 1, err
		}

		return 0, nil
	})
	if err != nil {
		panic(fmt.Errorf("failed to create cron looper for expression %q: %w", cfg.Migration.RegularCron, err))
	}

	slog.Info("Starting main loop")
	cronLooper.Run()
}

func createAPIService(apiCfg configuration.API) *exporter.Service {
	httpClient := http.DefaultClient
	exportClient := exporter.NewClient(apiCfg.ExporterHost, httpClient)
	exportService := exporter.NewService(apiCfg.ExporterHost, exportClient)

	return exportService
}

type k8sClients struct {
	pvcClient corev1.PersistentVolumeClaimInterface
	podClient corev1.PodInterface
	jobClient batchv1.JobInterface
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
	k8sPVCClient := k8sCoreClient.PersistentVolumeClaims(namespace)
	k8sPodClient := k8sCoreClient.Pods(namespace)

	k8sJobClient := k8sClientSet.BatchV1().Jobs(namespace)

	return k8sClients{
		pvcClient: k8sPVCClient,
		podClient: k8sPodClient,
		jobClient: k8sJobClient,
	}, nil
}
