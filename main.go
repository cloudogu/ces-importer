package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	ecoSystemV2 "github.com/cloudogu/k8s-dogu-operator/v2/api/ecoSystem"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/cloudogu/ces-importer/api/exporter"
	"github.com/cloudogu/ces-importer/api/importer"
	"github.com/cloudogu/ces-importer/configuration"
	"github.com/cloudogu/ces-importer/cron"
)

var hostProtocolScheme = "https://"

func main() {
	ctx := context.Background()

	config, err := configuration.ReadConfigFromEnv()
	if err != nil {
		panic(fmt.Errorf("failed to read config: %w", err))
	}

	configureLogger(config)

	logUsedConfig(ctx, config)

	cronLikeExpr := "0,30 * * * * *"
	cronLooper, err := cron.New(cronLikeExpr) // gronx supports 6 cron-style digits for seconds while regular cron only supports 5 // digits.
	if err != nil {
		panic(fmt.Errorf("failed to create cron looper for expression %q: %w", cronLikeExpr, err))
	}

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

	err = runMainLoop(ctx, config, cronLooper, exportApiCli, doguStartStopper, doguStartStopper)
	if err != nil {
		slog.Error("ces-importer main process restarts now because of an error", "error", err.Error())
		os.Exit(1)
	}
}

func runMainLoop(ctx context.Context, config configuration.Configuration, cronLooper looper, exportApiCli exporterApiClient, doguStart doguStarter, doguStop doguStopper) error {

	// Wait for interrupt signals to gracefully shut down the server with a timeout of 5 seconds.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	go func() {
		<-quit
		slog.Info("Shutdown Server ...")

		<-ctx.Done()
		cronLooper.Stop()
		slog.Info("shutdown-timeout of 5 seconds reached")
		slog.Info("exiting")
	}()

	slog.Log(ctx, slog.LevelInfo, "Starting main loop")
	cronLooper.Run(createMainLoop(config, exportApiCli, doguStart, doguStop))

	return nil
}

func createMainLoop(config configuration.Configuration, exportApiCli exporterApiClient, doguStart doguStarter, doguStop doguStopper) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		isExporterSyncReady, err := isApiExportReady(ctx, config.ExporterHost, exportApiCli)
		if err != nil {
			// This error is recoverable except for misconfiguration, which may be detected by analyzing the logs.
			slog.Log(ctx, slog.LevelError, fmt.Sprintf("Error while checking export sync readiness: %s", err.Error()))
			slog.Log(ctx, slog.LevelInfo, "Waiting for the next run...")
			return nil
		}

		if !isExporterSyncReady {
			// This condition is recoverable, but it is still unclear when the ready status will be triggered
			slog.Log(ctx, slog.LevelInfo, "Exporter does not seem to be ready. Waiting for the next run...")
			return nil // continue to the next main loop iteration
		}

		systemInfo, err := fetchExporterSystemInfo(ctx, config.ExporterHost, exportApiCli)
		if err != nil {
			// this error is recoverable, the exporter system API might be down, or the API server errs
			slog.Log(ctx, slog.LevelError, fmt.Sprintf("Failed to fetch the system info from the exporter: %s", err.Error()))
			slog.Log(ctx, slog.LevelInfo, "Waiting for the next run...")
			return nil
		}

		err = deactivateImporterDogus(ctx, systemInfo, doguStop)
		if err != nil {
			return err
		}

		err = syncDogus(ctx, systemInfo, config)
		if err != nil {
			return err
		}

		err = activateImporterDogus(ctx, systemInfo, doguStart)
		if err != nil {
			return err
		}

		slog.Log(ctx, slog.LevelInfo, "Sync successful")

		return nil
	}
}

func deactivateImporterDogus(ctx context.Context, systemInfo *exporter.SystemInfo, doguStop doguStopper) error {
	for _, dogu := range systemInfo.Dogus {
		slog.Log(ctx, slog.LevelInfo, "Deactivating dogu ", "doguName", dogu.Name)

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
		slog.Log(ctx, slog.LevelInfo, "Activating dogu", "doguName", dogu.Name)

		err := doguStart.StartDogu(ctx, dogu)
		if err != nil {
			// this error does not seem recoverable because the dogu must be down to avoid copy data problems
			return fmt.Errorf("failed to activate dogu %s in the importer: %w", dogu.Name, err)
		}
	}

	return nil
}

func syncDogus(ctx context.Context, systemInfo *exporter.SystemInfo, config configuration.Configuration) error {
	// FIXME: #4: actually implement the core functionality in a proper way. This is part of an upcoming feature

	//for _, dogu := range systemInfo.Dogus {
	//	slog.Log(ctx, slog.LevelInfo, "Starting sync for dogu ", "doguName", dogu.Name)
	//	// TODO in upcoming feature: Interpret the actual target data from the exporter API
	//	exporterSource, importerDestination, exporterPort := func(dogu exporter.Dogu) (string, string, string) {
	//		return "your exporterAPIResult here", "and here", "and here"
	//	}(dogu)
	//
	//	syncer := sync.NewRsyncSyncer(config.ExporterHost, exporterPort, config.ExporterSSHUser, config.ImporterPrivateSSHKeyPath)
	//
	//	if err := syncer.Sync(exporterSource, importerDestination); err != nil {
	//		// TODO: is this error recoverable? If so, log the error and continue
	//		return fmt.Errorf("failed to sync source %s to destination %s: %w", exporterSource, importerDestination, err)
	//	}
	//	slog.Log(ctx, slog.LevelInfo, "Syncing for dogu %s successful")
	//}
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

func logUsedConfig(ctx context.Context, config configuration.Configuration) {
	slog.Log(ctx, slog.LevelInfo, "                     ./////,                    ")
	slog.Log(ctx, slog.LevelInfo, "                 ./////==//////,                ")
	slog.Log(ctx, slog.LevelInfo, "                ////.  ___   ////.              ")
	slog.Log(ctx, slog.LevelInfo, "         ,OO,. ////  ,////A,  */// ,OO,.        ")
	slog.Log(ctx, slog.LevelInfo, "    ,/////////////*  */////*  *////////////A    ")
	slog.Log(ctx, slog.LevelInfo, "   ////'        `VA.   '|'   .///'       '///*  ")
	slog.Log(ctx, slog.LevelInfo, "  *///  .*///*,         |         .*//*,   ///* ")
	slog.Log(ctx, slog.LevelInfo, "  (///  (//////)**--_./////_----*//////)   ///) ")
	slog.Log(ctx, slog.LevelInfo, "   V///   '°°°°      (/////)      °°°°'   ////  ")
	slog.Log(ctx, slog.LevelInfo, "    V/////(////////o. '°°°' ./////////(///(/'   ")
	slog.Log(ctx, slog.LevelInfo, "       'V/(/////////////////////////////V'      ")

	slog.Log(ctx, slog.LevelInfo, "ces-importer started using this configuration:",
		"config", fmt.Sprintf("%#v", config),
	)
}

func configureLogger(conf configuration.Configuration) {
	var level slog.Level
	var err = level.UnmarshalText([]byte(conf.LogLevel))
	if err != nil {
		slog.Error("error parsing log level. Setting log level to INFO.", "err", err)
		level = slog.LevelInfo
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		AddSource: false,
		Level:     level,
	}))
	slog.SetDefault(logger)

	slog.Info("configured logger", "level", level.String())
}
