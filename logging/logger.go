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
	logFilesMode   = os.O_CREATE | os.O_WRONLY | os.O_TRUNC
	logFilesPerm   = 0666
)

type LogInitializer struct {
	open           osOpenFile
	newMultiWriter createMultiWriter
	config         configuration.Coordinator
}

func NewLogInitializer(cfg configuration.Configuration) *LogInitializer {
	return &LogInitializer{
		open: func(name string, flag int, perm os.FileMode) (file, error) {
			return os.OpenFile(name, flag, perm)
		},
		newMultiWriter: io.MultiWriter,
		config:         cfg,
	}
}

func (li LogInitializer) Initialize() error {
	var level slog.Level
	if err := level.UnmarshalText([]byte(li.config.Logging.LogLevel)); err != nil {
		slog.New(slog.NewTextHandler(os.Stderr, nil)).
			Error("Error parsing log level. Setting to INFO.", "err", err)
		level = slog.LevelInfo
	}

	logFile, err := li.open(PathAppLogFile, logFilesMode, logFilesPerm)
	if err != nil {
		return fmt.Errorf("failed to open app log file: %w", err)
	}

	multiWriter := li.newMultiWriter(os.Stderr, logFile)

	handler := slog.NewTextHandler(multiWriter, &slog.HandlerOptions{
		Level: level,
	})

	logger := slog.New(handler)
	slog.SetDefault(logger)

	slog.Info("Configured logger", "level", level.String())

	return nil
}
