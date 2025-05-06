package mail

import (
	"fmt"
	"net/smtp"
	"os"
	"strings"
)

const (
	stateMigrationSuccess = "erfolgreich"
	stateMigrationFailure = "nicht erfolgreich"
)

const (
	envSmtpServer = ""
	envSmtpPort
	envSmtpUsername
	envSmtpPassword
	envSmtpFrom
	envSmtpTo
)

const (
	mailSubject = "Migration war %s."
	mailBody    = "Die Migration von '%s' zu '%s' war %s.\n\nAlle weiteren Informationen finden Sie in der Log-Datei im Anhang."
)

type MailSenderService func(addr string, a smtp.Auth, from string, to []string, msg []byte) error

type SmtpConfig struct {
	server   string
	port     string
	username string
	password string
	from     string
	to       []string
}

type Sender struct {
	config        SmtpConfig
	senderService MailSenderService
}

func SmtpConfigFromEnv() (SmtpConfig, error) {
	server := os.Getenv(envSmtpServer)
	if server == "" {
		return SmtpConfig{}, fmt.Errorf("smtp server address is not configured")
	}
	port := os.Getenv(envSmtpPort)
	if port == "" {
		return SmtpConfig{}, fmt.Errorf("smtp port is not configured")
	}

	username := os.Getenv(envSmtpUsername)
	password := os.Getenv(envSmtpPassword)

	from := os.Getenv(envSmtpFrom)
	if from == "" {

	}
	toAsStr := os.Getenv(envSmtpTo)
	to := strings.Split(toAsStr, ",")

	return SmtpConfig{
		server:   server,
		port:     port,
		username: username,
		password: password,
		from:     from,
		to:       to,
	}, nil
}

func CreateSender(config SmtpConfig, senderService MailSenderService) *Sender {
	return &Sender{
		config,
		senderService,
	}
}

func (s *Sender) formatMessage() string {
	body := fmt.Sprintf(mailBody, "", "", "")
	subject := fmt.Sprintf(mailSubject, "")
	return fmt.Sprintf("From: %s\nTO: %s\nSubject: %s\n\n%s", s.config.from, strings.Join(s.config.to, ","), subject, body)
}

func (s *Sender) Send() error {
	var auth smtp.Auth
	if s.config.username != "" || s.config.password != "" {
		auth = smtp.PlainAuth("", s.config.username, s.config.password, s.config.server)
	}

	return s.senderService(
		s.config.server,
		auth,
		s.config.from,
		s.config.to,
		[]byte(s.formatMessage()),
	)
}
