package fqdn

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// newEcosystemCertificate creates a new ecosystemCertificate with the given mock repository
func newEcosystemCertificate(repo secretRepository) *ecosystemCertificate {
	return &ecosystemCertificate{
		repo: repo,
	}
}

// createSecret creates a test secret with the given name and data
func createSecret(name string, certData, keyData []byte) *corev1.Secret {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Data: map[string][]byte{
			certFileName: certData,
			keyFileName:  keyData,
		},
		StringData: map[string]string{},
	}
	return secret
}

// createAlreadyExistsError creates a Kubernetes AlreadyExists error
func createAlreadyExistsError() error {
	return apierrors.NewAlreadyExists(schema.GroupResource{Resource: "secrets"}, "already-exists")
}

// createNotFoundError creates a Kubernetes NotFound error
func createNotFoundError() error {
	return apierrors.NewNotFound(schema.GroupResource{Resource: "secrets"}, "not-found")
}

func TestBackup(t *testing.T) {
	testCases := []struct {
		name          string
		setupMock     func(*mockSecretRepository)
		expectedError bool
		errorContains string
	}{
		{
			name: "Success - Create backup",
			setupMock: func(m *mockSecretRepository) {
				// Setup Get to return a secret
				certData := []byte("cert-data")
				keyData := []byte("key-data")
				secret := createSecret(ecosystemCertificateName, certData, keyData)

				m.EXPECT().Get(mock.Anything, ecosystemCertificateName, mock.Anything).Return(secret, nil)

				// The backup secret should have the same data but a different name
				backupSecret := createSecret(certificateBackupName, certData, keyData)
				m.EXPECT().Create(mock.Anything, mock.MatchedBy(func(s *corev1.Secret) bool {
					return s.Name == certificateBackupName
				}), mock.Anything).Return(backupSecret, nil)
			},
			expectedError: false,
		},
		{
			name: "Error - Get fails",
			setupMock: func(m *mockSecretRepository) {
				m.EXPECT().Get(mock.Anything, ecosystemCertificateName, mock.Anything).Return(nil, errors.New("get error"))
			},
			expectedError: true,
			errorContains: "failed to get secret for certificate",
		},
		{
			name: "Success - ConfigChange already exists, update succeeds",
			setupMock: func(m *mockSecretRepository) {
				// Setup Get to return a secret
				certData := []byte("cert-data")
				keyData := []byte("key-data")
				secret := createSecret(ecosystemCertificateName, certData, keyData)

				m.EXPECT().Get(mock.Anything, ecosystemCertificateName, mock.Anything).Return(secret, nil)

				// Create fails with AlreadyExists
				m.EXPECT().Create(mock.Anything, mock.Anything, mock.Anything).Return(nil, createAlreadyExistsError())

				// Get for update
				backupSecret := createSecret(certificateBackupName, certData, keyData)
				m.EXPECT().Get(mock.Anything, certificateBackupName, mock.Anything).Return(backupSecret, nil)

				// Update succeeds
				m.EXPECT().Update(mock.Anything, mock.Anything, mock.Anything).Return(backupSecret, nil)
			},
			expectedError: false,
		},
		{
			name: "Error - ConfigChange already exists, update fails",
			setupMock: func(m *mockSecretRepository) {
				// Setup Get to return a secret
				certData := []byte("cert-data")
				keyData := []byte("key-data")
				secret := createSecret(ecosystemCertificateName, certData, keyData)

				m.EXPECT().Get(mock.Anything, ecosystemCertificateName, mock.Anything).Return(secret, nil)

				// Create fails with AlreadyExists
				m.EXPECT().Create(mock.Anything, mock.Anything, mock.Anything).Return(nil, createAlreadyExistsError())

				// Get for update
				backupSecret := createSecret(certificateBackupName, certData, keyData)
				m.EXPECT().Get(mock.Anything, certificateBackupName, mock.Anything).Return(backupSecret, nil)

				// Update fails
				m.EXPECT().Update(mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.New("update error"))
			},
			expectedError: true,
			errorContains: "failed to update backup for certificate",
		},
		{
			name: "Error - Create fails with other error",
			setupMock: func(m *mockSecretRepository) {
				// Setup Get to return a secret
				certData := []byte("cert-data")
				keyData := []byte("key-data")
				secret := createSecret(ecosystemCertificateName, certData, keyData)

				m.EXPECT().Get(mock.Anything, ecosystemCertificateName, mock.Anything).Return(secret, nil)

				// Create fails with other error
				m.EXPECT().Create(mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.New("create error"))
			},
			expectedError: true,
			errorContains: "failed to create secret for backup certificate",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create mock
			mockRepo := newMockSecretRepository(t)

			// Setup mock expectations
			tc.setupMock(mockRepo)

			// Create ecosystemCertificate with mock
			cert := newEcosystemCertificate(mockRepo)

			// Call ConfigChange
			err := cert.Backup(context.Background())

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

func TestRestore(t *testing.T) {
	testCases := []struct {
		name          string
		setupMock     func(*mockSecretRepository)
		expectedError bool
		errorContains string
	}{
		{
			name: "Success - Restore from backup",
			setupMock: func(m *mockSecretRepository) {
				// Setup Get to return a backup secret
				certData := []byte("cert-data")
				keyData := []byte("key-data")
				backupSecret := createSecret(certificateBackupName, certData, keyData)

				m.EXPECT().Get(mock.Anything, certificateBackupName, mock.Anything).Return(backupSecret, nil)

				// Get for update
				originalSecret := createSecret(ecosystemCertificateName, []byte("old-cert"), []byte("old-key"))
				m.EXPECT().Get(mock.Anything, ecosystemCertificateName, mock.Anything).Return(originalSecret, nil)

				// Update succeeds
				m.EXPECT().Update(mock.Anything, mock.Anything, mock.Anything).Return(originalSecret, nil)
			},
			expectedError: false,
		},
		{
			name: "Error - Get backup fails",
			setupMock: func(m *mockSecretRepository) {
				m.EXPECT().Get(mock.Anything, certificateBackupName, mock.Anything).Return(nil, errors.New("get error"))
			},
			expectedError: true,
			errorContains: "failed to get secret for backup certificate",
		},
		{
			name: "Error - Update fails",
			setupMock: func(m *mockSecretRepository) {
				// Setup Get to return a backup secret
				certData := []byte("cert-data")
				keyData := []byte("key-data")
				backupSecret := createSecret(certificateBackupName, certData, keyData)

				m.EXPECT().Get(mock.Anything, certificateBackupName, mock.Anything).Return(backupSecret, nil)

				// Get for update
				originalSecret := createSecret(ecosystemCertificateName, []byte("old-cert"), []byte("old-key"))
				m.EXPECT().Get(mock.Anything, ecosystemCertificateName, mock.Anything).Return(originalSecret, nil)

				// Update fails
				m.EXPECT().Update(mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.New("update error"))
			},
			expectedError: true,
			errorContains: "failed to update certificate",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create mock
			mockRepo := newMockSecretRepository(t)

			// Setup mock expectations
			tc.setupMock(mockRepo)

			// Create ecosystemCertificate with mock
			cert := newEcosystemCertificate(mockRepo)

			// Call Restore
			err := cert.Restore(context.Background())

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

func TestUpdate(t *testing.T) {
	testCases := []struct {
		name          string
		setupMock     func(*mockSecretRepository)
		expectedError bool
		errorContains string
	}{
		{
			name: "Success - Update certificate",
			setupMock: func(m *mockSecretRepository) {
				// Setup Get to return a secret
				secret := createSecret(ecosystemCertificateName, []byte("old-cert"), []byte("old-key"))

				m.EXPECT().Get(mock.Anything, ecosystemCertificateName, mock.Anything).Return(secret, nil)

				// Update succeeds
				m.EXPECT().Update(mock.Anything, mock.Anything, mock.Anything).Return(secret, nil)
			},
			expectedError: false,
		},
		{
			name: "Error - Get fails",
			setupMock: func(m *mockSecretRepository) {
				m.EXPECT().Get(mock.Anything, ecosystemCertificateName, mock.Anything).Return(nil, errors.New("get error"))
			},
			expectedError: true,
			errorContains: "failed to get secret for certificate",
		},
		{
			name: "Error - Update fails",
			setupMock: func(m *mockSecretRepository) {
				// Setup Get to return a secret
				secret := createSecret(ecosystemCertificateName, []byte("old-cert"), []byte("old-key"))

				m.EXPECT().Get(mock.Anything, ecosystemCertificateName, mock.Anything).Return(secret, nil)

				// Update fails
				m.EXPECT().Update(mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.New("update error"))
			},
			expectedError: true,
			errorContains: "failed to update secret for certificate",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create mock
			mockRepo := newMockSecretRepository(t)

			// Setup mock expectations
			tc.setupMock(mockRepo)

			// Create ecosystemCertificate with mock
			cert := newEcosystemCertificate(mockRepo)

			// Create test SecretChange
			tlsKeyPair := SecretChange{
				Certificate:    []byte("new-cert"),
				CertificateKey: []byte("new-key"),
			}

			// Call Update
			err := cert.Update(context.Background(), tlsKeyPair)

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

func TestDeleteBackup(t *testing.T) {
	testCases := []struct {
		name          string
		setupMock     func(*mockSecretRepository)
		expectedError bool
		errorContains string
	}{
		{
			name: "Success - Delete backup",
			setupMock: func(m *mockSecretRepository) {
				m.EXPECT().Delete(mock.Anything, certificateBackupName, mock.Anything).Return(nil)
			},
			expectedError: false,
		},
		{
			name: "Success - ConfigChange not found",
			setupMock: func(m *mockSecretRepository) {
				m.EXPECT().Delete(mock.Anything, certificateBackupName, mock.Anything).Return(createNotFoundError())
			},
			expectedError: false,
		},
		{
			name: "Error - Delete fails with other error",
			setupMock: func(m *mockSecretRepository) {
				m.EXPECT().Delete(mock.Anything, certificateBackupName, mock.Anything).Return(errors.New("delete error"))
			},
			expectedError: true,
			errorContains: "failed to delete secret for certificate backup",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create mock
			mockRepo := newMockSecretRepository(t)

			// Setup mock expectations
			tc.setupMock(mockRepo)

			// Create ecosystemCertificate with mock
			cert := newEcosystemCertificate(mockRepo)

			// Call DeleteBackup
			err := cert.DeleteBackup(context.Background())

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
