package mail

import (
	"context"
	"fmt"
	"github.com/cloudogu/ces-importer/configuration"
	regConfig "github.com/cloudogu/k8s-registry-lib/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"net/smtp"
	"testing"
	"time"
)

func TestSender(t *testing.T) {
	t.Run("create auth if username and password are set", func(t *testing.T) {
		config := configuration.Smtp{
			Username: "user",
			Password: "pw",
		}

		mGlobalConfigRepo := newMockGlobalConfigRepo(t)

		sender := CreateSender(config, "source", []string{}, mGlobalConfigRepo)

		auth := sender.auth()
		assert.NotNil(t, auth)
	})

	t.Run("return nil auth if empty", func(t *testing.T) {
		config := configuration.Smtp{}
		mGlobalConfigRepo := newMockGlobalConfigRepo(t)

		sender := CreateSender(config, "source", []string{}, mGlobalConfigRepo)

		auth := sender.auth()
		assert.Nil(t, auth)
	})

	t.Run("will create server address", func(t *testing.T) {
		config := configuration.Smtp{
			Port:   "123",
			Server: "server",
		}
		mGlobalConfigRepo := newMockGlobalConfigRepo(t)

		sender := CreateSender(config, "source", []string{}, mGlobalConfigRepo)

		assert.Equal(t, "server:123", sender.server())
	})

	t.Run("will create subject", func(t *testing.T) {
		config := configuration.Smtp{}
		mGlobalConfigRepo := newMockGlobalConfigRepo(t)

		sender := CreateSender(config, "source", []string{}, mGlobalConfigRepo)

		successSubject := sender.subject(true)
		failureSubject := sender.subject(false)

		assert.Equal(t, "Subject: Migration war erfolgreich.\r\n", successSubject)
		assert.Equal(t, "Subject: Migration war nicht erfolgreich.\r\n", failureSubject)
	})

	t.Run("will create body", func(t *testing.T) {
		config := configuration.Smtp{}
		mGlobalConfigRepo := newMockGlobalConfigRepo(t)

		sender := CreateSender(config, "source", []string{}, mGlobalConfigRepo)

		timeA, _ := time.Parse("15:04", "13:01")
		timeB := timeA.Add(5 * time.Minute)

		successFinal := sender.body(nil, "instance-a", "instance-b", timeA, timeB, true)
		failureFinal := sender.body(fmt.Errorf("error"), "instance-a", "instance-b", timeA, timeB, true)
		successNonFinal := sender.body(nil, "instance-a", "instance-b", timeA, timeB, true)
		failureNonFinal := sender.body(fmt.Errorf("error"), "instance-a", "instance-b", timeA, timeB, true)

		assert.Contains(t, successFinal, "Die finale Migration von der Instanz instance-a zu der Instanz instance-b war erfolgreich.\n\nStartzeitpunkt: 13:01\nEndzeitpunkt: 13:06\n\nAlle weiteren Informationen finden Sie in der Log-Datei im Anhang.\r\n")
		assert.Contains(t, failureFinal, "Die finale Migration von der Instanz instance-a zu der Instanz instance-b war nicht erfolgreich.\n\nStartzeitpunkt: 13:01\nEndzeitpunkt: 13:06\n\nDie Fehlermeldung ist: error\n\nAlle weiteren Informationen finden Sie in der Log-Datei im Anhang.\r\n")
		assert.Contains(t, successNonFinal, "Die finale Migration von der Instanz instance-a zu der Instanz instance-b war erfolgreich.\n\nStartzeitpunkt: 13:01\nEndzeitpunkt: 13:06\n\nAlle weiteren Informationen finden Sie in der Log-Datei im Anhang.\r\n")
		assert.Contains(t, failureNonFinal, "Die finale Migration von der Instanz instance-a zu der Instanz instance-b war nicht erfolgreich.\n\nStartzeitpunkt: 13:01\nEndzeitpunkt: 13:06\n\nDie Fehlermeldung ist: error\n\nAlle weiteren Informationen finden Sie in der Log-Datei im Anhang.\r\n")
	})
}

