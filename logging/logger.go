package logging

import (
	"fmt"
	"github.com/cloudogu/ces-importer/configuration"
	"io"
	"log/slog"
	"os"
)

const (
	PathAppLogFile = "/home/ces-importer/migration-log/log.log"
	appLogFileMode = os.O_CREATE | os.O_WRONLY | os.O_APPEND
	appLogPerm     = 0666
)

var createWriter = func() (io.Writer, error) {
	logFile, err := os.OpenFile(PathAppLogFile, appLogFileMode, appLogPerm)
	if err != nil {
		return nil, fmt.Errorf("failed to create app log file: %w", err)
	}

	return io.MultiWriter(os.Stderr, logFile), nil
}

func Initialize(conf configuration.Configuration) error {
	var level slog.Level
	if err := level.UnmarshalText([]byte(conf.LogLevel)); err != nil {
		slog.New(slog.NewTextHandler(os.Stderr, nil)).
			Error("Error parsing log level. Setting to INFO.", "err", err)
		level = slog.LevelInfo
	}

	multiWriter, err := createWriter()
	if err != nil {
		return fmt.Errorf("failed to create multiwriter: %w", err)
	}

	handler := slog.NewTextHandler(multiWriter, &slog.HandlerOptions{
		Level: level,
	})

	logger := slog.New(handler)
	slog.SetDefault(logger)

	slog.Info("Configured logger", "level", level.String())

	return nil
}
