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

func (ts *tlsSender) sendMailWithStartTLS(addr string, auth smtp.Auth, from string, to []string, msg []byte) error {
	serverName := strings.Split(addr, ":")[0]

	conn, err := smtp.Dial(addr)
	if err != nil {
		return err
	}

	defer func(conn *smtp.Client) {
		err := conn.Quit()
		if err != nil {
			slog.Error(fmt.Sprintf("Failed to quit smtp mail client: %v", err))
		}
	}(conn)

	hostname, _ := os.Hostname()
	if err = conn.Hello(hostname); err != nil {
		return fmt.Errorf("failed to register conn: %w", err)
	}

	if ok, _ := conn.Extension("STARTTLS"); !ok {
		return fmt.Errorf("SMTP server does not support STARTTLS")
	}

	tlsConfig, err := createTLSConfig(serverName, ts.config.SkipTLSVerify)
	if err != nil {
		return err
	}

	if err = conn.StartTLS(tlsConfig); err != nil {
		return fmt.Errorf("failed to init starttls: %w", err)
	}

	if auth != nil {
		if ok, _ := conn.Extension("AUTH"); ok {
			if err = conn.Auth(auth); err != nil {
				return fmt.Errorf("failed to authenticate: %w", err)
			}
		}
	}

	if err = conn.Mail(from); err != nil {
		return fmt.Errorf("failed to set sender: %w", err)
	}

	for _, addr := range to {
		if err = conn.Rcpt(addr); err != nil {
			return fmt.Errorf("failed to set recipient: %w", err)
		}
	}

	w, err := conn.Data()
	if err != nil {
		return fmt.Errorf("failed to get writer from mail server: %w", err)
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
	rootCAs, err := x509.SystemCertPool()
	if err != nil {
		rootCAs = x509.NewCertPool()
	}

	caCert, err := os.ReadFile(customCAPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			slog.Info(fmt.Sprintf("No custom CA found at %s, using system certs only", customCAPath))
		} else {
			return nil, fmt.Errorf("failed to read custom CA file: %w", err)
		}
	} else {
		if ok := rootCAs.AppendCertsFromPEM(caCert); !ok {
			slog.Warn(fmt.Sprintf("No certificates could be parsed from %s", customCAPath))
		}
	}

	return &tls.Config{
		ServerName:         serverName,
		RootCAs:            rootCAs,
		InsecureSkipVerify: insecureSkipVerify,
	}, nil
}
