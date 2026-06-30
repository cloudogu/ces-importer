package importer

import (
	"context"
	"fmt"
	"log/slog"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"

	blueprintv3 "github.com/cloudogu/k8s-blueprint-lib/v3/api/v3"
)

type BlueprintInterface interface {
	// Get takes name of the blueprint, and returns the corresponding blueprint object, and an error if there is any.
	Get(ctx context.Context, name string, opts metav1.GetOptions) (*blueprintv3.Blueprint, error)
	// List takes label and field selectors, and returns the list of Blueprints that match those selectors.
	List(ctx context.Context, opts metav1.ListOptions) (*blueprintv3.BlueprintList, error)
	// Patch applies the patch and returns the patched blueprint.
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *blueprintv3.Blueprint, err error)
}

type BlueprintControl struct {
	blueprintCli      BlueprintInterface
	stoppedBlueprints []string
}

var (
	patchBlueprintStop  = []byte(`{"spec":{"stopped":true}}`)
	patchBlueprintStart = []byte(`{"spec":{"stopped":false}}`)
)

// NewBlueprintControl creates a new client that operates on dogu deployments on the importer system.
func NewBlueprintControl(blueprintCli BlueprintInterface) *BlueprintControl {
	return &BlueprintControl{
		blueprintCli:      blueprintCli,
		stoppedBlueprints: []string{},
	}
}

// StopBlueprint stopps all blueprints in the importer system.
func (bc *BlueprintControl) StopBlueprint(ctx context.Context) error {
	slog.Info("Stopping all blueprints")
	list, err := bc.blueprintCli.List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list all blueprints: %w", err)
	}
	for _, blueprint := range list.Items {
		changed, err := bc.startStop(ctx, blueprint.Name, true)
		if err != nil {
			return fmt.Errorf("failed to stop blueprint: %w", err)
		}
		if changed {
			bc.stoppedBlueprints = append(bc.stoppedBlueprints, blueprint.Name)
		}
	}
	slog.Debug("Received list with blueprints", "length", len(list.Items))
	return nil
}

// StartBlueprint start all blueprints, which are stopped by this BlueprintControl
func (bc *BlueprintControl) StartBlueprint(ctx context.Context) error {
	slog.Info("Starting all blueprints")
	for _, blueprintName := range bc.stoppedBlueprints {
		_, err := bc.startStop(ctx, blueprintName, false)
		if err != nil {
			return fmt.Errorf("failed to start blueprint: %w", err)
		}
	}
	// If all blueprints have been started or the blueprints to be started no longer exist, clean up the list.
	bc.stoppedBlueprints = bc.stoppedBlueprints[:0]
	return nil
}

// Helper for start or stop blueprint, do nothing if desired state matches current-state
func (bc *BlueprintControl) startStop(ctx context.Context, blueprintName string, shouldStop bool) (bool, error) {
	blueprint, err := bc.blueprintCli.Get(ctx, blueprintName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			slog.Warn("Cannot start/stop blueprint because it does not exist", "blueprint", blueprintName)
			return false, nil // if there is no longer a deployment, there is no longer a problem ¯\_(ツ)_/¯
		}
		return false, fmt.Errorf("failed to get blueprint %s: %w", blueprintName, err)
	}

	if ptr.Deref(blueprint.Spec.Stopped, false) == shouldStop {
		return false, nil
	}

	patchData := patchBlueprintStart
	if shouldStop {
		patchData = patchBlueprintStop
	}

	_, err = bc.blueprintCli.Patch(ctx, blueprintName, types.MergePatchType, patchData, metav1.PatchOptions{})
	if err != nil {
		return false, fmt.Errorf("failed to patch blueprint %s (shouldStop: %t): %w", blueprintName, shouldStop, err)
	}

	return true, nil
}
