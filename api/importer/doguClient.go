package importer

import (
	"context"
	"fmt"
	"log/slog"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	cescommons "github.com/cloudogu/ces-commons-lib/dogu"
	doguV2 "github.com/cloudogu/k8s-dogu-operator/v2/api/v2"

	"github.com/cloudogu/ces-importer/api/exporter"
)

type DoguInterface interface {
	// Get returns a single dogu CR if it exists in the k8s cluster.
	Get(ctx context.Context, name string, opts metav1.GetOptions) (*doguV2.Dogu, error)
	// UpdateSpecWithRetry tries to update the provided dogu with the given update function and returns the updated
	// copy. If a conflict happens, the update will be retried with the same function.
	UpdateSpecWithRetry(ctx context.Context, dogu *doguV2.Dogu, updateFunc func(spec doguV2.DoguSpec) doguV2.DoguSpec, opts metav1.UpdateOptions) (*doguV2.Dogu, error)
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
	err := dc.scaleDogu(ctx, dogu, true)
	if err != nil {
		return fmt.Errorf("failed to stop dogu: %w", err)
	}

	return nil
}

// StartDogu starts the given dogu in the importer system by scaling up the deployment.
func (dc *doguClient) StartDogu(ctx context.Context, dogu exporter.Dogu) error {
	err := dc.scaleDogu(ctx, dogu, false)
	if err != nil {
		return fmt.Errorf("failed to start dogu: %w", err)
	}

	return nil
}

func (dc *doguClient) scaleDogu(ctx context.Context, exporterDogu exporter.Dogu, shouldStop bool) error {
	fullyQualifiedDoguName, err := cescommons.QualifiedNameFromString(exporterDogu.Name)
	if err != nil {
		return err
	}

	doguName := fullyQualifiedDoguName.SimpleName.String()

	dogu, err := dc.doguCli.Get(ctx, doguName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			slog.Warn("Cannot start/stop dogu because it does not exist", "dogu", fullyQualifiedDoguName)
			return nil // if there is no longer a deployment, there is no longer a problem ¯\_(ツ)_/¯
		}
		return fmt.Errorf("failed to get dogu %s: %w", fullyQualifiedDoguName, err)
	}

	if dogu.Spec.Stopped == shouldStop {
		return nil
	}

	_, err = dc.doguCli.UpdateSpecWithRetry(ctx, dogu, func(spec doguV2.DoguSpec) doguV2.DoguSpec {
		spec.Stopped = shouldStop
		return spec
	}, metav1.UpdateOptions{})

	if err != nil {
		return fmt.Errorf("failed to update dogu %s (shouldStop: %t): %w", fullyQualifiedDoguName, shouldStop, err)
	}

	return nil
}
