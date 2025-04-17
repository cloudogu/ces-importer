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

	"github.com/cloudogu/ces-importer/api"
	"github.com/cloudogu/ces-importer/configuration"
	"github.com/cloudogu/ces-importer/cron"
	"github.com/cloudogu/ces-importer/sync"
)

var hostProtocolScheme = "https://"

type exporterApiClient interface {
	// DoGetRequest allows issuing HTTP requests towards the exporter API. The result will be a byte slice that must
	// be parsed by the caller respectively.
	DoGetRequest(ctx context.Context, url string) ([]byte, error)
}

func main() {
	config, err := configuration.ReadConfigFromEnv()
	if err != nil {
		panic(fmt.Errorf("ces-importer main process failed to read config from env: %w", err))
	}

	configureLogger(config)

	ctx := context.Background()
	logUsedConfig(ctx, config)

	looper := cron.NewMainLooper("0,30 * * * * *") // gronx supports 6 cron-style digits for seconds

	err = runMainLoop(ctx, config, looper)
	if err != nil {
		slog.Error("ces-importer main process restarts now because of an error: %s", err.Error())
		os.Exit(1)
	}
}

func runMainLoop(ctx context.Context, config configuration.Configuration, looper *cron.MainLooper) error {
	httpClient := http.Client{}
	exportApiCli := api.NewClient(config.ExporterApiKey, httpClient)

	// Wait for interrupt signals to gracefully shut down the server with a timeout of 5 seconds.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	<-quit
	slog.Info("Shutdown Server ...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	defer looper.Stop()

	<-ctx.Done()
	slog.Info("shutdown-timeout of 5 seconds reached")
	slog.Info("exiting")

	slog.Log(ctx, slog.LevelInfo, "Starting main loop")
	err := looper.Run(func(ctx context.Context) error {
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
		}

		err = deactivateImporterDogus(ctx, systemInfo, config)
		if err != nil {
			return err
		}

		err = syncDogus(ctx, systemInfo, config)
		if err != nil {
			return err
		}

		slog.Log(ctx, slog.LevelInfo, "Sync successful")

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to sync from exporter: %w", err)
	}

	return nil
}

func deactivateImporterDogus(ctx context.Context, systemInfo *api.SystemInfo, config configuration.Configuration) error {
	for _, dogu := range systemInfo.Dogus {
		slog.Log(ctx, slog.LevelInfo, "Starting sync for dogu %s...", dogu.Name)

		err := deactivateDogu(ctx, config, dogu)
		if err != nil {
			// this error does not seem recoverable because the dogu must be down to avoid copy data problems
			return fmt.Errorf("failed to deactivate dogu %s in the importer: %w", dogu.Name, err)
		}
	}

	return nil
}

func deactivateDogu(ctx context.Context, config configuration.Configuration, dogu api.Dogu) error {
	return nil
}

func syncDogus(ctx context.Context, systemInfo *api.SystemInfo, config configuration.Configuration) error {
	for _, dogu := range systemInfo.Dogus {
		slog.Log(ctx, slog.LevelInfo, "Starting sync for dogu %s...", dogu.Name)

		err := deactivateDogu(ctx, config, dogu)
		if err != nil {
			// this error does not seem recoverable because the dogu must be down to avoid copy data problems
			return fmt.Errorf("failed to deactivate dogu %s in the importer: %w", dogu.Name, err)
		}

		// TODO in upcoming feature: Interpret the actual target data from the exporter API
		exporterSource, importerDestination, exporterPort := func(dogu api.Dogu) (string, string, string) {
			return "your exporterAPIResult here", "and here", "and here"
		}(dogu)

		syncer := sync.NewRsyncSyncer(config.ExporterHost, exporterPort, config.ExporterSSHUser, config.ImporterPrivateSSHKeyPath)

		if err := syncer.Sync(exporterSource, importerDestination); err != nil {
			// TODO: is this error recoverable? If so, log the error and continue
			return fmt.Errorf("failed to sync source %s to destination %s: %w", exporterSource, importerDestination, err)
		}
		slog.Log(ctx, slog.LevelInfo, "Syncing for dogu %s successful")
	}
	return nil
}

func isApiExportReady(ctx context.Context, hostname string, apiCli exporterApiClient) (isActive bool, err error) {
	exporterUrl := hostProtocolScheme + hostname + api.EndpointExportMode

	result, err := apiCli.DoGetRequest(ctx, exporterUrl)
	if err != nil {
		return false, fmt.Errorf("failed to check whether exporter is export ready: %w", err)
	}

	var exportMode api.ExportMode
	err = json.Unmarshal(result, &exportMode)
	if err != nil {
		return false, fmt.Errorf("failed to parse export mode response: %q: %w", result, err)
	}

	return exportMode.IsActive, nil
}

func fetchExporterSystemInfo(ctx context.Context, hostname string, apiCli exporterApiClient) (*api.SystemInfo, error) {
	exporterUrl := hostProtocolScheme + hostname + api.EndpointSystemInfo

	result, err := apiCli.DoGetRequest(ctx, exporterUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch exporter system info: %w", err)
	}

	var systemInfo *api.SystemInfo
	err = json.Unmarshal(result, systemInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to parse system info response: %q: %w", result, err)
	}

	return systemInfo, nil
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

	slog.Log(ctx, slog.LevelInfo, "ces-importer started using this configuration:", "LogLevel", config.LogLevel,
		"ExporterHost", config.ExporterHost,
		"ExporterSSHUser", config.ExporterSSHUser,
		"MigrationRegularCron", config.MigrationRegularCron,
		"MigrationFinalTimestamp", config.MigrationFinalTimestamp,
		"ImporterPrivateSSHKeyPath", config.ImporterPrivateSSHKeyPath)
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
