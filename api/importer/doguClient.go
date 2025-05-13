package importer

import (
	"context"
	"fmt"
	"log/slog"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	cescommons "github.com/cloudogu/ces-commons-lib/dogu"
	doguV2 "github.com/cloudogu/k8s-dogu-operator/v3/api/v2"
)

type DoguInterface interface {
	// List takes label and field selectors and returns the list of Dogus that match those selectors.
	List(ctx context.Context, opts metav1.ListOptions) (result *doguV2.DoguList, err error)
	// Get returns a single dogu CR if it exists in the k8s cluster.
	Get(ctx context.Context, name string, opts metav1.GetOptions) (*doguV2.Dogu, error)
	// UpdateSpecWithRetry tries to update the provided dogu with the given update function and returns the updated
	// copy. If a conflict happens, the update will be retried with the same function.
	UpdateSpecWithRetry(ctx context.Context, dogu *doguV2.Dogu, updateFunc func(spec doguV2.DoguSpec) doguV2.DoguSpec, opts metav1.UpdateOptions) (*doguV2.Dogu, error)
}

type doguClient struct {
	doguCli DoguInterface
}

// NewDoguClient creates a new client that operates on dogu deployments on the importer system.
func NewDoguClient(doguCli DoguInterface) *doguClient {
	return &doguClient{
		doguCli: doguCli,
	}
}

// StopAll stopps all dogus in the importer system.
func (dc *doguClient) StopAll(ctx context.Context) error {
	list, err := dc.doguCli.List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list all dogus: %w", err)
	}

	for _, dogu := range list.Items {
		err := dc.startStop(ctx, dogu.Name, true)
		if err != nil {
			return fmt.Errorf("failed to stop dogu: %w", err)
		}
	}

	return nil
}

// StopDogu stopps the given dogu in the importer system.
func (dc *doguClient) StopDogu(ctx context.Context, doguName string) error {
	err := dc.startStop(ctx, doguName, true)
	if err != nil {
		return fmt.Errorf("failed to stop dogu: %w", err)
	}

	return nil
}

// StartAll starts all dogus in the importer system.
func (dc *doguClient) StartAll(ctx context.Context) error {
	list, err := dc.doguCli.List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list all dogus: %w", err)
	}

	for _, dogu := range list.Items {
		err := dc.startStop(ctx, dogu.Name, false)
		if err != nil {
			return fmt.Errorf("failed to start dogu: %w", err)
		}
	}

	return nil
}

// StartDogu starts the given dogu in the importer system.
func (dc *doguClient) StartDogu(ctx context.Context, doguName string) error {
	err := dc.startStop(ctx, doguName, false)
	if err != nil {
		return fmt.Errorf("failed to start dogu: %w", err)
	}

	return nil
}

func (dc *doguClient) startStop(ctx context.Context, exporterDoguName string, shouldStop bool) error {
	fullyQualifiedDoguName, err := cescommons.QualifiedNameFromString(exporterDoguName)
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
