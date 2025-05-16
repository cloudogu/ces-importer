package logging

import (
	"bytes"
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"io"
	"log/slog"
	"os"
	"testing"
)

var testCtx = context.Background()

func Test_configureLogger(t *testing.T) {
	t.Run("should fallback to INFO on config error", func(t *testing.T) {
		// given
		mockOpen := newMockOsOpenFile(t)
		mockOpen.EXPECT().Execute(PathAppLogFile, mock.Anything, mock.Anything).Return(nil, nil)
		mockWriter := newMockIoWriter(t)
		mockWriter.EXPECT().Write(mock.Anything).Return(0, nil)
		mockWrite := newMockCreateMultiWriter(t)
		mockWrite.EXPECT().Execute(mock.Anything, mock.Anything).Return(mockWriter)

		// when
		initializer := NewLogInitializer("banana")
		initializer.open = func(name string, flag int, perm os.FileMode) (file, error) {
			return mockOpen.Execute(name, flag, perm)
		}
		initializer.newMultiWriter = func(writers ...io.Writer) io.Writer {
			return mockWrite.Execute(writers...)
		}
		err := initializer.InitializeWithLogFile()
		require.NoError(t, err)

		// then
		assert.True(t, slog.Default().Enabled(testCtx, slog.LevelError))
		assert.True(t, slog.Default().Enabled(testCtx, slog.LevelWarn))
		assert.True(t, slog.Default().Enabled(testCtx, slog.LevelInfo))
		assert.False(t, slog.Default().Enabled(testCtx, slog.LevelDebug))
	})
	t.Run("should return error if file cannot be opened", func(t *testing.T) {
		// given
		mockOpen := newMockOsOpenFile(t)
		mockOpen.EXPECT().Execute(PathAppLogFile, mock.Anything, mock.Anything).Return(nil, fmt.Errorf("testerror"))
		mockWrite := newMockCreateMultiWriter(t)

		// when
		initializer := NewLogInitializer("banana")
		initializer.open = func(name string, flag int, perm os.FileMode) (file, error) {
			return mockOpen.Execute(name, flag, perm)
		}
		initializer.newMultiWriter = func(writers ...io.Writer) io.Writer {
			return mockWrite.Execute(writers...)
		}
		err := initializer.InitializeWithLogFile()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to open app log file: testerror")

		// then
		assert.True(t, slog.Default().Enabled(testCtx, slog.LevelError))
		assert.True(t, slog.Default().Enabled(testCtx, slog.LevelWarn))
		assert.True(t, slog.Default().Enabled(testCtx, slog.LevelInfo))
		assert.False(t, slog.Default().Enabled(testCtx, slog.LevelDebug))
	})
	t.Run("should set loglevel to ERROR", func(t *testing.T) {
		// given
		mockOpen := newMockOsOpenFile(t)
		mockOpen.EXPECT().Execute(PathAppLogFile, mock.Anything, mock.Anything).Return(nil, nil)
		mockWriter := newMockIoWriter(t)
		mockWrite := newMockCreateMultiWriter(t)
		mockWrite.EXPECT().Execute(mock.Anything, mock.Anything).Return(mockWriter)

		// when
		initializer := NewLogInitializer("ERROR")
		initializer.open = func(name string, flag int, perm os.FileMode) (file, error) {
			return mockOpen.Execute(name, flag, perm)
		}
		initializer.newMultiWriter = func(writers ...io.Writer) io.Writer {
			return mockWrite.Execute(writers...)
		}
		err := initializer.InitializeWithLogFile()
		require.NoError(t, err)

		// then
		assert.True(t, slog.Default().Enabled(testCtx, slog.LevelError))
		assert.False(t, slog.Default().Enabled(testCtx, slog.LevelWarn))
		assert.False(t, slog.Default().Enabled(testCtx, slog.LevelInfo))
		assert.False(t, slog.Default().Enabled(testCtx, slog.LevelDebug))
	})
	t.Run("should set loglevel to WARN", func(t *testing.T) {
		// given
		mockOpen := newMockOsOpenFile(t)
		mockOpen.EXPECT().Execute(PathAppLogFile, mock.Anything, mock.Anything).Return(nil, nil)
		mockWriter := newMockIoWriter(t)
		mockWrite := newMockCreateMultiWriter(t)
		mockWrite.EXPECT().Execute(mock.Anything, mock.Anything).Return(mockWriter)

		// when
		initializer := NewLogInitializer("WARN")
		initializer.open = func(name string, flag int, perm os.FileMode) (file, error) {
			return mockOpen.Execute(name, flag, perm)
		}
		initializer.newMultiWriter = func(writers ...io.Writer) io.Writer {
			return mockWrite.Execute(writers...)
		}
		err := initializer.InitializeWithLogFile()
		require.NoError(t, err)

		// then
		assert.True(t, slog.Default().Enabled(testCtx, slog.LevelError))
		assert.True(t, slog.Default().Enabled(testCtx, slog.LevelWarn))
		assert.False(t, slog.Default().Enabled(testCtx, slog.LevelInfo))
		assert.False(t, slog.Default().Enabled(testCtx, slog.LevelDebug))
	})
}

func TestLogInitializer_Initialize(t *testing.T) {
	t.Run("should initialize with stdout and level WARN", func(t *testing.T) {
		// given
		originalStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w
		defer func() {
			os.Stdout = originalStdout
		}()

		initializer := NewLogInitializer("WARN")
		err := initializer.Initialize()
		require.NoError(t, err)
		slog.Warn("test log msg")
		err = w.Close()
		require.NoError(t, err)

		// then
		assert.True(t, slog.Default().Enabled(testCtx, slog.LevelError))
		assert.True(t, slog.Default().Enabled(testCtx, slog.LevelWarn))
		assert.False(t, slog.Default().Enabled(testCtx, slog.LevelInfo))
		assert.False(t, slog.Default().Enabled(testCtx, slog.LevelDebug))

		// Read captured output
		var buf bytes.Buffer
		_, err = io.Copy(&buf, r)
		require.NoError(t, err)
		assert.Contains(t, buf.String(), "level=WARN msg=\"test log msg\"\n")

	})
}
