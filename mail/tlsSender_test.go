package mail

import (
	"net/smtp"
	"os"
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

		config := configuration.Smtp{SkipTLSVerify: true}
		tlsconfig, _ := createTLSConfig("localhost", true)

		mockSMTPClientFactory := NewMockSMTPClientFactory(t)
		mockSMTPClientFactory.EXPECT().NewTLSClient(addr, tlsconfig).Return(nil, assert.AnError)

		ts := &tlsSender{config: config, factory: mockSMTPClientFactory}

		err := ts.sendMailWithTls(addr, a, from, to, msg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create smtp tls client")
	})
}

func Test_sendMailWithStarttls(t *testing.T) {
	t.Run("should fail to send mail with StartTLS", func(t *testing.T) {
		addr := "localhost:1"
		var a smtp.Auth = nil
		from := "from@example.com"
		to := []string{"to@example.com"}
		msg := []byte("Subject: test\r\n\r\nbody")

		config := configuration.Smtp{SkipTLSVerify: true}

		mockSMTPClientFactory := NewMockSMTPClientFactory(t)
		mockSMTPClientFactory.EXPECT().NewClient(addr).Return(nil, assert.AnError)

		ts := &tlsSender{config: config, factory: mockSMTPClientFactory}

		err := ts.sendMailWithStartTLS(addr, a, from, to, msg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create smtp client")
	})
	t.Run("should fail to send mail with StartTLS on register server", func(t *testing.T) {
		addr := "localhost:1"
		var a smtp.Auth = nil
		from := "from@example.com"
		to := []string{"to@example.com"}
		msg := []byte("Subject: test\r\n\r\nbody")

		config := configuration.Smtp{SkipTLSVerify: true}

		mockSMTPClientFactory := NewMockSMTPClientFactory(t)
		mockSMTPClient := NewMockSMTPClient(t)
		mockSMTPClientFactory.EXPECT().NewClient(addr).Return(mockSMTPClient, nil)
		ts := &tlsSender{config: config, factory: mockSMTPClientFactory}

		hostname, _ := os.Hostname()
		mockSMTPClient.EXPECT().Hello(hostname).Return(assert.AnError)
		mockSMTPClient.EXPECT().Quit().Return(nil)

		err := ts.sendMailWithStartTLS(addr, a, from, to, msg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to register conn")
	})
	t.Run("should fail to send mail with StartTLS on get starttls extensions", func(t *testing.T) {
		addr := "localhost:1"
		var a smtp.Auth = nil
		from := "from@example.com"
		to := []string{"to@example.com"}
		msg := []byte("Subject: test\r\n\r\nbody")

		config := configuration.Smtp{SkipTLSVerify: true}

		mockSMTPClientFactory := NewMockSMTPClientFactory(t)
		mockSMTPClient := NewMockSMTPClient(t)
		mockSMTPClientFactory.EXPECT().NewClient(addr).Return(mockSMTPClient, nil)
		ts := &tlsSender{config: config, factory: mockSMTPClientFactory}

		hostname, _ := os.Hostname()
		mockSMTPClient.EXPECT().Hello(hostname).Return(nil)

		mockSMTPClient.EXPECT().Extension("STARTTLS").Return(false, "")

		mockSMTPClient.EXPECT().Quit().Return(nil)

		err := ts.sendMailWithStartTLS(addr, a, from, to, msg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "SMTP server does not support STARTTLS")
	})
	t.Run("should fail to send mail with StartTLS on starttls", func(t *testing.T) {
		addr := "localhost:1"
		var a smtp.Auth = nil
		from := "from@example.com"
		to := []string{"to@example.com"}
		msg := []byte("Subject: test\r\n\r\nbody")

		config := configuration.Smtp{SkipTLSVerify: true}

		mockSMTPClientFactory := NewMockSMTPClientFactory(t)
		mockSMTPClient := NewMockSMTPClient(t)
		mockSMTPClientFactory.EXPECT().NewClient(addr).Return(mockSMTPClient, nil)
		ts := &tlsSender{config: config, factory: mockSMTPClientFactory}
		tlsconfig, _ := createTLSConfig("localhost", true)

		hostname, _ := os.Hostname()
		mockSMTPClient.EXPECT().Hello(hostname).Return(nil)

		mockSMTPClient.EXPECT().Extension("STARTTLS").Return(true, "")

		mockSMTPClient.EXPECT().StartTLS(tlsconfig).Return(assert.AnError)

		mockSMTPClient.EXPECT().Quit().Return(nil)

		err := ts.sendMailWithStartTLS(addr, a, from, to, msg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to init starttls")
	})
	t.Run("should fail to send mail with StartTLS on mail", func(t *testing.T) {
		addr := "localhost:1"
		var a smtp.Auth = nil
		from := "from@example.com"
		to := []string{"to@example.com"}
		msg := []byte("Subject: test\r\n\r\nbody")

		config := configuration.Smtp{SkipTLSVerify: true}

		mockSMTPClientFactory := NewMockSMTPClientFactory(t)
		mockSMTPClient := NewMockSMTPClient(t)
		mockSMTPClientFactory.EXPECT().NewClient(addr).Return(mockSMTPClient, nil)
		ts := &tlsSender{config: config, factory: mockSMTPClientFactory}
		tlsconfig, _ := createTLSConfig("localhost", true)

		hostname, _ := os.Hostname()
		mockSMTPClient.EXPECT().Hello(hostname).Return(nil)

		mockSMTPClient.EXPECT().Extension("STARTTLS").Return(true, "")

		mockSMTPClient.EXPECT().StartTLS(tlsconfig).Return(nil)

		mockSMTPClient.EXPECT().Mail(from).Return(assert.AnError)

		mockSMTPClient.EXPECT().Quit().Return(nil)

		err := ts.sendMailWithStartTLS(addr, a, from, to, msg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to set sender")
	})
	t.Run("should fail to send mail with StartTLS on setting recipients", func(t *testing.T) {
		addr := "localhost:1"
		var a smtp.Auth = nil
		from := "from@example.com"
		to := []string{"to@example.com"}
		msg := []byte("Subject: test\r\n\r\nbody")

		config := configuration.Smtp{SkipTLSVerify: true}

		mockSMTPClientFactory := NewMockSMTPClientFactory(t)
		mockSMTPClient := NewMockSMTPClient(t)
		mockSMTPClientFactory.EXPECT().NewClient(addr).Return(mockSMTPClient, nil)
		ts := &tlsSender{config: config, factory: mockSMTPClientFactory}
		tlsconfig, _ := createTLSConfig("localhost", true)

		hostname, _ := os.Hostname()
		mockSMTPClient.EXPECT().Hello(hostname).Return(nil)

		mockSMTPClient.EXPECT().Extension("STARTTLS").Return(true, "")

		mockSMTPClient.EXPECT().StartTLS(tlsconfig).Return(nil)

		mockSMTPClient.EXPECT().Mail(from).Return(nil)

		mockSMTPClient.EXPECT().Quit().Return(nil)

		for _, tor := range to {
			mockSMTPClient.EXPECT().Rcpt(tor).Return(assert.AnError)
		}

		err := ts.sendMailWithStartTLS(addr, a, from, to, msg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to set recipient to@example.com")
	})
	t.Run("should fail to send mail with StartTLS on setting data", func(t *testing.T) {
		addr := "localhost:1"
		var a smtp.Auth = nil
		from := "from@example.com"
		to := []string{"to@example.com"}
		msg := []byte("Subject: test\r\n\r\nbody")

		config := configuration.Smtp{SkipTLSVerify: true}

		mockSMTPClientFactory := NewMockSMTPClientFactory(t)
		mockSMTPClient := NewMockSMTPClient(t)
		mockSMTPClientFactory.EXPECT().NewClient(addr).Return(mockSMTPClient, nil)
		ts := &tlsSender{config: config, factory: mockSMTPClientFactory}
		tlsconfig, _ := createTLSConfig("localhost", true)

		hostname, _ := os.Hostname()
		mockSMTPClient.EXPECT().Hello(hostname).Return(nil)

		mockSMTPClient.EXPECT().Extension("STARTTLS").Return(true, "")

		mockSMTPClient.EXPECT().StartTLS(tlsconfig).Return(nil)

		mockSMTPClient.EXPECT().Mail(from).Return(nil)

		mockSMTPClient.EXPECT().Quit().Return(nil)

		for _, tor := range to {
			mockSMTPClient.EXPECT().Rcpt(tor).Return(nil)
		}

		mockSMTPClient.EXPECT().Data().Return(nil, assert.AnError)

		err := ts.sendMailWithStartTLS(addr, a, from, to, msg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get writer from mail server")
	})
	t.Run("should fail to send mail with StartTLS on setting data", func(t *testing.T) {
		addr := "localhost:1"
		var a smtp.Auth = nil
		from := "from@example.com"
		to := []string{"to@example.com"}
		msg := []byte("Subject: test\r\n\r\nbody")

		config := configuration.Smtp{SkipTLSVerify: true}

		mockSMTPClientFactory := NewMockSMTPClientFactory(t)
		mockSMTPClient := NewMockSMTPClient(t)
		mockSMTPClientFactory.EXPECT().NewClient(addr).Return(mockSMTPClient, nil)
		ts := &tlsSender{config: config, factory: mockSMTPClientFactory}
		tlsconfig, _ := createTLSConfig("localhost", true)

		hostname, _ := os.Hostname()
		mockSMTPClient.EXPECT().Hello(hostname).Return(nil)

		mockSMTPClient.EXPECT().Extension("STARTTLS").Return(true, "")

		mockSMTPClient.EXPECT().StartTLS(tlsconfig).Return(nil)

		mockSMTPClient.EXPECT().Mail(from).Return(nil)

		mockSMTPClient.EXPECT().Quit().Return(nil)

		for _, tor := range to {
			mockSMTPClient.EXPECT().Rcpt(tor).Return(nil)
		}

		mockSMTPClient.EXPECT().Data().Return(nil, assert.AnError)

		err := ts.sendMailWithStartTLS(addr, a, from, to, msg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get writer from mail server")
	})
}
