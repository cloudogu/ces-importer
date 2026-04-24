package mail

import (
	"crypto/x509"
	"fmt"
	"net/smtp"
	"os"
	"path/filepath"
	"testing"

	"github.com/cloudogu/ces-importer/configuration"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockWriter struct {
	data       []byte
	closed     bool
	faileClose bool
}

func (m *mockWriter) Write(p []byte) (int, error) {
	if m.closed {
		return 0, fmt.Errorf("write on closed writer")
	}
	m.data = append(m.data, p...)
	return len(p), nil
}

func (m *mockWriter) Close() error {
	if m.faileClose {
		return fmt.Errorf("fail on closed writer")
	}
	m.closed = true
	return nil
}

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
	t.Run("should pass on getting root cert by default", func(t *testing.T) {
		oldCertPool := systemCertPool
		defer func() { systemCertPool = oldCertPool }()

		systemCertPool = func() (*x509.CertPool, error) {
			return nil, assert.AnError
		}

		origiCaPath := customCAPath
		defer func() {
			customCAPath = origiCaPath
		}()
		customCAPath = "../testdata/api/test.crt"

		_, err := createTLSConfig("myServer", false)

		require.NoError(t, err)

	})
	t.Run("should fail on invalid pem", func(t *testing.T) {
		oldCertPool := systemCertPool
		defer func() { systemCertPool = oldCertPool }()

		systemCertPool = func() (*x509.CertPool, error) {
			return nil, assert.AnError
		}

		origiCaPath := customCAPath
		defer func() {
			customCAPath = origiCaPath
		}()
		tmpDir := t.TempDir()
		customCAPath = filepath.Join(tmpDir, "ca.pem")

		// absichtlich kaputt
		os.WriteFile(customCAPath, []byte("not a certificate"), 0644)

		_, err := createTLSConfig("myServer", false)

		require.NoError(t, err)

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
	t.Run("should fail to send mail with implicit tls on creating client", func(t *testing.T) {
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
	t.Run("should fail to send mail with implicit tls on set sender", func(t *testing.T) {
		addr := "localhost:1"
		var a smtp.Auth = nil
		from := "from@example.com"
		to := []string{"to@example.com"}
		msg := []byte("Subject: test\r\n\r\nbody")

		config := configuration.Smtp{SkipTLSVerify: true}
		tlsconfig, _ := createTLSConfig("localhost", true)

		mockSMTPClientFactory := NewMockSMTPClientFactory(t)
		mockSMTPClient := NewMockSMTPClient(t)
		mockSMTPClientFactory.EXPECT().NewTLSClient(addr, tlsconfig).Return(mockSMTPClient, nil)
		ts := &tlsSender{config: config, factory: mockSMTPClientFactory}

		mockSMTPClient.EXPECT().Mail(from).Return(assert.AnError)

		mockSMTPClient.EXPECT().Quit().Return(nil)

		err := ts.sendMailWithTls(addr, a, from, to, msg)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to set sender")
	})
	t.Run("should fail to send mail with implicit tls on set sender", func(t *testing.T) {
		addr := "localhost:1"
		var a smtp.Auth = nil
		from := "from@example.com"
		to := []string{"to@example.com"}
		msg := []byte("Subject: test\r\n\r\nbody")

		config := configuration.Smtp{SkipTLSVerify: true}
		tlsconfig, _ := createTLSConfig("localhost", true)

		mockSMTPClientFactory := NewMockSMTPClientFactory(t)
		mockSMTPClient := NewMockSMTPClient(t)
		mockSMTPClientFactory.EXPECT().NewTLSClient(addr, tlsconfig).Return(mockSMTPClient, nil)
		ts := &tlsSender{config: config, factory: mockSMTPClientFactory}

		mockSMTPClient.EXPECT().Mail(from).Return(nil)

		for _, tor := range to {
			mockSMTPClient.EXPECT().Rcpt(tor).Return(assert.AnError)
		}

		mockSMTPClient.EXPECT().Quit().Return(nil)

		err := ts.sendMailWithTls(addr, a, from, to, msg)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to set recipient to@example.com")
	})
	t.Run("should fail to send mail with implicit tls on set data", func(t *testing.T) {
		addr := "localhost:1"
		var a smtp.Auth = nil
		from := "from@example.com"
		to := []string{"to@example.com"}
		msg := []byte("Subject: test\r\n\r\nbody")

		config := configuration.Smtp{SkipTLSVerify: true}
		tlsconfig, _ := createTLSConfig("localhost", true)

		mockSMTPClientFactory := NewMockSMTPClientFactory(t)
		mockSMTPClient := NewMockSMTPClient(t)
		mockSMTPClientFactory.EXPECT().NewTLSClient(addr, tlsconfig).Return(mockSMTPClient, nil)
		ts := &tlsSender{config: config, factory: mockSMTPClientFactory}

		mockSMTPClient.EXPECT().Mail(from).Return(nil)

		for _, tor := range to {
			mockSMTPClient.EXPECT().Rcpt(tor).Return(nil)
		}

		mockSMTPClient.EXPECT().Data().Return(nil, assert.AnError)

		mockSMTPClient.EXPECT().Quit().Return(nil)

		err := ts.sendMailWithTls(addr, a, from, to, msg)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get writer from mail server")
	})
	t.Run("should fail to send mail with implicit tls on write data", func(t *testing.T) {
		addr := "localhost:1"
		var a smtp.Auth = nil
		from := "from@example.com"
		to := []string{"to@example.com"}
		msg := []byte("Subject: test\r\n\r\nbody")

		config := configuration.Smtp{SkipTLSVerify: true}
		tlsconfig, _ := createTLSConfig("localhost", true)

		mockSMTPClientFactory := NewMockSMTPClientFactory(t)
		mockSMTPClient := NewMockSMTPClient(t)
		mockSMTPClientFactory.EXPECT().NewTLSClient(addr, tlsconfig).Return(mockSMTPClient, nil)
		ts := &tlsSender{config: config, factory: mockSMTPClientFactory}

		mockSMTPClient.EXPECT().Mail(from).Return(nil)

		for _, tor := range to {
			mockSMTPClient.EXPECT().Rcpt(tor).Return(nil)
		}

		writer := &mockWriter{
			closed: true, // force error on write
		}
		mockSMTPClient.EXPECT().Data().Return(writer, nil)

		mockSMTPClient.EXPECT().Quit().Return(nil)

		err := ts.sendMailWithTls(addr, a, from, to, msg)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to write message: write on closed writer")
	})
	t.Run("should fail to send mail with implicit tls on close writer", func(t *testing.T) {
		addr := "localhost:1"
		var a smtp.Auth = nil
		from := "from@example.com"
		to := []string{"to@example.com"}
		msg := []byte("Subject: test\r\n\r\nbody")

		config := configuration.Smtp{SkipTLSVerify: true}
		tlsconfig, _ := createTLSConfig("localhost", true)

		mockSMTPClientFactory := NewMockSMTPClientFactory(t)
		mockSMTPClient := NewMockSMTPClient(t)
		mockSMTPClientFactory.EXPECT().NewTLSClient(addr, tlsconfig).Return(mockSMTPClient, nil)
		ts := &tlsSender{config: config, factory: mockSMTPClientFactory}

		mockSMTPClient.EXPECT().Mail(from).Return(nil)

		for _, tor := range to {
			mockSMTPClient.EXPECT().Rcpt(tor).Return(nil)
		}

		writer := &mockWriter{
			closed:     false,
			faileClose: true,
		}
		mockSMTPClient.EXPECT().Data().Return(writer, nil)

		mockSMTPClient.EXPECT().Quit().Return(nil)

		err := ts.sendMailWithTls(addr, a, from, to, msg)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to close message writer: fail on closed writer")
	})
	t.Run("should pass to send mail with implicit tls ", func(t *testing.T) {
		addr := "localhost:1"
		var a smtp.Auth = nil
		from := "from@example.com"
		to := []string{"to@example.com"}
		msg := []byte("Subject: test\r\n\r\nbody")

		config := configuration.Smtp{SkipTLSVerify: true}
		tlsconfig, _ := createTLSConfig("localhost", true)

		mockSMTPClientFactory := NewMockSMTPClientFactory(t)
		mockSMTPClient := NewMockSMTPClient(t)
		mockSMTPClientFactory.EXPECT().NewTLSClient(addr, tlsconfig).Return(mockSMTPClient, nil)
		ts := &tlsSender{config: config, factory: mockSMTPClientFactory}

		mockSMTPClient.EXPECT().Mail(from).Return(nil)

		for _, tor := range to {
			mockSMTPClient.EXPECT().Rcpt(tor).Return(nil)
		}

		writer := &mockWriter{
			closed:     false,
			faileClose: false,
		}
		mockSMTPClient.EXPECT().Data().Return(writer, nil)

		mockSMTPClient.EXPECT().Quit().Return(nil)

		err := ts.sendMailWithTls(addr, a, from, to, msg)

		require.NoError(t, err)
	})
	t.Run("should pass to send mail with implicit tls an error in quit", func(t *testing.T) {
		addr := "localhost:1"
		var a smtp.Auth = nil
		from := "from@example.com"
		to := []string{"to@example.com"}
		msg := []byte("Subject: test\r\n\r\nbody")

		config := configuration.Smtp{SkipTLSVerify: true}
		tlsconfig, _ := createTLSConfig("localhost", true)

		mockSMTPClientFactory := NewMockSMTPClientFactory(t)
		mockSMTPClient := NewMockSMTPClient(t)
		mockSMTPClientFactory.EXPECT().NewTLSClient(addr, tlsconfig).Return(mockSMTPClient, nil)
		ts := &tlsSender{config: config, factory: mockSMTPClientFactory}

		mockSMTPClient.EXPECT().Mail(from).Return(nil)

		for _, tor := range to {
			mockSMTPClient.EXPECT().Rcpt(tor).Return(nil)
		}

		writer := &mockWriter{
			closed:     false,
			faileClose: false,
		}
		mockSMTPClient.EXPECT().Data().Return(writer, nil)

		mockSMTPClient.EXPECT().Quit().Return(assert.AnError)

		err := ts.sendMailWithTls(addr, a, from, to, msg)

		require.NoError(t, err)
	})
	t.Run("fail on createConfig", func(t *testing.T) {
		oldCertPool := systemCertPool
		defer func() { systemCertPool = oldCertPool }()
		oldReadFile := readFile
		defer func() { readFile = oldReadFile }()

		readFile = func(name string) ([]byte, error) {
			return nil, assert.AnError
		}
		addr := "localhost:1"
		var a smtp.Auth = nil
		from := "from@example.com"
		to := []string{"to@example.com"}
		msg := []byte("Subject: test\r\n\r\nbody")

		config := configuration.Smtp{SkipTLSVerify: true}

		mockSMTPClientFactory := NewMockSMTPClientFactory(t)

		ts := &tlsSender{config: config, factory: mockSMTPClientFactory}

		err := ts.sendMailWithTls(addr, a, from, to, msg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create TLS config")
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
	t.Run("should fail to send mail with StartTLS on getting data", func(t *testing.T) {
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
	t.Run("should fail to send mail with StartTLS on write data", func(t *testing.T) {
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

		writer := &mockWriter{
			closed: true,
		}
		mockSMTPClient.EXPECT().Data().Return(writer, nil)

		err := ts.sendMailWithStartTLS(addr, a, from, to, msg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to write message: write on closed writer")
	})
	t.Run("should fail to send mail with StartTLS on close writer", func(t *testing.T) {
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

		writer := &mockWriter{
			closed:     false,
			faileClose: true,
		}
		mockSMTPClient.EXPECT().Data().Return(writer, nil)

		err := ts.sendMailWithStartTLS(addr, a, from, to, msg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to close message writer")
	})
	t.Run("should pass to send mail with StartTLS", func(t *testing.T) {
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

		writer := &mockWriter{
			closed:     false,
			faileClose: false,
		}
		mockSMTPClient.EXPECT().Data().Return(writer, nil)

		err := ts.sendMailWithStartTLS(addr, a, from, to, msg)
		require.NoError(t, err)
	})

	t.Run("fail on createConfig", func(t *testing.T) {
		oldCertPool := systemCertPool
		defer func() { systemCertPool = oldCertPool }()
		oldReadFile := readFile
		defer func() { readFile = oldReadFile }()

		readFile = func(name string) ([]byte, error) {
			return nil, assert.AnError
		}
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

		mockSMTPClient.EXPECT().Extension("STARTTLS").Return(true, "")

		mockSMTPClient.EXPECT().Quit().Return(nil)

		err := ts.sendMailWithStartTLS(addr, a, from, to, msg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create TLS config")
	})

	t.Run("should pass to send mail with StartTLS with error on quit", func(t *testing.T) {
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

		mockSMTPClient.EXPECT().Quit().Return(assert.AnError)

		for _, tor := range to {
			mockSMTPClient.EXPECT().Rcpt(tor).Return(nil)
		}

		writer := &mockWriter{
			closed:     false,
			faileClose: false,
		}
		mockSMTPClient.EXPECT().Data().Return(writer, nil)

		err := ts.sendMailWithStartTLS(addr, a, from, to, msg)
		require.NoError(t, err)
	})

}
