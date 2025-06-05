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
	General
}

func ReadJobConfig() (Job, error) {
	if err := newValidator(); err != nil {
		return Job{}, fmt.Errorf("failed to initialize validator: %w", err)
	}

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

	generalConfig := General{
		ExcludedDogus: GetExcludedDogus(),
		Namespace:     namespace,
	}

	return Job{
		Logging:   loggingConfig,
		API:       apiConfig,
		SSH:       sshConfig,
		JobConfig: jobConfig,
		General:   generalConfig,
	}, nil
}
