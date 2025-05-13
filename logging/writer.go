package logging

import (
	"fmt"
	"io"
	"os"
)

const (
	PathJobLogFile = "/home/ces-importer/migration-log/job.log"
)

type Writer struct {
	path   string
	remove osRemove
	create osCreate
	copy   ioCopy
}

func NewWriter(path string, remove osRemove, create osCreate, copy ioCopy) *Writer {
	return &Writer{
		path:   path,
		remove: remove,
		create: create,
		copy:   copy,
	}
}

func (w Writer) Write(readCloser io.ReadCloser) error {
	defer func() {
		_ = readCloser.Close()
	}()

	err := w.remove(w.path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to clear old log file: %w", err)
	}

	logFile, err := w.create(w.path)
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
