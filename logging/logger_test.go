package logging

import (
	"github.com/cloudogu/ces-importer"
	"github.com/cloudogu/ces-importer/configuration"
	"github.com/stretchr/testify/assert"
	"log/slog"
	"testing"
)

func Test_configureLogger(t *testing.T) {
	t.Run("should fallback to INFO on config error", func(t *testing.T) {
		// given
		brokenConfig := configuration.Configuration{LogLevel: "banana"}

		// when
		init(brokenConfig)

		// then
		assert.True(t, slog.Default().Enabled(main.testCtx, slog.LevelError))
		assert.True(t, slog.Default().Enabled(main.testCtx, slog.LevelWarn))
		assert.True(t, slog.Default().Enabled(main.testCtx, slog.LevelInfo))
		assert.False(t, slog.Default().Enabled(main.testCtx, slog.LevelDebug))
	})
	t.Run("should set loglevel to ERROR", func(t *testing.T) {
		// given
		brokenConfig := configuration.Configuration{LogLevel: "ERROR"}

		// when
		init(brokenConfig)

		// then
		assert.True(t, slog.Default().Enabled(main.testCtx, slog.LevelError))
		assert.False(t, slog.Default().Enabled(main.testCtx, slog.LevelWarn))
		assert.False(t, slog.Default().Enabled(main.testCtx, slog.LevelInfo))
		assert.False(t, slog.Default().Enabled(main.testCtx, slog.LevelDebug))
	})
	t.Run("should set loglevel to WARN", func(t *testing.T) {
		// given
		config := configuration.Configuration{LogLevel: "WARN"}

		// when
		init(config)

		// then
		assert.True(t, slog.Default().Enabled(main.testCtx, slog.LevelError))
		assert.True(t, slog.Default().Enabled(main.testCtx, slog.LevelWarn))
		assert.False(t, slog.Default().Enabled(main.testCtx, slog.LevelInfo))
		assert.False(t, slog.Default().Enabled(main.testCtx, slog.LevelDebug))
	})
}
