package configuration

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"path"
	"testing"
)

// Helper function to create valid config files
func createValidConfig(t *testing.T, dir string, filename string) {
	var content string
	switch filename {
	case fileLoggingConfig:
		content = `---
level: "DEBUG"
`
	case fileAPIConfig:
		content = `---
apiKey: "testAPIKey"
host: "classic-ces.exporter"
skipTLSVerify: true
secretName: "ces-exporter-secret"
secretDataKey: "apiKey"
`
	case fileMigrationConfig:
		content = `---
finalSchedule: "2025-04-03T12:34:56Z"
regularSchedule: "0 4 * * *"
changeFQDN: true
maintenanceModeMessage:
  title: "Migration completed."
  text: "The migration of your instance has been completed."
`
	case fileSSHConfig:
		content = `---
privateKeyPath: "/.ssh/privateKey"
secretDataKey: "privateKey"
secretName: "ces-importer-secret"
user: "root"
`
	case fileJobContainerConfig:
		content = `---
image:
  registry: "docker.io"
  repository: "cloudogu/ces-importer"
  tag: "0.0.1"
imagePullPolicy: "IfNotPresent"
imagePullSecrets:
  - name: "ces-container-registries"
resources:
  limits:
    cpu: "500m"
    memory: "256Mi"
  requests:
    cpu: "100m"
    memory: "128Mi"
jobConfigMap: "ces-importer-job-config"
jobServiceAccount: "ces-importer-main-manager"
`
	case fileJobConfig:
		content = `---
doguVolumeBasePath: "/data"
exclude:
  - dogu: "jenkins"
    pattern: "JENKINS_PATTERN"
  - dogu: "redmine"
    pattern: "REDMINE_PATTERN"
`
	case fileSMTPConfig:
		content = `---
server: 192.168.56.1
port: 1025
username:
password:
from: importer@ces.com
to: []
`
	}

	err := os.WriteFile(path.Join(dir, filename), []byte(content), 0600)
	require.NoError(t, err)
}

// Helper function to write invalid YAML
func writeInvalidYaml(t *testing.T, dir string, filename string) {
	err := os.WriteFile(path.Join(dir, filename), []byte("invalid: yaml: }{"), 0600)
	require.NoError(t, err)
}

func Test_validateAPIConfig(t *testing.T) {
	const testString129 = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789abcde"

	tests := []struct {
		name      string
		inConfig  string
		expectErr bool
	}{
		{
			name: "valid api config",
			inConfig: `---
apiKey: "testAPIKey"
host: "classic-ces.exporter"
skipTLSVerify: true
secretName: "ces-exporter-secret"
secretDataKey: "apiKey"
`,
			expectErr: false,
		},
		{
			name: "error - empty host",
			inConfig: `---
apiKey: "testAPIKey"
host: ""
skipTLSVerify: true
secretName: "ces-exporter-secret"
secretDataKey: "apiKey"
`,
			expectErr: true,
		},
		{
			name: "error - invalid host",
			inConfig: `---
apiKey: "testAPIKey"
host: ".invalid$Host_*"
skipTLSVerify: true
secretName: "ces-exporter-secret"
secretDataKey: "apiKey"
`,
			expectErr: true,
		},
		{
			name: "error - empty apiKey",
			inConfig: `---
apiKey: ""
host: "classic-ces.exporter"
skipTLSVerify: true
secretName: "ces-exporter-secret"
secretDataKey: "apiKey"
`,
			expectErr: true,
		},
		{
			name: "error - apiKey too short",
			inConfig: `---
apiKey: "test"
host: "classic-ces.exporter"
skipTLSVerify: true
secretName: "ces-exporter-secret"
secretDataKey: "apiKey"
`,
			expectErr: true,
		},
		{
			name: "error - apiKey too long",
			inConfig: fmt.Sprintf(`---
apiKey: "%s"
host: "classic-ces.exporter"
skipTLSVerify: true
secretName: "ces-exporter-secret"
secretDataKey: "apiKey"
`, testString129),
			expectErr: true,
		},
		{
			name: "error - empty secretName",
			inConfig: `---
apiKey: "testAPIKey"
host: "classic-ces.exporter"
skipTLSVerify: true
secretName: ""
secretDataKey: "apiKey"
`,
			expectErr: true,
		},
		{
			name: "error - invalid secretName",
			inConfig: `---
apiKey: "testAPIKey"
host: "classic-ces.exporter"
skipTLSVerify: true
secretName: "ces/exporter/secret!!!"
secretDataKey: "apiKey"
`,
			expectErr: true,
		},
		{
			name: "error - empty secretDataKey",
			inConfig: `---
apiKey: "testAPIKey"
host: "classic-ces.exporter"
skipTLSVerify: true
secretName: "ces-exporter-secret"
secretDataKey: ""
`,
			expectErr: true,
		},
		{
			name: "error - invalid secretDataKey",
			inConfig: `---
apiKey: "testAPIKey"
host: "classic-ces.exporter"
skipTLSVerify: true
secretName: "ces-exporter-secret"
secretDataKey: "ces/apiKey/secret!!!"
`,
			expectErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			configPath := path.Join(t.TempDir(), fileAPIConfig)
			err := os.WriteFile(configPath, []byte(tc.inConfig), 0600)
			require.NoError(t, err)

			_, err = readConfigYAML[API](configPath)

			assert.Equal(t, tc.expectErr, err != nil)
		})
	}
}

