package mail

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"net/smtp"
	"os"
	"testing"
	"time"
)

func TestSmtpConfigFromEnv(t *testing.T) {
	t.Run("can get config from env", func(t *testing.T) {
		_ = os.Setenv(envSmtpServer, "server")
		_ = os.Setenv(envSmtpPort, "port")
		_ = os.Setenv(envSmtpUsername, "username")
		_ = os.Setenv(envSmtpPassword, "password")
		_ = os.Setenv(envSmtpFrom, "from")
		_ = os.Setenv(envSmtpTo, "to")

		config, err := SmtpConfigFromEnv()
		require.NoError(t, err)
		assert.Equal(t, "server", config.Server)
		assert.Equal(t, "port", config.Port)
		assert.Equal(t, "username", config.Username)
		assert.Equal(t, "password", config.Password)
		assert.Equal(t, "from", config.From)
		assert.Equal(t, []string{"to"}, config.To)
	})

	t.Run("fail on unset server", func(t *testing.T) {
		_ = os.Unsetenv(envSmtpServer)
		_ = os.Setenv(envSmtpPort, "port")
		_ = os.Setenv(envSmtpUsername, "username")
		_ = os.Setenv(envSmtpPassword, "password")
		_ = os.Setenv(envSmtpFrom, "from")
		_ = os.Setenv(envSmtpTo, "to")

		_, err := SmtpConfigFromEnv()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "smtp Server address is not configured")
	})

	t.Run("fallback to 25 on unset port", func(t *testing.T) {
		_ = os.Setenv(envSmtpServer, "server")
		_ = os.Unsetenv(envSmtpPort)
		_ = os.Setenv(envSmtpUsername, "username")
		_ = os.Setenv(envSmtpPassword, "password")
		_ = os.Setenv(envSmtpFrom, "from")
		_ = os.Setenv(envSmtpTo, "to")

		config, err := SmtpConfigFromEnv()
		require.NoError(t, err)
		assert.Equal(t, "25", config.Port)
	})

	t.Run("fail on unset from", func(t *testing.T) {
		_ = os.Setenv(envSmtpServer, "server")
		_ = os.Setenv(envSmtpPort, "port")
		_ = os.Setenv(envSmtpUsername, "username")
		_ = os.Setenv(envSmtpPassword, "password")
		_ = os.Unsetenv(envSmtpFrom)
		_ = os.Setenv(envSmtpTo, "to")

		_, err := SmtpConfigFromEnv()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "smtp from is not configured")
	})
}

