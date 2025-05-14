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
	path     string
	copy     ioCopy
	openFile osOpenFile
}

func NewWriter(path string) *Writer {
	return &Writer{
		path: path,
		copy: io.Copy,
		openFile: func(name string, flag int, perm os.FileMode) (file, error) {
			return os.OpenFile(name, flag, perm)
		},
	}
}

func (w Writer) Write(readCloser io.ReadCloser) error {
	defer func() {
		_ = readCloser.Close()
	}()

	logFile, err := w.openFile(w.path, logFilesMode, logFilesPerm)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
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
