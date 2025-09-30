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
	config configuration.Smtp
}

func (ts *tlsSender) sendMailWithTls(addr string, a smtp.Auth, from string, to []string, msg []byte) error {
	slog.Debug("sending mail with TLS enabled")
	// addr contains the server and port
	serverName := strings.Split(addr, ":")[0]

	tlsConfig, err := createTLSConfig(serverName, ts.config.SkipTLSVerify)
	if err != nil {
		return fmt.Errorf("failed to create TLS config: %w", err)
	}

	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		return fmt.Errorf("failed to connect to mail server: %w", err)
	}
	defer func(conn *tls.Conn) {
		_ = conn.Close()
	}(conn)

	c, err := smtp.NewClient(conn, serverName)
	if err != nil {
		return fmt.Errorf("failed to create smtp client: %w", err)
	}
	defer func(c *smtp.Client) {
		err := c.Quit()
		if err != nil {
			slog.Error(fmt.Sprintf("Failed to quit smtp mail client: %v", err))
		}
	}(c)

	if err = c.Mail(from); err != nil {
		return fmt.Errorf("failed to create message on mail server: %w", err)
	}
	for _, addr := range to {
		if err = c.Rcpt(addr); err != nil {
			return fmt.Errorf("failed to add recipient %s message: %w", addr, err)
		}
	}
	w, err := c.Data()
	if err != nil {
		return fmt.Errorf("failed to  get writer from mail server: %w", err)
	}
	_, err = w.Write(msg)
	if err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}
	err = w.Close()
	if err != nil {
		return fmt.Errorf("failed to close message writer: %w", err)
	}

	return nil
}

func createTLSConfig(serverName string, insecureSkipVerify bool) (*tls.Config, error) {
	caCert, err := os.ReadFile(customCAPath)
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

	// use system cert pool if it exists
	rootCAs, err := x509.SystemCertPool()
	if err != nil {
		rootCAs = x509.NewCertPool()
	}

	if ok := rootCAs.AppendCertsFromPEM(caCert); !ok {
		slog.Warn("Could not add custom CAs. They might already be included.")
	}

	return &tls.Config{
		ServerName:         serverName,
		RootCAs:            rootCAs,
		InsecureSkipVerify: insecureSkipVerify,
	}, nil
}
