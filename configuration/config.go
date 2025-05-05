package configuration

import (
	"fmt"
	"os"
)

const (
	logLevelEnv                 = "LOG_LEVEL"
	importerNamespaceKeyEnv     = "IMPORTER_NAMESPACE"
	migrationRegularScheduleEnv = "MIGRATION_REGULAR_SCHEDULE"
	migrationFinalScheduleEnv   = "MIGRATION_FINAL_SCHEDULE"
)

const errorFormat = "environment variable %s is not set"

// Configuration consists of configuration data. The most fields are obtained from the Helm chart
// values file through a configmap, while others are hardcoded or obtained from secrets.
type Configuration struct {
	API
	SSH
	// ImporterNamespace contains the k8s namespace in which the importer Cloudogu EcoSystem is running., f. i.
	// "ecosystem". This value is required but inferred from the used Helm chart.
	ImporterNamespace string
	// LogLevel manages to granularity of log output. Values are (all in uppercase) in decreasing verbosity:
	// DEBUG, INFO, WARN, ERROR
	//  This value is optional and will default to INFO.
	LogLevel string
	// MigrationRegularCron triggers recurring migration jobs while the whole source system is running.
	// Uses CRON notation f. e. "0 4 * * *"
	// This value is required.
	MigrationRegularCron string
	// MigrationFinalTimestamp triggers the finishing migration job while the source system is supposed to be void of
	// active users.
	// Uses RFC 3339 notation f. e. "2025-04-03 12:34:56Z"
	// This value is optional, but a final migration without this value will then be impossible.
	MigrationFinalTimestamp string
}

func ReadConfigFromEnv() (Configuration, error) {
	conf := Configuration{}

	apiConf, err := ReadAPIConfiguration()
	if err != nil {
		return conf, fmt.Errorf("failed to read configuration for Exporter API: %w", err)
	}

	conf.API = apiConf

	sshConf, err := ReadSSHConfiguration()
	if err != nil {
		return conf, fmt.Errorf("failed to read configuration for SSH: %w", err)
	}

	conf.SSH = sshConf

	conf.LogLevel = os.Getenv(logLevelEnv)
	if conf.LogLevel == "" {
		conf.LogLevel = "INFO"
	}

	conf.MigrationRegularCron = os.Getenv(migrationRegularScheduleEnv)
	if conf.MigrationRegularCron == "" {
		return conf, fmt.Errorf(errorFormat, migrationRegularScheduleEnv)
	}

	conf.MigrationFinalTimestamp = os.Getenv(migrationFinalScheduleEnv)
	if conf.MigrationFinalTimestamp == "" {
		return conf, fmt.Errorf(errorFormat, migrationFinalScheduleEnv)
	}

	conf.ImporterNamespace = os.Getenv(importerNamespaceKeyEnv)
	if conf.ImporterNamespace == "" {
		return conf, fmt.Errorf(errorFormat, importerNamespaceKeyEnv)
	}

	return conf, nil
}
