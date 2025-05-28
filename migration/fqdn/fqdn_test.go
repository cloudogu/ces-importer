package fqdn

import (
	"context"
	"errors"
	"github.com/stretchr/testify/mock"
	"strings"
	"testing"

	regConfig "github.com/cloudogu/k8s-registry-lib/config"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func removeLeadingSlash(path string) string {
	return strings.TrimPrefix(path, "/")
}

var (
	mockGlobalCfgFQDNKey     = regConfig.Key(removeLeadingSlash(globalCfgFQDNKey))
	mockGlobalCfgCertTypeKey = regConfig.Key(removeLeadingSlash(globalCfgCertTypeKey))
)

// createFQDNConfigMap creates a test ConfigMap with the given name and data
func createFQDNConfigMap(name string, fqdn, certType string) *corev1.ConfigMap {
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Data: map[string]string{
			fqdnKey:     fqdn,
			certTypeKey: certType,
		},
	}
	return configMap
}

// createFQDNAlreadyExistsError creates a Kubernetes AlreadyExists error for ConfigMaps
func createFQDNAlreadyExistsError() error {
	return apierrors.NewAlreadyExists(schema.GroupResource{Resource: "configmaps"}, "already-exists")
}

// createFQDNNotFoundError creates a Kubernetes NotFound error for ConfigMaps
func createFQDNNotFoundError() error {
	return apierrors.NewNotFound(schema.GroupResource{Resource: "configmaps"}, "not-found")
}

// createTestFQDNBackup creates a test backup for FQDN
func createTestFQDNBackup() ConfigChange {
	return ConfigChange{
		FQDN:     "example.com",
		CertType: "self-signed",
	}
}

// mockFQDNManager is a mock implementation of the fqdnManager that overrides the methods to return nil (success)
type mockFQDNManager struct {
	*fqdnManager
}

func newMockFQDNManager(repo *mockConfigMapRepository, configAPI *mockConfigGetter, globalConfigRepo *mockGlobalConfigRepo) *mockFQDNManager {
	return &mockFQDNManager{
		fqdnManager: &fqdnManager{
			repo:             repo,
			globalConfigRepo: globalConfigRepo,
		},
	}
}

func (m *mockFQDNManager) Backup(ctx context.Context) error {
	// Create a test backup
	backup := createTestFQDNBackup()

	// Create a backup ConfigMap
	backupConfigMap := createFQDNConfigMap(fqdnBackupConfigMapName, backup.FQDN, backup.CertType)

	// Mock the repo.Create call
	_, err := m.repo.Create(ctx, backupConfigMap, metav1.CreateOptions{})
	return err
}

func (m *mockFQDNManager) Restore(ctx context.Context) error {
	return nil
}

func (m *mockFQDNManager) Update(ctx context.Context, c ConfigChange) error {
	return nil
}

