package logging

import (
	"context"
	"github.com/cloudogu/ces-importer/configuration"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"log/slog"
	"os"
	"testing"
)

var testCtx = context.Background()

func Test_configureLogger(t *testing.T) {
	t.Run("should fallback to INFO on config error", func(t *testing.T) {
		originalWriter := createWriter
		createWriter = func() (io.Writer, error) {
			return os.Stderr, nil
		}

		defer func() { createWriter = originalWriter }()

		// given
		brokenConfig := configuration.Configuration{LogLevel: "banana"}

		// when
		err := Initialize(brokenConfig)
		require.NoError(t, err)

		// then
		assert.True(t, slog.Default().Enabled(testCtx, slog.LevelError))
		assert.True(t, slog.Default().Enabled(testCtx, slog.LevelWarn))
		assert.True(t, slog.Default().Enabled(testCtx, slog.LevelInfo))
		assert.False(t, slog.Default().Enabled(testCtx, slog.LevelDebug))
	})
	t.Run("should set loglevel to ERROR", func(t *testing.T) {
		originalWriter := createWriter
		createWriter = func() (io.Writer, error) {
			return os.Stderr, nil
		}

		defer func() { createWriter = originalWriter }()
		// given
		brokenConfig := configuration.Configuration{LogLevel: "ERROR"}

		// when
		err := Initialize(brokenConfig)
		require.NoError(t, err)

		// then
		assert.True(t, slog.Default().Enabled(testCtx, slog.LevelError))
		assert.False(t, slog.Default().Enabled(testCtx, slog.LevelWarn))
		assert.False(t, slog.Default().Enabled(testCtx, slog.LevelInfo))
		assert.False(t, slog.Default().Enabled(testCtx, slog.LevelDebug))
	})
	t.Run("should set loglevel to WARN", func(t *testing.T) {
		originalWriter := createWriter
		createWriter = func() (io.Writer, error) {
			return os.Stderr, nil
		}

		defer func() { createWriter = originalWriter }()

		// given
		config := configuration.Configuration{LogLevel: "WARN"}

		// when
		err := Initialize(config)
		require.NoError(t, err)

		// then
		assert.True(t, slog.Default().Enabled(testCtx, slog.LevelError))
		assert.True(t, slog.Default().Enabled(testCtx, slog.LevelWarn))
		assert.False(t, slog.Default().Enabled(testCtx, slog.LevelInfo))
		assert.False(t, slog.Default().Enabled(testCtx, slog.LevelDebug))
	})
}
