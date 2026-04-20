package mail

import (
	"net/smtp"
	"testing"

	"github.com/cloudogu/ces-importer/configuration"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_createTLSConfig(t *testing.T) {
	t.Run("should create TLS config", func(t *testing.T) {
		config, err := createTLSConfig("myServer", true)

		require.NoError(t, err)

		assert.Equal(t, "myServer", config.ServerName)
		assert.True(t, config.InsecureSkipVerify)
		assert.Nil(t, config.RootCAs)
	})

	t.Run("should create TLS config", func(t *testing.T) {
		origiCaPath := customCAPath
		defer func() {
			customCAPath = origiCaPath
		}()
		customCAPath = "../testdata/api/test.crt"

		config, err := createTLSConfig("myServer", false)

		require.NoError(t, err)

		assert.Equal(t, "myServer", config.ServerName)
		assert.False(t, config.InsecureSkipVerify)
		assert.NotNil(t, config.RootCAs)
	})
}

func Test_sendMailWithTls(t *testing.T) {
	t.Run("should fail to send mail with TLS", func(t *testing.T) {
		addr := "localhost:1"
		var a smtp.Auth = nil
		from := "from@example.com"
		to := []string{"to@example.com"}
		msg := []byte("Subject: test\r\n\r\nbody")

		ts := &tlsSender{config: configuration.Smtp{SkipTLSVerify: true}}

		err := ts.sendMailWithTls(addr, a, from, to, msg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to connect to mail server")
	})
}

func Test_sendMailWithStarttls(t *testing.T) {
	t.Run("should fail to send mail with StartTLS", func(t *testing.T) {
		addr := "localhost:1"
		var a smtp.Auth = nil
		from := "from@example.com"
		to := []string{"to@example.com"}
		msg := []byte("Subject: test\r\n\r\nbody")

		ts := &tlsSender{config: configuration.Smtp{SkipTLSVerify: true}}

		err := ts.sendMailWithStartTLS(addr, a, from, to, msg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "connect: connection refused")
	})
}
