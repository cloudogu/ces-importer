package logging

import (
	"fmt"
	"io"
)

const (
	PathJobLogFile = "/home/ces-importer/migration-log/job.log"
)

type Writer struct {
	path     string
	copy     ioCopy
	openFile osOpenFile
}

func NewWriter(path string, copy ioCopy, openFile osOpenFile) *Writer {
	return &Writer{
		path:     path,
		copy:     copy,
		openFile: openFile,
	}
}

func (w Writer) Write(readCloser io.ReadCloser) error {
	defer func() {
		_ = readCloser.Close()
	}()

	logFile, err := w.openFile(w.path, logFilesMode, logFilesPerm)
	if err != nil {
		return fmt.Errorf("failed to create log file: %w", err)
	}
	defer func() {
		_ = logFile.Close()
	}()

	_, err = w.copy(logFile, readCloser)
	if err != nil {
		return fmt.Errorf("failed to copy log to path %s: %w", w.path, err)
	}

	return nil
}
