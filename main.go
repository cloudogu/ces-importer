package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/cloudogu/ces-importer/configuration"
	"github.com/cloudogu/ces-importer/sync"
)

// http constants
const (
	// apiKeyAuthName contains the name of the header key to authenticate against the exporter API without basic auth.
	apiKeyAuthName = "X-CES-EXPORTER-API-KEY"
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
	exporterSource, importerDestination, exporterPort, err := fetchExporterAPIConfig(ctx, config)
	if err != nil {
		return fmt.Errorf("failed to fetch API configuration from the exporter: %w", err)
	}

	httpClient := http.Client{}

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
			return fmt.Errorf("failed to sync source %s to destination %s: %w", exporterSource, importerDestination)
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

func fetchExporterAPIConfig(ctx context.Context, config configuration.Configuration) (string, string, string, error) {
	return "", "", "", nil
}

func checkExportSyncState(ctx context.Context, source string, port string, config configuration.Configuration, httpClient http.Client) (isReady bool, err error) {
	endpoint := "/export/mode"
	exporterUrl := source + ":" + port + endpoint

	request, err := http.NewRequest(http.MethodGet, exporterUrl, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create request to %s: %w", exporterUrl, err)
	}

	request.Header.Set(apiKeyAuthName, config.ExporterApiKey)

	response, err := httpClient.Do(request)
	if err != nil {
		return false, fmt.Errorf("request to %s failed with an error: %w", exporterUrl, err)
	}

	defer func() { _ = response.Body.Close() }()
	responseMsg, err := io.ReadAll(response.Body)
	if err != nil {
		return false, fmt.Errorf("failed to read response body for %s", exporterUrl)
	}

	if response.StatusCode != http.StatusOK {
		return false, fmt.Errorf("received unexpected response to %s (wanted %d got %d): %s",
			exporterUrl, http.StatusOK, response.StatusCode, string(responseMsg))
	}

	slog.Log(ctx, slog.LevelDebug, "Successfully called %s with response %#v", responseMsg)

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
