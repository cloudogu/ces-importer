package fqdn

import (
	"context"
	"fmt"
	regConfig "github.com/cloudogu/k8s-registry-lib/config"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
	"log/slog"
)

const (
	fqdnBackupConfigMapName = "ecosystem-fqdn-backup"
	fqdnKey                 = "fqdn"
	certTypeKey             = "certificate-type"
)

type fqdnManager struct {
	repo             configMapRepository
	globalConfigRepo globalConfigRepository
}

// ConfigChange represents the fqdn and certificate type that should be changed.
type ConfigChange struct {
	FQDN     string
	CertType string
}

func createFQDNBackup(globalConfig regConfig.GlobalConfig) (ConfigChange, error) {
	fqdn, ok := globalConfig.Get(globalCfgFQDNKey)
	if !ok {
		return ConfigChange{}, fmt.Errorf("could not find fqdn in global config")
	}

	certType, ok := globalConfig.Get(globalCfgCertTypeKey)
	if !ok {
		return ConfigChange{}, fmt.Errorf("could not find certificate/type in global config")
	}

	return ConfigChange{
		FQDN:     fqdn.String(),
		CertType: certType.String(),
	}, nil
}

func createFQDNBackupConfigMap(backup ConfigChange) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: fqdnBackupConfigMapName},
		Data: map[string]string{
			fqdnKey:     backup.FQDN,
			certTypeKey: backup.CertType,
		},
	}
}

// Backup creates a backup of the current fqdn and certificate type from the global config.
func (f *fqdnManager) Backup(ctx context.Context) error {
	return retry.OnError(retry.DefaultRetry, retriable, func() error {
		return f.backup(ctx)
	})
}

func (f *fqdnManager) backup(ctx context.Context) error {
	slog.Debug("Backing up FQDN")

	globalConfigMap, err := f.globalConfigRepo.Get(ctx)
	if err != nil {
		return fmt.Errorf("could not get global config: %w", err)
	}

	slog.Debug("Got global config map")

	fqdnBackup, err := createFQDNBackup(globalConfigMap)
	if err != nil {
		return fmt.Errorf("could not create fqdn backup from global config: %w", err)
	}

	_, err = f.repo.Create(ctx, createFQDNBackupConfigMap(fqdnBackup), metav1.CreateOptions{})
	if err != nil {
		if apierrors.IsAlreadyExists(err) {
			slog.Debug("ConfigChange for fqdn already exists, updating")

			uppErr := f.updateBackup(ctx, fqdnBackup)
			if uppErr != nil {
				return fmt.Errorf("failed to update backup for fqdn: %w", uppErr)
			}

			slog.Debug("Updated backup for fqdn")

			return nil
		}

		return fmt.Errorf("failed to create config map for fqdn backup: %w", err)
	}

	slog.Debug("Created backup for fqdn", "name", fqdnBackupConfigMapName)

	return nil
}

func (f *fqdnManager) updateBackup(ctx context.Context, b ConfigChange) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		backupCM, err := f.repo.Get(ctx, fqdnBackupConfigMapName, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("could not get backup config map for fqdn: %w", err)
		}

		backupCM.Data[certTypeKey] = b.CertType
		backupCM.Data[fqdnKey] = b.FQDN

		_, err = f.repo.Update(ctx, backupCM, metav1.UpdateOptions{})
		if err != nil {
			return fmt.Errorf("failed to update config map for fqdn: %w", err)
		}

		return nil
	})
}

// Restore restores the fqdn and certificate type from the backup.
func (f *fqdnManager) Restore(ctx context.Context) error {
	return retry.OnError(retry.DefaultRetry, retriable, func() error {
		return f.restore(ctx)
	})
}

func (f *fqdnManager) restore(ctx context.Context) error {
	slog.Debug("Restoring fqdn from backup")

	backup, err := f.repo.Get(ctx, fqdnBackupConfigMapName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get config mapo for fqdn backup: %w", err)
	}

	slog.Debug("Got backup for fqdn", "name", fqdnBackupConfigMapName)

	change := ConfigChange{
		FQDN:     backup.Data[fqdnKey],
		CertType: backup.Data[certTypeKey],
	}

	err = f.Update(ctx, change)
	if err != nil {
		return fmt.Errorf("failed to update fqdn: %w", err)
	}

	slog.Debug("Updated fqdn from backup")

	return nil
}

// Update updates the fqdn and certificate type in the global config.
func (f *fqdnManager) Update(ctx context.Context, c ConfigChange) error {
	slog.Debug("Updating FQDN")

	globalConfig, err := f.globalConfigRepo.Get(ctx)
	if err != nil {
		return fmt.Errorf("could not get global config: %w", err)
	}

	globalConfig.Config, err = globalConfig.Set(globalCfgFQDNKey, regConfig.Value(c.FQDN))
	if err != nil {
		return fmt.Errorf("failed to set new fqdn in global config: %w", err)
	}

	globalConfig.Config, err = globalConfig.Set(globalCfgCertTypeKey, regConfig.Value(c.CertType))
	if err != nil {
		return fmt.Errorf("failed to set new cert type in global config: %w", err)
	}

	slog.Debug("Set new fqdn in global config")

	_, err = f.globalConfigRepo.SaveOrMerge(ctx, globalConfig)
	if err != nil {
		return fmt.Errorf("failed to save global config: %w", err)
	}

	slog.Debug("Updated global config")

	return nil
}

// DeleteBackup deletes the backup of the fqdn and certificate type.
func (f *fqdnManager) DeleteBackup(ctx context.Context) error {
	err := retry.OnError(retry.DefaultRetry, retriable, func() error {
		return f.repo.Delete(ctx, fqdnBackupConfigMapName, metav1.DeleteOptions{})
	})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}

		return fmt.Errorf("failed to delete config map for fqdn backup: %w", err)
	}

	return nil
}
