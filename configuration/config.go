package configuration

import (
	"fmt"
	"github.com/cloudogu/ces-importer/mail"
	"os"
	"strings"
)

const (
	logLevelEnv                 = "LOG_LEVEL"
	exporterHostEnv             = "EXPORTER_HOST"
	exporterSSHUserEnv          = "EXPORTER_SSH_USER"
	exporterApiKeyEnv           = "EXPORTER_API_KEY"
	importerNamespaceKeyEnv     = "IMPORTER_NAMESPACE"
	migrationRegularScheduleEnv = "MIGRATION_REGULAR_SCHEDULE"
	migrationFinalScheduleEnv   = "MIGRATION_FINAL_SCHEDULE"
)

const (
	envSmtpServer   = "SMTP_SERVER"
	envSmtpPort     = "SMTP_PORT"
	envSmtpUsername = "SMTP_USERNAME"
	envSmtpPassword = "SMTP_PASSWORD"
	envSmtpFrom     = "SMTP_FROM"
	envSmtpTo       = "SMTP_TO"
)

const errorFormat = "environment variable %s is not set"

// Configuration consists of configuration data. The most fields are obtained from the Helm chart
// values file through a configmap, while others are hardcoded or obtained from secrets.
type Configuration struct {
	// ExporterHost configures the FQDN under which the exporter will be available for CES data export. The importer
	// will contact the exporter API which returns all required data like data paths etc.
	// The exporter API endpoint is fixed and will be routed on exporter side. This value is required.
	ExporterHost string
	// ExporterSSHUser contains the SSH account name that will be used during copying the data from the source to the
	// target system. This is usually the root user. This value is required.
	ExporterSSHUser string
	// ExporterApiKey contains the API key to authenticate against the source system's exporter system info endpoint.
	// This value is required.
	ExporterApiKey string
	// ImporterPrivateSSHKeyPath contains the file path inside the container to the SSH private key used to identify
	// against the source system.  This value is required but hardcoded in the respective Helm chart.
	ImporterPrivateSSHKeyPath string
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
	// MailConfig contains the smtp configuration for the mail server to which the migration log is sent
	MailConfig mail.SmtpConfig
}

func ReadConfigFromEnv() (Configuration, error) {
	var err error
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

	conf.ImporterNamespace = os.Getenv(importerNamespaceKeyEnv)
	if conf.ImporterNamespace == "" {
		return conf, fmt.Errorf(errorFormat, importerNamespaceKeyEnv)
	}

	conf.MailConfig, err = SmtpConfigFromEnv()
	if err != nil {
		return conf, fmt.Errorf("failed to get smtp config: %w", err)
	}

	return conf, nil
}

// SmtpConfigFromEnv reads SMTP configuration from environment variables and returns
// a SmtpConfig struct. Returns an error if required fields like server or from address are missing.
//
// Expected environment variables:
//   - SMTP_SERVER
//   - SMTP_PORT (optional, defaults to "25")
//   - SMTP_USERNAME
//   - SMTP_PASSWORD
//   - SMTP_FROM
//   - SMTP_TO (comma-separated list of recipient emails)
func SmtpConfigFromEnv() (mail.SmtpConfig, error) {
	server := os.Getenv(envSmtpServer)
	if server == "" {
		return mail.SmtpConfig{}, fmt.Errorf("smtp Server address is not configured")
	}
	port := os.Getenv(envSmtpPort)
	if port == "" {
		port = "25"
	}

	username := os.Getenv(envSmtpUsername)
	password := os.Getenv(envSmtpPassword)

	from := os.Getenv(envSmtpFrom)
	if from == "" {
		return mail.SmtpConfig{}, fmt.Errorf("smtp from is not configured")
	}
	toAsStr := os.Getenv(envSmtpTo)
	to := strings.Split(toAsStr, ",")

	return mail.SmtpConfig{
		Server:   server,
		Port:     port,
		Username: username,
		Password: password,
		From:     from,
		To:       to,
	}, nil
}
