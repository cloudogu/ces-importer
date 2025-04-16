package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/cloudogu/ces-importer/api"
	"github.com/cloudogu/ces-importer/configuration"
	"github.com/cloudogu/ces-importer/sync"
)

func main() {
	config, err := configuration.ReadConfigFromEnv()
	if err != nil {
		panic(fmt.Errorf("ces-importer main process failed to read config from env: %w", err))
	}

	configureLogger(config)

	ctx := context.Background()
	logUsedConfig(ctx, config)

	err = runMain(ctx, config)
	if err != nil {
		slog.Error("ces-importer main process restarts now because of an error: %s", err.Error())
		os.Exit(1)
	}
}

func runMain(ctx context.Context, config configuration.Configuration) error {
	httpClient := http.Client{}

	exporterSource, importerDestination, exporterPort, err := fetchExporterAPIConfig(ctx, config, httpClient)
	if err != nil {
		return fmt.Errorf("failed to fetch API configuration from the exporter: %w", err)
	}

	for {
		isExporterSyncReady, err := checkExportSyncState(ctx, exporterSource, exporterPort, config, httpClient)
		if err != nil {
			// This error is recoverable except for misconfiguration which may be detected by analyzing the logs.
			// Fall-through to sleep and avoid adding load to the log output AND the CPU.
			slog.Log(ctx, slog.LevelError, fmt.Sprintf("Error while checking export sync readiness: %s", err.Error()))
		}

		if !isExporterSyncReady {
			// FIXME: do proper cron ticks here
			time.Sleep(60 * time.Second)
			continue
		}

		syncer := sync.NewRsyncSyncer(config.ExporterHost, exporterPort, config.ExporterSSHUser, config.ImporterPrivateSSHKeyPath)

		if err := syncer.Sync(exporterSource, importerDestination); err != nil {
			// TODO: is this error recoverable? If so, log the error and continue
			return fmt.Errorf("failed to sync source %s to destination %s: %w", exporterSource, importerDestination, err)
		}

		slog.Info("Sync successful")

		//// Wait for interrupt signal to gracefully shut down the server with a timeout of 5 seconds.
		//quit := make(chan os.Signal, 1)
		//signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		//<-quit
		//slog.Info("Shutdown Server ...")
		//
		//ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		//defer cancel()
		//
		//<-ctx.Done()
		//slog.Info("shutdown-timeout of 5 seconds reached")
		//slog.Info("exiting")
	}
}

func fetchExporterAPIConfig(ctx context.Context, config configuration.Configuration, httpClient http.Client) (exporterSource, importerDestination, exporterPort string, err error) {
	exporterUrl := "https://" + config.ExporterHost + api.EndpointExportMode

	result, err := api.DoGetRequest(ctx, exporterUrl, config.ExporterApiKey, httpClient)
	if err != nil {
		return "", "", "", err
	}

	return "", "", "", nil
}

func checkExportSyncState(ctx context.Context, source string, port string, config configuration.Configuration, httpClient http.Client) (isReady bool, err error) {
	exporterUrl := "https://" + source + ":" + port + api.EndpointExportMode

	result, err := api.DoGetRequest(ctx, exporterUrl, config.ExporterApiKey, httpClient)
	if err != nil {
		return false, err
	}

	return true, nil
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
