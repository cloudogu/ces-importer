package configuration

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"path"
	"testing"
)

func TestReadCoordinatorConfig(t *testing.T) {
	t.Run("read config for coordinator", func(t *testing.T) {
		t.Setenv(EnvBaseConfigPathKey, "../testdata/config")
		t.Setenv(EnvImporterNamespaceKey, "test")
		t.Setenv("API_KEY", "testAPIKEY")

		cfg, err := ReadCoordinatorConfig()
		assert.NoError(t, err)

		// namespace
		assert.Equal(t, "test", cfg.Namespace)

		// logging
		assert.Equal(t, Logging{
			Level: "DEBUG",
		}, cfg.Logging)

		// api
		assert.Equal(t, API{
			ExporterHost:   "classic-ces.exporter",
			ExporterApiKey: "testAPIKEY",
			SkipTLSVerify:  true,
		}, cfg.API)

		// migration
		assert.Equal(t, Migration{
			RegularCron:    "0 4 * * *",
			FinalTimestamp: "2025-04-03 12:34:56Z",
			ChangeFQDN:     true,
		}, cfg.Migration)

		// ssh
		assert.Equal(t, SSH{
			User:              "root",
			PrivateSSHKeyPath: "/.ssh/privateKey",
			SecretName:        "ces-importer-secret",
			SecretDataKey:     "privateKey",
		}, cfg.SSH)

		// job
		assert.Equal(t, JobConfig{
			DoguVolumeBasePath: "/data",
			Exclude: []ExcludePattern{
				{
					DoguName: "jenkins",
					Pattern:  "JENKINS_PATTERN",
				},
				{
					DoguName: "redmine",
					Pattern:  "REDMINE_PATTERN",
				},
			},
		}, cfg.JobConfig)

		// job-container
		assert.Equal(t, JobContainer{
			Image: ContainerImage{
				Registry:   "docker.io",
				Repository: "cloudogu/ces-importer",
				Tag:        "0.0.1",
			},
			ImagePullPolicy: "IfNotPresent",
			ImagePullSecrets: []ImagePullSecret{
				{Name: "ces-container-registries"},
			},
			Resources: ResourceRequirements{
				Limits: ResourceList{
					CPU:    "500m",
					Memory: "256Mi",
				},
				Requests: ResourceList{
					CPU:    "100m",
					Memory: "128Mi",
				},
			},
			JobConfigMap:      "ces-importer-job-config",
			JobServiceAccount: "ces-importer-main-manager",
		}, cfg.JobContainer)
	})

	t.Run("error while reading config", func(t *testing.T) {
		tests := map[string]struct {
			setupEnv       func(t *testing.T)
			setupFiles     func(t *testing.T, tmpDir string)
			expectedErrMsg string
		}{
			"should fail when CONFIG_PATH env var is not set": {
				setupEnv: func(t *testing.T) {
					t.Setenv(EnvImporterNamespaceKey, "test-namespace")
				},
				setupFiles:     func(t *testing.T, tmpDir string) {},
				expectedErrMsg: "environment variable CONFIG_PATH is not set",
			},
			"should fail when IMPORTER_NAMESPACE env var is not set": {
				setupEnv: func(t *testing.T) {
					t.Setenv(EnvBaseConfigPathKey, "/some/path")
				},
				setupFiles:     func(t *testing.T, tmpDir string) {},
				expectedErrMsg: "environment variable IMPORTER_NAMESPACE is not set",
			},
			"should fail when logging config is missing": {
				setupEnv: func(t *testing.T) {
					t.Setenv(EnvBaseConfigPathKey, "/tmp")
					t.Setenv(EnvImporterNamespaceKey, "test-namespace")
				},
				setupFiles: func(t *testing.T, tmpDir string) {
					// Don't create logging config
					createValidConfig(t, tmpDir, fileAPIConfig)
					createValidConfig(t, tmpDir, fileMigrationConfig)
					createValidConfig(t, tmpDir, fileSSHConfig)
					createValidConfig(t, tmpDir, fileJobConfig)
					createValidConfig(t, tmpDir, fileJobContainerConfig)
					createValidConfig(t, tmpDir, fileSMTPConfig)
				},
				expectedErrMsg: "failed to read logging configuration",
			},
			"should fail when api config is missing": {
				setupEnv: func(t *testing.T) {
					t.Setenv(EnvBaseConfigPathKey, "/tmp")
					t.Setenv(EnvImporterNamespaceKey, "test-namespace")
				},
				setupFiles: func(t *testing.T, tmpDir string) {
					createValidConfig(t, tmpDir, fileLoggingConfig)
					// Don't create API config
					createValidConfig(t, tmpDir, fileMigrationConfig)
					createValidConfig(t, tmpDir, fileSSHConfig)
					createValidConfig(t, tmpDir, fileJobConfig)
					createValidConfig(t, tmpDir, fileJobContainerConfig)
					createValidConfig(t, tmpDir, fileSMTPConfig)
				},
				expectedErrMsg: "failed to read API configuration",
			},
			"should fail when migration config is missing": {
				setupEnv: func(t *testing.T) {
					t.Setenv(EnvBaseConfigPathKey, "/tmp")
					t.Setenv(EnvImporterNamespaceKey, "test-namespace")
				},
				setupFiles: func(t *testing.T, tmpDir string) {
					createValidConfig(t, tmpDir, fileLoggingConfig)
					createValidConfig(t, tmpDir, fileAPIConfig)
					// Don't create migration config
					createValidConfig(t, tmpDir, fileSSHConfig)
					createValidConfig(t, tmpDir, fileJobConfig)
					createValidConfig(t, tmpDir, fileJobContainerConfig)
					createValidConfig(t, tmpDir, fileSMTPConfig)
				},
				expectedErrMsg: "failed to read migration configuration",
			},
			"should fail when ssh config is missing": {
				setupEnv: func(t *testing.T) {
					t.Setenv(EnvBaseConfigPathKey, "/tmp")
					t.Setenv(EnvImporterNamespaceKey, "test-namespace")
				},
				setupFiles: func(t *testing.T, tmpDir string) {
					createValidConfig(t, tmpDir, fileLoggingConfig)
					createValidConfig(t, tmpDir, fileAPIConfig)
					createValidConfig(t, tmpDir, fileMigrationConfig)
					// Don't create SSH config
					createValidConfig(t, tmpDir, fileJobConfig)
					createValidConfig(t, tmpDir, fileJobContainerConfig)
					createValidConfig(t, tmpDir, fileSMTPConfig)
				},
				expectedErrMsg: "failed to read ssh configuration",
			},
			"should fail when job config is missing": {
				setupEnv: func(t *testing.T) {
					t.Setenv(EnvBaseConfigPathKey, "/tmp")
					t.Setenv(EnvImporterNamespaceKey, "test-namespace")
				},
				setupFiles: func(t *testing.T, tmpDir string) {
					createValidConfig(t, tmpDir, fileLoggingConfig)
					createValidConfig(t, tmpDir, fileAPIConfig)
					createValidConfig(t, tmpDir, fileMigrationConfig)
					createValidConfig(t, tmpDir, fileSSHConfig)
					// Don't create Job config
					createValidConfig(t, tmpDir, fileJobContainerConfig)
					createValidConfig(t, tmpDir, fileSMTPConfig)
				},
				expectedErrMsg: "failed to read job configuration",
			},
			"should fail when job container config is missing": {
				setupEnv: func(t *testing.T) {
					t.Setenv(EnvBaseConfigPathKey, "/tmp")
					t.Setenv(EnvImporterNamespaceKey, "test-namespace")
				},
				setupFiles: func(t *testing.T, tmpDir string) {
					createValidConfig(t, tmpDir, fileLoggingConfig)
					createValidConfig(t, tmpDir, fileAPIConfig)
					createValidConfig(t, tmpDir, fileMigrationConfig)
					createValidConfig(t, tmpDir, fileSSHConfig)
					createValidConfig(t, tmpDir, fileJobConfig)
					createValidConfig(t, tmpDir, fileSMTPConfig)
					// Don't create job container config
				},
				expectedErrMsg: "failed to read job container configuration",
			},
			"should fail when logging config has invalid yaml": {
				setupEnv: func(t *testing.T) {
					t.Setenv(EnvBaseConfigPathKey, "/tmp")
					t.Setenv(EnvImporterNamespaceKey, "test-namespace")
				},
				setupFiles: func(t *testing.T, tmpDir string) {
					writeInvalidYaml(t, tmpDir, fileLoggingConfig)
					createValidConfig(t, tmpDir, fileAPIConfig)
					createValidConfig(t, tmpDir, fileMigrationConfig)
					createValidConfig(t, tmpDir, fileSSHConfig)
					createValidConfig(t, tmpDir, fileJobConfig)
					createValidConfig(t, tmpDir, fileJobContainerConfig)
					createValidConfig(t, tmpDir, fileSMTPConfig)
				},
				expectedErrMsg: "failed to read logging configuration: failed to unmarshal config",
			},
			"should fail when api config has invalid yaml": {
				setupEnv: func(t *testing.T) {
					t.Setenv(EnvBaseConfigPathKey, "/tmp")
					t.Setenv(EnvImporterNamespaceKey, "test-namespace")
				},
				setupFiles: func(t *testing.T, tmpDir string) {
					createValidConfig(t, tmpDir, fileLoggingConfig)
					writeInvalidYaml(t, tmpDir, fileAPIConfig)
					createValidConfig(t, tmpDir, fileMigrationConfig)
					createValidConfig(t, tmpDir, fileSSHConfig)
					createValidConfig(t, tmpDir, fileJobConfig)
					createValidConfig(t, tmpDir, fileJobContainerConfig)
					createValidConfig(t, tmpDir, fileSMTPConfig)
				},
				expectedErrMsg: "failed to read API configuration: failed to unmarshal config",
			},
			"should fail when migration config has invalid yaml": {
				setupEnv: func(t *testing.T) {
					t.Setenv(EnvBaseConfigPathKey, "/tmp")
					t.Setenv(EnvImporterNamespaceKey, "test-namespace")
				},
				setupFiles: func(t *testing.T, tmpDir string) {
					createValidConfig(t, tmpDir, fileLoggingConfig)
					createValidConfig(t, tmpDir, fileAPIConfig)
					writeInvalidYaml(t, tmpDir, fileMigrationConfig)
					createValidConfig(t, tmpDir, fileSSHConfig)
					createValidConfig(t, tmpDir, fileJobConfig)
					createValidConfig(t, tmpDir, fileJobContainerConfig)
					createValidConfig(t, tmpDir, fileSMTPConfig)
				},
				expectedErrMsg: "failed to read migration configuration: failed to unmarshal config",
			},
			"should fail when ssh config has invalid yaml": {
				setupEnv: func(t *testing.T) {
					t.Setenv(EnvBaseConfigPathKey, "/tmp")
					t.Setenv(EnvImporterNamespaceKey, "test-namespace")
				},
				setupFiles: func(t *testing.T, tmpDir string) {
					createValidConfig(t, tmpDir, fileLoggingConfig)
					createValidConfig(t, tmpDir, fileAPIConfig)
					createValidConfig(t, tmpDir, fileMigrationConfig)
					writeInvalidYaml(t, tmpDir, fileSSHConfig)
					createValidConfig(t, tmpDir, fileJobConfig)
					createValidConfig(t, tmpDir, fileJobContainerConfig)
					createValidConfig(t, tmpDir, fileSMTPConfig)
				},
				expectedErrMsg: "failed to read ssh configuration: failed to unmarshal config",
			},
			"should fail when job config has invalid yaml": {
				setupEnv: func(t *testing.T) {
					t.Setenv(EnvBaseConfigPathKey, "/tmp")
					t.Setenv(EnvImporterNamespaceKey, "test-namespace")
				},
				setupFiles: func(t *testing.T, tmpDir string) {
					createValidConfig(t, tmpDir, fileLoggingConfig)
					createValidConfig(t, tmpDir, fileAPIConfig)
					createValidConfig(t, tmpDir, fileMigrationConfig)
					createValidConfig(t, tmpDir, fileSSHConfig)
					writeInvalidYaml(t, tmpDir, fileJobConfig)
					createValidConfig(t, tmpDir, fileJobContainerConfig)
					createValidConfig(t, tmpDir, fileSMTPConfig)
				},
				expectedErrMsg: "failed to read job configuration: failed to unmarshal config",
			},
			"should fail when job container config has invalid yaml": {
				setupEnv: func(t *testing.T) {
					t.Setenv(EnvBaseConfigPathKey, "/tmp")
					t.Setenv(EnvImporterNamespaceKey, "test-namespace")
				},
				setupFiles: func(t *testing.T, tmpDir string) {
					createValidConfig(t, tmpDir, fileLoggingConfig)
					createValidConfig(t, tmpDir, fileAPIConfig)
					createValidConfig(t, tmpDir, fileMigrationConfig)
					createValidConfig(t, tmpDir, fileSSHConfig)
					createValidConfig(t, tmpDir, fileJobConfig)
					writeInvalidYaml(t, tmpDir, fileJobContainerConfig)
					createValidConfig(t, tmpDir, fileSMTPConfig)
				},
				expectedErrMsg: "failed to read job container configuration: failed to unmarshal config",
			},
		}

		for name, tc := range tests {
			t.Run(name, func(t *testing.T) {
				// Setup
				tmpDir := t.TempDir()
				tc.setupEnv(t)

				// If CONFIG_PATH is set to /tmp, update it to the actual temp directory
				if os.Getenv(EnvBaseConfigPathKey) == "/tmp" {
					t.Setenv(EnvBaseConfigPathKey, tmpDir)
				}

				tc.setupFiles(t, tmpDir)

				// Execute
				config, err := ReadCoordinatorConfig()

				// Assert
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedErrMsg)
				assert.Empty(t, config)
			})
		}
	})
}