func Test_validateLoggingConfig(t *testing.T) {
	tests := []struct {
		name      string
		inConfig  string
		expectErr bool
	}{
		{
			name: "valid logging config - DEBUG",
			inConfig: `---
level: "DEBUG"
`,
			expectErr: false,
		},
		{
			name: "valid logging config - INFO",
			inConfig: `---
level: "INFO"
`,
			expectErr: false,
		},
		{
			name: "valid logging config - WARN",
			inConfig: `---
level: "WARN"
`,
			expectErr: false,
		},
		{
			name: "valid logging config - ERROR",
			inConfig: `---
level: "ERROR"
`,
			expectErr: false,
		},
		{
			name: "error - level empty",
			inConfig: `---
level: ""
`,
			expectErr: true,
		},
		{
			name: "error - invalid level",
			inConfig: `---
level: "TRACE"
`,
			expectErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			configPath := path.Join(t.TempDir(), fileLoggingConfig)
			err := os.WriteFile(configPath, []byte(tc.inConfig), 0600)
			require.NoError(t, err)

			_, err = readConfigYAML[Logging](configPath)

			assert.Equal(t, tc.expectErr, err != nil)
		})
	}
}

func Test_validateMigrationConfig(t *testing.T) {
	tests := []struct {
		name      string
		inConfig  string
		expectErr bool
	}{
		{
			name: "valid migration config",
			inConfig: `---
finalSchedule: "2025-04-03T12:34:56Z"
regularSchedule: "0 4 * * *"
changeFQDN: true
maintenanceModeMessage:
  title: "Migration completed."
  text: "The migration of your instance has been completed."
`,
			expectErr: false,
		},
		{
			name: "valid migration config - empty finalSchedule timestamp",
			inConfig: `---
finalSchedule: ""
regularSchedule: "0 4 * * *"
changeFQDN: true
maintenanceModeMessage:
  title: "Migration completed."
  text: "The migration of your instance has been completed."
`,
			expectErr: false,
		},
		{
			name: "valid migration config - empty maintenanceModeMessage when finaleSchedule is empty",
			inConfig: `---
finalSchedule: ""
regularSchedule: "0 4 * * *"
changeFQDN: true
`,
			expectErr: false,
		},
		{
			name: "error - wrong finalSchedule timestamp format",
			inConfig: `---
finalSchedule: "2025-04-03  12:34:56Z"
regularSchedule: "0 4 * * *"
changeFQDN: true
maintenanceModeMessage:
  title: "Migration completed."
  text: "The migration of your instance has been completed."
`,
			expectErr: true,
		},
		{
			name: "error - empty regularSchedule",
			inConfig: `---
finalSchedule: "2025-04-03T12:34:56Z"
regularSchedule: ""
changeFQDN: true
maintenanceModeMessage:
  title: "Migration completed."
  text: "The migration of your instance has been completed."
`,
			expectErr: true,
		},
		{
			name: "error - invalid cron format in regularSchedule",
			inConfig: `---
finalSchedule: "2025-04-03T12:34:56Z"
regularSchedule: "invalid"
changeFQDN: true
maintenanceModeMessage:
  title: "Migration completed."
  text: "The migration of your instance has been completed."
`,
			expectErr: true,
		},
		{
			name: "error - empty maintenanceModeMessage when finalSchedule is set",
			inConfig: `---
finalSchedule: "2025-04-03T12:34:56Z"
regularSchedule: "0 4 * * *"
changeFQDN: true
`,
			expectErr: true,
		},
		{
			name: "error - empty maintenanceModeMessage title when finalSchedule is set",
			inConfig: `---
finalSchedule: "2025-04-03T12:34:56Z"
regularSchedule: "0 4 * * *"
changeFQDN: true
maintenanceModeMessage:
  title: ""
  text: "The migration of your instance has been completed."
`,
			expectErr: true,
		},
		{
			name: "error - empty maintenanceModeMessage message when finalSchedule is set",
			inConfig: `---
finalSchedule: "2025-04-03T12:34:56Z"
regularSchedule: "0 4 * * *"
changeFQDN: true
maintenanceModeMessage:
  title: "Migration completed."
  text: ""
`,
			expectErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			configPath := path.Join(t.TempDir(), fileMigrationConfig)
			err := os.WriteFile(configPath, []byte(tc.inConfig), 0600)
			require.NoError(t, err)

			_, err = readConfigYAML[Migration](configPath)

			assert.Equal(t, tc.expectErr, err != nil)
		})
	}
}

