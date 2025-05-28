package configuration

import (
	"fmt"
	"os"
	"path"
)

// Job consists of configuration data. The most fields are obtained from the Helm chart
// values file through the YAML files.
type Job struct {
	Logging
	API
	SSH
	JobConfig

	// Namespace contains the k8s namespace in which the importer Cloudogu EcoSystem is running., f. i.
	// "ecosystem". This value is required but inferred from the used Helm chart.
	Namespace string
}

func ReadJobConfig() (Job, error) {
	configBaseDir := os.Getenv(EnvBaseConfigPathKey)
	if configBaseDir == "" {
		return Job{}, fmt.Errorf(errorFormat, EnvBaseConfigPathKey)
	}

	namespace := os.Getenv(EnvImporterNamespaceKey)
	if namespace == "" {
		return Job{}, fmt.Errorf(errorFormat, EnvImporterNamespaceKey)
	}

	loggingConfig, err := readConfigYAML[Logging](path.Join(configBaseDir, fileLoggingConfig))
	if err != nil {
		return Job{}, fmt.Errorf("failed to read logging configuration: %w", err)
	}

	apiConfig, err := readConfigYAML[API](path.Join(configBaseDir, fileAPIConfig))
	if err != nil {
		return Job{}, fmt.Errorf("failed to read API configuration: %w", err)
	}

	sshConfig, err := readConfigYAML[SSH](path.Join(configBaseDir, fileSSHConfig))
	if err != nil {
		return Job{}, fmt.Errorf("failed to read ssh configuration: %w", err)
	}

	jobConfig, err := readConfigYAML[JobConfig](path.Join(configBaseDir, fileJobConfig))
	if err != nil {
		return Job{}, fmt.Errorf("failed to read job configuration: %w", err)
	}

	return Job{
		Logging:   loggingConfig,
		API:       apiConfig,
		SSH:       sshConfig,
		JobConfig: jobConfig,
		Namespace: namespace,
	}, nil
}
