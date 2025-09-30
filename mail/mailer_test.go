package mail

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"mime/multipart"
	"net/smtp"
	"strings"
	"testing"
	"time"

	"github.com/cloudogu/ces-importer/configuration"
	regConfig "github.com/cloudogu/k8s-registry-lib/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
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
			Port:   123,
			Server: "server",
		}
		mGlobalConfigRepo := newMockGlobalConfigRepo(t)

		sender := CreateSender(config, "source", []string{}, mGlobalConfigRepo)

		assert.Equal(t, "server:123", sender.server())
	})

	t.Run("will build subject", func(t *testing.T) {
		successSubject := buildSubject(true, "source")
		failureSubject := buildSubject(false, "source")

		assert.Equal(t, "Die Migration der Instanz source war erfolgreich.", successSubject)
		assert.Equal(t, "Die Migration der Instanz source war nicht erfolgreich.", failureSubject)
	})
}

func TestSendMigrationResult(t *testing.T) {
	t.Run("send migration result", func(t *testing.T) {
		config := configuration.Smtp{
			Server:   "server",
			Port:     25,
			Username: "username",
			Password: "password",
			From:     "from",
			To: []string{
				"a@test.de",
				"b@test.de",
			},
		}
		senderFunc := NewMockSenderService(t)
		senderFunc.EXPECT().Execute("server:25", mock.Anything, "from", []string{"a@test.de", "b@test.de"}, mock.Anything).Run(func(addr string, a smtp.Auth, from string, to []string, msg []byte) {
			body := string(msg)
			assert.Contains(t, body, "Subject: Die Migration der Instanz source war erfolgreich.\r\n")
			assert.Contains(t, body, "MIME-Version: 1.0\r\n")
			assert.Contains(t, body, "Content-Type: multipart/mixed; boundary=")
			assert.Contains(t, body, "Die finale Migration von der Instanz https://source zu der Instanz https://=\r\ntarget war erfolgreich.\r\n\r\nStartzeitpunkt: 01.01.0000 13:01 (UTC +0000)\r\nEndzeitpunkt: 01.01.0000 13:06 (UTC +0000)\r\n\r\nAlle weiteren Informationen finden Sie in der Log-Datei im Anhang.")
			assert.Contains(t, body, "Content-Disposition: attachment; filename=\"a\"\r\nContent-Transfer-Encoding: base64\r\nContent-Type: application/octet-stream\r\n\r\n")
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

	t.Run("should send migration result with unauthenticated server", func(t *testing.T) {
		config := configuration.Smtp{
			Server: "server",
			Port:   25,
			From:   "from",
			To: []string{
				"a@test.de",
				"b@test.de",
			},
		}
		senderFunc := NewMockSenderService(t)
		senderFunc.EXPECT().Execute("server:25", nil, "from", []string{"a@test.de", "b@test.de"}, mock.Anything).Return(nil)

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

	t.Run("should fail to send migration result for error getting target-instance", func(t *testing.T) {
		config := configuration.Smtp{
			Server:   "server",
			Port:     25,
			Username: "username",
			Password: "password",
			From:     "from",
			To: []string{
				"a@test.de",
				"b@test.de",
			},
		}
		senderFunc := NewMockSenderService(t)

		globalConfig := regConfig.CreateGlobalConfig(map[regConfig.Key]regConfig.Value{
			"fqdn": "target",
		})

		mGlobalConfigRepo := newMockGlobalConfigRepo(t)
		mGlobalConfigRepo.EXPECT().Get(context.Background()).Return(globalConfig, assert.AnError)
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

		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to get target instance")
	})

	t.Run("should fail to send migration result for error getting target-instance while writing body-text", func(t *testing.T) {
		config := configuration.Smtp{
			Server:   "server",
			Port:     25,
			Username: "username",
			Password: "password",
			From:     "from",
			To: []string{
				"a@test.de",
				"b@test.de",
			},
		}
		senderFunc := NewMockSenderService(t)

		globalConfig := regConfig.CreateGlobalConfig(map[regConfig.Key]regConfig.Value{
			"fqdn": "target",
		})

		counter := 0
		mGlobalConfigRepo := newMockGlobalConfigRepo(t)
		mGlobalConfigRepo.EXPECT().Get(context.Background()).RunAndReturn(func(ctx context.Context) (regConfig.GlobalConfig, error) {
			counter++
			if counter <= 1 {
				return globalConfig, nil
			}

			return globalConfig, assert.AnError
		})
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

		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to write body text: failed to get target instance")
	})

	t.Run("should not fail to send migration result for error reading attachment-file", func(t *testing.T) {
		config := configuration.Smtp{
			Server:   "server",
			Port:     25,
			Username: "username",
			Password: "password",
			From:     "from",
			To: []string{
				"a@test.de",
				"b@test.de",
			},
		}
		senderFunc := NewMockSenderService(t)
		senderFunc.EXPECT().Execute("server:25", mock.Anything, "from", []string{"a@test.de", "b@test.de"}, mock.Anything).Return(nil)

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
			return []byte("filecontent"), assert.AnError
		}

		timeA, _ := time.Parse("15:04", "13:01")
		timeB := timeA.Add(5 * time.Minute)

		err := sender.Send(context.Background(), true, nil, timeA, timeB)

		require.NoError(t, err)
	})

	t.Run("should not send mail when server not configured", func(t *testing.T) {
		originalLogger := slog.Default()
		sb := new(strings.Builder)
		testLogger := slog.New(slog.NewTextHandler(sb, nil))
		slog.SetDefault(testLogger)

		defer func() {
			slog.SetDefault(originalLogger)
		}()

		config := configuration.Smtp{
			Port: 25,
		}
		sender := CreateSender(config, "source", []string{"a", "b"}, nil)

		timeA, _ := time.Parse("15:04", "13:01")
		timeB := timeA.Add(5 * time.Minute)
		err := sender.Send(context.Background(), true, nil, timeA, timeB)

		require.NoError(t, err)
		assert.Contains(t, sb.String(), "SMTP server not configured. Not sending mail.")
	})

	t.Run("should not send mail when port not configured", func(t *testing.T) {
		originalLogger := slog.Default()
		sb := new(strings.Builder)
		testLogger := slog.New(slog.NewTextHandler(sb, nil))
		slog.SetDefault(testLogger)

		defer func() {
			slog.SetDefault(originalLogger)
		}()

		config := configuration.Smtp{
			Server: "myserver",
		}
		sender := CreateSender(config, "source", []string{"a", "b"}, nil)

		timeA, _ := time.Parse("15:04", "13:01")
		timeB := timeA.Add(5 * time.Minute)
		err := sender.Send(context.Background(), true, nil, timeA, timeB)

		require.NoError(t, err)
		assert.Contains(t, sb.String(), "SMTP server not configured. Not sending mail.")
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

func TestSender_writeBodyText(t *testing.T) {
	testCtx := context.Background()

	t.Run("write body text for successful final migration", func(t *testing.T) {
		config := configuration.Smtp{}
		mGlobalConfigRepo := newMockGlobalConfigRepo(t)
		mGlobalConfigRepo.EXPECT().Get(testCtx).Return(
			regConfig.CreateGlobalConfig(map[regConfig.Key]regConfig.Value{
				GLOBAL_CONFIG_FQDN_KEY: "instance-b",
			}), nil,
		)

		sender := CreateSender(config, "instance-a", []string{}, mGlobalConfigRepo)

		timeA, _ := time.Parse("15:04", "13:01")
		timeB := timeA.Add(5 * time.Minute)

		var body bytes.Buffer
		multipartWriter := multipart.NewWriter(&body)

		err := sender.writeBodyText(testCtx, multipartWriter, nil, timeA, timeB, true)

		require.NoError(t, err)

		assert.Contains(t, body.String(), "Die finale Migration von der Instanz https://instance-a zu der Instanz http=\r\ns://instance-b war erfolgreich.\r\n\r\nStartzeitpunkt: 01.01.0000 13:01 (UTC +0000)\r\nEndzeitpunkt: 01.01.0000 13:06 (UTC +0000)\r\n\r\nAlle weiteren Informationen finden Sie in der Log-Datei im Anhang.")
	})

	t.Run("write body text for delta migration with error", func(t *testing.T) {
		config := configuration.Smtp{}
		mGlobalConfigRepo := newMockGlobalConfigRepo(t)
		mGlobalConfigRepo.EXPECT().Get(testCtx).Return(
			regConfig.CreateGlobalConfig(map[regConfig.Key]regConfig.Value{
				GLOBAL_CONFIG_FQDN_KEY: "instance-b",
			}), nil,
		)

		sender := CreateSender(config, "instance-a", []string{}, mGlobalConfigRepo)

		timeA, _ := time.Parse("15:04", "13:01")
		timeB := timeA.Add(5 * time.Minute)

		var body bytes.Buffer
		multipartWriter := multipart.NewWriter(&body)

		err := sender.writeBodyText(testCtx, multipartWriter, fmt.Errorf("migration-error"), timeA, timeB, false)

		require.NoError(t, err)

		assert.Contains(t, body.String(), "Die Delta-Migration von der Instanz https://instance-a zu der Instanz https=\r\n://instance-b war nicht erfolgreich.\r\n\r\nStartzeitpunkt: 01.01.0000 13:01 (UTC +0000)\r\nEndzeitpunkt: 01.01.0000 13:06 (UTC +0000)\r\n\r\nDie Fehlermeldung ist: migration-error\r\n\r\nAlle weiteren Informationen finden Sie in der Log-Datei im Anhang.")
	})
}