func TestSender(t *testing.T) {
	t.Run("create auth if username and password are set", func(t *testing.T) {
		config := SmtpConfig{
			Username: "user",
			Password: "pw",
		}
		sender := CreateSender(
			config,
			func(addr string, a smtp.Auth, from string, to []string, msg []byte) error {
				return nil
			},
			func(name string) ([]byte, error) {
				return []byte(""), nil
			})

		auth := sender.auth()
		assert.NotNil(t, auth)
	})

	t.Run("return nil auth if empty", func(t *testing.T) {
		config := SmtpConfig{}
		sender := CreateSender(
			config,
			func(addr string, a smtp.Auth, from string, to []string, msg []byte) error {
				return nil
			},
			func(name string) ([]byte, error) {
				return []byte(""), nil
			})

		auth := sender.auth()
		assert.Nil(t, auth)
	})

	t.Run("will create server address", func(t *testing.T) {
		config := SmtpConfig{
			Port:   "123",
			Server: "server",
		}
		sender := CreateSender(
			config,
			func(addr string, a smtp.Auth, from string, to []string, msg []byte) error {
				return nil
			},
			func(name string) ([]byte, error) {
				return []byte(""), nil
			})

		assert.Equal(t, "server:123", sender.server())
	})

	t.Run("will create subject", func(t *testing.T) {
		config := SmtpConfig{}
		sender := CreateSender(
			config,
			func(addr string, a smtp.Auth, from string, to []string, msg []byte) error {
				return nil
			},
			func(name string) ([]byte, error) {
				return []byte(""), nil
			})

		successSubject := sender.subject(true)
		failureSubject := sender.subject(false)

		assert.Equal(t, "Subject: Migration war erfolgreich.\r\n", successSubject)
		assert.Equal(t, "Subject: Migration war nicht erfolgreich.\r\n", failureSubject)
	})

	t.Run("will create body", func(t *testing.T) {
		config := SmtpConfig{}
		sender := CreateSender(
			config,
			func(addr string, a smtp.Auth, from string, to []string, msg []byte) error {
				return nil
			},
			func(name string) ([]byte, error) {
				return []byte(""), nil
			})

		timeA, _ := time.Parse("15:04", "13:01")
		timeB := timeA.Add(5 * time.Minute)

		successFinal := sender.body(true, "instance-a", "instance-b", timeA, timeB, true)
		failureFinal := sender.body(false, "instance-a", "instance-b", timeA, timeB, true)
		successNonFinal := sender.body(true, "instance-a", "instance-b", timeA, timeB, true)
		failureNonFinal := sender.body(false, "instance-a", "instance-b", timeA, timeB, true)

		assert.Contains(t, successFinal, "Die finale Migration von der Instanz instance-a zu der Instanz instance-b war erfolgreich.\n\nStartzeitpunkt: 13:01\nEndzeitpunkt: 13:06\n\nAlle weiteren Informationen finden Sie in der Log-Datei im Anhang.\r\n")
		assert.Contains(t, failureFinal, "Die finale Migration von der Instanz instance-a zu der Instanz instance-b war nicht erfolgreich.\n\nStartzeitpunkt: 13:01\nEndzeitpunkt: 13:06\n\nAlle weiteren Informationen finden Sie in der Log-Datei im Anhang.\r\n")
		assert.Contains(t, successNonFinal, "Die finale Migration von der Instanz instance-a zu der Instanz instance-b war erfolgreich.\n\nStartzeitpunkt: 13:01\nEndzeitpunkt: 13:06\n\nAlle weiteren Informationen finden Sie in der Log-Datei im Anhang.\r\n")
		assert.Contains(t, failureNonFinal, "Die finale Migration von der Instanz instance-a zu der Instanz instance-b war nicht erfolgreich.\n\nStartzeitpunkt: 13:01\nEndzeitpunkt: 13:06\n\nAlle weiteren Informationen finden Sie in der Log-Datei im Anhang.\r\n")
	})
}

func TestSendMigrationResult(t *testing.T) {
	t.Run("asdf", func(t *testing.T) {
		config := SmtpConfig{
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
			assert.Contains(t, body, "Die finale Migration von der Instanz  zu der Instanz  war erfolgreich.\n\nStartzeitpunkt: 13:01\nEndzeitpunkt: 13:06\n\nAlle weiteren Informationen finden Sie in der Log-Datei im Anhang.\r\n\r\n\r\n")
			assert.Contains(t, body, "--MIME_BOUNDARY_CES_IMPORTER\r\nContent-Type: application/octet-stream\r\nContent-Transfer-Encoding: base64\r\nContent-Disposition: attachment; filename=\"a\"\r\n\r\nZmlsZWNvbnRlbnQ=\r\n\r\n--MIME_BOUNDARY_CES_IMPORTER\r\nContent-Type: application/octet-stream\r\nContent-Transfer-Encoding: base64\r\nContent-Disposition: attachment; filename=\"b\"\r\n\r\nZmlsZWNvbnRlbnQ=\r\n--MIME_BOUNDARY_CES_IMPORTER")
		}).Return(nil)
		sender := CreateSender(
			config,
			func(addr string, a smtp.Auth, from string, to []string, msg []byte) error {
				return senderFunc.Execute(addr, a, from, to, msg)
			},
			func(name string) ([]byte, error) {
				return []byte("filecontent"), nil
			})

		timeA, _ := time.Parse("15:04", "13:01")
		timeB := timeA.Add(5 * time.Minute)

		err := sender.SendMigrationResult(true, []string{"a", "b"}, "", "", timeA, timeB, true)
		require.NoError(t, err)
	})
}
