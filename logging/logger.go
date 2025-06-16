package logging

import (
	"fmt"
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
	logLevel       string
	component      string
}

func NewLogInitializer(logLevel string, component string) *LogInitializer {
	return &LogInitializer{
		open: func(name string, flag int, perm os.FileMode) (file, error) {
			return os.OpenFile(name, flag, perm)
		},
		newMultiWriter: io.MultiWriter,
		logLevel:       logLevel,
		component:      component,
	}
}

func (li LogInitializer) Initialize() error {
	return li.initialize(os.Stdout)
}

func (li LogInitializer) InitializeWithLogFile() error {
	logFile, err := li.open(PathAppLogFile, logFilesMode, logFilesPerm)
	if err != nil {
		return fmt.Errorf("failed to open app log file: %w", err)
	}

	multiWriter := li.newMultiWriter(os.Stdout, logFile)

	return li.initialize(multiWriter)
}

func (li LogInitializer) initialize(writer io.Writer) error {
	var level slog.Level
	if err := level.UnmarshalText([]byte(li.logLevel)); err != nil {
		slog.New(slog.NewTextHandler(os.Stderr, nil)).
			Error("Error parsing log level. Setting to INFO.", "err", err)
		level = slog.LevelInfo
	}

	handler := slog.NewTextHandler(writer, &slog.HandlerOptions{
		Level: level,
	})

	componentHandler := handler.WithAttrs([]slog.Attr{
		slog.String("component", li.component),
	})

	logger := slog.New(componentHandler)
	slog.SetDefault(logger)

	slog.Info("Configured logger", "level", level.String())

	return nil
}
