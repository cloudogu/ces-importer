package configuration

import (
	"context"
	"errors"
	"fmt"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
	"os"
	"path"
)

type secretGetter interface {
	Get(ctx context.Context, name string, opts metav1.GetOptions) (*corev1.Secret, error)
}

// Coordinator consists of configuration data. The most fields are obtained from the Helm chart
// values file through a configmap, while others are hardcoded or obtained from secrets.
type Coordinator struct {
	Logging
	API
	Migration
	SSH
	JobConfig
	JobContainer
	Smtp

	// Namespace contains the k8s namespace in which the importer Cloudogu EcoSystem is running., f. i.
	// "ecosystem". This value is required but inferred from the used Helm chart.
	Namespace string
}

// ValidateSecrets validates whether the secrets with their corresponding names exist as well as the defined data keys.
func (c Coordinator) ValidateSecrets(ctx context.Context, sg secretGetter) error {
	secretMap := make(map[string][]string)

	apiConfig := c.API
	secretMap[apiConfig.SecretName] = append(secretMap[apiConfig.SecretName], apiConfig.SecretDataKey)

	sshConfig := c.SSH
	secretMap[sshConfig.SecretName] = append(secretMap[sshConfig.SecretName], sshConfig.SecretDataKey)

	smtpConfig := c.Smtp
	if smtpConfig.Server != "" {
		secretMap[smtpConfig.SecretName] = append(secretMap[smtpConfig.SecretName], smtpConfig.SecretDataKey)
	}

	// retry when the error is other than NotFound
	retriable := func(err error) bool {
		return !apierrors.IsNotFound(err)
	}

	return retry.OnError(retry.DefaultRetry, retriable, func() error {
		var resultErr error

		for secretName, secretDataKeyList := range secretMap {
			secret, err := sg.Get(ctx, secretName, metav1.GetOptions{})
			if err != nil {
				resultErr = errors.Join(resultErr, fmt.Errorf("failed to get secret %s: %w", secretName, err))
				continue
			}

			for _, secretDataKey := range secretDataKeyList {
				_, ok := secret.Data[secretDataKey]
				if !ok {
					resultErr = errors.Join(resultErr, fmt.Errorf("secret %s does not contain key %s", secretName, secretDataKey))
					continue
				}
			}
		}

		return resultErr
	})
}

func ReadCoordinatorConfig() (Coordinator, error) {
	configBaseDir := os.Getenv(EnvBaseConfigPathKey)
	if configBaseDir == "" {
		return Coordinator{}, fmt.Errorf(errorFormat, EnvBaseConfigPathKey)
	}

	namespace := os.Getenv(EnvImporterNamespaceKey)
	if namespace == "" {
		return Coordinator{}, fmt.Errorf(errorFormat, EnvImporterNamespaceKey)
	}

	loggingConfig, err := readConfigYAML[Logging](path.Join(configBaseDir, fileLoggingConfig))
	if err != nil {
		return Coordinator{}, fmt.Errorf("failed to read logging configuration: %w", err)
	}

	apiConfig, err := readConfigYAML[API](path.Join(configBaseDir, fileAPIConfig))
	if err != nil {
		return Coordinator{}, fmt.Errorf("failed to read API configuration: %w", err)
	}

	migrationConfig, err := readConfigYAML[Migration](path.Join(configBaseDir, fileMigrationConfig))
	if err != nil {
		return Coordinator{}, fmt.Errorf("failed to read migration configuration: %w", err)
	}

	sshConfig, err := readConfigYAML[SSH](path.Join(configBaseDir, fileSSHConfig))
	if err != nil {
		return Coordinator{}, fmt.Errorf("failed to read ssh configuration: %w", err)
	}

	jobConfig, err := readConfigYAML[JobConfig](path.Join(configBaseDir, fileJobConfig))
	if err != nil {
		return Coordinator{}, fmt.Errorf("failed to read job configuration: %w", err)
	}

	jobContainerConfig, err := readConfigYAML[JobContainer](path.Join(configBaseDir, fileJobContainerConfig))
	if err != nil {
		return Coordinator{}, fmt.Errorf("failed to read job container configuration: %w", err)
	}

	smtpConfig, err := readConfigYAML[Smtp](path.Join(configBaseDir, fileSMTPConfig))
	if err != nil {
		return Coordinator{}, fmt.Errorf("failed to read smtp configuration: %w", err)
	}

	return Coordinator{
		Logging:      loggingConfig,
		API:          apiConfig,
		Migration:    migrationConfig,
		SSH:          sshConfig,
		JobConfig:    jobConfig,
		JobContainer: jobContainerConfig,
		Smtp:         smtpConfig,
		Namespace:    namespace,
	}, nil
}
