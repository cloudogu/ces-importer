package logging

import (
	"errors"
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

var (
	writer io.Writer = os.Stdout
)

func GetWriter() io.Writer {
	return writer
}

type logger struct {
	level      slog.Level
	attributes []slog.Attr
	writer     io.Writer
}

type LoggerOption func(*logger) error

func WithLevel(level string) LoggerOption {
	return func(l *logger) error {
		var logLevel slog.Level

		if err := logLevel.UnmarshalText([]byte(level)); err != nil {
			return fmt.Errorf("failed to parse log level, fallback to INFO: %w", err)
		}

		l.level = logLevel

		return nil
	}
}

func WithComponent(component string) LoggerOption {
	return func(l *logger) error {
		l.attributes = append(l.attributes, slog.String("component", component))

		return nil
	}
}

func WithFile(file string) LoggerOption {
	return func(l *logger) error {
		logFile, err := os.OpenFile(file, logFilesMode, logFilesPerm)
		if err != nil {
			return fmt.Errorf("failed to open log file: %w", err)
		}

		l.writer = io.MultiWriter(l.writer, logFile)

		return nil
	}
}

func InitStructuredLogger(options ...LoggerOption) error {
	l := &logger{
		level:      slog.LevelInfo,
		attributes: []slog.Attr{},
		writer:     os.Stdout,
	}

	var err error

	for _, option := range options {
		if oErr := option(l); oErr != nil {
			err = errors.Join(err, oErr)
		}
	}

	textHandler := slog.NewTextHandler(l.writer, &slog.HandlerOptions{
		Level: l.level,
	})

	handler := textHandler.WithAttrs(l.attributes)

	slog.SetDefault(slog.New(handler))
	slog.Info("Configured logger", "level", l.level.String())

	if err != nil {
		return fmt.Errorf("failed to set options: %w", err)
	}

	// Set writer from logger
	writer = l.writer

	return nil
}
