package migration

import (
	"context"
	"log/slog"
	"os"
)

type migrationContextKeyType uint8

const (
	finalMigrationKey migrationContextKeyType = iota
	triggerFQDNChange
)

const envTriggerFQDNChange = "TRIGGER_FQDN_CHANGE"

func SetFinalMigration(ctx context.Context) context.Context {
	return context.WithValue(ctx, finalMigrationKey, true)
}

func IsFinalMigration(ctx context.Context) bool {
	isSet, ok := ctx.Value(finalMigrationKey).(bool)
	if !ok {
		return false
	}

	return isSet
}

func SetTriggerFQDNChange(ctx context.Context) context.Context {
	return context.WithValue(ctx, triggerFQDNChange, true)
}

func SetTriggerFQDNChangeFromEnv(ctx context.Context) context.Context {
	if os.Getenv(envTriggerFQDNChange) != "" {
		ctx = SetTriggerFQDNChange(ctx)
		slog.Debug("Trigger for fqdn change has been set.")
	}

	return ctx
}

func TriggerFQDNChange(ctx context.Context) bool {
	isSet, ok := ctx.Value(triggerFQDNChange).(bool)
	if !ok {
		return false
	}

	return isSet
}
