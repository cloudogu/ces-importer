package fqdn

import (
	"context"
	"errors"
	"fmt"
	"github.com/cloudogu/ces-importer/api/exporter"
	"log/slog"
)

const (
	globalCfgFQDNKey     = "/fqdn"
	globalCfgCertTypeKey = "/certificate/type"
	globalCfgCertKey     = "/certificate/server.crt"
	globalCfgPrivateKey  = "/certificate/server.key"
)

type backuper interface {
	Backup(ctx context.Context) error
	DeleteBackup(ctx context.Context) error
}

type restorer interface {
	Restore(ctx context.Context) error
}

type fqdnUpdater interface {
	Update(ctx context.Context, c ConfigChange) error
}

type certificateUpdater interface {
	Update(ctx context.Context, tlsKeyPair SecretChange) error
}

type fqdnChanger interface {
	backuper
	restorer
	fqdnUpdater
}

type certificateChanger interface {
	backuper
	restorer
	certificateUpdater
}

// Change represents a change request for fqdn and certificate.
type Change struct {
	ConfigChange
	SecretChange
}

// isValid checks if the change request is valid by checking if the fqdn, certificate type, certificate and certificate key are set.
func (c Change) isValid() error {
	if c.FQDN == "" {
		return fmt.Errorf("fqdn is empty")
	}

	if c.CertType == "" {
		return fmt.Errorf("certificate type is empty")
	}

	if len(c.Certificate) == 0 {
		return fmt.Errorf("certificate is empty")
	}

	if len(c.CertificateKey) == 0 {
		return fmt.Errorf("certificate key is empty")
	}

	return nil
}

// createChangeRequest creates a change request from the global config of the exporter.
func createChangeRequest(globalCfg exporter.GlobalConfig) (Change, error) {
	var change Change

	for _, kv := range globalCfg {
		switch kv.Key {
		case globalCfgFQDNKey:
			change.FQDN = kv.Value
		case globalCfgCertTypeKey:
			change.CertType = kv.Value
		case globalCfgCertKey:
			change.Certificate = Certificate(kv.Value)
		case globalCfgPrivateKey:
			change.CertificateKey = CertificateKey(kv.Value)
		}
	}

	if err := change.isValid(); err != nil {
		return Change{}, fmt.Errorf("invalid global config: %w", err)
	}

	return change, nil
}

// restoreError is a custom error type for the restore operation that wraps the error that occurred during the update of the fqdn and certificate.
type restoreError struct {
	err       error
	updateErr error
}

func (r restoreError) Error() string {
	return r.updateErr.Error()
}

// Service is the service that handles the fqdn change.
type Service struct {
	configService configGetter
	fqdn          fqdnChanger
	cert          certificateChanger
}

// NewService creates a new instance of the fqdn change service.
func NewService(configService configGetter, globalConfigRepository globalConfigRepository, configMapRepository configMapRepository, secretRepository secretRepository) *Service {
	return &Service{
		configService: configService,
		fqdn: &fqdnManager{
			repo:             configMapRepository,
			globalConfigRepo: globalConfigRepository,
		},
		cert: &ecosystemCertificate{repo: secretRepository},
	}
}

// ChangeFQDN changes the fqdn and certificate of the importing system by taking over the configuration of the exporter.
// It creates a backup of the fqdn and certificate before the change and restores it if the change fails.
func (s *Service) ChangeFQDN(ctx context.Context) (err error) {
	slog.Info("FQDN change started.")

	if err = s.backup(ctx); err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}

	defer func() {
		var rErr restoreError

		if errors.As(err, &rErr) {
			slog.Error("restore from backup has failed", "error", rErr.err)
			return
		}

		if dErr := s.deleteBackup(ctx); dErr != nil {
			slog.Warn("failed to delete backup", "error", dErr)
		}
	}()

	slog.Debug("created backup while updating fqdn")

	exporterConfig, err := s.configService.GetConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to get config from exporter: %w", err)
	}

	globalCfgDTO := exporterConfig.GlobalConfig

	slog.Debug("got global config from exporter")

	changeReq, err := createChangeRequest(globalCfgDTO)
	if err != nil {
		return fmt.Errorf("failed to create change request for fqdn: %w", err)
	}

	slog.Debug("extracted relevant config parameters from exporter for fqdn change")

	if err = s.update(ctx, changeReq); err != nil {
		slog.Warn("update fqdn has failed, try to restore backup")

		if rErr := s.restore(ctx); rErr != nil {
			return restoreError{err: rErr, updateErr: fmt.Errorf("failed to update fqdn or certificate: %w", err)}
		}

		slog.Info("successfully restored backup")

		return fmt.Errorf("failed to update fqdn or certificate: %w", err)
	}

	return nil
}

func (s *Service) backup(ctx context.Context) error {
	if err := s.fqdn.Backup(ctx); err != nil {
		return fmt.Errorf("failed to backup fqdn: %w", err)
	}

	if err := s.cert.Backup(ctx); err != nil {
		return fmt.Errorf("failed to backup certificate: %w", err)
	}

	return nil
}

func (s *Service) deleteBackup(ctx context.Context) error {
	if err := s.fqdn.DeleteBackup(ctx); err != nil {
		return fmt.Errorf("failed to delete backup for fqdn: %w", err)
	}

	if err := s.cert.DeleteBackup(ctx); err != nil {
		return fmt.Errorf("failed to delete backup for certificate: %w", err)
	}

	return nil
}

func (s *Service) update(ctx context.Context, c Change) error {
	if err := s.cert.Update(ctx, c.SecretChange); err != nil {
		return fmt.Errorf("failed to update certificate: %w", err)
	}

	if err := s.fqdn.Update(ctx, c.ConfigChange); err != nil {
		return fmt.Errorf("failed to update fqdn: %w", err)
	}

	return nil
}

func (s *Service) restore(ctx context.Context) error {
	if err := s.cert.Restore(ctx); err != nil {
		return fmt.Errorf("failed to restore certificate: %w", err)
	}

	if err := s.fqdn.Restore(ctx); err != nil {
		return fmt.Errorf("failed to restore fqdn: %w", err)
	}

	return nil
}
