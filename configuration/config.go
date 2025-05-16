package configuration

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"os"
	"path"
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
	Level string `yaml:"level"`
}

// API contains the configuration data for the API connection to the source system.
type API struct {
	// ExporterHost configures the FQDN under which the exporter will be available for CES data export. The importer
	// will contact the exporter API which returns all required data like data paths etc.
	// The exporter API endpoint is fixed and will be routed on exporter side. This value is required.
	ExporterHost string `yaml:"host"`
	// ExporterApiKey contains the API key to authenticate against the source system's exporter system info endpoint.
	// This value is required.
	ExporterApiKey string `yaml:"apiKey"`
}

// Migration contains the configuration data for the migration schedule.
type Migration struct {
	// RegularCron triggers recurring migration jobs while the whole source system is running.
	// Uses CRON notation f. e. "0 4 * * *"
	// This value is required.
	RegularCron string `yaml:"regularSchedule"`
	// FinalTimestamp triggers the finishing migration job while the source system is supposed to be void of
	// active users.
	// Uses RFC 3339 notation f. e. "2025-04-03 12:34:56Z"
	// This value is optional, but a final migration without this value will then be impossible.
	FinalTimestamp string `yaml:"finalSchedule"`
}

// SSH contains the configuration data for the SSH connection to the source system.
type SSH struct {
	// User contains the SSH account name that will be used during copying the data from the source to the
	// target system. This is usually the root user. This value is required.
	User string `yaml:"user"`
	// PrivateSSHKeyPath contains the file path inside the container to the SSH private key used to identify
	// against the source system.  This value is required but hardcoded in the respective Helm chart.
	PrivateSSHKeyPath string `yaml:"privateKeyPath"`
	// SecretName specifies the Kubernetes secret name containing private SSH key data for authentication.
	SecretName string `yaml:"secretName"`
	// SecretDataKey specifies the key inside the secret containing the private SSH key data.
	SecretDataKey string `yaml:"secretDataKey"`
}

// ContainerImage contains the container image information
type ContainerImage struct {
	// Registry specifies the container registry to pull the image from
	Registry string `yaml:"registry"`
	// Repository specifies the image repository
	Repository string `yaml:"repository"`
	// Tag specifies the image version tag
	Tag string `yaml:"tag"`
}

// ImagePullSecret contains the name of a secret used for pulling images from private registries
type ImagePullSecret struct {
	Name string `yaml:"name"`
}

// ResourceRequirements defines the compute resources required by the container
type ResourceRequirements struct {
	// Limits specify the maximum resources that can be consumed
	Limits ResourceList `yaml:"limits"`
	// Requests specify the minimum resources that must be available
	Requests ResourceList `yaml:"requests"`
}

// ResourceList specifies CPU and memory resources
type ResourceList struct {
	// CPU specifies the amount of CPU the container can use
	CPU string `yaml:"cpu"`
	// Memory specifies the amount of memory the container can use
	Memory string `yaml:"memory"`
}

// JobContainer defines the configuration for the container that runs migration jobs.
// It includes image details, pull policy, secrets, and resource requirements.
type JobContainer struct {
	// JobConfigMap specifies the name of the configmap containing the job configuration.
	JobConfigMap string `yaml:"jobConfigMap"`
	// JobServiceAccount specifies the Kubernetes service account to be used for running the migration job pod.
	JobServiceAccount string `yaml:"jobServiceAccount"`
	// Image contains the container image information
	Image ContainerImage `yaml:"image"`
	// ImagePullPolicy defines when the kubelet should pull the image (Always, IfNotPresent, Never)
	ImagePullPolicy string `yaml:"imagePullPolicy"`
	// ImagePullSecrets contains names of secrets used for pulling the image from private registries
	ImagePullSecrets []ImagePullSecret `yaml:"imagePullSecrets"`
	// Resources defines the compute resources required by the container
	Resources ResourceRequirements `yaml:"resources"`
}

// ExcludePattern defines a pattern for files that should not be synchronized for a specific dogu
type ExcludePattern struct {
	// DoguName specifies the name of the dogu for which the files should not be synchronized.
	DoguName string `yaml:"dogu"`
	// Pattern specifies the file pattern for the excluded files.
	Pattern string `yaml:"pattern"`
}

// JobConfig contains the configuration data for the job container.
type JobConfig struct {
	// DoguVolumeBasePath specifies the base path for the Dogu volumes mounted in the job.
	DoguVolumeBasePath string `yaml:"doguVolumeBasePath"`
	// Exclude specifies a list of dogus for which specific files should not be synchronized.
	Exclude []ExcludePattern `yaml:"exclude"`
	// Verbose makes the sync process log on verbose mode
	Verbose bool `yaml:"verbose"`
}

// Smtp holds SMTP server configuration details required for sending emails.
type Smtp struct {
	Server   string   `yaml:"server"`   // SMTP server address (e.g., smtp.example.com)
	Port     string   `yaml:"port"`     // SMTP server port (default is "25" if not specified)
	Username string   `yaml:"username"` // Username for SMTP authentication
	Password string   `yaml:"password"` // Password for SMTP authentication
	From     string   `yaml:"from"`     // Sender's email address
	To       []string `yaml:"to"`       // List of recipient email addresses
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

type ConfigTypes interface {
	API | Logging | Migration | JobContainer | JobConfig | SSH | Smtp
}

func readConfigYAML[T ConfigTypes](configPath string) (T, error) {
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

	return config, nil
}
