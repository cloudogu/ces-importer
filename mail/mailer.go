package mail

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log/slog"
	"maps"
	"mime/multipart"
	"mime/quotedprintable"
	"net/smtp"
	"net/textproto"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/cloudogu/ces-importer/configuration"
	"github.com/cloudogu/k8s-registry-lib/config"
)

const (
	stateMigrationSuccess = "erfolgreich"
	stateMigrationFailure = "nicht erfolgreich"
)

const (
	typeMigrationFinal   = "finale Migration"
	typeMigrationPartial = "Delta-Migration"
)

const (
	mailSubject      = "Die Migration der Instanz %s war %s."
	mailBody         = "Die %s von der Instanz %s zu der Instanz %s war %s.\n\nStartzeitpunkt: %v\nEndzeitpunkt: %v\n\n%sAlle weiteren Informationen finden Sie in der Log-Datei im Anhang."
	errorMsgTemplate = "Die Fehlermeldung ist: %s\n\n"
)

const GLOBAL_CONFIG_FQDN_KEY = "fqdn"

var customCAPath = "/etc/custom-certs/mail/%s"

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
	config           configuration.Smtp // SMTP configuration
	sourceInstance   string             // Source instance URL of the exporters
	senderService    SenderService      // Function to send email
	readFile         OsReadFile         // Function to read email content from a file
	attachments      []string           // List of files to attach to each mail
	globalConfigRepo globalConfigRepo   // Repository for global configuration
}

