package main

import (
	"github.com/cloudogu/ces-importer/configuration"
	"github.com/cloudogu/ces-importer/sync"
	"log/slog"
	"os"
)

func main() {
	config, err := configuration.ReadConfigFromEnv()
	if err != nil {
		panic(err)
	}

	configureLogger(config)

	syncer := sync.NewRsyncSyncer(config.ExporterHost, config.ExporterPort, config.ExporterSSHUser, config.ImporterPrivateSSHKeyPath)

	if err := syncer.Sync(config.ExporterSource, config.ImporterDestination); err != nil {
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
