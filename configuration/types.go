package configuration

import (
	"errors"
	"fmt"
	"github.com/go-playground/validator/v10"
	"gopkg.in/yaml.v3"
	"os"
)

const (
	EnvBaseConfigPathKey    = "CONFIG_PATH"
	EnvImporterNamespaceKey = "IMPORTER_NAMESPACE"
)

const (
	fileLoggingConfig      = "logging.yaml"
	fileAPIConfig          = "api.yaml"
	fileMigrationConfig    = "migration.yaml"
	fileSSHConfig          = "ssh.yaml"
	fileJobContainerConfig = "job-container.yaml"
	fileJobConfig          = "job.yaml"
	fileSMTPConfig         = "smtp.yaml"
)

const errorFormat = "environment variable %s is not set"

// Logging contains the configuration data for the logging.
type Logging struct {
	// Level manages to granularity of log output. Values are (all in uppercase) in decreasing verbosity:
	// DEBUG, INFO, WARN, ERROR
	Level string `yaml:"level" validate:"oneof=DEBUG INFO WARN ERROR"`
}

// API contains the configuration data for the API connection to the source system.
type API struct {
	// ExporterHost configures the FQDN under which the exporter will be available for CES data export. The importer
	// will contact the exporter API which returns all required data like data paths etc.
	// The exporter API endpoint is fixed and will be routed on the exporter side. This value is required.
	ExporterHost string `yaml:"host" validate:"required,hostname_rfc1123"`
	// ExporterApiKey contains the API key to authenticate against the source system's exporter system info endpoint.
	// This value is required.
	ExporterApiKey string `yaml:"apiKey" validate:"min=8,max=128"`
	// SkipTLSVerify controls whether to skip check the server's certificate
	SkipTLSVerify bool `yaml:"skipTLSVerify"`
	// SecretName specifies the Kubernetes secret name containing the exporter API key for authentication.
	SecretName string `yaml:"secretName" validate:"required,k8sSecretName"`
	// SecretDataKey specifies the key inside the secret containing the exporter API key.
	SecretDataKey string `yaml:"secretDataKey" validate:"required,k8sSecretDataKey"`
}

// Migration contains the configuration data for the migration schedule.
type Migration struct {
	// RegularCron triggers recurring migration jobs while the whole source system is running.
	// Uses CRON notation f. e. "0 4 * * *"
	// This value is required.
	RegularCron string `yaml:"regularSchedule" validate:"required,cron"`
	// FinalTimestamp triggers the finishing migration job while the source system is supposed to be void of
	// active users.
	// Uses RFC 3339 notation f. e. "2025-04-03T12:34:56Z"
	// This value is optional, but a final migration without this value will then be impossible.
	FinalTimestamp string `yaml:"finalSchedule" validate:"omitempty,datetime=2006-01-02T15:04:05Z07:00"`
	// ChangeFQDN triggers a fqdn change.
	// The certificates and the fqdn get migrated from the exporter to the import site.
	// This is only taken into account when the job runs as final migration.
	// Default: False
	ChangeFQDN bool `yaml:"changeFQDN"`
	// MaintenanceModeMessage is the message to be shown at the source system when the maintenance mode gets activated.
	MaintenanceModeMessage *MaintenanceModeMessage `yaml:"maintenanceModeMessage" validate:"required_with=FinalTimestamp"`
}

// MaintenanceModeMessage is the message to be shown at the source system when the maintenance mode gets activated.
type MaintenanceModeMessage struct {
	// Title to be shown in the maintenance mode message.
	Title string `yaml:"title" validate:"required"`
	// Text to be shown in the maintenance mode message.
	Text string `yaml:"text" validate:"required"`
}

// SSH contains the configuration data for the SSH connection to the source system.
type SSH struct {
	// User contains the SSH account name that will be used during copying the data from the source to the
	// target system. This is usually the root user. This value is required.
	User string `yaml:"user" validate:"required"`
	// PrivateSSHKeyPath contains the file path inside the container to the SSH private key used to identify
	// against the source system.  This value is required but hardcoded in the respective Helm chart.
	PrivateSSHKeyPath string `yaml:"privateKeyPath" validate:"required"`
	// SecretName specifies the Kubernetes secret name containing private SSH key data for authentication.
	SecretName string `yaml:"secretName" validate:"required,k8sSecretName"`
	// SecretDataKey specifies the key inside the secret containing the private SSH key data.
	SecretDataKey string `yaml:"secretDataKey" validate:"required,k8sSecretDataKey"`
}

// ContainerImage contains the container image information
type ContainerImage struct {
	// Registry specifies the container registry to pull the image from
	Registry string `yaml:"registry" validate:"required,hostname_rfc1123"`
	// Repository specifies the image repository
	Repository string `yaml:"repository" validate:"required"`
	// Tag specifies the image version tag
	Tag string `yaml:"tag" validate:"required"`
}