// CreateSender initializes and returns a new Sender instance with the provided configuration,
// sender service, and file reader.
func CreateSender(config configuration.Smtp, sourceInstance string, attachments []string, globalConfigRepo globalConfigRepo) (*Sender, error) {
	slog.Info(fmt.Sprintf("Mailer useTLS Config: %v", config.UseTls))
	var senderService SenderService
	if config.UseTls == configuration.TLSModeImplicit || config.UseTls == "true" {
		ts := &tlsSender{config: config, factory: &realSMTPFactory{}}
		senderService = ts.sendMailWithTls
	} else if config.UseTls == configuration.TLSModeStartTLS {
		ts := &tlsSender{config: config, factory: &realSMTPFactory{}}
		senderService = ts.sendMailWithStartTLS
	} else if config.UseTls == configuration.TLSModeNone || config.UseTls == "false" || config.UseTls == "" {
		senderService = smtp.SendMail
	} else {
		return nil, fmt.Errorf("invalid useTLS(implicit, starttls, none): %v", config.UseTls)
	}

	customCAPath = fmt.Sprintf(customCAPath, config.TLSCertificateName)

	return &Sender{
		config,
		sourceInstance,
		senderService,
		os.ReadFile,
		attachments,
		globalConfigRepo,
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
// :
// Returns an error if email composition or sending fails.
func (s *Sender) Send(ctx context.Context, isFinal bool, migrationResult error, start time.Time, end time.Time) error {
	if !s.config.Enabled {
		slog.Info("Sending mail is disabled in configuration. Not sending mail.")
		return nil
	}

	if s.config.Server == "" || s.config.Port <= 0 {
		slog.Warn("SMTP server not configured. Not sending mail.", "server", s.config.Server, "port", s.config.Port)
		return nil
	}

	slog.Info("Sending migration result via mail...")
	slog.Debug(fmt.Sprintf("Mail is sent from: %s", s.config.From))
	slog.Debug(fmt.Sprintf("Mail is sent to: %v", s.config.To))
	slog.Debug(fmt.Sprintf("Mail is sent to server: %s", s.server()))

	if s.auth() != nil {
		slog.Info("Using authentication for mail server")
	} else {
		slog.Info("Mail server is unauthenticated")
	}

	migrationSuccessful := migrationResult == nil

	targetInstance, err := s.getTargetInstance(ctx)
	if err != nil {
		return fmt.Errorf("failed to get target instance: %w", err)
	}

	// Create buffer and multipart writer
	var body bytes.Buffer
	multipartWriter := multipart.NewWriter(&body)

	// Write headers
	headers := make(map[string]string)
	headers["From"] = s.config.From
	headers["To"] = strings.Join(s.config.To, ", ")
	headers["Subject"] = buildSubject(migrationSuccessful, s.sourceInstance)
	headers["MIME-Version"] = "1.0"
	headers["Content-Type"] = "multipart/mixed; boundary=" + multipartWriter.Boundary()
	headers["Message-ID"] = buildMessageId(targetInstance)
	headers["Date"] = getDate()

	for _, key := range slices.Sorted(maps.Keys(headers)) {
		body.WriteString(fmt.Sprintf("%s: %s\r\n", key, headers[key]))
	}

	// empty line needed between header and content
	body.WriteString("\r\n")

	err = s.writeBodyText(ctx, multipartWriter, migrationResult, start, end, isFinal)
	if err != nil {
		return fmt.Errorf("failed to write body text: %w", err)
	}

	for _, file := range s.attachments {
		if err := s.writeAttachment(multipartWriter, file); err != nil {
			slog.Error("failed to add attachment to mail", "err", err)
			continue
		}
		slog.Debug("added attachment to mail", "file", file)
	}

	if err := multipartWriter.Close(); err != nil {
		return fmt.Errorf("failed to close multipart writer: %w", err)
	}

	return s.senderService(
		s.server(),
		s.auth(),
		s.config.From,
		s.config.To,
		body.Bytes(),
	)
}

func (s *Sender) auth() smtp.Auth {
	var auth smtp.Auth

	if s.config.Username != "" && s.config.Password != "" {
		auth = smtp.PlainAuth("", s.config.Username, s.config.Password, s.config.Server)
	}

	return auth
}

func (s *Sender) server() string {
	return fmt.Sprintf("%s:%d", s.config.Server, s.config.Port)
}

func buildSubject(success bool, sourceInstance string) string {
	result := stateMigrationSuccess
	if !success {
		result = stateMigrationFailure
	}
	return fmt.Sprintf(mailSubject, sourceInstance, result)
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

func (s *Sender) writeBodyText(ctx context.Context, writer *multipart.Writer, migrationResult error, start time.Time, end time.Time, isFinal bool) error {
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

	targetInstance, err := s.getTargetInstance(ctx)
	if err != nil {
		return fmt.Errorf("failed to get target instance: %w", err)
	}

	bodyText := fmt.Sprintf(
		mailBody,
		migrationType,
		formatAsUrl(s.sourceInstance),
		formatAsUrl(targetInstance),
		result,
		start.Format("02.01.2006 15:04 (MST -0700)"),
		end.Format("02.01.2006 15:04 (MST -0700)"),
		errorMsg,
	)

	textPart, err := writer.CreatePart(textproto.MIMEHeader{
		"Content-Type":              {"text/plain; charset=utf-8"},
		"Content-Transfer-Encoding": {"quoted-printable"},
	})
	if err != nil {
		return fmt.Errorf("failed to create text part: %w", err)
	}

	qp := quotedprintable.NewWriter(textPart)
	if _, err := qp.Write([]byte(bodyText)); err != nil {
		return fmt.Errorf("failed to write body-text: %w", err)
	}

	if err := qp.Close(); err != nil {
		return fmt.Errorf("failed to close quoted-printable writer: %w", err)
	}

	return nil
}

func (s *Sender) writeAttachment(writer *multipart.Writer, filename string) error {
	fileData, err := s.readFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read file for attachment: %w", err)
	}

	attachmentPart, err := writer.CreatePart(textproto.MIMEHeader{
		"Content-Type":              {"application/octet-stream"},
		"Content-Disposition":       {`attachment; filename="` + filepath.Base(filename) + `"`},
		"Content-Transfer-Encoding": {"base64"},
	})
	if err != nil {
		return fmt.Errorf("failed to create attachment part: %w", err)
	}

	b := make([]byte, base64.StdEncoding.EncodedLen(len(fileData)))
	base64.StdEncoding.Encode(b, fileData)

	if _, err := attachmentPart.Write(b); err != nil {
		return fmt.Errorf("failed to write attachment: %w", err)
	}

	return nil
}

func buildMessageId(fqdn string) string {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		slog.Error(fmt.Sprintf("failed to generate random ID: %v", err))
	}

	timestamp := time.Now().UTC().Format("20060102150405.999999999")
	identifier := fmt.Sprintf("%s.%x", timestamp, b)
	return fmt.Sprintf("<%s@%s>", identifier, fqdn)
}

// getDate returns the current date time formatted for the email related date fields. This format uses the date RFC 1123 formatting which matches the date format used for the email `date` field as described by RFC 5322 and other relevant mail RFCs .
func getDate() string {
	return time.Now().Format(time.RFC1123)
}

func formatAsUrl(instance string) string {
	return fmt.Sprintf("https://%s", instance)
}
