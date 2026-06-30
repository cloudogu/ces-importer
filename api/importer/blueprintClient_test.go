package importer

import (
	"context"
	"testing"
	"time"

	blueprintv3 "github.com/cloudogu/k8s-blueprint-lib/v3/api/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
)

var blueprintNotFoundErr = apierrors.NewNotFound(schema.GroupResource{Resource: "blueprints"}, "not-found")

func TestNewBlueprintDeploymentClient(t *testing.T) {
	client := NewBlueprintControl(nil)

	require.NotNil(t, client)
}

func Test_blueprintClient_StopBlueprint(t *testing.T) {
	testCtx, _ = context.WithTimeout(context.Background(), 1*time.Second)
	t.Run("should stop blueprint", func(t *testing.T) {
		// given
		stoppedA := false
		v3BlueprintA := blueprintv3.Blueprint{
			ObjectMeta: metav1.ObjectMeta{Name: "Blueprint A"},
			Spec:       blueprintv3.BlueprintSpec{Stopped: &stoppedA},
		}
		stoppedB := false
		v3BlueprintB := blueprintv3.Blueprint{
			ObjectMeta: metav1.ObjectMeta{Name: "Blueprint B"},
			Spec:       blueprintv3.BlueprintSpec{Stopped: &stoppedB},
		}

		blueprintCli := NewMockBlueprintInterface(t)
		blueprintCli.EXPECT().List(testCtx, mock.Anything).Return(&blueprintv3.BlueprintList{Items: []blueprintv3.Blueprint{v3BlueprintA, v3BlueprintB}}, nil)
		blueprintCli.EXPECT().Get(testCtx, "Blueprint A", mock.Anything).Return(&v3BlueprintA, nil)
		blueprintCli.EXPECT().Get(testCtx, "Blueprint B", mock.Anything).Return(&v3BlueprintB, nil)
		blueprintCli.EXPECT().Patch(testCtx, "Blueprint A", types.MergePatchType, patchBlueprintStop, mock.Anything).Return(&v3BlueprintA, nil)
		blueprintCli.EXPECT().Patch(testCtx, "Blueprint B", types.MergePatchType, patchBlueprintStop, mock.Anything).Return(&v3BlueprintB, nil)

		sut := NewBlueprintControl(blueprintCli)

		// when
		err := sut.StopBlueprint(testCtx)

		// then
		require.NoError(t, err)
		assert.Equal(t, 2, len(sut.stoppedBlueprints))
		assert.Equal(t, "Blueprint A", sut.stoppedBlueprints[0])
		assert.Equal(t, "Blueprint B", sut.stoppedBlueprints[1])
	})
	t.Run("should fail to stop blueprint for error in list", func(t *testing.T) {
		// given
		blueprintCli := NewMockBlueprintInterface(t)
		blueprintCli.EXPECT().List(testCtx, mock.Anything).Return(nil, assert.AnError)

		sut := NewBlueprintControl(blueprintCli)

		// when
		err := sut.StopBlueprint(testCtx)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to list all blueprints:")
	})
	t.Run("should fail to stop all dogus for error in startStop", func(t *testing.T) {
		// given
		// given
		stoppedA := false
		v3BlueprintA := blueprintv3.Blueprint{
			ObjectMeta: metav1.ObjectMeta{Name: "Blueprint A"},
			Spec:       blueprintv3.BlueprintSpec{Stopped: &stoppedA},
		}
		stoppedB := false
		v3BlueprintB := blueprintv3.Blueprint{
			ObjectMeta: metav1.ObjectMeta{Name: "Blueprint B"},
			Spec:       blueprintv3.BlueprintSpec{Stopped: &stoppedB},
		}

		blueprintCli := NewMockBlueprintInterface(t)
		blueprintCli.EXPECT().List(testCtx, mock.Anything).Return(&blueprintv3.BlueprintList{Items: []blueprintv3.Blueprint{v3BlueprintA, v3BlueprintB}}, nil)
		blueprintCli.EXPECT().Get(testCtx, "Blueprint A", mock.Anything).Return(&v3BlueprintA, assert.AnError)

		sut := NewBlueprintControl(blueprintCli)

		// when
		err := sut.StopBlueprint(testCtx)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to stop blueprint: failed to get blueprint Blueprint A:")
	})
	t.Run("should skip already stopped blueprint", func(t *testing.T) {
		// given
		stoppedA := false
		v3BlueprintA := blueprintv3.Blueprint{
			ObjectMeta: metav1.ObjectMeta{Name: "Blueprint A"},
			Spec:       blueprintv3.BlueprintSpec{Stopped: &stoppedA},
		}
		stoppedB := true
		v3BlueprintB := blueprintv3.Blueprint{
			ObjectMeta: metav1.ObjectMeta{Name: "Blueprint B"},
			Spec:       blueprintv3.BlueprintSpec{Stopped: &stoppedB},
		}

		blueprintCli := NewMockBlueprintInterface(t)
		blueprintCli.EXPECT().List(testCtx, mock.Anything).Return(&blueprintv3.BlueprintList{Items: []blueprintv3.Blueprint{v3BlueprintA, v3BlueprintB}}, nil)
		blueprintCli.EXPECT().Get(testCtx, "Blueprint A", mock.Anything).Return(&v3BlueprintA, nil)
		blueprintCli.EXPECT().Get(testCtx, "Blueprint B", mock.Anything).Return(&v3BlueprintB, nil)
		blueprintCli.EXPECT().Patch(testCtx, "Blueprint A", types.MergePatchType, patchBlueprintStop, mock.Anything).Return(&v3BlueprintA, nil)

		sut := NewBlueprintControl(blueprintCli)

		// when
		err := sut.StopBlueprint(testCtx)

		// then
		require.NoError(t, err)
		assert.Equal(t, 1, len(sut.stoppedBlueprints))
		assert.Equal(t, "Blueprint A", sut.stoppedBlueprints[0])
	})
	t.Run("should fail on getting blueprint", func(t *testing.T) {
		// given
		stoppedA := false
		v3BlueprintA := blueprintv3.Blueprint{
			ObjectMeta: metav1.ObjectMeta{Name: "Blueprint A"},
			Spec:       blueprintv3.BlueprintSpec{Stopped: &stoppedA},
		}
		stoppedB := true
		v3BlueprintB := blueprintv3.Blueprint{
			ObjectMeta: metav1.ObjectMeta{Name: "Blueprint B"},
			Spec:       blueprintv3.BlueprintSpec{Stopped: &stoppedB},
		}

		blueprintCli := NewMockBlueprintInterface(t)
		blueprintCli.EXPECT().List(testCtx, mock.Anything).Return(&blueprintv3.BlueprintList{Items: []blueprintv3.Blueprint{v3BlueprintA, v3BlueprintB}}, nil)
		blueprintCli.EXPECT().Get(testCtx, "Blueprint A", mock.Anything).Return(&v3BlueprintA, assert.AnError)

		sut := NewBlueprintControl(blueprintCli)

		// when
		err := sut.StopBlueprint(testCtx)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to get blueprint Blueprint A")
	})
	t.Run("should fail with no found blueprint", func(t *testing.T) {
		// given
		stoppedA := false
		v3BlueprintA := blueprintv3.Blueprint{
			ObjectMeta: metav1.ObjectMeta{Name: "Blueprint A"},
			Spec:       blueprintv3.BlueprintSpec{Stopped: &stoppedA},
		}

		blueprintCli := NewMockBlueprintInterface(t)
		blueprintCli.EXPECT().List(testCtx, mock.Anything).Return(&blueprintv3.BlueprintList{Items: []blueprintv3.Blueprint{v3BlueprintA}}, nil)
		blueprintCli.EXPECT().Get(testCtx, "Blueprint A", mock.Anything).Return(&v3BlueprintA, blueprintNotFoundErr)

		sut := NewBlueprintControl(blueprintCli)

		// when
		err := sut.StopBlueprint(testCtx)

		// then
		require.NoError(t, err)
		assert.Equal(t, 0, len(sut.stoppedBlueprints))
	})
	t.Run("should fail on update", func(t *testing.T) {
		// given
		stoppedA := false
		v3BlueprintA := blueprintv3.Blueprint{
			ObjectMeta: metav1.ObjectMeta{Name: "Blueprint A"},
			Spec:       blueprintv3.BlueprintSpec{Stopped: &stoppedA},
		}

		blueprintCli := NewMockBlueprintInterface(t)
		blueprintCli.EXPECT().List(testCtx, mock.Anything).Return(&blueprintv3.BlueprintList{Items: []blueprintv3.Blueprint{v3BlueprintA}}, nil)
		blueprintCli.EXPECT().Get(testCtx, "Blueprint A", mock.Anything).Return(&v3BlueprintA, nil)
		blueprintCli.EXPECT().Patch(testCtx, "Blueprint A", types.MergePatchType, patchBlueprintStop, mock.Anything).Return(&v3BlueprintA, assert.AnError)

		sut := NewBlueprintControl(blueprintCli)

		// when
		err := sut.StopBlueprint(testCtx)

		// then
		require.Error(t, err)
		assert.Equal(t, 0, len(sut.stoppedBlueprints))
		assert.ErrorContains(t, err, "failed to patch blueprint Blueprint A (shouldStop: true)")
	})
}

