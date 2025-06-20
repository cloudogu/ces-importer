package logging

import (
	"bytes"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"log/slog"
	"os"
	"testing"
)

func TestWithLevel(t *testing.T) {
	tests := []struct {
		name     string
		level    string
		expected string
		expErr   bool
		errMsg   string
	}{
		{
			name:     "level is set to DEBUG",
			level:    "DEBUG",
			expected: "DEBUG",
		},
		{
			name:     "level is set to INFO",
			level:    "INFO",
			expected: "INFO",
		},
		{
			name:     "level is set to WARN",
			level:    "WARN",
			expected: "WARN",
		},
		{
			name:     "level is set to ERROR",
			level:    "ERROR",
			expected: "ERROR",
		},
		{
			name:     "Fallback to INFO on parsing error",
			level:    "INVALID",
			expected: "INFO",
			expErr:   true,
			errMsg:   "failed to parse log level",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defaultLogger := &logger{}

			err := WithLevel(tt.level)(defaultLogger)

			if tt.expErr {
				assert.Error(t, err)
				assert.ErrorContains(t, err, tt.errMsg)
				return
			}

			assert.Equal(t, tt.expected, defaultLogger.level.String())
		})
	}
}

func TestWithFile(t *testing.T) {
	t.Run("should initialize with file", func(t *testing.T) {
		var byteBuffer bytes.Buffer

		testFile, err := os.CreateTemp(t.TempDir(), "test-log-file")
		require.NoError(t, err)

		defer func() { _ = testFile.Close() }()

		defaultLogger := &logger{writer: &byteBuffer}
		err = WithFile(testFile.Name())(defaultLogger)
		assert.NoError(t, err)

		_, err = fmt.Fprint(defaultLogger.writer, "test")
		assert.NoError(t, err)

		testFileOutput, err := io.ReadAll(testFile)
		require.NoError(t, err)

		assert.Equal(t, byteBuffer.String(), string(testFileOutput))
		assert.Equal(t, "test", string(testFileOutput))
	})

	t.Run("should fail with permission denied", func(t *testing.T) {
		testFile, err := os.CreateTemp(t.TempDir(), "test-log-file")
		require.NoError(t, err)

		err = testFile.Chmod(0000)
		require.NoError(t, err)

		defer func() { _ = testFile.Close() }()

		defaultLogger := &logger{writer: os.Stdout}
		err = WithFile(testFile.Name())(defaultLogger)

		assert.Error(t, err)
		assert.ErrorContains(t, err, "failed to open log file")
	})
}

func TestWithComponent(t *testing.T) {
	defaultLogger := &logger{}
	err := WithComponent("test")(defaultLogger)

	assert.NoError(t, err)
	assert.Len(t, defaultLogger.attributes, 1)
	assert.Equal(t, "component=test", defaultLogger.attributes[0].String())
}

func TestInitStructuredLogger(t *testing.T) {
	oldWriter := writer
	cleanUp := func() { writer = oldWriter }

	t.Run("Setup Default Logger", func(t *testing.T) {
		defer cleanUp()

		err := InitStructuredLogger()
		assert.NoError(t, err)

		assert.Equal(t, os.Stdout, writer)

		defaultLogger := slog.Default()
		assert.NotNil(t, defaultLogger)

		handler := defaultLogger.Handler()
		_, ok := handler.(*slog.TextHandler)
		assert.True(t, ok)
	})

	t.Run("Setup Logger with file writer", func(t *testing.T) {
		defer cleanUp()

		testFile, err := os.CreateTemp(t.TempDir(), "test-log-file")
		require.NoError(t, err)

		defer func() { _ = testFile.Close() }()

		err = InitStructuredLogger(
			WithFile(testFile.Name()),
		)
		assert.NoError(t, err)

		slog.Info("test")

		_, err = testFile.Seek(0, 0)
		require.NoError(t, err)
		output, err := io.ReadAll(testFile)
		require.NoError(t, err)
		assert.Contains(t, string(output), "test")
	})

	t.Run("Setup Logger with ERROR level", func(t *testing.T) {
		defer cleanUp()

		testFile, err := os.CreateTemp(t.TempDir(), "test-log-file")
		require.NoError(t, err)

		defer func() { _ = testFile.Close() }()

		err = InitStructuredLogger(
			WithLevel("ERROR"),
			WithFile(testFile.Name()),
		)
		assert.NoError(t, err)

		slog.Info("cloudogu")
		slog.Error("testERROR")

		_, err = testFile.Seek(0, 0)
		require.NoError(t, err)
		output, err := io.ReadAll(testFile)
		require.NoError(t, err)
		assert.NotContains(t, string(output), "cloudogu")
		assert.Contains(t, string(output), "testERROR")
	})

	t.Run("Fallback to INFO level", func(t *testing.T) {
		defer cleanUp()

		testFile, err := os.CreateTemp(t.TempDir(), "test-log-file")
		require.NoError(t, err)

		defer func() { _ = testFile.Close() }()

		err = InitStructuredLogger(
			WithLevel("INVALID"),
			WithFile(testFile.Name()),
		)
		assert.Error(t, err)

		slog.Info("cloudogu")

		_, err = testFile.Seek(0, 0)
		require.NoError(t, err)
		output, err := io.ReadAll(testFile)
		require.NoError(t, err)
		assert.Contains(t, string(output), "cloudogu")
	})

	t.Run("Return multiple errors", func(t *testing.T) {
		defer cleanUp()

		testFile, err := os.CreateTemp(t.TempDir(), "test-log-file")
		require.NoError(t, err)

		err = testFile.Chmod(0000)
		require.NoError(t, err)

		defer func() { _ = testFile.Close() }()

		err = InitStructuredLogger(
			WithLevel("INVALID"),
			WithFile(testFile.Name()),
		)
		assert.Error(t, err)
		assert.ErrorContains(t, err, "failed to parse log level")
		assert.ErrorContains(t, err, "failed to open log file")
	})
}

func TestGetWriter(t *testing.T) {
	t.Run("should return stdout writer on default", func(t *testing.T) {
		w := GetWriter()
		assert.Equal(t, os.Stdout, w)
	})

	t.Run("override writer", func(t *testing.T) {
		oldWriter := writer
		defer func() { writer = oldWriter }()

		writer = &bytes.Buffer{}
		w := GetWriter()
		assert.Equal(t, writer, w)
	})
}
