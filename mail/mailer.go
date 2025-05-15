package mail

import (
	"context"
	"encoding/base64"
	"fmt"
	"github.com/cloudogu/ces-importer/configuration"
	"github.com/cloudogu/k8s-registry-lib/config"
	"log/slog"
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
	mailSubject      = "Migration war %s."
	mailBody         = "Die %s Migration von der Instanz %s zu der Instanz %s war %s.\n\nStartzeitpunkt: %v\nEndzeitpunkt: %v\n\n%sAlle weiteren Informationen finden Sie in der Log-Datei im Anhang."
	errorMsgTemplate = "Die Fehlermeldung ist: %s\n\n"
)

const GLOBAL_CONFIG_FQDN_KEY = "fqdn"

type globalConfigRepo interface {
	Get(ctx context.Context) (config.GlobalConfig, error)
}

// OsReadFile defines a function type for reading a file from the given name and returning its contents as bytes.
type OsReadFile func(name string) ([]byte, error)

// SenderService defines a function type for sending an email using SMTP with the provided address, authentication,
// sender, recipients, and message body.
type SenderService func(addr string, a smtp.Auth, from string, to []string, msg []byte) error

// Sender provides functionality to send emails using a configured SMTP service.
type Sender struct {
	config           configuration.SmtpConfig // SMTP configuration
	sourceInstance   string                   // Source instance URL of the exporter
	senderService    SenderService            // Function to send email
	readFile         OsReadFile               // Function to read email content from a file
	attachments      []string                 // List of files to attach to each mail
	globalConfigRepo globalConfigRepo         // Repository for global configuration
}

// CreateSender initializes and returns a new Sender instance with the provided configuration,
// sender service, and file reader.
func CreateSender(config configuration.SmtpConfig, sourceInstance string, attachments []string, globalConfigRepo globalConfigRepo) *Sender {
	return &Sender{
		config,
		sourceInstance,
		smtp.SendMail,
		os.ReadFile,
		attachments,
		globalConfigRepo,
	}
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
func (s *Sender) Send(ctx context.Context, isFinal bool, migrationResult error, start time.Time, end time.Time) error {
	slog.Info("Sending migration result via mail...")
	slog.Info(fmt.Sprintf("Mail is sent from: %s", s.config.From))
	slog.Info(fmt.Sprintf("Mail is sent to: %v", s.config.To))
	slog.Info(fmt.Sprintf("Mail is sent to server: %s", s.server()))
	if s.auth() != nil {
		slog.Info("Using authentication for mail server")
	} else {
		slog.Info("Mail server is unauthenticated")
	}

	targetInstance, err := s.getTargetInstance(ctx)
	if err != nil {
		slog.Error(fmt.Sprintf("failed to get target instance: %v", err))
	}

	from := fmt.Sprintf("From: %s\r\n", s.config.From)
	body := s.body(migrationResult, s.sourceInstance, targetInstance, start, end, isFinal)
	boundary := "MIME_BOUNDARY_CES_IMPORTER"
	mime := fmt.Sprintf("MIME-Version: 1.0\r\nContent-Type: multipart/mixed; boundary=%s\r\n\r\n", boundary)
	message := mime +
		"--" + boundary + "\r\n" +
		"Content-Type: text/plain; charset=utf-8\r\n\r\n" +
		body + "\r\n"

	for _, file := range s.attachments {
		attachment, err := s.buildAttachment(file, boundary)
		if err != nil {
			slog.Error(fmt.Sprintf("failed to add attachment to mail: %v", err))
			continue
		}
		slog.Info(fmt.Sprintf("Added attachment to mail: %s", attachment))
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

func (s *Sender) body(migrationResult error, sourceInstance string, targetInstance string, start time.Time, end time.Time, isFinal bool) string {
	result := stateMigrationSuccess
	errorMsg := ""
	if migrationResult != nil {
		result = stateMigrationFailure
		errorMsg = fmt.Sprintf(errorMsgTemplate, migrationResult.Error())
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
			errorMsg,
		),
	)
}

func (s *Sender) buildAttachment(filename, boundary string) (string, error) {
	data, err := s.readFile(filename)
	if err != nil {
		return "", fmt.Errorf("failed to read file for attachment: %w", err)
	}

	encoded := base64.StdEncoding.EncodeToString(data)

	attachment := "\r\n--" + boundary + "\r\n" +
		"Content-Type: application/octet-stream\r\n" +
		"Content-Transfer-Encoding: base64\r\n" +
		"Content-Disposition: attachment; filename=\"" + filename + "\"\r\n\r\n" +
		chunkSplit(encoded, 76) + "\r\n"

	return attachment, nil
}

func (s *Sender) getTargetInstance(ctx context.Context) (string, error) {
	cfg, err := s.globalConfigRepo.Get(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get global config: %w", err)
	}

	fqdn, exists := cfg.Get(GLOBAL_CONFIG_FQDN_KEY)
	if !exists {
		return "", fmt.Errorf("global config does not contain key %s", GLOBAL_CONFIG_FQDN_KEY)
	}

	return fqdn.String(), nil
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
