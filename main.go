package main

import (
	"github.com/cloudogu/ces-importer/rsync"
	"log/slog"
	"os"
)

func main() {

	configureLogger()

	err := rsync.Sync("localhost:/data/", "/home/bernst/temp/migration/copy")
	if err != nil {
		panic(err)
	}

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

func configureLogger() {
	var level slog.Level
	level = slog.LevelInfo

	//var err = level.UnmarshalText([]byte(conf.LogLevel))
	//if err != nil {
	//	slog.Error("error parsing log level. Setting log level to INFO.", "err", err)
	//	level = slog.LevelInfo
	//}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		AddSource: false,
		Level:     level,
	}))
	slog.SetDefault(logger)

	slog.Info("configured logger", "level", level.String())
}
