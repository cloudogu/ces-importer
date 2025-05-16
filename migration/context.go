package migration

import "context"

type migrationContextKeyType uint8

const finalMigrationKey migrationContextKeyType = iota

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
