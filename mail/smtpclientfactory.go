package mail

import (
	"crypto/tls"
	"net/smtp"
	"strings"
)

type SMTPClientFactory interface {
	NewTLSClient(addr string, tlsConfig *tls.Config) (SMTPClient, error)
	NewClient(addr string) (SMTPClient, error)
}

type realSMTPFactory struct{}

func (f *realSMTPFactory) NewTLSClient(addr string, tlsConfig *tls.Config) (SMTPClient, error) {
	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		return nil, err
	}

	serverName := strings.Split(addr, ":")[0]

	c, err := smtp.NewClient(conn, serverName)
	if err != nil {
		return nil, err
	}

	return &realSMTPClient{client: c}, nil
}

func (f *realSMTPFactory) NewClient(addr string) (SMTPClient, error) {
	c, err := smtp.Dial(addr)
	if err != nil {
		return nil, err
	}

	return &realSMTPClient{client: c}, nil
}