func Test_validateSSHConfig(t *testing.T) {
	tests := []struct {
		name      string
		inConfig  string
		expectErr bool
	}{
		{
			name: "valid ssh config",
			inConfig: `---
user: "root"
privateKeyPath: "/.ssh/privateKey"
secretName: "ces-importer-secret"
secretDataKey: "privateKey"
`,
			expectErr: false,
		},
		{
			name: "error - empty user",
			inConfig: `---
user: ""
privateKeyPath: "/.ssh/privateKey"
secretName: "ces-importer-secret"
secretDataKey: "privateKey"
`,
			expectErr: true,
		},
		{
			name: "error - empty privateKeyPath",
			inConfig: `---
user: "root"
privateKeyPath: ""
secretName: "ces-importer-secret"
secretDataKey: "privateKey"
`,
			expectErr: true,
		},
		{
			name: "error - empty secretName",
			inConfig: `---
user: "root"
privateKeyPath: "/.ssh/privateKey"
secretName: ""
secretDataKey: "privateKey"
`,
			expectErr: true,
		},
		{
			name: "error - invalid secretName",
			inConfig: `---
user: "root"
privateKeyPath: "/.ssh/privateKey"
secretName: "ces/exporter/secret!!!"
secretDataKey: "privateKey"
`,
			expectErr: true,
		},
		{
			name: "error - empty secretDataKey",
			inConfig: `---
user: "root"
privateKeyPath: "/.ssh/privateKey"
secretName: "ces-importer-secret"
secretDataKey: ""
`,
			expectErr: true,
		},
		{
			name: "error - invalid secretDataKey",
			inConfig: `---
user: "root"
privateKeyPath: "/.ssh/privateKey"
secretName: "ces-importer-secret"
secretDataKey: "ces/apiKey/secret!!!"
`,
			expectErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			configPath := path.Join(t.TempDir(), fileSSHConfig)
			err := os.WriteFile(configPath, []byte(tc.inConfig), 0600)
			require.NoError(t, err)

			_, err = readConfigYAML[SSH](configPath)

			assert.Equal(t, tc.expectErr, err != nil)
		})
	}
}

