package configuration

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

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
		t.Setenv(importerNamespaceKeyEnv, "ecosystem")

		// when
		actualCfg, err := ReadConfigFromEnv()

		// then
		require.NoError(t, err)
		assert.Equal(t, Configuration{
			API: API{
				ExporterHost:   "source.net",
				ExporterApiKey: "example1-1234-5678-102938475",
			},
			SSH: SSH{
				ExporterSSHUser:           "root",
				ImporterPrivateSSHKeyPath: "/importerSshPrivateKey",
			},
			LogLevel:                "DEBUG",
			MigrationRegularCron:    "0 4 * * *",
			MigrationFinalTimestamp: "2025-04-03 12:34:56Z",
			ImporterNamespace:       "ecosystem",
		}, actualCfg)
	})
}

func TestReadConfigFromEnv_Errors(t *testing.T) {
	tests := []struct {
		name    string
		setEnv  []string
		wantErr assert.ErrorAssertionFunc
	}{
		{"host unset", []string{}, assert.Error},
		{"ssh user unset", []string{exporterHostEnv}, assert.Error},
		{"api key unset", []string{exporterHostEnv, exporterSSHUserEnv}, assert.Error},
		{"loglevel unset", []string{exporterHostEnv, exporterSSHUserEnv, exporterApiKeyEnv}, assert.Error},
		{"regular sched unset", []string{exporterHostEnv, exporterSSHUserEnv, exporterApiKeyEnv, logLevelEnv}, assert.Error},
		{"final sched unset", []string{exporterHostEnv, exporterSSHUserEnv, exporterApiKeyEnv, logLevelEnv, migrationRegularScheduleEnv}, assert.Error},
		{"namespace unset", []string{exporterHostEnv, exporterSSHUserEnv, exporterApiKeyEnv, logLevelEnv, migrationRegularScheduleEnv, migrationFinalScheduleEnv}, assert.Error},
		{"no errors", []string{exporterHostEnv, exporterSSHUserEnv, exporterApiKeyEnv, logLevelEnv, migrationRegularScheduleEnv, migrationFinalScheduleEnv, importerNamespaceKeyEnv}, assert.NoError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, env := range tt.setEnv {
				t.Setenv(env, "aTestValueFor:"+env)
			}
			// when
			_, err := ReadConfigFromEnv()
			if !tt.wantErr(t, err, fmt.Sprintf("ReadConfigFromEnv()")) {
				return
			}
		})
	}
}
