package fqdn

import (
	"context"
	"fmt"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
	"log/slog"
)

const (
	ecosystemCertificateName = "ecosystem-certificate"
	certFileName             = "tls.crt"
	keyFileName              = "tls.key"
)

var (
	certificateBackupName = fmt.Sprintf("%s-backup", ecosystemCertificateName)
)

type SecretChange struct {
	Certificate
	CertificateKey
}

type Certificate []byte

type CertificateKey []byte

type ecosystemCertificate struct {
	repo secretRepository
}

func retriable(err error) bool {
	return !apierrors.IsNotFound(err)
}

func (c *ecosystemCertificate) Backup(ctx context.Context) error {
	return retry.OnError(retry.DefaultRetry, retriable, func() error {
		return c.backup(ctx)
	})
}

func (c *ecosystemCertificate) backup(ctx context.Context) error {
	slog.Debug("Backing up ecosystem certificate")

	certificateSecret, err := c.repo.Get(ctx, ecosystemCertificateName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get secret for certificate: %w", err)
	}

	slog.Debug("Got original certificate secret", "name", ecosystemCertificateName)

	backup := certificateSecret.DeepCopy()
	backup.Name = certificateBackupName

	_, err = c.repo.Create(ctx, backup, metav1.CreateOptions{})
	if err != nil {
		if apierrors.IsAlreadyExists(err) {
			slog.Debug("ConfigChange certificate already exists, updating")

			uppErr := c.updateBackup(ctx, SecretChange{
				Certificate:    backup.Data[certFileName],
				CertificateKey: backup.Data[keyFileName],
			})
			if uppErr != nil {
				return fmt.Errorf("failed to update backup for certificate: %w", uppErr)
			}

			slog.Debug("Updated backup for certificate")

			return nil
		}

		return fmt.Errorf("failed to create secret for backup certificate: %w", err)
	}

	slog.Debug("Created backup for certificate secret", "name", certificateBackupName)

	return nil
}

func (c *ecosystemCertificate) updateBackup(ctx context.Context, tlsKeyPair SecretChange) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		return c.update(ctx, certificateBackupName, tlsKeyPair)
	})
}

func (c *ecosystemCertificate) Restore(ctx context.Context) error {
	return retry.OnError(retry.DefaultRetry, retriable, func() error {
		return c.restore(ctx)
	})
}

func (c *ecosystemCertificate) restore(ctx context.Context) error {
	slog.Debug("Restoring ecosystem certificate")

	backup, err := c.repo.Get(ctx, certificateBackupName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get secret for backup certificate: %w", err)
	}

	slog.Debug("Got backup for certificate secret", "name", certificateBackupName)

	restoredCertificate := SecretChange{
		Certificate:    backup.Data[certFileName],
		CertificateKey: backup.Data[keyFileName],
	}

	err = c.Update(ctx, restoredCertificate)
	if err != nil {
		return fmt.Errorf("failed to update certificate: %w", err)
	}

	slog.Debug("Updated certificate with backup")

	return nil
}

func (c *ecosystemCertificate) Update(ctx context.Context, tlsKeyPair SecretChange) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		return c.update(ctx, ecosystemCertificateName, tlsKeyPair)
	})
}

func (c *ecosystemCertificate) update(ctx context.Context, secretName string, tlsKeyPair SecretChange) error {
	certificateSecret, err := c.repo.Get(ctx, secretName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get secret for certificate: %w", err)
	}

	certificateSecret.StringData[certFileName] = string(tlsKeyPair.Certificate)
	certificateSecret.StringData[keyFileName] = string(tlsKeyPair.CertificateKey)

	_, err = c.repo.Update(ctx, certificateSecret, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update secret for certificate: %w", err)
	}

	return nil
}

func (c *ecosystemCertificate) DeleteBackup(ctx context.Context) error {
	err := retry.OnError(retry.DefaultRetry, retriable, func() error {
		return c.repo.Delete(ctx, certificateBackupName, metav1.DeleteOptions{})
	})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}

		return fmt.Errorf("failed to delete secret for certificate backup: %w", err)
	}

	return nil
}