func Test_validateJobContainerConfig(t *testing.T) {
	tests := []struct {
		name      string
		inConfig  string
		expectErr bool
	}{
		{
			name: "valid job container config",
			inConfig: `---
image:
  registry: "docker.io"
  repository: "cloudogu/ces-importer"
  tag: "0.0.1"
imagePullPolicy: "IfNotPresent"
imagePullSecrets:
  - name: "ces-container-registries"
resources:
  limits:
    cpu: "500m"
    memory: "256Mi"
  requests:
    cpu: "100m"
    memory: "128Mi"
jobConfigMap: "ces-importer-job-config"
jobServiceAccount: "ces-importer-main-manager"
`,
			expectErr: false,
		},
		{
			name: "valid job container config - imagePullPolicy Never",
			inConfig: `---
image:
  registry: "docker.io"
  repository: "cloudogu/ces-importer"
  tag: "0.0.1"
imagePullPolicy: "Never"
imagePullSecrets:
  - name: "ces-container-registries"
resources:
  limits:
    cpu: "500m"
    memory: "256Mi"
  requests:
    cpu: "100m"
    memory: "128Mi"
jobConfigMap: "ces-importer-job-config"
jobServiceAccount: "ces-importer-main-manager"
`,
			expectErr: false,
		},
		{
			name: "valid job container config - imagePullPolicy Always",
			inConfig: `---
image:
  registry: "docker.io"
  repository: "cloudogu/ces-importer"
  tag: "0.0.1"
imagePullPolicy: "Always"
imagePullSecrets:
  - name: "ces-container-registries"
resources:
  limits:
    cpu: "500m"
    memory: "256Mi"
  requests:
    cpu: "100m"
    memory: "128Mi"
jobConfigMap: "ces-importer-job-config"
jobServiceAccount: "ces-importer-main-manager"
`,
			expectErr: false,
		},
		{
			name: "error - empty imagePullPolicy",
			inConfig: `---
image:
  registry: "docker.io"
  repository: "cloudogu/ces-importer"
  tag: "0.0.1"
imagePullPolicy: ""
imagePullSecrets:
  - name: "ces-container-registries"
resources:
  limits:
    cpu: "500m"
    memory: "256Mi"
  requests:
    cpu: "100m"
    memory: "128Mi"
jobConfigMap: "ces-importer-job-config"
jobServiceAccount: "ces-importer-main-manager"
`,
			expectErr: true,
		},
		{
			name: "error - invalid imagePullPolicy",
			inConfig: `---
image:
  registry: "docker.io"
  repository: "cloudogu/ces-importer"
  tag: "0.0.1"
imagePullPolicy: "JUST PULL"
imagePullSecrets:
  - name: "ces-container-registries"
resources:
  limits:
    cpu: "500m"
    memory: "256Mi"
  requests:
    cpu: "100m"
    memory: "128Mi"
jobConfigMap: "ces-importer-job-config"
jobServiceAccount: "ces-importer-main-manager"
`,
			expectErr: true,
		},
		{
			name: "error - empty jobConfigMap",
			inConfig: `---
image:
  registry: "docker.io"
  repository: "cloudogu/ces-importer"
  tag: "0.0.1"
imagePullPolicy: "IfNotPresent"
imagePullSecrets:
  - name: "ces-container-registries"
resources:
  limits:
    cpu: "500m"
    memory: "256Mi"
  requests:
    cpu: "100m"
    memory: "128Mi"
jobConfigMap: ""
jobServiceAccount: "ces-importer-main-manager"
`,
			expectErr: true,
		},
		{
			name: "error - invalid jobConfigMap",
			inConfig: `---
image:
  registry: "docker.io"
  repository: "cloudogu/ces-importer"
  tag: "0.0.1"
imagePullPolicy: "IfNotPresent"
imagePullSecrets:
  - name: "ces-container-registries"
resources:
  limits:
    cpu: "500m"
    memory: "256Mi"
  requests:
    cpu: "100m"
    memory: "128Mi"
jobConfigMap: "!!!!ces-importer-job-config!!!!"
jobServiceAccount: "ces-importer-main-manager"
`,
			expectErr: true,
		},
		{
			name: "error - empty jobServiceAccount",
			inConfig: `---
image:
  registry: "docker.io"
  repository: "cloudogu/ces-importer"
  tag: "0.0.1"
imagePullPolicy: "IfNotPresent"
imagePullSecrets:
  - name: "ces-container-registries"
resources:
  limits:
    cpu: "500m"
    memory: "256Mi"
  requests:
    cpu: "100m"
    memory: "128Mi"
jobConfigMap: "ces-importer-job-config"
jobServiceAccount: ""
`,
			expectErr: true,
		},
		{
			name: "error - invalid jobServiceAccount",
			inConfig: `---
image:
  registry: "docker.io"
  repository: "cloudogu/ces-importer"
  tag: "0.0.1"
imagePullPolicy: "IfNotPresent"
imagePullSecrets:
  - name: "ces-container-registries"
resources:
  limits:
    cpu: "500m"
    memory: "256Mi"
  requests:
    cpu: "100m"
    memory: "128Mi"
jobConfigMap: "ces-importer-job-config"
jobServiceAccount: "!!!!!ces-importer-main-manager!!!!!"
`,
			expectErr: true,
		},
		{
			name: "error - empty image",
			inConfig: `---
imagePullPolicy: "IfNotPresent"
imagePullSecrets:
  - name: "ces-container-registries"
resources:
  limits:
    cpu: "500m"
    memory: "256Mi"
  requests:
    cpu: "100m"
    memory: "128Mi"
jobConfigMap: "ces-importer-job-config"
jobServiceAccount: "ces-importer-main-manager"
`,
			expectErr: true,
		},
		{
			name: "error - empty registry in image",
			inConfig: `---
image:
  registry: ""
  repository: "cloudogu/ces-importer"
  tag: "0.0.1"
imagePullPolicy: "IfNotPresent"
imagePullSecrets:
  - name: "ces-container-registries"
resources:
  limits:
    cpu: "500m"
    memory: "256Mi"
  requests:
    cpu: "100m"
    memory: "128Mi"
jobConfigMap: "ces-importer-job-config"
jobServiceAccount: "ces-importer-main-manager"
`,
			expectErr: true,
		},
		{
			name: "error - invalid registry in image",
			inConfig: `---
image:
  registry: "!!!!docker.io!!!!"
  repository: "cloudogu/ces-importer"
  tag: "0.0.1"
imagePullPolicy: "IfNotPresent"
imagePullSecrets:
  - name: "ces-container-registries"
resources:
  limits:
    cpu: "500m"
    memory: "256Mi"
  requests:
    cpu: "100m"
    memory: "128Mi"
jobConfigMap: "ces-importer-job-config"
jobServiceAccount: "ces-importer-main-manager"
`,
			expectErr: true,
		},
		{
			name: "error -empty repository in image",
			inConfig: `---
image:
  registry: "docker.io"
  repository: ""
  tag: "0.0.1"
imagePullPolicy: "IfNotPresent"
imagePullSecrets:
  - name: "ces-container-registries"
resources:
  limits:
    cpu: "500m"
    memory: "256Mi"
  requests:
    cpu: "100m"
    memory: "128Mi"
jobConfigMap: "ces-importer-job-config"
jobServiceAccount: "ces-importer-main-manager"
`,
			expectErr: true,
		},
		{
			name: "error - empty tag in image",
			inConfig: `---
image:
  registry: "docker.io"
  repository: "cloudogu/ces-importer"
  tag: ""
imagePullPolicy: "IfNotPresent"
imagePullSecrets:
  - name: "ces-container-registries"
resources:
  limits:
    cpu: "500m"
    memory: "256Mi"
  requests:
    cpu: "100m"
    memory: "128Mi"
jobConfigMap: "ces-importer-job-config"
jobServiceAccount: "ces-importer-main-manager"
`,
			expectErr: true,
		},
		{
			name: "error empty resources",
			inConfig: `---
image:
  registry: "docker.io"
  repository: "cloudogu/ces-importer"
  tag: "0.0.1"
imagePullPolicy: "IfNotPresent"
imagePullSecrets:
  - name: "ces-container-registries"
jobConfigMap: "ces-importer-job-config"
jobServiceAccount: "ces-importer-main-manager"
`,
			expectErr: true,
		},
		{
			name: "error - empty limits in resources",
			inConfig: `---
image:
  registry: "docker.io"
  repository: "cloudogu/ces-importer"
  tag: "0.0.1"
imagePullPolicy: "IfNotPresent"
imagePullSecrets:
  - name: "ces-container-registries"
resources:
  requests:
    cpu: "100m"
    memory: "128Mi"
jobConfigMap: "ces-importer-job-config"
jobServiceAccount: "ces-importer-main-manager"
`,
			expectErr: true,
		},
		{
			name: "error - empty requests in resources",
			inConfig: `---
image:
  registry: "docker.io"
  repository: "cloudogu/ces-importer"
  tag: "0.0.1"
imagePullPolicy: "IfNotPresent"
imagePullSecrets:
  - name: "ces-container-registries"
resources:
  limits:
    cpu: "500m"
    memory: "256Mi"
jobConfigMap: "ces-importer-job-config"
jobServiceAccount: "ces-importer-main-manager"
`,
			expectErr: true,
		},
		{
			name: "error - empty cpu limit in resources",
			inConfig: `---
image:
  registry: "docker.io"
  repository: "cloudogu/ces-importer"
  tag: "0.0.1"
imagePullPolicy: "IfNotPresent"
imagePullSecrets:
  - name: "ces-container-registries"
resources:
  limits:
    cpu: ""
    memory: "256Mi"
  requests:
    cpu: "100m"
    memory: "128Mi"
jobConfigMap: "ces-importer-job-config"
jobServiceAccount: "ces-importer-main-manager"
`,
			expectErr: true,
		},
		{
			name: "error - empty memory limit in resources",
			inConfig: `---
image:
  registry: "docker.io"
  repository: "cloudogu/ces-importer"
  tag: "0.0.1"
imagePullPolicy: "IfNotPresent"
imagePullSecrets:
  - name: "ces-container-registries"
resources:
  limits:
    cpu: "500m"
    memory: ""
  requests:
    cpu: "100m"
    memory: "128Mi"
jobConfigMap: "ces-importer-job-config"
jobServiceAccount: "ces-importer-main-manager"
`,
			expectErr: true,
		},
		{
			name: "error - empty cpu request in resources",
			inConfig: `---
image:
  registry: "docker.io"
  repository: "cloudogu/ces-importer"
  tag: "0.0.1"
imagePullPolicy: "IfNotPresent"
imagePullSecrets:
  - name: "ces-container-registries"
resources:
  limits:
    cpu: "500m"
    memory: "256Mi"
  requests:
    cpu: ""
    memory: "128Mi"
jobConfigMap: "ces-importer-job-config"
jobServiceAccount: "ces-importer-main-manager"
`,
			expectErr: true,
		},
		{
			name: "error - empty memory request in resources",
			inConfig: `---
image:
  registry: "docker.io"
  repository: "cloudogu/ces-importer"
  tag: "0.0.1"
imagePullPolicy: "IfNotPresent"
imagePullSecrets:
  - name: "ces-container-registries"
resources:
  limits:
    cpu: "500m"
    memory: "256Mi"
  requests:
    cpu: "100m"
    memory: ""
jobConfigMap: "ces-importer-job-config"
jobServiceAccount: "ces-importer-main-manager"
`,
			expectErr: true,
		},
		{
			name: "error - invalid cpu limit in resources",
			inConfig: `---
image:
  registry: "docker.io"
  repository: "cloudogu/ces-importer"
  tag: "0.0.1"
imagePullPolicy: "IfNotPresent"
imagePullSecrets:
  - name: "ces-container-registries"
resources:
  limits:
    cpu: "!!!!500m!!!!"
    memory: "256Mi"
  requests:
    cpu: "100m"
    memory: "128Mi"
jobConfigMap: "ces-importer-job-config"
jobServiceAccount: "ces-importer-main-manager"
`,
			expectErr: true,
		},
		{
			name: "error - invalid memory limit in resources",
			inConfig: `---
image:
  registry: "docker.io"
  repository: "cloudogu/ces-importer"
  tag: "0.0.1"
imagePullPolicy: "IfNotPresent"
imagePullSecrets:
  - name: "ces-container-registries"
resources:
  limits:
    cpu: "500m"
    memory: "!!!!256Mi!!!!"
  requests:
    cpu: "100m"
    memory: "128Mi"
jobConfigMap: "ces-importer-job-config"
jobServiceAccount: "ces-importer-main-manager"
`,
			expectErr: true,
		},
		{
			name: "error - invalid cpu request in resources",
			inConfig: `---
image:
  registry: "docker.io"
  repository: "cloudogu/ces-importer"
  tag: "0.0.1"
imagePullPolicy: "IfNotPresent"
imagePullSecrets:
  - name: "ces-container-registries"
resources:
  limits:
    cpu: "500m"
    memory: "256Mi"
  requests:
    cpu: "!!!!100m!!!!"
    memory: "128Mi"
jobConfigMap: "ces-importer-job-config"
jobServiceAccount: "ces-importer-main-manager"
`,
			expectErr: true,
		},
		{
			name: "error - invalid memory request in resources",
			inConfig: `---
image:
  registry: "docker.io"
  repository: "cloudogu/ces-importer"
  tag: "0.0.1"
imagePullPolicy: "IfNotPresent"
imagePullSecrets:
  - name: "ces-container-registries"
resources:
  limits:
    cpu: "500m"
    memory: "256Mi"
  requests:
    cpu: "100m"
    memory: "!!!!128Mi!!!!"
jobConfigMap: "ces-importer-job-config"
jobServiceAccount: "ces-importer-main-manager"
`,
			expectErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			configPath := path.Join(t.TempDir(), fileJobContainerConfig)
			err := os.WriteFile(configPath, []byte(tc.inConfig), 0600)
			require.NoError(t, err)

			_, err = readConfigYAML[JobContainer](configPath)

			assert.Equal(t, tc.expectErr, err != nil)
		})
	}
}

