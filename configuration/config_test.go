package configuration

import (
	"fmt"
	"os"
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
		t.Setenv("SMTP_SERVER", "server")
		t.Setenv("SMTP_PORT", "1")
		t.Setenv("SMTP_FROM", "from")

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
			ImporterNamespace:         "ecosystem",
			MailConfig: SmtpConfig{
				Server: "server",
				Port:   "1",
				From:   "from",
				To:     []string{""},
			},
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
		{"no errors", []string{exporterHostEnv, exporterSSHUserEnv, exporterApiKeyEnv, logLevelEnv, migrationRegularScheduleEnv, migrationFinalScheduleEnv, importerNamespaceKeyEnv, "SMTP_SERVER", "SMTP_PORT", "SMTP_FROM"}, assert.NoError},
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

func TestSmtpConfigFromEnv(t *testing.T) {
	t.Run("can get config from env", func(t *testing.T) {
		_ = os.Setenv(envSmtpServer, "server")
		_ = os.Setenv(envSmtpPort, "port")
		_ = os.Setenv(envSmtpUsername, "username")
		_ = os.Setenv(envSmtpPassword, "password")
		_ = os.Setenv(envSmtpFrom, "from")
		_ = os.Setenv(envSmtpTo, "to")

		config, err := SmtpConfigFromEnv()
		require.NoError(t, err)
		assert.Equal(t, "server", config.Server)
		assert.Equal(t, "port", config.Port)
		assert.Equal(t, "username", config.Username)
		assert.Equal(t, "password", config.Password)
		assert.Equal(t, "from", config.From)
		assert.Equal(t, []string{"to"}, config.To)
	})

	t.Run("fail on unset server", func(t *testing.T) {
		_ = os.Unsetenv(envSmtpServer)
		_ = os.Setenv(envSmtpPort, "port")
		_ = os.Setenv(envSmtpUsername, "username")
		_ = os.Setenv(envSmtpPassword, "password")
		_ = os.Setenv(envSmtpFrom, "from")
		_ = os.Setenv(envSmtpTo, "to")

		_, err := SmtpConfigFromEnv()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "smtp Server address is not configured")
	})

	t.Run("fallback to 25 on unset port", func(t *testing.T) {
		_ = os.Setenv(envSmtpServer, "server")
		_ = os.Unsetenv(envSmtpPort)
		_ = os.Setenv(envSmtpUsername, "username")
		_ = os.Setenv(envSmtpPassword, "password")
		_ = os.Setenv(envSmtpFrom, "from")
		_ = os.Setenv(envSmtpTo, "to")

		config, err := SmtpConfigFromEnv()
		require.NoError(t, err)
		assert.Equal(t, "25", config.Port)
	})

	t.Run("fail on unset from", func(t *testing.T) {
		_ = os.Setenv(envSmtpServer, "server")
		_ = os.Setenv(envSmtpPort, "port")
		_ = os.Setenv(envSmtpUsername, "username")
		_ = os.Setenv(envSmtpPassword, "password")
		_ = os.Unsetenv(envSmtpFrom)
		_ = os.Setenv(envSmtpTo, "to")

		_, err := SmtpConfigFromEnv()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "smtp from is not configured")
	})
}
