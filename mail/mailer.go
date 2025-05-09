package mail

import (
	"encoding/base64"
	"fmt"
	"net/smtp"
	"os"
	"strings"
	"time"
)

const (
	stateMigrationSuccess = "erfolgreich"
	stateMigrationFailure = "nicht erfolgreich"
)

const (
	typeMigrationFinal   = "finale"
	typeMigrationPartial = "partielle"
)

const (
	envSmtpServer   = "SMTP_SERVER"
	envSmtpPort     = "SMTP_PORT"
	envSmtpUsername = "SMTP_USERNAME"
	envSmtpPassword = "SMTP_PASSWORD"
	envSmtpFrom     = "SMTP_FROM"
	envSmtpTo       = "SMTP_TO"
)

const (
	mailSubject = "Migration war %s."
	mailBody    = "Die %s Migration von der Instanz %s zu der Instanz %s war %s.\n\nStartzeitpunkt: %v\nEndzeitpunkt: %v\n\nAlle weiteren Informationen finden Sie in der Log-Datei im Anhang."
)

type OsReadFile func(name string) ([]byte, error)

type SenderService func(addr string, a smtp.Auth, from string, to []string, msg []byte) error

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
	senderService SenderService
	readFile      OsReadFile
}

func CreateSender(config SmtpConfig, senderService SenderService, readFile OsReadFile) *Sender {
	return &Sender{
		config,
		senderService,
		readFile,
	}
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

func (s *Sender) auth() smtp.Auth {
	var auth smtp.Auth

	if s.config.username != "" || s.config.password != "" {
		auth = smtp.PlainAuth("", s.config.username, s.config.password, s.config.server)
	}

	return auth
}

func (s *Sender) server() string {
	return fmt.Sprintf("%s:%s", s.config.server, s.config.port)
}

func (s *Sender) subject(success bool) string {
	result := stateMigrationSuccess
	if !success {
		result = stateMigrationFailure
	}
	return fmt.Sprintf("Subject: %s\r\n", fmt.Sprintf(mailSubject, result))
}

func (s *Sender) body(success bool, sourceInstance string, targetInstance string, start time.Time, end time.Time, isFinal bool) string {
	result := stateMigrationSuccess
	if !success {
		result = stateMigrationFailure
	}

	migrationType := typeMigrationPartial
	if isFinal {
		migrationType = typeMigrationFinal
	}

	return fmt.Sprintf("%s\r\n", fmt.Sprintf(mailBody, migrationType, sourceInstance, targetInstance, result, start, end))
}

func (s *Sender) SendMigrationResult(success bool, attachments []string, sourceInstance string, targetInstance string, start time.Time, end time.Time, isFinal bool) error {
	from := fmt.Sprintf("From: %s\r\n", s.config.from)
	body := s.body(success, sourceInstance, targetInstance, start, end, isFinal)
	boundary := "MIME_BOUNDARY_CES_IMPORTER"
	mime := fmt.Sprintf("MIME-Version: 1.0\r\nContent-Type: multipart/mixed; boundary=%s\r\n\r\n", boundary)
	message := mime +
		"--" + boundary + "\r\n" +
		"Content-Type: text/plain; charset=utf-8\r\n\r\n" +
		body + "\r\n"

	for _, file := range attachments {
		attachment, err := s.buildAttachment(file, boundary)
		if err != nil {
			return fmt.Errorf("failed to add attachment: %w", err)
		}
		message += attachment
	}

	message += "--" + boundary

	return s.senderService(
		s.server(),
		s.auth(),
		s.config.from,
		s.config.to,
		[]byte(from+s.subject(success)+message),
	)
}

func (s *Sender) buildAttachment(filename, boundary string) (string, error) {
	data, err := s.readFile(filename)
	if err != nil {
		return "", err
	}

	encoded := base64.StdEncoding.EncodeToString(data)

	attachment := "\r\n--" + boundary + "\r\n" +
		"Content-Type: application/octet-stream\r\n" +
		"Content-Transfer-Encoding: base64\r\n" +
		"Content-Disposition: attachment; filename=\"" + filename + "\"\r\n\r\n" +
		chunkSplit(encoded, 76) + "\r\n"

	return attachment, nil
}

func chunkSplit(body string, limit int) string {
	var chunked []string
	for i := 0; i < len(body); i += limit {
		end := i + limit
		if end > len(body) {
			end = len(body)
		}
		chunked = append(chunked, body[i:end])
	}
	return strings.Join(chunked, "\r\n")
}