func Test_validateJobConfig(t *testing.T) {
	tests := []struct {
		name      string
		inConfig  string
		expectErr bool
	}{
		{
			name: "valid job config",
			inConfig: `---
doguVolumeBasePath: "/data"
exclude:
  - dogu: "jenkins"
    pattern: "JENKINS_PATTERN"
  - dogu: "redmine"
    pattern: "REDMINE_PATTERN"
verbose: true
`,
			expectErr: false,
		},
		{
			name: "valid job config - verbose not set",
			inConfig: `---
doguVolumeBasePath: "/data"
exclude:
  - dogu: "jenkins"
    pattern: "JENKINS_PATTERN"
  - dogu: "redmine"
    pattern: "REDMINE_PATTERN"
`,
			expectErr: false,
		},
		{
			name: "valid job config - exclude empty",
			inConfig: `---
doguVolumeBasePath: "/data"
exclude: []
verbose: true
`,
			expectErr: false,
		},
		{
			name: "error - empty doguVolumeBasePath",
			inConfig: `---
doguVolumeBasePath: ""
exclude:
  - dogu: "jenkins"
    pattern: "JENKINS_PATTERN"
  - dogu: "redmine"
    pattern: "REDMINE_PATTERN"
verbose: true
`,
			expectErr: true,
		},
		{
			name: "error - empty dogu in exclude",
			inConfig: `---
doguVolumeBasePath: "/data"
exclude:
  - dogu: ""
    pattern: "JENKINS_PATTERN"
verbose: true
`,
			expectErr: true,
		},
		{
			name: "error - empty pattern in exclude",
			inConfig: `---
doguVolumeBasePath: "/data"
exclude:
  - dogu: "jenkins"
    pattern: "JENKINS_PATTERN"
  - dogu: "redmine"
    pattern: ""
verbose: true
`,
			expectErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			configPath := path.Join(t.TempDir(), fileJobConfig)
			err := os.WriteFile(configPath, []byte(tc.inConfig), 0600)
			require.NoError(t, err)

			_, err = readConfigYAML[JobConfig](configPath)

			assert.Equal(t, tc.expectErr, err != nil)
		})
	}
}

