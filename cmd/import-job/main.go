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

	exporterService := createAPIService(jobConfig.API)

	slog.Debug("Successfully created service for exporter API")

	k8sClientSet, err := importer.CreateK8SClientSet(jobConfig.General.Namespace)
	if err != nil {
		slog.Error("failed to create k8s client set", "cause", err)
		return 1
	}

	slog.Debug("Successfully created k8s client set")

	importJob := NewImportExecuter(jobConfig, exporterService, k8sClientSet)

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

	return exporter.NewClient(apiCfg.ExporterHost, apiCfg.ExporterApiKey, options...)
}

type apiService struct {
	config *exporter.ConfigService
	dogu   *exporter.ExportDoguClient
	system *exporter.SystemInfoClient
}

func createAPIService(apiCfg configuration.API) apiService {
	exportClient := createAPIClient(apiCfg)
	exportService := exporter.NewService(exportClient)

	exportDoguApiClient := exporter.NewExportDoguClient(exportClient)
	systemInfoApiClient := exporter.NewSystemInfoClient(exportClient)

	return apiService{
		config: exportService.ConfigService,
		dogu:   exportDoguApiClient,
		system: systemInfoApiClient,
	}
}
