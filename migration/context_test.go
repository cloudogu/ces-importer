package migration

import (
	"context"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSetFinalMigration(t *testing.T) {
	ctx := SetFinalMigration(context.Background())

	result, ok := ctx.Value(finalMigrationKey).(bool)
	assert.True(t, ok)
	assert.True(t, result)
}

func TestIsFinalMigration(t *testing.T) {
	t.Run("Final migration should be true", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), finalMigrationKey, true)
		assert.True(t, IsFinalMigration(ctx))
	})

	t.Run("Final migration should be false", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), finalMigrationKey, false)
		assert.False(t, IsFinalMigration(ctx))
	})

	t.Run("Final migration should be false when not set", func(t *testing.T) {
		assert.False(t, IsFinalMigration(context.Background()))
	})
}

func TestSetTriggerFQDNChange(t *testing.T) {
	ctx := SetTriggerFQDNChange(context.Background())

	result, ok := ctx.Value(triggerFQDNChange).(bool)
	assert.True(t, ok)
	assert.True(t, result)
}

func TestSetTriggerFQDNChangeFromEnv(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		expected bool
	}{
		{
			name:     "env to trigger fqdn change is set",
			envValue: "true",
			expected: true,
		},
		{
			name:     "env to trigger fqdn change is empty",
			envValue: "",
			expected: false,
		},
		{
			name:     "env to trigger fqdn change is set to false",
			envValue: "false",
			expected: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(envTriggerFQDNChange, tc.envValue)

			ctx := SetTriggerFQDNChangeFromEnv(context.Background())

			_, ok := ctx.Value(triggerFQDNChange).(bool)
			assert.Equal(t, tc.expected, ok)
		})
	}
}

func TestTriggerFQDNChange(t *testing.T) {
	t.Run("Trigger fqdn change should be true", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), triggerFQDNChange, true)
		assert.True(t, TriggerFQDNChange(ctx))
	})

	t.Run("Trigger fqdn change should be false", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), triggerFQDNChange, false)
		assert.False(t, TriggerFQDNChange(ctx))
	})

	t.Run("Trigger fqdn change should be false when not set", func(t *testing.T) {
		assert.False(t, TriggerFQDNChange(context.Background()))
	})
}
