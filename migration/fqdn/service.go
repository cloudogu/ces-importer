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

type Change struct {
	ConfigChange
	SecretChange
}

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

type restoreError struct {
	err       error
	updateErr error
}

func (r restoreError) Error() string {
	return r.updateErr.Error()
}

type Service struct {
	configService configGetter
	fqdn          fqdnChanger
	cert          certificateChanger
}

func NewService(configService configGetter, fqdn fqdnChanger, cert certificateChanger) *Service {
	return &Service{
		configService: configService,
		fqdn:          fqdn,
		cert:          cert,
	}
}

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

	slog.Debug("extracted relevant config parameters from exporter for fqdn change", "changeRequest", changeReq)

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
