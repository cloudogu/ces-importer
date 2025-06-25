package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/cloudogu/ces-importer/api/exporter"
	"github.com/cloudogu/ces-importer/configuration"
	"github.com/cloudogu/ces-importer/systeminfo"
	"golang.org/x/crypto/ssh"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"log/slog"
	"time"
)

const (
	exporterSSHPort = "7022"
)

// runPreflightCheck checks the most important services and configurations before the migration takes place.
// Checks:
//
// * health status of the exporter
//
// * ssh connection to the exporting system
//
// * access to the importing systems k8s resources
func runPreflightCheck(ctx context.Context, service *exporter.Service, cfg configuration.Coordinator, systemInfoProvider *systeminfo.Provider, secretClient corev1.SecretInterface) error {
	var result error
	slog.Info("Running preflight migration checks:")
	// check api
	healthy, exporterApiErr := service.HealthService.GetIsHealthy(ctx)
	if exporterApiErr != nil || !healthy {
		result = errors.Join(result, fmt.Errorf("unable to determine exporter health status: %w", exporterApiErr))
	} else {
		slog.Info("Successfully reached exporter api")
	}

	sshError := testSSHConnection(ctx, cfg, secretClient)
	if sshError != nil {
		result = errors.Join(result, fmt.Errorf("unable to test ssh connection: %w", sshError))
	} else {
		slog.Info("Successfully connected to the exporter via ssh")
	}

	// check k8s access
	_, sysInfoErr := systemInfoProvider.GetImporterSystemInfo(ctx)
	if sysInfoErr != nil {
		result = errors.Join(result, fmt.Errorf("unable to retrieve current systems system info: %w", sysInfoErr))
	} else {
		slog.Info("Successfully retrieved system information")
	}
	return result
}

// testSSHConnection creates an ssh connection to the exporting system and performs an echo command to test the connection
func testSSHConnection(_ context.Context, cfg configuration.Coordinator, secretClient corev1.SecretInterface) error {
	// get ssh private key from k8s secret
	secret, err := secretClient.Get(context.Background(), cfg.SSH.SecretName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("could not get secret '%s': %w", cfg.SSH.SecretName, err)
	}
	val, ok := secret.Data[cfg.SSH.SecretDataKey]
	if !ok {
		return fmt.Errorf("secret '%s' does not contain data key '%s': %w", cfg.SSH.SecretName, cfg.SSH.SecretDataKey, err)
	}
	signer, err := ssh.ParsePrivateKey(val)
	if err != nil {
		return fmt.Errorf("could not parse private ssh key: %w", err)
	}

	config := &ssh.ClientConfig{
		User: cfg.SSH.User,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         5 * time.Second,
	}

	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:%s", cfg.API.ExporterHost, exporterSSHPort), config)
	if err != nil {
		return fmt.Errorf("could not open ssh connection to exporter: %w", err)
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("could not create ssh session: %w", err)
	}
	defer session.Close()

	_, err = session.CombinedOutput("echo 'Importer was able to connect successfully via ssh'")
	if err != nil {
		return fmt.Errorf("could not execute echo command on the exporter: %w", err)
	}
	return nil
}