func TestReadJobConfig(t *testing.T) {
	t.Run("read config for job", func(t *testing.T) {
		t.Setenv(EnvBaseConfigPathKey, "../testdata/config")
		t.Setenv(EnvImporterNamespaceKey, "test")
		t.Setenv("API_KEY", "testAPIKEY")

		cfg, err := ReadJobConfig()
		assert.NoError(t, err)

		// namespace
		assert.Equal(t, "test", cfg.Namespace)

		// logging
		assert.Equal(t, Logging{
			Level: "DEBUG",
		}, cfg.Logging)

		// api
		assert.Equal(t, API{
			ExporterHost:   "classic-ces.exporter",
			ExporterApiKey: "testAPIKEY",
			SkipTLSVerify:  true,
		}, cfg.API)

		// ssh
		assert.Equal(t, SSH{
			User:              "root",
			PrivateSSHKeyPath: "/.ssh/privateKey",
			SecretName:        "ces-importer-secret",
			SecretDataKey:     "privateKey",
		}, cfg.SSH)

		// job
		assert.Equal(t, JobConfig{
			DoguVolumeBasePath: "/data",
			Exclude: []ExcludePattern{
				{
					DoguName: "jenkins",
					Pattern:  "JENKINS_PATTERN",
				},
				{
					DoguName: "redmine",
					Pattern:  "REDMINE_PATTERN",
				},
			},
		}, cfg.JobConfig)
	})

	t.Run("error while reading config", func(t *testing.T) {
		tests := map[string]struct {
			setupEnv       func(t *testing.T)
			setupFiles     func(t *testing.T, tmpDir string)
			expectedErrMsg string
		}{
			"should fail when CONFIG_PATH env var is not set": {
				setupEnv: func(t *testing.T) {
					t.Setenv(EnvImporterNamespaceKey, "test-namespace")
				},
				setupFiles:     func(t *testing.T, tmpDir string) {},
				expectedErrMsg: "environment variable CONFIG_PATH is not set",
			},
			"should fail when IMPORTER_NAMESPACE env var is not set": {
				setupEnv: func(t *testing.T) {
					t.Setenv(EnvBaseConfigPathKey, "/some/path")
				},
				setupFiles:     func(t *testing.T, tmpDir string) {},
				expectedErrMsg: "environment variable IMPORTER_NAMESPACE is not set",
			},
			"should fail when logging config is missing": {
				setupEnv: func(t *testing.T) {
					t.Setenv(EnvBaseConfigPathKey, "/tmp")
					t.Setenv(EnvImporterNamespaceKey, "test-namespace")
				},
				setupFiles: func(t *testing.T, tmpDir string) {
					// Don't create logging config
					createValidConfig(t, tmpDir, fileAPIConfig)
					createValidConfig(t, tmpDir, fileSSHConfig)
					createValidConfig(t, tmpDir, fileJobConfig)
				},
				expectedErrMsg: "failed to read logging configuration",
			},
			"should fail when api config is missing": {
				setupEnv: func(t *testing.T) {
					t.Setenv(EnvBaseConfigPathKey, "/tmp")
					t.Setenv(EnvImporterNamespaceKey, "test-namespace")
				},
				setupFiles: func(t *testing.T, tmpDir string) {
					createValidConfig(t, tmpDir, fileLoggingConfig)
					// Don't create API config
					createValidConfig(t, tmpDir, fileSSHConfig)
					createValidConfig(t, tmpDir, fileJobConfig)
				},
				expectedErrMsg: "failed to read API configuration",
			},
			"should fail when job config is missing": {
				setupEnv: func(t *testing.T) {
					t.Setenv(EnvBaseConfigPathKey, "/tmp")
					t.Setenv(EnvImporterNamespaceKey, "test-namespace")
				},
				setupFiles: func(t *testing.T, tmpDir string) {
					createValidConfig(t, tmpDir, fileLoggingConfig)
					createValidConfig(t, tmpDir, fileAPIConfig)
					createValidConfig(t, tmpDir, fileSSHConfig)
					// Don't create job config
				},
				expectedErrMsg: "failed to read job configuration",
			},
			"should fail when ssh config is missing": {
				setupEnv: func(t *testing.T) {
					t.Setenv(EnvBaseConfigPathKey, "/tmp")
					t.Setenv(EnvImporterNamespaceKey, "test-namespace")
				},
				setupFiles: func(t *testing.T, tmpDir string) {
					createValidConfig(t, tmpDir, fileLoggingConfig)
					createValidConfig(t, tmpDir, fileAPIConfig)
					// Don't create SSH config
					createValidConfig(t, tmpDir, fileJobConfig)
				},
				expectedErrMsg: "failed to read ssh configuration",
			},
			"should fail when logging config has invalid yaml": {
				setupEnv: func(t *testing.T) {
					t.Setenv(EnvBaseConfigPathKey, "/tmp")
					t.Setenv(EnvImporterNamespaceKey, "test-namespace")
				},
				setupFiles: func(t *testing.T, tmpDir string) {
					writeInvalidYaml(t, tmpDir, fileLoggingConfig)
					createValidConfig(t, tmpDir, fileAPIConfig)
					createValidConfig(t, tmpDir, fileSSHConfig)
					createValidConfig(t, tmpDir, fileJobConfig)
				},
				expectedErrMsg: "failed to read logging configuration: failed to unmarshal config",
			},
			"should fail when api config has invalid yaml": {
				setupEnv: func(t *testing.T) {
					t.Setenv(EnvBaseConfigPathKey, "/tmp")
					t.Setenv(EnvImporterNamespaceKey, "test-namespace")
				},
				setupFiles: func(t *testing.T, tmpDir string) {
					createValidConfig(t, tmpDir, fileLoggingConfig)
					writeInvalidYaml(t, tmpDir, fileAPIConfig)
					createValidConfig(t, tmpDir, fileSSHConfig)
					createValidConfig(t, tmpDir, fileJobConfig)
				},
				expectedErrMsg: "failed to read API configuration: failed to unmarshal config",
			},
			"should fail when job config has invalid yaml": {
				setupEnv: func(t *testing.T) {
					t.Setenv(EnvBaseConfigPathKey, "/tmp")
					t.Setenv(EnvImporterNamespaceKey, "test-namespace")
				},
				setupFiles: func(t *testing.T, tmpDir string) {
					createValidConfig(t, tmpDir, fileLoggingConfig)
					createValidConfig(t, tmpDir, fileAPIConfig)
					createValidConfig(t, tmpDir, fileSSHConfig)
					writeInvalidYaml(t, tmpDir, fileJobConfig)
				},
				expectedErrMsg: "failed to read job configuration: failed to unmarshal config",
			},
			"should fail when ssh config has invalid yaml": {
				setupEnv: func(t *testing.T) {
					t.Setenv(EnvBaseConfigPathKey, "/tmp")
					t.Setenv(EnvImporterNamespaceKey, "test-namespace")
				},
				setupFiles: func(t *testing.T, tmpDir string) {
					createValidConfig(t, tmpDir, fileLoggingConfig)
					createValidConfig(t, tmpDir, fileAPIConfig)
					writeInvalidYaml(t, tmpDir, fileSSHConfig)
					createValidConfig(t, tmpDir, fileJobConfig)
				},
				expectedErrMsg: "failed to read ssh configuration: failed to unmarshal config",
			},
		}

		for name, tc := range tests {
			t.Run(name, func(t *testing.T) {
				// Setup
				tmpDir := t.TempDir()
				tc.setupEnv(t)

				// If CONFIG_PATH is set to /tmp, update it to the actual temp directory
				if os.Getenv(EnvBaseConfigPathKey) == "/tmp" {
					t.Setenv(EnvBaseConfigPathKey, tmpDir)
				}

				tc.setupFiles(t, tmpDir)

				// Execute
				config, err := ReadJobConfig()

				// Assert
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedErrMsg)
				assert.Empty(t, config)
			})
		}
	})
}

// Helper function to create valid config files
func createValidConfig(t *testing.T, dir string, filename string) {
	var content string
	switch filename {
	case fileLoggingConfig:
		content = "level: INFO"
	case fileAPIConfig:
		content = "host: test-host\napiKey: test-key\nskipTLSVerify: true"
	case fileMigrationConfig:
		content = "regularSchedule: \"0 * * * *\""
	case fileSSHConfig:
		content = "user: test-user\nprivateKeyPath: /test/path"
	case fileJobContainerConfig:
		content = "image:\n  registry: test-registry\n  repository: test-repo\n  tag: latest"
	case fileJobConfig:
		content = "doguVolumeBasePath: /data\nexclude:\n- dogu: jenkins\n  pattern: JENKINS_PATTERN\n- dogu: redmine\n  pattern: REDMINE_PATTERN"
	}

	err := os.WriteFile(path.Join(dir, filename), []byte(content), 0600)
	require.NoError(t, err)
}

// Helper function to write invalid YAML
func writeInvalidYaml(t *testing.T, dir string, filename string) {
	err := os.WriteFile(path.Join(dir, filename), []byte("invalid: yaml: }{"), 0600)
	require.NoError(t, err)
}
