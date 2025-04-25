package importer

import (
	"context"
	"fmt"
	"log/slog"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	cescommons "github.com/cloudogu/ces-commons-lib/dogu"
	"github.com/cloudogu/k8s-dogu-operator/v3/api/ecoSystem"
	doguV2 "github.com/cloudogu/k8s-dogu-operator/v3/api/v2"

	"github.com/cloudogu/ces-importer/api/exporter"
)

type DoguInterface interface {
	ecoSystem.DoguInterface
}

type doguClient struct {
	doguCli DoguInterface
}

// NewDoguDeploymentClient creates a new client that operates on dogu deployments on the importer system.
func NewDoguDeploymentClient(doguCli DoguInterface) *doguClient {
	return &doguClient{
		doguCli: doguCli,
	}
}

// StopDogu stopps the given dogu in the importer system by scaling down the deployment.
func (dc *doguClient) StopDogu(ctx context.Context, dogu exporter.Dogu) error {
	fullyQualifiedDoguName, err := cescommons.QualifiedNameFromString(dogu.Name)
	if err != nil {
		return fmt.Errorf("failed to stop dogu: %w", err)
	}

	doguName := fullyQualifiedDoguName.SimpleName.String()

	err = dc.scaleDogu(ctx, doguName, true)
	if err != nil {
		return fmt.Errorf("failed to stop dogu: %w", err)
	}

	return nil
}

// StartDogu starts the given dogu in the importer system by scaling up the deployment.
func (dc *doguClient) StartDogu(ctx context.Context, dogu exporter.Dogu) error {
	fullyQualifiedDoguName, err := cescommons.QualifiedNameFromString(dogu.Name)
	if err != nil {
		return fmt.Errorf("failed to start dogu: %w", err)
	}

	doguName := fullyQualifiedDoguName.SimpleName.String()

	err = dc.scaleDogu(ctx, doguName, false)
	if err != nil {
		return fmt.Errorf("failed to start dogu: %w", err)
	}

	return nil
}

func (dc *doguClient) getDoguByName(ctx context.Context, simpleDoguName string) (found bool, dogu *doguV2.Dogu, err error) {
	dogu, err = dc.doguCli.Get(ctx, simpleDoguName, metav1.GetOptions{})

	if err != nil {
		if errors.IsNotFound(err) {
			return false, nil, nil
		}
		return false, nil, fmt.Errorf("failed to fetch deployment for dogu %q: %w", simpleDoguName, err)
	}

	return true, dogu, nil
}

func (dc *doguClient) scaleDogu(ctx context.Context, doguName string, shouldStop bool) error {
	dogu, err := dc.doguCli.Get(ctx, doguName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			slog.Log(ctx, slog.LevelWarn, "Cannot start/stop dogu because it does not exist", "dogu", doguName)
			return nil // if there is no longer a deployment, there is no longer a problem ¯\_(ツ)_/¯
		}
		return fmt.Errorf("failed to get dogu %s: %w", doguName, err)
	}

	if dogu.Spec.Stopped == shouldStop {
		return nil
	}

	_, err = dc.doguCli.UpdateSpecWithRetry(ctx, dogu, func(spec doguV2.DoguSpec) doguV2.DoguSpec {
		spec.Stopped = shouldStop
		return spec
	}, metav1.UpdateOptions{})

	if err != nil {
		return fmt.Errorf("failed to update dogu %s (shouldStop: %t): %w", doguName, shouldStop, err)
	}

	return nil
}
