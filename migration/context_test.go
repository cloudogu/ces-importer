package migration

import (
	"context"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSetFinalMigration(t *testing.T) {
	ctx := SetFinalMigration(context.Background())

	result, ok := ctx.Value(finalMigrationKey).(finalMigration)
	assert.True(t, ok)
	assert.True(t, bool(result))
}

func TestIsFinalMigration(t *testing.T) {
	t.Run("Final migration should be true", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), finalMigrationKey, finalMigration(true))
		assert.True(t, IsFinalMigration(ctx))
	})

	t.Run("Final migration should be false", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), finalMigrationKey, finalMigration(false))
		assert.False(t, IsFinalMigration(ctx))
	})

	t.Run("Final migration should be false when not set", func(t *testing.T) {
		assert.False(t, IsFinalMigration(context.Background()))
	})
}
