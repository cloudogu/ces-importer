package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/cloudogu/ces-importer/logging"
	"github.com/cloudogu/ces-importer/systeminfo"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	ctrl "sigs.k8s.io/controller-runtime"
	"syscall"

	ecoSystemV2 "github.com/cloudogu/k8s-dogu-operator/v3/api/ecoSystem"

	"github.com/cloudogu/ces-importer/api/exporter"
	"github.com/cloudogu/ces-importer/api/importer"
	"github.com/cloudogu/ces-importer/configuration"
	"github.com/cloudogu/ces-importer/cron"
	"github.com/cloudogu/ces-importer/sync"
)

var hostProtocolScheme = "https://"

func main() {
	ctx := context.Background()

	config, err := configuration.ReadConfigFromEnv()
	if err != nil {
		panic(fmt.Errorf("failed to read config: %w", err))
	}

	err = logging.Initialize(config)
	if err != nil {
		panic(err)
	}

	logUsedConfig(config)

	httpClient := &http.Client{}
	exportApiCli := exporter.NewClient(config.ExporterApiKey, httpClient)

	k8sRestConfig, err := ctrl.GetConfig()
	if err != nil {
		panic(fmt.Errorf("failed to read kube config: %w", err))
	}

	doguCli, err := ecoSystemV2.NewForConfig(k8sRestConfig)
	if err != nil {
		panic(fmt.Errorf("failed to create dogu client: %w", err))
	}

	doguClient := doguCli.Dogus(config.ImporterNamespace)
	doguStartStopper := importer.NewDoguDeploymentClient(doguClient)

	syncer := sync.NewRsyncSyncer(config.ExporterHost, config.ExporterSSHUser, config.ImporterPrivateSSHKeyPath)

	provider, err := systeminfo.NewSystemInfoProvider(config.ImporterNamespace)
	if err != nil {
		panic(fmt.Errorf("failed to create system info provider: %w", err))
	}
	validator := systeminfo.NewValidator(config, config.ImporterNamespace, provider)

	mainLoop := createMainLoop(config, exportApiCli, doguStartStopper, doguStartStopper, syncer, validator)
	cronLooper, err := cron.New(ctx, config.MigrationRegularCron, mainLoop)
	if err != nil {
		panic(fmt.Errorf("failed to create cron looper for expression %q: %w", config.MigrationRegularCron, err))
	}

	// Wait for interrupt signals to gracefully shut down the server with a timeout of 5 seconds.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	go func() {
		<-quit
		slog.Info("Shutdown Server ...")

		<-ctx.Done()
		cronLooper.Stop()
		slog.Warn("shutdown reached: exiting")
	}()

	slog.Info("Starting main loop")
	cronLooper.Run()
}

type systemInfoValidator interface {
	ValidateSystemInfo(ctx context.Context) error
}

func createMainLoop(config configuration.Configuration, exportApiCli exporterApiClient, doguStart doguStarter, doguStop doguStopper, syncer doguVolumeSyncer, sysInfoValidator systemInfoValidator) func(ctx context.Context) (int, error) {
	return func(ctx context.Context) (int, error) {
		isExporterSyncReady, err := isApiExportReady(ctx, config.ExporterHost, exportApiCli)
		if err != nil {
			// This error is recoverable except for misconfiguration, which may be detected by analyzing the logs.
			slog.Error(fmt.Sprintf("Error while checking export sync readiness: %s", err.Error()))
			slog.Info("Waiting for the next run...")
			return 0, nil
		}

		if !isExporterSyncReady {
			// This condition is recoverable, but it is still unclear when the ready status will be triggered
			slog.Info("Exporter does not seem to be ready. Waiting for the next run...")
			return 0, nil // continue to the next main loop iteration
		}

		systemInfo, err := fetchExporterSystemInfo(ctx, config.ExporterHost, exportApiCli)
		if err != nil {
			// this error is recoverable, the exporter system API might be down, or the API server errs
			slog.Error(fmt.Sprintf("Failed to fetch the system info from the exporter: %s", err.Error()))
			slog.Info("Waiting for the next run...")
			return 0, nil
		}

		err = sysInfoValidator.ValidateSystemInfo(ctx)
		if err != nil {
			slog.Log(ctx, slog.LevelError, fmt.Sprintf("Failed to validate importer system info: %s", err.Error()))
			// TODO should this break the main loop or not?
			slog.Log(ctx, slog.LevelInfo, "Waiting for the next run...")
			return 0, nil
		}

		err = deactivateImporterDogus(ctx, systemInfo, doguStop)
		if err != nil {
			return 1, err
		}

		err = syncDogus(ctx, systemInfo, exportApiCli, syncer)
		if err != nil {
			return 2, err
		}

		err = activateImporterDogus(ctx, systemInfo, doguStart)
		if err != nil {
			return 3, err
		}

		slog.Info("Sync successful")

		return 0, nil
	}
}

