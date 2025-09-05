package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/cloudogu/ces-importer/configuration"
	"github.com/cloudogu/ces-importer/migration"
	"golang.org/x/crypto/ssh"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"log/slog"
	"time"
)

const (
	exporterSSHPort = "7022"
)

type secretClient interface {
	Get(ctx context.Context, name string, opts metav1.GetOptions) (*v1.Secret, error)
}

type systemInfoGetter interface {
	GetImporterSystemInfo(ctx context.Context) (*migration.SystemInfo, error)
}

type healthClient interface {
	GetIsHealthy(ctx context.Context) (bool, error)
}

type exportDoguClient interface {
	SetExportDogu(ctx context.Context, doguName string) (*migration.DoguExport, error)
}

type testSSHConnection func(ctx context.Context, cfg configuration.Coordinator, secretClient secretClient, exportDoguClient exportDoguClient) error

type PreflightExecuter struct {
	healthClient     healthClient
	exportDoguClient exportDoguClient
	systemInfoGetter systemInfoGetter
	secretClient     secretClient
	testSSHConnection
}

func newPreflightExecuter(healthClient healthClient, exportDoguClient exportDoguClient, systemInfoGetter systemInfoGetter, secretClient secretClient) *PreflightExecuter {
	return &PreflightExecuter{
		healthClient:      healthClient,
		exportDoguClient:  exportDoguClient,
		systemInfoGetter:  systemInfoGetter,
		secretClient:      secretClient,
		testSSHConnection: sshConnectionTest,
	}
}

// runPreflightCheck checks the most important services and configurations before the migration takes place.
// Checks:
//
// * health status of the exporter
//
// * ssh connection to the exporting system
//
// * access to the importing systems k8s resources
func (p *PreflightExecuter) runPreflightCheck(ctx context.Context, cfg configuration.Coordinator) error {
	var result error
	slog.Info("Running preflight migration checks:")
	// check api
	healthy, exporterApiErr := p.healthClient.GetIsHealthy(ctx)
	if exporterApiErr != nil {
		result = errors.Join(result, fmt.Errorf("unable to determine exporter health status: %w", exporterApiErr))
	} else if !healthy {
		result = errors.Join(result, fmt.Errorf("exporter health status is unhealthy"))
	} else {
		slog.Info("Successfully reached exporter api")
	}

	sshError := p.testSSHConnection(ctx, cfg, p.secretClient, p.exportDoguClient)
	if sshError != nil {
		result = errors.Join(result, fmt.Errorf("unable to test ssh connection: %w", sshError))
	} else {
		slog.Info("Successfully connected to the exporter via ssh")
	}

	// check k8s access
	_, sysInfoErr := p.systemInfoGetter.GetImporterSystemInfo(ctx)
	if sysInfoErr != nil {
		result = errors.Join(result, fmt.Errorf("unable to retrieve current systems system info: %w", sysInfoErr))
	} else {
		slog.Info("Successfully retrieved system information")
	}
	return result
}

// testSSHConnection creates an ssh connection to the exporting system and performs an echo command to test the connection
func sshConnectionTest(ctx context.Context, cfg configuration.Coordinator, secretClient secretClient, exportDoguClient exportDoguClient) error {
	// set an export dogu in the source instance
	doguName := "cas"
	_, err := exportDoguClient.SetExportDogu(ctx, doguName)
	if err != nil {
		slog.Warn(fmt.Sprintf("Could not set dogu %s as export dogu. If the exporting system is not a multinode system, this might lead to further errors. Otherwise this can be ignored.", doguName))
	}

	// get ssh private key from k8s secret
	secret, err := secretClient.Get(ctx, cfg.SSH.SecretName, metav1.GetOptions{})
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

	return nil
}
