package mail

import (
	"github.com/cloudogu/ces-importer/configuration"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"net/smtp"
	"testing"
	"time"
)

func TestSender(t *testing.T) {
	t.Run("create auth if username and password are set", func(t *testing.T) {
		config := configuration.SmtpConfig{
			Username: "user",
			Password: "pw",
		}
		sender := CreateSender(config, []string{})

		auth := sender.auth()
		assert.NotNil(t, auth)
	})

	t.Run("return nil auth if empty", func(t *testing.T) {
		config := configuration.SmtpConfig{}
		sender := CreateSender(config, []string{})

		auth := sender.auth()
		assert.Nil(t, auth)
	})

	t.Run("will create server address", func(t *testing.T) {
		config := configuration.SmtpConfig{
			Port:   "123",
			Server: "server",
		}
		sender := CreateSender(config, []string{})

		assert.Equal(t, "server:123", sender.server())
	})

	t.Run("will create subject", func(t *testing.T) {
		config := configuration.SmtpConfig{}
		sender := CreateSender(config, []string{})

		successSubject := sender.subject(true)
		failureSubject := sender.subject(false)

		assert.Equal(t, "Subject: Migration war erfolgreich.\r\n", successSubject)
		assert.Equal(t, "Subject: Migration war nicht erfolgreich.\r\n", failureSubject)
	})

	t.Run("will create body", func(t *testing.T) {
		config := configuration.SmtpConfig{}
		sender := CreateSender(config, []string{})

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
	t.Run("send migration result", func(t *testing.T) {
		config := configuration.SmtpConfig{
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
		sender := CreateSender(config, []string{"a", "b"})

		sender.senderService = func(addr string, a smtp.Auth, from string, to []string, msg []byte) error {
			return senderFunc.Execute(addr, a, from, to, msg)
		}

		sender.readFile = func(name string) ([]byte, error) {
			return []byte("filecontent"), nil
		}

		timeA, _ := time.Parse("15:04", "13:01")
		timeB := timeA.Add(5 * time.Minute)

		err := sender.Send(true, nil, "source", "target", timeA, timeB)
		require.NoError(t, err)
	})
}
