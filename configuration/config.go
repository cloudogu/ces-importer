package configuration

import (
	"fmt"
	"os"
)

const logLevelEnv = "LOG_LEVEL"
const exporterHostEnv = "EXPORTER_HOST"
const exporterPortEnv = "EXPORTER_PORT"
const exporterSSHUserEnv = "EXPORTER_SSH_USER"
const exporterSourceEnv = "EXPORTER_SOURCE"
const importerPrivateSSHKeyPathEnv = "IMPORTER_PRIVATE_SSH_KEY_PATH"
const importerDestinationEnv = "IMPORTER_DESTINATION"

const errorFormat = "environment variable %s is not set"

type Configuration struct {
	ExporterHost              string
	ExporterPort              string
	ExporterSSHUser           string
	ExporterSource            string
	ImporterPrivateSSHKeyPath string
	ImporterDestination       string
	LogLevel                  string
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

	conf.ExporterPort = os.Getenv(exporterPortEnv)
	if conf.ExporterPort == "" {
		return conf, fmt.Errorf(errorFormat, exporterPortEnv)
	}

	conf.ExporterSSHUser = os.Getenv(exporterSSHUserEnv)
	if conf.ExporterSSHUser == "" {
		return conf, fmt.Errorf(errorFormat, exporterSSHUserEnv)
	}

	conf.ExporterSource = os.Getenv(exporterSourceEnv)
	if conf.ExporterSource == "" {
		return conf, fmt.Errorf(errorFormat, exporterSourceEnv)
	}

	conf.ImporterPrivateSSHKeyPath = os.Getenv(importerPrivateSSHKeyPathEnv)
	if conf.ImporterPrivateSSHKeyPath == "" {
		return conf, fmt.Errorf(errorFormat, importerPrivateSSHKeyPathEnv)
	}

	conf.ImporterDestination = os.Getenv(importerDestinationEnv)
	if conf.ImporterDestination == "" {
		return conf, fmt.Errorf(errorFormat, importerDestinationEnv)
	}

	return conf, nil
}