func Test_validateSMTPConfig(t *testing.T) {
	tests := []struct {
		name      string
		inConfig  string
		expectErr bool
	}{
		{
			name: "valid smtp config",
			inConfig: `---
server: 192.168.56.1
port: 1025
username:
password:
from: importer@ces.com
to:
  - recipient1@example.com
  - recipient2@example.com
`,
			expectErr: false,
		},
		{
			name: "valid smtp config - empty server",
			inConfig: `---
server: ""
port: 
username:
password:
from:
to:
`,
			expectErr: false,
		},
		{
			name: "error - invalid server",
			inConfig: `---
server: ".invalid$Host_*"
port: 1025
username:
password:
from: importer@ces.com
to:
  - recipient1@example.com
  - recipient2@example.com
`,
			expectErr: true,
		},
		{
			name: "error - empty port when server is set",
			inConfig: `---
server: 192.168.56.1
port: 
username:
password:
from: importer@ces.com
to:
  - recipient1@example.com
  - recipient2@example.com
`,
			expectErr: true,
		},
		{
			name: "error - empty from when server is set",
			inConfig: `---
server: 192.168.56.1
port: 1025
username:
password:
from: 
to:
  - recipient1@example.com
  - recipient2@example.com
`,
			expectErr: true,
		},
		{
			name: "error - invalid from when server is set",
			inConfig: `---
server: 192.168.56.1
port: 1025
username:
password:
from: invalid
to:
  - recipient1@example.com
  - recipient2@example.com
`,
			expectErr: true,
		},
		{
			name: "error - empty (nil) to when server is set",
			inConfig: `---
server: 192.168.56.1
port: 1025
username:
password:
from: importer@ces.com
to:
`,
			expectErr: true,
		},
		{
			name: "error - invalid first mail in to when server is set",
			inConfig: `---
server: 192.168.56.1
port: 1025
username:
password:
from: importer@ces.com
to:
  - invalidMailAddress
  - recipient2@example.com 
`,
			expectErr: true,
		},
		{
			name: "error - invalid second mail in to when server is set",
			inConfig: `---
server: 192.168.56.1
port: 1025
username:
password:
from: importer@ces.com
to:
  - recipient2@example.com 
  - invalidMailAddress
`,
			expectErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			configPath := path.Join(t.TempDir(), fileSMTPConfig)
			err := os.WriteFile(configPath, []byte(tc.inConfig), 0600)
			require.NoError(t, err)

			_, err = readConfigYAML[Smtp](configPath)

			assert.Equal(t, tc.expectErr, err != nil)
		})
	}
}
