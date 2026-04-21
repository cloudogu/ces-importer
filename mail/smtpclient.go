package mail

import (
	"crypto/tls"
	"io"
	"net/smtp"
)

type DialFunc func(addr string) (SMTPClient, error)

type SMTPClient interface {
	Mail(from string) error
	Rcpt(to string) error
	Data() (io.WriteCloser, error)
	Quit() error
	Hello(localName string) error
	Extension(ext string) (bool, string)
	StartTLS(config *tls.Config) error
	Auth(auth smtp.Auth) error
}

type realSMTPClient struct {
	client *smtp.Client
}

func (r *realSMTPClient) Mail(from string) error {
	return r.client.Mail(from)
}

func (r *realSMTPClient) Rcpt(to string) error {
	return r.client.Rcpt(to)
}

func (r *realSMTPClient) Data() (io.WriteCloser, error) {
	return r.client.Data()
}

func (r *realSMTPClient) Quit() error {
	return r.client.Quit()
}

func (r *realSMTPClient) Hello(localName string) error {
	return r.client.Hello(localName)
}

func (r *realSMTPClient) Extension(ext string) (bool, string) {
	return r.client.Extension(ext)
}

func (r *realSMTPClient) StartTLS(config *tls.Config) error {
	return r.client.StartTLS(config)
}

func (r *realSMTPClient) Auth(auth smtp.Auth) error {
	return r.client.Auth(auth)
}
