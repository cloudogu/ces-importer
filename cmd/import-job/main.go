package main

import (
	"context"
	"github.com/cloudogu/ces-importer/api/exporter"
	"github.com/cloudogu/ces-importer/api/importer"
	"github.com/cloudogu/ces-importer/configuration"
	"github.com/cloudogu/ces-importer/logging"
	"github.com/cloudogu/ces-importer/migration"
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
	ctx = migration.SetTriggerFQDNChangeFromEnv(ctx)

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

	exportAPIService := exporter.NewServiceFromConfig(
		exporter.APIHost(jobConfig.ExporterHost),
		exporter.APIKey(jobConfig.ExporterApiKey),
		exporter.SkipTLSVerification(jobConfig.SkipTLSVerify),
	)
	slog.Debug("Successfully created service for exporter API")

	k8sRestConfig, err := ctrl.GetConfig()
	if err != nil {
		slog.Error("failed to read kube config", "cause", err)
		return 1
	}

	k8sClientSet, err := importer.CreateK8SClientSet(k8sRestConfig, jobConfig.General.Namespace)
	if err != nil {
		slog.Error("failed to create k8s client set", "cause", err)
		return 1
	}

	slog.Debug("Successfully created k8s client set")

	importJob := NewImportExecuter(jobConfig, exportAPIService, k8sClientSet)

	slog.Info("Import executer created, start data synchronization...")

	err = importJob.Start(ctx)
	if err != nil {
		slog.Error("Import job failed", "cause", err)
		return 1
	}

	slog.Info("Import job finished.")
	return 0
}
