package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/cloudogu/ces-importer/configuration"
	"github.com/cloudogu/ces-importer/sync"
)

func main() {
	config, err := configuration.ReadConfigFromEnv()
	if err != nil {
		panic(fmt.Errorf("failed to read config from env: %w", err))
	}

	configureLogger(config)

	//TODO: remove in next feature, this is only to demonstrate the config arrival
	demoCtx := context.Background()
	slog.Log(demoCtx, slog.LevelWarn, "========================")
	slog.Log(demoCtx, slog.LevelWarn, "hooray! configuration arrived!", "config", config)
	slog.Log(demoCtx, slog.LevelWarn, "========================")

	// TODO in upcoming feature: Interpret the actual target data from the exporter API
	exporterSource, importerDestination, exporterPort := func() (string, string, string) {
		return "your exporterAPIResult here", "and here", "and here"
	}()

	syncer := sync.NewRsyncSyncer(config.ExporterHost, exporterPort, config.ExporterSSHUser, config.ImporterPrivateSSHKeyPath)

	if err := syncer.Sync(exporterSource, importerDestination); err != nil {
		panic(err)
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
