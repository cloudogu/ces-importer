package configuration

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"os"
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
			SecretName:     "ces-exporter-secret",
			SecretDataKey:  "apiKey",
		}, cfg.API)

		// migration
		assert.Equal(t, Migration{
			RegularCron:    "0 4 * * *",
			FinalTimestamp: "2025-04-03T12:34:56Z",
			ChangeFQDN:     true,
			MaintenanceModeMessage: &MaintenanceModeMessage{
				Title: "Migration completed.",
				Text:  "The migration of your instance has been completed.",
			},
			ExecutePreflightCheck: true,
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
			AdditionalExcludedDogus: []string{
				"DOGU_EXCLUDE_1",
				"DOGU_EXCLUDE_2",
			},
			Exclude: []ExcludePattern{
				{
					DoguName: "jenkins",
					Pattern:  []string{"JENKINS_PATTERN"},
				},
				{
					DoguName: "redmine",
					Pattern: []string{
						"REDMINE_PATTERN_1",
						"REDMINE_PATTERN_2",
					},
				},
			},
			Verbose: true,
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

		// smtp
		assert.Equal(t, Smtp{
			Server:   "192.168.56.1",
			Port:     1025,
			Username: "",
			Password: "",
			From:     "importer@ces.com",
			To: []string{
				"recipient1@example.com",
				"recipient2@example.com",
			},
			SecretName:    "ces-importer-secret",
			SecretDataKey: "mailPassword",
		}, cfg.Smtp)
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
			"should fail when smtp config is missing": {
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
					createValidConfig(t, tmpDir, fileJobContainerConfig)
					// Don't create SMTP config
				},
				expectedErrMsg: "failed to read smtp configuration",
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
			"should fail when smtp config has invalid yaml": {
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
					createValidConfig(t, tmpDir, fileJobContainerConfig)
					writeInvalidYaml(t, tmpDir, fileSMTPConfig)
				},
				expectedErrMsg: "failed to read smtp configuration: failed to unmarshal config",
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

func TestCoordinator_ValidateSecrets(t *testing.T) {
	tests := []struct {
		name          string
		setupMock     func(*mockSecretGetter)
		inCoordinator Coordinator
		expErr        bool
		errorContains string
	}{
		{
			name: "successful validation - API, SSH and SMTP Keys in single secret",
			setupMock: func(mockSecretGetter *mockSecretGetter) {
				mockSecretGetter.EXPECT().Get(mock.Anything, mock.Anything, metav1.GetOptions{}).Return(&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{Name: "valid-secret"},
					Data: map[string][]byte{
						"apiKey":       []byte("testAPIKEY"),
						"privateKey":   []byte("testPrivateKey"),
						"mailPassword": []byte("testMailPassword"),
					},
				}, nil)
			},
			inCoordinator: Coordinator{
				API: API{
					SecretName:    "valid-secret",
					SecretDataKey: "apiKey",
				},
				SSH: SSH{
					SecretName:    "valid-secret",
					SecretDataKey: "privateKey",
				},
				Smtp: Smtp{
					Server:        "testSMTPServer",
					Username:      "testUser",
					SecretName:    "valid-secret",
					SecretDataKey: "mailPassword",
				},
			},
			expErr: false,
		},
		{
			name: "successful validation - API, SSH and SMTP Keys in separate secrets",
			setupMock: func(mockSecretGetter *mockSecretGetter) {
				mockSecretGetter.EXPECT().Get(mock.Anything, "valid-api-secret", metav1.GetOptions{}).Return(&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{Name: "valid-api-secret"},
					Data: map[string][]byte{
						"apiKey": []byte("testAPIKEY"),
					},
				}, nil)

				mockSecretGetter.EXPECT().Get(mock.Anything, "valid-ssh-secret", metav1.GetOptions{}).Return(&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{Name: "valid-ssh-secret"},
					Data: map[string][]byte{
						"privateKey": []byte("testPrivateKey"),
					},
				}, nil)

				mockSecretGetter.EXPECT().Get(mock.Anything, "valid-smtp-secret", metav1.GetOptions{}).Return(&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{Name: "valid-ssh-secret"},
					Data: map[string][]byte{
						"mailPassword": []byte("testMailPassword"),
					},
				}, nil)
			},
			inCoordinator: Coordinator{
				API: API{
					SecretName:    "valid-api-secret",
					SecretDataKey: "apiKey",
				},
				SSH: SSH{
					SecretName:    "valid-ssh-secret",
					SecretDataKey: "privateKey",
				},
				Smtp: Smtp{
					Server:        "testSMTPServer",
					Username:      "testUser",
					SecretName:    "valid-smtp-secret",
					SecretDataKey: "mailPassword",
				},
			},
			expErr: false,
		},
		{
			name: "successful validation - SMTP Key when Server is not set",
			setupMock: func(mockSecretGetter *mockSecretGetter) {
				mockSecretGetter.EXPECT().Get(mock.Anything, mock.Anything, metav1.GetOptions{}).Return(&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{Name: "valid-secret"},
					Data: map[string][]byte{
						"apiKey":     []byte("testAPIKEY"),
						"privateKey": []byte("testPrivateKey"),
					},
				}, nil)
			},
			inCoordinator: Coordinator{
				API: API{
					SecretName:    "valid-secret",
					SecretDataKey: "apiKey",
				},
				SSH: SSH{
					SecretName:    "valid-secret",
					SecretDataKey: "privateKey",
				},
				Smtp: Smtp{
					Server: "",
				},
			},
			expErr: false,
		},
		{
			name: "successful validation - SMTP when Server is set but no user",
			setupMock: func(mockSecretGetter *mockSecretGetter) {
				mockSecretGetter.EXPECT().Get(mock.Anything, mock.Anything, metav1.GetOptions{}).Return(&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{Name: "valid-secret"},
					Data: map[string][]byte{
						"apiKey":     []byte("testAPIKEY"),
						"privateKey": []byte("testPrivateKey"),
					},
				}, nil)
			},
			inCoordinator: Coordinator{
				API: API{
					SecretName:    "valid-secret",
					SecretDataKey: "apiKey",
				},
				SSH: SSH{
					SecretName:    "valid-secret",
					SecretDataKey: "privateKey",
				},
				Smtp: Smtp{
					Server: "testSMTPServer",
				},
			},
			expErr: false,
		},
		{
			name: "Error - secret is missing",
			setupMock: func(mockSecretGetter *mockSecretGetter) {
				mockSecretGetter.EXPECT().Get(mock.Anything, "valid-secret", metav1.GetOptions{}).Return(nil, apierrors.NewNotFound(schema.GroupResource{}, "valid-secret"))
			},
			inCoordinator: Coordinator{
				API: API{
					SecretName:    "valid-secret",
					SecretDataKey: "apiKey",
				},
				SSH: SSH{
					SecretName:    "valid-secret",
					SecretDataKey: "privateKey",
				},
				Smtp: Smtp{
					Server:        "testSMTPServer",
					Username:      "testUser",
					SecretName:    "valid-secret",
					SecretDataKey: "mailPassword",
				},
			},
			expErr:        true,
			errorContains: "failed to get secret valid-secret",
		},
		{
			name: "Error - API data key is missing",
			setupMock: func(mockSecretGetter *mockSecretGetter) {
				mockSecretGetter.EXPECT().Get(mock.Anything, mock.Anything, metav1.GetOptions{}).Return(&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{Name: "valid-secret"},
					Data: map[string][]byte{
						"privateKey":   []byte("testPrivateKey"),
						"mailPassword": []byte("testMailPassword"),
					},
				}, nil)
			},
			inCoordinator: Coordinator{
				API: API{
					SecretName:    "valid-secret",
					SecretDataKey: "apiKey",
				},
				SSH: SSH{
					SecretName:    "valid-secret",
					SecretDataKey: "privateKey",
				},
				Smtp: Smtp{
					Server:        "testSMTPServer",
					Username:      "testUser",
					SecretName:    "valid-secret",
					SecretDataKey: "mailPassword",
				},
			},
			expErr:        true,
			errorContains: "secret valid-secret does not contain key apiKey",
		},
		{
			name: "Error - SSH data key is missing",
			setupMock: func(mockSecretGetter *mockSecretGetter) {
				mockSecretGetter.EXPECT().Get(mock.Anything, mock.Anything, metav1.GetOptions{}).Return(&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{Name: "valid-secret"},
					Data: map[string][]byte{
						"apiKey":       []byte("testAPIKEY"),
						"mailPassword": []byte("testMailPassword"),
					},
				}, nil)
			},
			inCoordinator: Coordinator{
				API: API{
					SecretName:    "valid-secret",
					SecretDataKey: "apiKey",
				},
				SSH: SSH{
					SecretName:    "valid-secret",
					SecretDataKey: "privateKey",
				},
				Smtp: Smtp{
					Server:        "testSMTPServer",
					Username:      "testUser",
					SecretName:    "valid-secret",
					SecretDataKey: "mailPassword",
				},
			},
			expErr:        true,
			errorContains: "secret valid-secret does not contain key privateKey",
		},
		{
			name: "Error - MailPassword data key is missing",
			setupMock: func(mockSecretGetter *mockSecretGetter) {
				mockSecretGetter.EXPECT().Get(mock.Anything, mock.Anything, metav1.GetOptions{}).Return(&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{Name: "valid-secret"},
					Data: map[string][]byte{
						"apiKey":     []byte("testAPIKEY"),
						"privateKey": []byte("testPrivateKey"),
					},
				}, nil)
			},
			inCoordinator: Coordinator{
				API: API{
					SecretName:    "valid-secret",
					SecretDataKey: "apiKey",
				},
				SSH: SSH{
					SecretName:    "valid-secret",
					SecretDataKey: "privateKey",
				},
				Smtp: Smtp{
					Server:        "testSMTPServer",
					Username:      "testUser",
					SecretName:    "valid-secret",
					SecretDataKey: "mailPassword",
				},
			},
			expErr:        true,
			errorContains: "secret valid-secret does not contain key mailPassword",
		},
		{
			name: "Error - SSH data key, API data key and SMTP data key are missing",
			setupMock: func(mockSecretGetter *mockSecretGetter) {
				mockSecretGetter.EXPECT().Get(mock.Anything, mock.Anything, metav1.GetOptions{}).Return(&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{Name: "valid-secret"},
					Data:       map[string][]byte{},
				}, nil)
			},
			inCoordinator: Coordinator{
				API: API{
					SecretName:    "valid-secret",
					SecretDataKey: "apiKey",
				},
				SSH: SSH{
					SecretName:    "valid-secret",
					SecretDataKey: "privateKey",
				},
				Smtp: Smtp{
					Server:        "testSMTPServer",
					Username:      "testUser",
					SecretName:    "valid-secret",
					SecretDataKey: "mailPassword",
				},
			},
			expErr:        true,
			errorContains: "secret valid-secret does not contain key apiKey\nsecret valid-secret does not contain key privateKey\nsecret valid-secret does not contain key mailPassword",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			secretGetterMock := newMockSecretGetter(t)
			tt.setupMock(secretGetterMock)

			err := tt.inCoordinator.ValidateSecrets(context.TODO(), secretGetterMock)
			if tt.expErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorContains)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
