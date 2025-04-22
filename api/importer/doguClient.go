package importer

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	v1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/retry"
	"k8s.io/utils/ptr"

	cescommons "github.com/cloudogu/ces-commons-lib/dogu"

	"github.com/cloudogu/ces-importer/api/exporter"
)

type clientSet interface {
	kubernetes.Interface
}

// DoguStopper provides functions to stop a running dogu.
type DoguStopper interface {
	// StopDogu stopps the given dogu in the importer system. An error is expected if the dogu is in a non-healthy
	// condition except the dogu is already stopped.
	StopDogu(ctx context.Context, dogu exporter.Dogu) error
}

// DoguStarter provides functions to start a stopped dogu.
type DoguStarter interface {
	// StartDogu starts the given dogu in the importer system. An error is expected if the dogu is in a non-healthy
	// condition except when the dogu is stopped.
	StartDogu(ctx context.Context, dogu exporter.Dogu) error
}

type doguClient struct {
	k8sClientSet clientSet
	Namespace    string
}

// NewDoguDeploymentClient creates a new client that operates on dogu deployments on the importer system.
func NewDoguDeploymentClient(k8sClientSet clientSet, importerNamespace string) *doguClient {
	return &doguClient{
		k8sClientSet: k8sClientSet,
		Namespace:    importerNamespace,
	}
}

// StopDogu stopps the given dogu in the importer system by scaling down the deployment.
func (dc *doguClient) StopDogu(ctx context.Context, dogu exporter.Dogu) error {
	fullyQualifiedDoguName, err := cescommons.QualifiedNameFromString(dogu.Name)
	if err != nil {
		return fmt.Errorf("failed to stop dogu: %w", err)
	}

	doguName := fullyQualifiedDoguName.SimpleName.String()

	_, err = dc.scaleDeployment(ctx, doguName, 0)
	if err != nil {
		return fmt.Errorf("failed to scale down dogu deployment for dogu %q: %w", doguName, err)
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

	_, err = dc.scaleDeployment(ctx, doguName, 1)
	if err != nil {
		return fmt.Errorf("failed to start dogu deployment for dogu %q: %w", doguName, err)
	}

	return nil
}

func (dc *doguClient) getDeploymentByName(ctx context.Context, simpleDeploymentName string) (found bool, deployment *v1.Deployment, err error) {
	deployment, err = dc.k8sClientSet.
		AppsV1().Deployments(dc.Namespace).
		Get(ctx, simpleDeploymentName, metav1.GetOptions{})

	if err != nil {
		if errors.IsNotFound(err) {
			return false, nil, nil
		}
		return false, nil, fmt.Errorf("failed to fetch deployment for dogu %q: %w", simpleDeploymentName, err)
	}

	return true, deployment, nil
}

func (dc *doguClient) scaleDeployment(ctx context.Context, deployName string, newReplicas int32) (prevReplicas int32, err error) {
	prevReplicas = -1

	conflictBackoff := wait.Backoff{
		Duration: 1500 * time.Millisecond,
		Factor:   1.5,
		Jitter:   0,
		Steps:    9999,
		Cap:      30 * time.Second,
	}

	var currentDeployment *v1.Deployment
	err = retry.RetryOnConflict(conflictBackoff, func() error {
		found := false
		found, currentDeployment, err = dc.getDeploymentByName(ctx, deployName)
		if err != nil {
			return fmt.Errorf("failed to get deployment %q for scaling update: %w", deployName, err)
		}

		if !found {
			slog.Log(ctx, slog.LevelWarn, "Cannot scale down dogu deployment because it does not exist", "dogu", deployName)
			return nil // if there is no longer a deployment, there is no longer a problem ¯\_(ツ)_/¯
		}

		prevReplicas = *currentDeployment.Spec.Replicas

		newReplicasPtr := ptr.To(newReplicas)
		if *currentDeployment.Spec.Replicas == *newReplicasPtr {
			slog.Log(ctx, slog.LevelWarn, "Could not scale deployment because it already was at the target value", "deployment", deployName, "replicas", newReplicas)
			return nil
		}

		currentDeployment.Spec.Replicas = newReplicasPtr

		_, err = dc.k8sClientSet.AppsV1().Deployments(dc.Namespace).Update(ctx, currentDeployment, metav1.UpdateOptions{})
		if err != nil {
			return fmt.Errorf("failed to update deployment %q for scaling: %w", deployName, err)
		}

		return nil
	})

	if err != nil {
		return prevReplicas, err
	}

	return prevReplicas, nil
}
