package importer

import (
	"context"
	"fmt"
	"log/slog"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	blueprintv2 "github.com/cloudogu/k8s-blueprint-lib/v2/api/v2"
)

type BlueprintInterface interface {
	// Get takes name of the blueprint, and returns the corresponding blueprint object, and an error if there is any.
	Get(ctx context.Context, name string, opts metav1.GetOptions) (*blueprintv2.Blueprint, error)
	// List takes label and field selectors, and returns the list of Blueprints that match those selectors.
	List(ctx context.Context, opts metav1.ListOptions) (*blueprintv2.BlueprintList, error)
	// Update takes the representation of a blueprint and updates it. Returns the server's representation of the blueprint, and an error, if there is any.
	Update(ctx context.Context, blueprint *blueprintv2.Blueprint, opts metav1.UpdateOptions) (*blueprintv2.Blueprint, error)
}

type BlueprintControl struct {
	blueprintCli      BlueprintInterface
	stoppedBlueprints []string
}

// NewDoguControl creates a new client that operates on dogu deployments on the importer system.
func NewBlueprintControl(blueprintCli BlueprintInterface) *BlueprintControl {
	return &BlueprintControl{
		blueprintCli:      blueprintCli,
		stoppedBlueprints: []string{},
	}
}

// StopAll stopps all dogus in the importer system.
func (dc *BlueprintControl) StopBlueprint(ctx context.Context) error {
	slog.Info("Stopping all blueprints")
	list, err := dc.blueprintCli.List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list all blueprints: %w", err)
	}
	for _, blueprint := range list.Items {
		b := true
		err := dc.startStop(ctx, blueprint.Name, &b)
		if err != nil {
			return fmt.Errorf("failed to stop blueprint: %w", err)
		}
		dc.stoppedBlueprints = append(dc.stoppedBlueprints, blueprint.Name)
	}
	slog.Debug("Received list with blueprints", "length", len(list.Items))
	return nil
}

// StartBlueprint start all blueprint, which are stopped by this BlueprintControl
func (dc *BlueprintControl) StartBlueprint(ctx context.Context) error {
	slog.Info("Starting all blueprints")
	for _, blueprintName := range dc.stoppedBlueprints {
		b := false
		err := dc.startStop(ctx, blueprintName, &b)
		if err != nil {
			return fmt.Errorf("failed to start blueprint: %w", err)
		}
	}
	return nil
}

func (dc *BlueprintControl) startStop(ctx context.Context, blueprintName string, shouldStop *bool) error {
	blueprint, err := dc.blueprintCli.Get(ctx, blueprintName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			slog.Warn("Cannot start/stop blueprint because it does not exist", "blueprint", blueprintName)
			return nil // if there is no longer a deployment, there is no longer a problem ¯\_(ツ)_/¯
		}
		return fmt.Errorf("failed to get blueprint %s: %w", blueprintName, err)
	}

	if blueprint.Spec.Stopped == shouldStop {
		return nil
	}

	blueprint.Spec.Stopped = shouldStop
	_, err = dc.blueprintCli.Update(ctx, blueprint, metav1.UpdateOptions{})

	if err != nil {
		return fmt.Errorf("failed to update blueprint %s (shouldStop: %t): %w", blueprintName, shouldStop, err)
	}

	return nil
}