func TestSendMigrationResult(t *testing.T) {
	t.Run("send migration result", func(t *testing.T) {
		config := configuration.Smtp{
			Server:   "server",
			Port:     "port",
			Username: "username",
			Password: "password",
			From:     "from",
			To: []string{
				"a@test.de",
				"b@test.de",
			},
		}
		senderFunc := NewMockSenderService(t)
		senderFunc.EXPECT().Execute("server:port", mock.Anything, "from", []string{"a@test.de", "b@test.de"}, mock.Anything).Run(func(addr string, a smtp.Auth, from string, to []string, msg []byte) {
			body := string(msg)
			assert.Contains(t, body, "From: from\r\nSubject: Migration war erfolgreich.\r\nMIME-Version: 1.0\r\nContent-Type: multipart/mixed; boundary=MIME_BOUNDARY_CES_IMPORTER\r\n\r\n--MIME_BOUNDARY_CES_IMPORTER\r\nContent-Type: text/plain; charset=utf-8\r\n\r\n")
			assert.Contains(t, body, "Die finale Migration von der Instanz source zu der Instanz target war erfolgreich.\n\nStartzeitpunkt: 13:01\nEndzeitpunkt: 13:06\n\nAlle weiteren Informationen finden Sie in der Log-Datei im Anhang.\r\n\r\n")
			assert.Contains(t, body, "--MIME_BOUNDARY_CES_IMPORTER\r\nContent-Type: application/octet-stream\r\nContent-Transfer-Encoding: base64\r\nContent-Disposition: attachment; filename=\"b\"\r\n\r\nZmlsZWNvbnRlbnQ=\r\n--MIME_BOUNDARY_CES_IMPORTER")
		}).Return(nil)

		globalConfig := regConfig.CreateGlobalConfig(map[regConfig.Key]regConfig.Value{
			"fqdn": "target",
		})

		mGlobalConfigRepo := newMockGlobalConfigRepo(t)
		mGlobalConfigRepo.EXPECT().Get(context.Background()).Return(globalConfig, nil)
		sender := CreateSender(config, "source", []string{"a", "b"}, mGlobalConfigRepo)

		sender.senderService = func(addr string, a smtp.Auth, from string, to []string, msg []byte) error {
			return senderFunc.Execute(addr, a, from, to, msg)
		}

		sender.readFile = func(name string) ([]byte, error) {
			return []byte("filecontent"), nil
		}

		timeA, _ := time.Parse("15:04", "13:01")
		timeB := timeA.Add(5 * time.Minute)

		err := sender.Send(context.Background(), true, nil, timeA, timeB)
		require.NoError(t, err)
	})
}

func TestGetTargetInstance(t *testing.T) {
	tests := []struct {
		name       string
		mockConfig func(t *testing.T) globalConfigRepo
		expectFqdn string
		expectErr  bool
	}{
		{
			name: "successful fqdn retrieval",
			mockConfig: func(t *testing.T) globalConfigRepo {
				mockRepo := newMockGlobalConfigRepo(t)
				mockRepo.EXPECT().Get(context.Background()).Return(regConfig.CreateGlobalConfig(map[regConfig.Key]regConfig.Value{
					GLOBAL_CONFIG_FQDN_KEY: "target-instance",
				}), nil)
				return mockRepo
			},
			expectFqdn: "target-instance",
			expectErr:  false,
		},
		{
			name: "missing fqdn key",
			mockConfig: func(t *testing.T) globalConfigRepo {
				mockRepo := newMockGlobalConfigRepo(t)
				mockRepo.EXPECT().Get(context.Background()).Return(regConfig.CreateGlobalConfig(map[regConfig.Key]regConfig.Value{}), nil)
				return mockRepo
			},
			expectFqdn: "",
			expectErr:  true,
		},
		{
			name: "error fetching global config",
			mockConfig: func(t *testing.T) globalConfigRepo {
				mockRepo := newMockGlobalConfigRepo(t)
				mockRepo.EXPECT().Get(context.Background()).Return(regConfig.GlobalConfig{}, fmt.Errorf("global config error"))
				return mockRepo
			},
			expectFqdn: "",
			expectErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mGlobalConfigRepo := tt.mockConfig(t)

			sender := &Sender{
				globalConfigRepo: mGlobalConfigRepo,
			}

			fqdn, err := sender.getTargetInstance(context.Background())

			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectFqdn, fqdn)
			}
		})
	}
}