func Test_blueprintClient_StartBlueprint(t *testing.T) {
	testCtx, _ = context.WithTimeout(context.Background(), 1*time.Second)
	t.Run("should start blueprint", func(t *testing.T) {
		// given
		stoppedA := true
		v3BlueprintA := blueprintv3.Blueprint{
			ObjectMeta: metav1.ObjectMeta{Name: "Blueprint A"},
			Spec:       blueprintv3.BlueprintSpec{Stopped: &stoppedA},
		}
		stoppedB := true
		v3BlueprintB := blueprintv3.Blueprint{
			ObjectMeta: metav1.ObjectMeta{Name: "Blueprint B"},
			Spec:       blueprintv3.BlueprintSpec{Stopped: &stoppedB},
		}

		blueprintCli := NewMockBlueprintInterface(t)
		blueprintCli.EXPECT().Get(testCtx, "Blueprint A", mock.Anything).Return(&v3BlueprintA, nil)
		blueprintCli.EXPECT().Get(testCtx, "Blueprint B", mock.Anything).Return(&v3BlueprintB, nil)
		blueprintCli.EXPECT().Patch(testCtx, "Blueprint A", types.MergePatchType, patchBlueprintStart, mock.Anything).Return(&v3BlueprintA, nil)
		blueprintCli.EXPECT().Patch(testCtx, "Blueprint B", types.MergePatchType, patchBlueprintStart, mock.Anything).Return(&v3BlueprintA, nil)

		sut := NewBlueprintControl(blueprintCli)
		sut.stoppedBlueprints = []string{"Blueprint A", "Blueprint B"}

		// when
		err := sut.StartBlueprint(testCtx)

		// then
		require.NoError(t, err)
	})
	t.Run("should fail on starting blueprint", func(t *testing.T) {
		// given
		stoppedA := true
		v3BlueprintA := blueprintv3.Blueprint{
			ObjectMeta: metav1.ObjectMeta{Name: "Blueprint A"},
			Spec:       blueprintv3.BlueprintSpec{Stopped: &stoppedA},
		}

		blueprintCli := NewMockBlueprintInterface(t)
		blueprintCli.EXPECT().Get(testCtx, "Blueprint A", mock.Anything).Return(&v3BlueprintA, assert.AnError)

		sut := NewBlueprintControl(blueprintCli)
		sut.stoppedBlueprints = []string{"Blueprint A", "Blueprint B"}

		// when
		err := sut.StartBlueprint(testCtx)

		// then
		require.Error(t, err)
	})
}
