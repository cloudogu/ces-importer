package migration

import "context"

const finalMigrationKey = "finalMigration"

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
