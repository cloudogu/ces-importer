package migration

import "context"

const finalMigrationKey = "finalMigration"

type finalMigration bool

func SetFinalMigration(ctx context.Context) context.Context {
	return context.WithValue(ctx, finalMigrationKey, finalMigration(true))
}

func IsFinalMigration(ctx context.Context) bool {
	isSet, ok := ctx.Value(finalMigrationKey).(finalMigration)
	if !ok {
		return false
	}

	return bool(isSet)
}