// ImagePullSecret contains the name of a secret used for pulling images from private registries
type ImagePullSecret struct {
	Name string `yaml:"name"`
}

// ResourceRequirements defines the compute resources required by the container
type ResourceRequirements struct {
	// Limits specify the maximum resources that can be consumed
	Limits ResourceList `yaml:"limits" validate:"required"`
	// Requests specify the minimum resources that must be available
	Requests ResourceList `yaml:"requests" validate:"required"`
}

// ResourceList specifies CPU and memory resources
type ResourceList struct {
	// CPU specifies the amount of CPU the container can use
	CPU string `yaml:"cpu" validate:"required,alphanum"`
	// Memory specifies the amount of memory the container can use
	Memory string `yaml:"memory" validate:"required,alphanum"`
}

// JobContainer defines the configuration for the container that runs migration jobs.
// It includes image details, pull policy, secrets, and resource requirements.
type JobContainer struct {
	// JobConfigMap specifies the name of the configmap containing the job configuration.
	JobConfigMap string `yaml:"jobConfigMap" validate:"required,k8sSecretName"`
	// JobServiceAccount specifies the Kubernetes service account to be used for running the migration job pod.
	JobServiceAccount string `yaml:"jobServiceAccount" validate:"required,k8sSecretName"`
	// Image contains the container image information
	Image ContainerImage `yaml:"image" validate:"required"`
	// ImagePullPolicy defines when the kubelet should pull the image (Always, IfNotPresent, Never)
	ImagePullPolicy string `yaml:"imagePullPolicy" validate:"required,oneof=Always IfNotPresent Never"`
	// ImagePullSecrets contains names of secrets used for pulling the image from private registries
	ImagePullSecrets []ImagePullSecret `yaml:"imagePullSecrets"`
	// Resources defines the compute resources required by the container
	Resources ResourceRequirements `yaml:"resources" validate:"required"`
}

// ExcludePattern defines a pattern for files that should not be synchronized for a specific dogu
type ExcludePattern struct {
	// DoguName specifies the name of the dogu for which the files should not be synchronized.
	DoguName string `yaml:"dogu" validate:"required"`
	// Pattern specifies the file pattern for the excluded files.
	Pattern string `yaml:"pattern" validate:"required"`
}

// JobConfig contains the configuration data for the job container.
type JobConfig struct {
	// DoguVolumeBasePath specifies the base path for the Dogu volumes mounted in the job.
	DoguVolumeBasePath string `yaml:"doguVolumeBasePath" validate:"required"`
	// Exclude specifies a list of dogus for which specific files should not be synchronized.
	Exclude []ExcludePattern `yaml:"exclude" validate:"omitempty,dive"`
	// Verbose makes the sync process log in verbose mode
	Verbose bool `yaml:"verbose"`
}

// Smtp holds SMTP server configuration details required for sending emails.
type Smtp struct {
	// SMTP server address (e.g., smtp.example.com)
	Server string `yaml:"server" validate:"omitempty,hostname_rfc1123"`
	// SMTP server port (default is "25" if not specified)
	Port uint16 `yaml:"port" validate:"required_with=Server"`
	// Username for SMTP authentication
	Username string `yaml:"username"`
	// Password for SMTP authentication
	Password string `yaml:"password"`
	// Sender's email address
	From string `yaml:"from" validate:"required_with=Server,len=0|email"`
	// List of recipient email addresses
	To []string `yaml:"to" validate:"required_with=Server,dive,email"`
}

type Types interface {
	API | Logging | Migration | JobContainer | JobConfig | SSH | Smtp
}

func readConfigYAML[T Types](configPath string) (T, error) {
	var config T

	content, err := os.ReadFile(configPath)
	if err != nil {
		return config, fmt.Errorf("failed to read config file %s: %w", configPath, err)
	}

	// Expand environment variables like ${VAR}
	fullContent := os.ExpandEnv(string(content))

	if err = yaml.Unmarshal([]byte(fullContent), &config); err != nil {
		return config, fmt.Errorf("failed to unmarshal config file %s: %w", configPath, err)
	}

	if err = validate.Struct(config); err != nil {
		return config, fmt.Errorf("failed to validate config file %s: %v", configPath, joinValidationErrors(err))
	}

	return config, nil
}

func joinValidationErrors(err error) error {
	var vErrs validator.ValidationErrors
	errors.As(err, &vErrs)

	var allVErrors error

	for _, vErr := range vErrs {
		allVErrors = errors.Join(allVErrors, errors.New(vErr.Translate(trans)))
	}

	return allVErrors
}
