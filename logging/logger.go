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
	err        error
}

type LoggerOption func(*logger)

func WithLevel(level string) LoggerOption {
	return func(l *logger) {
		var logLevel slog.Level

		if err := logLevel.UnmarshalText([]byte(level)); err != nil {
			l.err = errors.Join(l.err, fmt.Errorf("failed to parse log level, fallback to INFO: %w", err))
		}

		l.level = logLevel
	}
}

func WithComponent(component string) LoggerOption {
	return func(l *logger) {
		l.attributes = append(l.attributes, slog.String("component", component))
	}
}

func WithFile(file string) LoggerOption {
	return func(l *logger) {
		logFile, err := os.OpenFile(file, logFilesMode, logFilesPerm)
		if err != nil {
			l.err = errors.Join(l.err, fmt.Errorf("failed to open log file: %w", err))
			return
		}

		l.writer = io.MultiWriter(l.writer, logFile)
	}
}

func InitStructuredLogger(options ...LoggerOption) error {
	l := &logger{
		level:      slog.LevelInfo,
		attributes: []slog.Attr{},
		writer:     os.Stdout,
	}

	for _, option := range options {
		option(l)
	}

	textHandler := slog.NewTextHandler(l.writer, &slog.HandlerOptions{
		Level: l.level,
	})

	handler := textHandler.WithAttrs(l.attributes)

	slog.SetDefault(slog.New(handler))
	slog.Info("Configured logger", "level", l.level.String())

	if l.err != nil {
		return l.err
	}

	// Set writer from logger
	writer = l.writer

	return nil
}
