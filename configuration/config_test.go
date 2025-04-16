package configuration

import (
	"github.com/stretchr/testify/assert"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReadConfigFromEnv(t *testing.T) {
	t.Run("should return with no error", func(t *testing.T) {
		// given
		t.Setenv(exporterHostEnv, "source.net")
		t.Setenv(exporterSSHUserEnv, "root")
		t.Setenv(exporterApiKeyEnv, "example1-1234-5678-102938475")
		t.Setenv(logLevelEnv, "DEBUG")
		t.Setenv(migrationRegularScheduleEnv, "0 4 * * *")
		t.Setenv(migrationFinalScheduleEnv, "2025-04-03 12:34:56Z")

		// when
		actualCfg, err := ReadConfigFromEnv()

		// then
		require.NoError(t, err)
		assert.Equal(t, Configuration{
			ExporterHost:              "source.net",
			ExporterSSHUser:           "root",
			ExporterApiKey:            "example1-1234-5678-102938475",
			ImporterPrivateSSHKeyPath: "/importerSshPrivateKey",
			LogLevel:                  "DEBUG",
			MigrationRegularCron:      "0 4 * * *",
			MigrationFinalTimestamp:   "2025-04-03 12:34:56Z",
		}, actualCfg)
	})

}