func deactivateImporterDogus(ctx context.Context, systemInfo *exporter.SystemInfo, doguStop doguStopper) error {
	for _, dogu := range systemInfo.Dogus {
		slog.Info("Deactivating dogu ", "doguName", dogu.Name)

		err := doguStop.StopDogu(ctx, dogu)
		if err != nil {
			// this error does not seem recoverable because the dogu must be down to avoid copy data problems
			return fmt.Errorf("failed to deactivate dogu %s in the importer: %w", dogu.Name, err)
		}
	}

	return nil
}

func activateImporterDogus(ctx context.Context, systemInfo *exporter.SystemInfo, doguStart doguStarter) error {
	for _, dogu := range systemInfo.Dogus {
		slog.Info("Activating dogu", "doguName", dogu.Name)

		err := doguStart.StartDogu(ctx, dogu)
		if err != nil {
			// this error does not seem recoverable because the dogu must be down to avoid copy data problems
			return fmt.Errorf("failed to activate dogu %s in the importer: %w", dogu.Name, err)
		}
	}

	return nil
}

func syncDogus(ctx context.Context, systemInfo *exporter.SystemInfo, _ exporterApiClient, syncer doguVolumeSyncer) error {
	// FIXME: #4: actually implement the core functionality in a proper way. This is part of an upcoming feature

	for _, dogu := range systemInfo.Dogus {
		slog.Info("Starting sync for dogu ", "doguName", dogu.Name)

		exporterSource, importerDestination, exporterPort := func(dogu exporter.Dogu) (string, string, string) {
			return "call your your exporterApiClient for data here", "and here", "and here"
		}(dogu)

		if err := syncer.SyncDogu(ctx, exporterPort, exporterSource, importerDestination); err != nil {
			// TODO: should we continue syncing other dogus on a best-effort basis?
			return fmt.Errorf("failed to sync source %s to destination %s: %w", exporterSource, importerDestination, err)
		}
		slog.Info("Syncing for dogu successful", "doguName", dogu.Name)
	}

	return nil

}

func isApiExportReady(ctx context.Context, hostname string, apiCli exporterApiClient) (isActive bool, err error) {
	exporterUrl := hostProtocolScheme + hostname + exporter.EndpointExportMode

	result, err := apiCli.DoGetRequest(ctx, exporterUrl)
	if err != nil {
		return false, fmt.Errorf("failed to check whether exporter is export ready: %w", err)
	}

	var exportMode exporter.ExportMode
	err = json.Unmarshal(result, &exportMode)
	if err != nil {
		return false, fmt.Errorf("failed to parse export mode response: %q: %w", result, err)
	}

	return exportMode.IsActive, nil
}

func fetchExporterSystemInfo(ctx context.Context, hostname string, apiCli exporterApiClient) (*exporter.SystemInfo, error) {
	exporterUrl := hostProtocolScheme + hostname + exporter.EndpointSystemInfo

	result, err := apiCli.DoGetRequest(ctx, exporterUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch exporter system info: %w", err)
	}

	var systemInfo exporter.SystemInfo
	err = json.Unmarshal(result, &systemInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to parse system info response: %q: %w", result, err)
	}

	return &systemInfo, nil
}

func logUsedConfig(config configuration.Configuration) {
	slog.Info("                     ./////,                    ")
	slog.Info("                 ./////==//////,                ")
	slog.Info("                ////.  ___   ////.              ")
	slog.Info("         ,OO,. ////  ,////A,  */// ,OO,.        ")
	slog.Info("    ,/////////////*  */////*  *////////////A    ")
	slog.Info("   ////'        `VA.   '|'   .///'       '///*  ")
	slog.Info("  *///  .*///*,         |         .*//*,   ///* ")
	slog.Info("  (///  (//////)**--_./////_----*//////)   ///) ")
	slog.Info("   V///   '°°°°      (/////)      °°°°'   ////  ")
	slog.Info("    V/////(////////o. '°°°' ./////////(///(/'   ")
	slog.Info("       'V/(/////////////////////////////V'      ")

	slog.Info("ces-importer started using this configuration:",
		"config", fmt.Sprintf("%#v", config),
	)
}
