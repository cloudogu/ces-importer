package configuration

import (
	"fmt"
	"os"
)

const (
	logLevelEnv                 = "LOG_LEVEL"
	exporterHostEnv             = "EXPORTER_HOST"
	exporterSSHUserEnv          = "EXPORTER_SSH_USER"
	exporterApiKeyEnv           = "EXPORTER_API_KEY"
	migrationRegularScheduleEnv = "MIGRATION_REGULAR_SCHEDULE"
	migrationFinalScheduleEnv   = "MIGRATION_FINAL_SCHEDULE"
)

const errorFormat = "environment variable %s is not set"

// Configuration consists of configuration data. The most fields are obtained from the Helm chart
// values file by means of a configmap, while others are hardcoded or obtained from secrets.
type Configuration struct {
	// ExporterHost configures the FQDN under which the exporter will be available for CES data export. The importer
	// will contact the exporter API which returns all required data like data paths etc.
	// The exporter API endpoint is fixed and will be routed on source side.
	ExporterHost string
	// ExporterSSHUser contains the SSH account name that will be used during copying the data from the source to the
	// target system. This is usually the root user.
	ExporterSSHUser string
	// ExporterApiKey contains the API key to authenticate against the source system's exporter system info endpoint.
	ExporterApiKey string
	// ImporterPrivateSSHKeyPath contains the file path to the SSH private key used to identify against the source
	// system.
	ImporterPrivateSSHKeyPath string
	// LogLevel manages to granularity of log output.
	LogLevel string
	// regular_schedule triggers recurring migration jobs while the whole source system is running.
	// Uses CRON notation f. e. "0 4 * * *"
	MigrationRegularCron string
	// final schedule triggers the finishing migration job while the source system is supposed to be void of active
	// users.
	// Uses RFC 3339 notation f. e. "2025-04-03 12:34:56Z"
	MigrationFinalTimestamp string
}

func ReadConfigFromEnv() (Configuration, error) {
	conf := Configuration{}

	conf.LogLevel = os.Getenv(logLevelEnv)
	if conf.LogLevel == "" {
		conf.LogLevel = "INFO"
	}

	conf.ExporterHost = os.Getenv(exporterHostEnv)
	if conf.ExporterHost == "" {
		return conf, fmt.Errorf(errorFormat, exporterHostEnv)
	}

	conf.ExporterSSHUser = os.Getenv(exporterSSHUserEnv)
	if conf.ExporterSSHUser == "" {
		return conf, fmt.Errorf(errorFormat, exporterSSHUserEnv)
	}

	conf.ExporterApiKey = os.Getenv(exporterApiKeyEnv)
	if conf.ExporterApiKey == "" {
		return conf, fmt.Errorf(errorFormat, exporterApiKeyEnv)
	}

	conf.ImporterPrivateSSHKeyPath = "/importerSshPrivateKey"

	conf.MigrationRegularCron = os.Getenv(migrationRegularScheduleEnv)
	if conf.MigrationRegularCron == "" {
		return conf, fmt.Errorf(errorFormat, migrationRegularScheduleEnv)
	}

	conf.MigrationFinalTimestamp = os.Getenv(migrationFinalScheduleEnv)
	if conf.MigrationFinalTimestamp == "" {
		return conf, fmt.Errorf(errorFormat, migrationFinalScheduleEnv)
	}

	return conf, nil
}
