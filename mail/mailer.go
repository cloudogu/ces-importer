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

// OsReadFile defines a function type for reading a file from the given name and returning its contents as bytes.
type OsReadFile func(name string) ([]byte, error)

// SenderService defines a function type for sending an email using SMTP with the provided address, authentication,
// sender, recipients, and message body.
type SenderService func(addr string, a smtp.Auth, from string, to []string, msg []byte) error

// SmtpConfig holds SMTP server configuration details required for sending emails.
type SmtpConfig struct {
	Server   string   // SMTP server address (e.g., smtp.example.com)
	Port     string   // SMTP server port (default is "25" if not specified)
	Username string   // Username for SMTP authentication
	Password string   // Password for SMTP authentication
	From     string   // Sender's email address
	To       []string // List of recipient email addresses
}

// Sender provides functionality to send emails using a configured SMTP service.
type Sender struct {
	config        SmtpConfig    // SMTP configuration
	senderService SenderService // Function to send email
	readFile      OsReadFile    // Function to read email content from a file
	attachments   []string      // List of files to attach to each mail
}

// CreateSender initializes and returns a new Sender instance with the provided configuration,
// sender service, and file reader.
func CreateSender(config SmtpConfig, senderService SenderService, readFile OsReadFile, attachments []string) *Sender {
	return &Sender{
		config,
		senderService,
		readFile,
		attachments,
	}
}

// SmtpConfigFromEnv reads SMTP configuration from environment variables and returns
// a SmtpConfig struct. Returns an error if required fields like server or from address are missing.
//
// Expected environment variables:
//   - SMTP_SERVER
//   - SMTP_PORT (optional, defaults to "25")
//   - SMTP_USERNAME
//   - SMTP_PASSWORD
//   - SMTP_FROM
//   - SMTP_TO (comma-separated list of recipient emails)
func SmtpConfigFromEnv() (SmtpConfig, error) {
	server := os.Getenv(envSmtpServer)
	if server == "" {
		return SmtpConfig{}, fmt.Errorf("smtp Server address is not configured")
	}
	port := os.Getenv(envSmtpPort)
	if port == "" {
		port = "25"
	}

	username := os.Getenv(envSmtpUsername)
	password := os.Getenv(envSmtpPassword)

	from := os.Getenv(envSmtpFrom)
	if from == "" {
		return SmtpConfig{}, fmt.Errorf("smtp from is not configured")
	}
	toAsStr := os.Getenv(envSmtpTo)
	to := strings.Split(toAsStr, ",")

	return SmtpConfig{
		Server:   server,
		Port:     port,
		Username: username,
		Password: password,
		From:     from,
		To:       to,
	}, nil
}

// Send composes and sends an email containing the result of a migration operation.
//
// The email includes a plain text body summarizing the migration status and optional file attachments.
// It uses multipart MIME encoding to support attachments and plain text.
//
// Parameters:
//   - success: Indicates whether the migration was successful.
//   - attachments: List of file paths to include as attachments in the email.
//   - sourceInstance: URL of the source system
//   - targetInstance: URL of the target system.
//   - start: Start time of the migration.
//   - end: End time of the migration.
//   - isFinal: Whether this is the final report of a migration process.
//
// Returns an error if email composition or sending fails.
func (s *Sender) Send(isFinal bool, migrationResult error, sourceInstance string, targetInstance string, start time.Time, end time.Time) error {
	from := fmt.Sprintf("From: %s\r\n", s.config.From)
	body := s.body(migrationResult == nil, sourceInstance, targetInstance, start, end, isFinal)
	boundary := "MIME_BOUNDARY_CES_IMPORTER"
	mime := fmt.Sprintf("MIME-Version: 1.0\r\nContent-Type: multipart/mixed; boundary=%s\r\n\r\n", boundary)
	message := mime +
		"--" + boundary + "\r\n" +
		"Content-Type: text/plain; charset=utf-8\r\n\r\n" +
		body + "\r\n"

	for _, file := range s.attachments {
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
		s.config.From,
		s.config.To,
		[]byte(from+s.subject(migrationResult == nil)+message),
	)
}

func (s *Sender) auth() smtp.Auth {
	var auth smtp.Auth

	if s.config.Username != "" || s.config.Password != "" {
		auth = smtp.PlainAuth("", s.config.Username, s.config.Password, s.config.Server)
	}

	return auth
}

func (s *Sender) server() string {
	return fmt.Sprintf("%s:%s", s.config.Server, s.config.Port)
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

	return fmt.Sprintf("%s\r\n",
		fmt.Sprintf(
			mailBody,
			migrationType,
			sourceInstance,
			targetInstance,
			result,
			start.Format("15:04"),
			end.Format("15:04"),
		),
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
