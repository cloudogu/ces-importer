package mail

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"log/slog"
	"net/smtp"
	"os"
	"strings"

	"github.com/cloudogu/ces-importer/configuration"
)

type tlsSender struct {
	config  configuration.Smtp
	factory SMTPClientFactory
}

var readFile = os.ReadFile
var systemCertPool = x509.SystemCertPool

func (ts *tlsSender) sendMailWithTls(addr string, a smtp.Auth, from string, to []string, msg []byte) error {
	slog.Debug("sending mail with TLS enabled")
	// addr contains the server and port
	serverName := strings.Split(addr, ":")[0]

	tlsConfig, err := createTLSConfig(serverName, ts.config.SkipTLSVerify)
	if err != nil {
		return fmt.Errorf("failed to create TLS config: %w", err)
	}

	client, err := ts.factory.NewTLSClient(addr, tlsConfig)
	if err != nil {
		return fmt.Errorf("failed to create smtp tls client: %w", err)
	}

	defer func() {
		if err := client.Quit(); err != nil {
			slog.Error(fmt.Sprintf("Failed to quit smtp mail client: %v", err))
		}
	}()

	return ts.prepareMail(client, from, to, msg)
}

func (ts *tlsSender) sendMailWithStartTLS(addr string, a smtp.Auth, from string, to []string, msg []byte) error {
	serverName := strings.Split(addr, ":")[0]

	client, err := ts.factory.NewClient(addr)
	if err != nil {
		return fmt.Errorf("failed to create smtp client: %w", err)
	}

	defer func() {
		if err := client.Quit(); err != nil {
			slog.Error(fmt.Sprintf("Failed to quit smtp mail client: %v", err))
		}
	}()

	hostname, _ := os.Hostname()
	if err = client.Hello(hostname); err != nil {
		return fmt.Errorf("failed to register conn: %w", err)
	}

	if ok, _ := client.Extension("STARTTLS"); !ok {
		return fmt.Errorf("SMTP server does not support STARTTLS")
	}

	tlsConfig, err := createTLSConfig(serverName, ts.config.SkipTLSVerify)
	if err != nil {
		return fmt.Errorf("failed to create TLS config: %w", err)
	}

	if err = client.StartTLS(tlsConfig); err != nil {
		return fmt.Errorf("failed to init starttls: %w", err)
	}

	if a != nil {
		if ok, _ := client.Extension("AUTH"); ok {
			if err = client.Auth(a); err != nil {
				return fmt.Errorf("failed to authenticate: %w", err)
			}
		}
	}

	return ts.prepareMail(client, from, to, msg)
}

func (ts *tlsSender) prepareMail(client SMTPClient, f string, t []string, msg []byte) error {
	if err := client.Mail(f); err != nil {
		return fmt.Errorf("failed to set sender: %w", err)
	}

	for _, rcpt := range t {
		if err := client.Rcpt(rcpt); err != nil {
			return fmt.Errorf("failed to set recipient %s: %w", rcpt, err)
		}
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("failed to get writer from mail server: %w", err)
	}

	if _, err = w.Write(msg); err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}

	if err = w.Close(); err != nil {
		return fmt.Errorf("failed to close message writer: %w", err)
	}
	return nil
}

func createTLSConfig(serverName string, insecureSkipVerify bool) (*tls.Config, error) {
	caCert, err := readFile(customCAPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			slog.Info(fmt.Sprintf("Skipping custom CAs as none were provided in %s", customCAPath))

			return &tls.Config{
				ServerName:         serverName,
				InsecureSkipVerify: insecureSkipVerify,
			}, nil
		} else if err != nil {
			return nil, fmt.Errorf("failed to read custom CA file: %w", err)
		}
	}

	rootCAs, err := systemCertPool()
	if err != nil {
		rootCAs = x509.NewCertPool()
	}

	if ok := rootCAs.AppendCertsFromPEM(caCert); !ok {
		slog.Warn(fmt.Sprintf("No certificates could be parsed from %s", customCAPath))
	}

	return &tls.Config{
		ServerName:         serverName,
		RootCAs:            rootCAs,
		InsecureSkipVerify: insecureSkipVerify,
	}, nil
}
