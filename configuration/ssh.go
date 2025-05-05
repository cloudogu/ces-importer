package configuration

import (
	"fmt"
	"os"
)

const importerPrivateSSHKeyPath = "/importerSshPrivateKey"

const exporterSSHUserEnv = "EXPORTER_SSH_USER"

// SSH contains the configuration data for the SSH connection to the source system.
type SSH struct {
	// ExporterSSHUser contains the SSH account name that will be used during copying the data from the source to the
	// target system. This is usually the root user. This value is required.
	ExporterSSHUser string
	// ImporterPrivateSSHKeyPath contains the file path inside the container to the SSH private key used to identify
	// against the source system.  This value is required but hardcoded in the respective Helm chart.
	ImporterPrivateSSHKeyPath string
}

func ReadSSHConfiguration() (SSH, error) {
	confExporterSSHUser := os.Getenv(exporterSSHUserEnv)
	if confExporterSSHUser == "" {
		return SSH{}, fmt.Errorf(errorFormat, exporterSSHUserEnv)
	}

	return SSH{
		ExporterSSHUser:           confExporterSSHUser,
		ImporterPrivateSSHKeyPath: importerPrivateSSHKeyPath,
	}, nil
}
