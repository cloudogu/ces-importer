package fqdn

import (
	"context"
	"github.com/cloudogu/ces-importer/migration"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
)

type exporterConfigTestCase uint8

const (
	validExporterConfig exporterConfigTestCase = iota
	emptyFqdn
	emptyCertType
	emptyCert
	emptyKey
)

func createExporterConfig(tc exporterConfigTestCase) *migration.Configuration {
	var globalConfigValues []migration.KeyValue

	switch tc {
	case validExporterConfig:
		globalConfigValues = []migration.KeyValue{
			{Key: globalCfgFQDNKey, Value: "test.com"},
			{Key: globalCfgCertTypeKey, Value: "self-signed"},
			{Key: globalCfgCertKey, Value: "certificate"},
			{Key: globalCfgPrivateKey, Value: "privateKey"},
		}
	case emptyFqdn:
		globalConfigValues = []migration.KeyValue{
			// empty fqdn
			{Key: globalCfgCertTypeKey, Value: "self-signed"},
			{Key: globalCfgCertKey, Value: "certificate"},
			{Key: globalCfgPrivateKey, Value: "privateKey"},
		}
	case emptyCertType:
		globalConfigValues = []migration.KeyValue{
			{Key: globalCfgFQDNKey, Value: "test.com"},
			// empty certificate type
			{Key: globalCfgCertKey, Value: "certificate"},
			{Key: globalCfgPrivateKey, Value: "privateKey"},
		}
	case emptyCert:
		globalConfigValues = []migration.KeyValue{
			{Key: globalCfgFQDNKey, Value: "test.com"},
			{Key: globalCfgCertTypeKey, Value: "self-signed"},
			// empty certificate
			{Key: globalCfgPrivateKey, Value: "privateKey"},
		}
	case emptyKey:
		globalConfigValues = []migration.KeyValue{
			{Key: globalCfgFQDNKey, Value: "test.com"},
			{Key: globalCfgCertTypeKey, Value: "self-signed"},
			{Key: globalCfgCertKey, Value: "certificate"},
			// empty key
		}
	}

	return &migration.Configuration{
		GlobalConfig: globalConfigValues,
	}
}