func TestFQDNBackup(t *testing.T) {
	testCases := []struct {
		name          string
		setupMock     func(*mockConfigMapRepository, *mockGlobalConfigRepo)
		expectedError bool
		errorContains string
	}{
		{
			name: "Success - Backup fqdn - create new config map",
			setupMock: func(mockRepo *mockConfigMapRepository, mockGlobalRepo *mockGlobalConfigRepo) {
				mockGlobalRepo.EXPECT().Get(mock.Anything).Return(regConfig.CreateGlobalConfig(map[regConfig.Key]regConfig.Value{
					mockGlobalCfgFQDNKey:     "old.example.com",
					mockGlobalCfgCertTypeKey: "signed",
				}), nil)

				mockRepo.EXPECT().Create(mock.Anything, mock.Anything, mock.Anything).Return(&corev1.ConfigMap{}, nil)
			},
			expectedError: false,
		},
		{
			name: "Success - Backup fqdn - update config map",
			setupMock: func(mockRepo *mockConfigMapRepository, mockGlobalRepo *mockGlobalConfigRepo) {
				mockGlobalRepo.EXPECT().Get(mock.Anything).Return(regConfig.CreateGlobalConfig(map[regConfig.Key]regConfig.Value{
					mockGlobalCfgFQDNKey:     "old.example.com",
					mockGlobalCfgCertTypeKey: "signed",
				}), nil)

				mockRepo.EXPECT().Create(mock.Anything, mock.Anything, mock.Anything).Return(nil, createFQDNAlreadyExistsError())
				mockRepo.EXPECT().Get(mock.Anything, fqdnBackupConfigMapName, mock.Anything).Return(&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{Name: fqdnBackupConfigMapName},
					Data:       map[string]string{},
				}, nil)

				mockRepo.EXPECT().Update(mock.Anything, mock.Anything, mock.Anything).Return(&corev1.ConfigMap{}, nil)
			},
			expectedError: false,
		},
		{
			name: "Error - Get global config fails",
			setupMock: func(mockRepo *mockConfigMapRepository, mockGlobalRepo *mockGlobalConfigRepo) {
				mockGlobalRepo.EXPECT().Get(mock.Anything).Return(regConfig.GlobalConfig{}, assert.AnError)
			},
			expectedError: true,
			errorContains: "could not get global config",
		},
		{
			name: "Error - fqdn not found in global config",
			setupMock: func(mockRepo *mockConfigMapRepository, mockGlobalRepo *mockGlobalConfigRepo) {
				mockGlobalRepo.EXPECT().Get(mock.Anything).Return(regConfig.CreateGlobalConfig(map[regConfig.Key]regConfig.Value{
					"fqdn/test":              "old.example.com",
					mockGlobalCfgCertTypeKey: "signed",
				}), nil)
			},
			expectedError: true,
			errorContains: "could not find fqdn in global config",
		},
		{
			name: "Error - certificate type not found in global config",
			setupMock: func(mockRepo *mockConfigMapRepository, mockGlobalRepo *mockGlobalConfigRepo) {
				mockGlobalRepo.EXPECT().Get(mock.Anything).Return(regConfig.CreateGlobalConfig(map[regConfig.Key]regConfig.Value{
					mockGlobalCfgFQDNKey: "old.example.com",
					"certificate":        "signed",
				}), nil)
			},
			expectedError: true,
			errorContains: "could not find certificate/type in global config",
		},
		{
			name: "Error - create config map",
			setupMock: func(mockRepo *mockConfigMapRepository, mockGlobalRepo *mockGlobalConfigRepo) {
				mockGlobalRepo.EXPECT().Get(mock.Anything).Return(regConfig.CreateGlobalConfig(map[regConfig.Key]regConfig.Value{
					mockGlobalCfgFQDNKey:     "old.example.com",
					mockGlobalCfgCertTypeKey: "signed",
				}), nil)

				mockRepo.EXPECT().Create(mock.Anything, mock.Anything, mock.Anything).Return(nil, assert.AnError)
			},
			expectedError: true,
		},
		{
			name: "Error - update config map - get current config map",
			setupMock: func(mockRepo *mockConfigMapRepository, mockGlobalRepo *mockGlobalConfigRepo) {
				mockGlobalRepo.EXPECT().Get(mock.Anything).Return(regConfig.CreateGlobalConfig(map[regConfig.Key]regConfig.Value{
					mockGlobalCfgFQDNKey:     "old.example.com",
					mockGlobalCfgCertTypeKey: "signed",
				}), nil)

				mockRepo.EXPECT().Create(mock.Anything, mock.Anything, mock.Anything).Return(nil, createFQDNAlreadyExistsError())
				mockRepo.EXPECT().Get(mock.Anything, fqdnBackupConfigMapName, mock.Anything).Return(nil, assert.AnError)
			},
			expectedError: true,
			errorContains: "failed to update backup for fqdn",
		},
		{
			name: "Error - update config map - get current config map",
			setupMock: func(mockRepo *mockConfigMapRepository, mockGlobalRepo *mockGlobalConfigRepo) {
				mockGlobalRepo.EXPECT().Get(mock.Anything).Return(regConfig.CreateGlobalConfig(map[regConfig.Key]regConfig.Value{
					mockGlobalCfgFQDNKey:     "old.example.com",
					mockGlobalCfgCertTypeKey: "signed",
				}), nil)

				mockRepo.EXPECT().Create(mock.Anything, mock.Anything, mock.Anything).Return(nil, createFQDNAlreadyExistsError())
				mockRepo.EXPECT().Get(mock.Anything, fqdnBackupConfigMapName, mock.Anything).Return(&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{Name: fqdnBackupConfigMapName},
					Data:       map[string]string{},
				}, nil)

				mockRepo.EXPECT().Update(mock.Anything, mock.Anything, mock.Anything).Return(nil, assert.AnError)
			},
			expectedError: true,
			errorContains: "failed to update backup for fqdn",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create mocks
			mockRepo := newMockConfigMapRepository(t)
			mockGlobalRepo := newMockGlobalConfigRepo(t)

			// Setup mock expectations
			tc.setupMock(mockRepo, mockGlobalRepo)

			// Create fqdnManager with mocks
			manager := &fqdnManager{
				repo:             mockRepo,
				globalConfigRepo: mockGlobalRepo,
			}

			// Call Backup
			err := manager.Backup(context.Background())

			// Check error
			if tc.expectedError {
				assert.Error(t, err)
				if tc.errorContains != "" {
					assert.Contains(t, err.Error(), tc.errorContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestFQDNRestore(t *testing.T) {
	testCases := []struct {
		name          string
		setupMock     func(*mockConfigMapRepository, *mockGlobalConfigRepo)
		expectedError bool
		errorContains string
	}{
		{
			name: "Success - restore from backup",
			setupMock: func(mockRepo *mockConfigMapRepository, mockGlobalRepo *mockGlobalConfigRepo) {
				mockRepo.EXPECT().Get(mock.Anything, fqdnBackupConfigMapName, mock.Anything).Return(&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name: fqdnBackupConfigMapName,
					},
					Data: map[string]string{
						fqdnKey:     "new.example.com",
						certTypeKey: "new-self-signed",
					},
				}, nil)

				mockGlobalRepo.EXPECT().Get(mock.Anything).Return(regConfig.CreateGlobalConfig(map[regConfig.Key]regConfig.Value{
					fqdnKey:              "old.example.com",
					globalCfgCertTypeKey: "signed",
				}), nil)

				mockGlobalRepo.EXPECT().SaveOrMerge(mock.Anything, mock.Anything).Return(regConfig.GlobalConfig{}, nil)
			},
			expectedError: false,
		},
		{
			name: "Error - get config map for backup",
			setupMock: func(mockRepo *mockConfigMapRepository, mockGlobalRepo *mockGlobalConfigRepo) {
				mockRepo.EXPECT().Get(mock.Anything, fqdnBackupConfigMapName, mock.Anything).Return(nil, assert.AnError)
			},
			expectedError: true,
			errorContains: "failed to get config mapo for fqdn backup",
		},
		{
			name: "Error - update with old data",
			setupMock: func(mockRepo *mockConfigMapRepository, mockGlobalRepo *mockGlobalConfigRepo) {
				mockRepo.EXPECT().Get(mock.Anything, fqdnBackupConfigMapName, mock.Anything).Return(&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name: fqdnBackupConfigMapName,
					},
					Data: map[string]string{
						fqdnKey:     "new.example.com",
						certTypeKey: "new-self-signed",
					},
				}, nil)

				mockGlobalRepo.EXPECT().Get(mock.Anything).Return(regConfig.GlobalConfig{}, assert.AnError)
			},
			expectedError: true,
			errorContains: "failed to update fqdn",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create mocks
			mockRepo := newMockConfigMapRepository(t)
			mockGlobalRepo := newMockGlobalConfigRepo(t)

			// Setup mock expectations
			tc.setupMock(mockRepo, mockGlobalRepo)

			// Create fqdnManager with mocks
			manager := &fqdnManager{
				repo:             mockRepo,
				globalConfigRepo: mockGlobalRepo,
			}

			// Call Restore
			err := manager.Restore(context.Background())

			// Check error
			if tc.expectedError {
				assert.Error(t, err)
				if tc.errorContains != "" {
					assert.Contains(t, err.Error(), tc.errorContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestFQDNUpdate(t *testing.T) {
	testCases := []struct {
		name          string
		setupMock     func(*mockGlobalConfigRepo)
		expectedError bool
		errorContains string
	}{
		{
			name: "Success - Update FQDN and certificate type",
			setupMock: func(mockGlobalRepo *mockGlobalConfigRepo) {
				mockGlobalRepo.EXPECT().Get(mock.Anything).Return(regConfig.CreateGlobalConfig(map[regConfig.Key]regConfig.Value{
					mockGlobalCfgFQDNKey:     "old.example.com",
					mockGlobalCfgCertTypeKey: "signed",
				}), nil)

				mockGlobalRepo.EXPECT().SaveOrMerge(mock.Anything, mock.Anything).Return(regConfig.GlobalConfig{}, nil)
			},
			expectedError: false,
		},
		{
			name: "Error - Get global config fails",
			setupMock: func(mockGlobalRepo *mockGlobalConfigRepo) {
				mockGlobalRepo.EXPECT().Get(mock.Anything).Return(regConfig.GlobalConfig{}, assert.AnError)
			},
			expectedError: true,
			errorContains: "could not get global config",
		},
		{
			name: "Error - Setting new fqdn",
			setupMock: func(mockGlobalRepo *mockGlobalConfigRepo) {
				mockGlobalRepo.EXPECT().Get(mock.Anything).Return(regConfig.CreateGlobalConfig(map[regConfig.Key]regConfig.Value{
					"fqdn/test":              "old.example.com",
					mockGlobalCfgCertTypeKey: "signed",
				}), nil)
			},
			expectedError: true,
			errorContains: "failed to set new fqdn in global config",
		},
		{
			name: "Error - Setting new certificate type",
			setupMock: func(mockGlobalRepo *mockGlobalConfigRepo) {
				mockGlobalRepo.EXPECT().Get(mock.Anything).Return(regConfig.CreateGlobalConfig(map[regConfig.Key]regConfig.Value{
					mockGlobalCfgFQDNKey: "old.example.com",
					"certificate":        "signed",
				}), nil)
			},
			expectedError: true,
			errorContains: "failed to set new cert type in global config",
		},
		{
			name: "Error - Save updated global config",
			setupMock: func(mockGlobalRepo *mockGlobalConfigRepo) {
				mockGlobalRepo.EXPECT().Get(mock.Anything).Return(regConfig.CreateGlobalConfig(map[regConfig.Key]regConfig.Value{
					mockGlobalCfgFQDNKey:     "old.example.com",
					mockGlobalCfgCertTypeKey: "signed",
				}), nil)

				mockGlobalRepo.EXPECT().SaveOrMerge(mock.Anything, mock.Anything).Return(regConfig.GlobalConfig{}, assert.AnError)
			},
			expectedError: true,
			errorContains: "failed to save global config",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create mocks
			mockGlobalRepo := newMockGlobalConfigRepo(t)

			// Setup mock expectations
			tc.setupMock(mockGlobalRepo)

			// Create fqdnManager with mocks
			manager := &fqdnManager{
				globalConfigRepo: mockGlobalRepo,
			}

			// Create test ConfigChange
			change := ConfigChange{
				FQDN:     "example.com",
				CertType: "self-signed",
			}

			// Call Update
			err := manager.Update(context.Background(), change)

			// Check error
			if tc.expectedError {
				assert.Error(t, err)
				if tc.errorContains != "" {
					assert.Contains(t, err.Error(), tc.errorContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestFQDNDeleteBackup(t *testing.T) {
	testCases := []struct {
		name          string
		setupMock     func(*mockConfigMapRepository)
		expectedError bool
		errorContains string
	}{
		{
			name: "Success - Delete backup",
			setupMock: func(mockRepo *mockConfigMapRepository) {
				mockRepo.EXPECT().Delete(mock.Anything, fqdnBackupConfigMapName, mock.Anything).Return(nil)
			},
			expectedError: false,
		},
		{
			name: "Success - Backup not found",
			setupMock: func(mockRepo *mockConfigMapRepository) {
				mockRepo.EXPECT().Delete(mock.Anything, fqdnBackupConfigMapName, mock.Anything).Return(createFQDNNotFoundError())
			},
			expectedError: false,
		},
		{
			name: "Error - Delete fails with other error",
			setupMock: func(mockRepo *mockConfigMapRepository) {
				mockRepo.EXPECT().Delete(mock.Anything, fqdnBackupConfigMapName, mock.Anything).Return(errors.New("delete error"))
			},
			expectedError: true,
			errorContains: "failed to delete config map for fqdn backup",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create mocks
			mockRepo := newMockConfigMapRepository(t)

			// Setup mock expectations
			tc.setupMock(mockRepo)

			// Create fqdnManager with mocks
			manager := fqdnManager{
				repo: mockRepo,
			}

			// Call DeleteBackup
			err := manager.DeleteBackup(context.Background())

			// Check error
			if tc.expectedError {
				assert.Error(t, err)
				if tc.errorContains != "" {
					assert.Contains(t, err.Error(), tc.errorContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