func TestService_ChangeFQDN(t *testing.T) {
	testCases := []struct {
		name          string
		setupMock     func(*mockConfigGetter, *mockFqdnChanger, *mockCertificateChanger)
		expectedError bool
		errorContains string
	}{
		{
			name: "Success - change FQDN",
			setupMock: func(getter *mockConfigGetter, fChanger *mockFqdnChanger, cChanger *mockCertificateChanger) {
				fChanger.EXPECT().Backup(mock.Anything).Return(nil)
				cChanger.EXPECT().Backup(mock.Anything).Return(nil)

				getter.EXPECT().GetConfig(mock.Anything).Return(createExporterConfig(validExporterConfig), nil)

				fChanger.EXPECT().Update(mock.Anything, mock.Anything).Return(nil)
				cChanger.EXPECT().Update(mock.Anything, mock.Anything).Return(nil)

				fChanger.EXPECT().DeleteBackup(mock.Anything).Return(nil)
				cChanger.EXPECT().DeleteBackup(mock.Anything).Return(nil)
			},
			expectedError: false,
		},
		{
			name: "Success - change FQDN - delete fqdn backup failed",
			setupMock: func(getter *mockConfigGetter, fChanger *mockFqdnChanger, cChanger *mockCertificateChanger) {
				fChanger.EXPECT().Backup(mock.Anything).Return(nil)
				cChanger.EXPECT().Backup(mock.Anything).Return(nil)

				getter.EXPECT().GetConfig(mock.Anything).Return(createExporterConfig(validExporterConfig), nil)

				fChanger.EXPECT().Update(mock.Anything, mock.Anything).Return(nil)
				cChanger.EXPECT().Update(mock.Anything, mock.Anything).Return(nil)

				fChanger.EXPECT().DeleteBackup(mock.Anything).Return(assert.AnError)
			},
			expectedError: false,
		},
		{
			name: "Success - change FQDN - delete certificate backup failed",
			setupMock: func(getter *mockConfigGetter, fChanger *mockFqdnChanger, cChanger *mockCertificateChanger) {
				fChanger.EXPECT().Backup(mock.Anything).Return(nil)
				cChanger.EXPECT().Backup(mock.Anything).Return(nil)

				getter.EXPECT().GetConfig(mock.Anything).Return(createExporterConfig(validExporterConfig), nil)

				fChanger.EXPECT().Update(mock.Anything, mock.Anything).Return(nil)
				cChanger.EXPECT().Update(mock.Anything, mock.Anything).Return(nil)

				fChanger.EXPECT().DeleteBackup(mock.Anything).Return(nil)
				cChanger.EXPECT().DeleteBackup(mock.Anything).Return(assert.AnError)
			},
			expectedError: false,
		},
		{
			name: "Error - backup for fqdn",
			setupMock: func(getter *mockConfigGetter, fChanger *mockFqdnChanger, cChanger *mockCertificateChanger) {
				fChanger.EXPECT().Backup(mock.Anything).Return(assert.AnError)
			},
			expectedError: true,
			errorContains: "failed to backup fqdn",
		},
		{
			name: "Error - backup for certificate",
			setupMock: func(getter *mockConfigGetter, fChanger *mockFqdnChanger, cChanger *mockCertificateChanger) {
				fChanger.EXPECT().Backup(mock.Anything).Return(nil)
				cChanger.EXPECT().Backup(mock.Anything).Return(assert.AnError)
			},
			expectedError: true,
			errorContains: "failed to backup certificate",
		},
		{
			name: "Error - get config from exporter",
			setupMock: func(getter *mockConfigGetter, fChanger *mockFqdnChanger, cChanger *mockCertificateChanger) {
				fChanger.EXPECT().Backup(mock.Anything).Return(nil)
				cChanger.EXPECT().Backup(mock.Anything).Return(nil)

				getter.EXPECT().GetConfig(mock.Anything).Return(nil, assert.AnError)

				fChanger.EXPECT().DeleteBackup(mock.Anything).Return(nil)
				cChanger.EXPECT().DeleteBackup(mock.Anything).Return(nil)
			},
			expectedError: true,
			errorContains: "failed to get config from exporter",
		},
		{
			name: "Error - invalid config: empty fqdn",
			setupMock: func(getter *mockConfigGetter, fChanger *mockFqdnChanger, cChanger *mockCertificateChanger) {
				fChanger.EXPECT().Backup(mock.Anything).Return(nil)
				cChanger.EXPECT().Backup(mock.Anything).Return(nil)

				getter.EXPECT().GetConfig(mock.Anything).Return(createExporterConfig(emptyFqdn), nil)

				fChanger.EXPECT().DeleteBackup(mock.Anything).Return(nil)
				cChanger.EXPECT().DeleteBackup(mock.Anything).Return(nil)
			},
			expectedError: true,
			errorContains: "fqdn is empty",
		},
		{
			name: "Error - invalid config: empty certificate type",
			setupMock: func(getter *mockConfigGetter, fChanger *mockFqdnChanger, cChanger *mockCertificateChanger) {
				fChanger.EXPECT().Backup(mock.Anything).Return(nil)
				cChanger.EXPECT().Backup(mock.Anything).Return(nil)

				getter.EXPECT().GetConfig(mock.Anything).Return(createExporterConfig(emptyCertType), nil)

				fChanger.EXPECT().DeleteBackup(mock.Anything).Return(nil)
				cChanger.EXPECT().DeleteBackup(mock.Anything).Return(nil)
			},
			expectedError: true,
			errorContains: "certificate type is empty",
		},
		{
			name: "Error - invalid config: empty certificate",
			setupMock: func(getter *mockConfigGetter, fChanger *mockFqdnChanger, cChanger *mockCertificateChanger) {
				fChanger.EXPECT().Backup(mock.Anything).Return(nil)
				cChanger.EXPECT().Backup(mock.Anything).Return(nil)

				getter.EXPECT().GetConfig(mock.Anything).Return(createExporterConfig(emptyCert), nil)

				fChanger.EXPECT().DeleteBackup(mock.Anything).Return(nil)
				cChanger.EXPECT().DeleteBackup(mock.Anything).Return(nil)
			},
			expectedError: true,
			errorContains: "certificate is empty",
		},
		{
			name: "Error - invalid config: empty certificate key",
			setupMock: func(getter *mockConfigGetter, fChanger *mockFqdnChanger, cChanger *mockCertificateChanger) {
				fChanger.EXPECT().Backup(mock.Anything).Return(nil)
				cChanger.EXPECT().Backup(mock.Anything).Return(nil)

				getter.EXPECT().GetConfig(mock.Anything).Return(createExporterConfig(emptyKey), nil)

				fChanger.EXPECT().DeleteBackup(mock.Anything).Return(nil)
				cChanger.EXPECT().DeleteBackup(mock.Anything).Return(nil)
			},
			expectedError: true,
			errorContains: "certificate key is empty",
		},
		{
			name: "Error - update certificate - restore successful",
			setupMock: func(getter *mockConfigGetter, fChanger *mockFqdnChanger, cChanger *mockCertificateChanger) {
				fChanger.EXPECT().Backup(mock.Anything).Return(nil)
				cChanger.EXPECT().Backup(mock.Anything).Return(nil)

				getter.EXPECT().GetConfig(mock.Anything).Return(createExporterConfig(validExporterConfig), nil)

				fChanger.EXPECT().Update(mock.Anything, mock.Anything).Return(nil)
				cChanger.EXPECT().Update(mock.Anything, mock.Anything).Return(assert.AnError)

				cChanger.EXPECT().Restore(mock.Anything).Return(nil)
				fChanger.EXPECT().Restore(mock.Anything).Return(nil)

				fChanger.EXPECT().DeleteBackup(mock.Anything).Return(nil)
				cChanger.EXPECT().DeleteBackup(mock.Anything).Return(nil)
			},
			expectedError: true,
			errorContains: "failed to update fqdn",
		},
		{
			name: "Error - update fqdn - restore successful",
			setupMock: func(getter *mockConfigGetter, fChanger *mockFqdnChanger, cChanger *mockCertificateChanger) {
				fChanger.EXPECT().Backup(mock.Anything).Return(nil)
				cChanger.EXPECT().Backup(mock.Anything).Return(nil)

				getter.EXPECT().GetConfig(mock.Anything).Return(createExporterConfig(validExporterConfig), nil)

				fChanger.EXPECT().Update(mock.Anything, mock.Anything).Return(assert.AnError)

				cChanger.EXPECT().Restore(mock.Anything).Return(nil)
				fChanger.EXPECT().Restore(mock.Anything).Return(nil)

				fChanger.EXPECT().DeleteBackup(mock.Anything).Return(nil)
				cChanger.EXPECT().DeleteBackup(mock.Anything).Return(nil)
			},
			expectedError: true,
			errorContains: "failed to update fqdn",
		},
		{
			name: "Error - update certificate - restore certificate failed",
			setupMock: func(getter *mockConfigGetter, fChanger *mockFqdnChanger, cChanger *mockCertificateChanger) {
				fChanger.EXPECT().Backup(mock.Anything).Return(nil)
				cChanger.EXPECT().Backup(mock.Anything).Return(nil)

				getter.EXPECT().GetConfig(mock.Anything).Return(createExporterConfig(validExporterConfig), nil)

				fChanger.EXPECT().Update(mock.Anything, mock.Anything).Return(nil)
				cChanger.EXPECT().Update(mock.Anything, mock.Anything).Return(assert.AnError)

				cChanger.EXPECT().Restore(mock.Anything).Return(assert.AnError)
			},
			expectedError: true,
			errorContains: "failed to update fqdn or certificate",
		},
		{
			name: "Error - update certificate - restore fqdn failed",
			setupMock: func(getter *mockConfigGetter, fChanger *mockFqdnChanger, cChanger *mockCertificateChanger) {
				fChanger.EXPECT().Backup(mock.Anything).Return(nil)
				cChanger.EXPECT().Backup(mock.Anything).Return(nil)

				getter.EXPECT().GetConfig(mock.Anything).Return(createExporterConfig(validExporterConfig), nil)

				fChanger.EXPECT().Update(mock.Anything, mock.Anything).Return(nil)
				cChanger.EXPECT().Update(mock.Anything, mock.Anything).Return(assert.AnError)

				cChanger.EXPECT().Restore(mock.Anything).Return(nil)
				fChanger.EXPECT().Restore(mock.Anything).Return(assert.AnError)
			},
			expectedError: true,
			errorContains: "failed to update fqdn or certificate",
		},
		{
			name: "Error - update fqdn - restore successful - delete fqdn backup failed",
			setupMock: func(getter *mockConfigGetter, fChanger *mockFqdnChanger, cChanger *mockCertificateChanger) {
				fChanger.EXPECT().Backup(mock.Anything).Return(nil)
				cChanger.EXPECT().Backup(mock.Anything).Return(nil)

				getter.EXPECT().GetConfig(mock.Anything).Return(createExporterConfig(validExporterConfig), nil)

				fChanger.EXPECT().Update(mock.Anything, mock.Anything).Return(assert.AnError)

				cChanger.EXPECT().Restore(mock.Anything).Return(nil)
				fChanger.EXPECT().Restore(mock.Anything).Return(nil)

				fChanger.EXPECT().DeleteBackup(mock.Anything).Return(assert.AnError)
			},
			expectedError: true,
			errorContains: "failed to update fqdn",
		},
		{
			name: "Error - update fqdn - restore successful - delete certificate backup failed",
			setupMock: func(getter *mockConfigGetter, fChanger *mockFqdnChanger, cChanger *mockCertificateChanger) {
				fChanger.EXPECT().Backup(mock.Anything).Return(nil)
				cChanger.EXPECT().Backup(mock.Anything).Return(nil)

				getter.EXPECT().GetConfig(mock.Anything).Return(createExporterConfig(validExporterConfig), nil)

				fChanger.EXPECT().Update(mock.Anything, mock.Anything).Return(assert.AnError)

				cChanger.EXPECT().Restore(mock.Anything).Return(nil)
				fChanger.EXPECT().Restore(mock.Anything).Return(nil)

				fChanger.EXPECT().DeleteBackup(mock.Anything).Return(nil)
				cChanger.EXPECT().DeleteBackup(mock.Anything).Return(assert.AnError)
			},
			expectedError: true,
			errorContains: "failed to update fqdn",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create mock
			configGetterMock := newMockConfigGetter(t)
			fqdnChangerMock := newMockFqdnChanger(t)
			certChangerMock := newMockCertificateChanger(t)

			// Setup mock expectations
			tc.setupMock(configGetterMock, fqdnChangerMock, certChangerMock)

			service := &Service{
				configService: configGetterMock,
				fqdn:          fqdnChangerMock,
				cert:          certChangerMock,
			}

			err := service.ChangeFQDN(context.TODO())

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

func TestNewService(t *testing.T) {
	service := NewService(newMockConfigGetter(t), newMockGlobalConfigRepo(t), newMockConfigMapRepository(t), newMockSecretRepository(t))

	assert.NotNil(t, service)
	assert.NotNil(t, service.fqdn)
	assert.NotNil(t, service.cert)
	assert.NotNil(t, service.configService)
}
