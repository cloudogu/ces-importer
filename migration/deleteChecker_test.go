package migration

import (
	"fmt"
	"testing"
	"time"

	cloudoguerrors "github.com/cloudogu/ces-commons-lib/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func newNotFoundError() error {
	return apierrors.NewNotFound(schema.GroupResource{Group: "", Resource: "configmaps"}, "test")
}

func Test_waitForDeletion(t *testing.T) {
	// speed up polling and timeout so the tests run fast
	oldTimeout := deleteConfigMapTimeout
	oldPollInterval := deleteConfigMapPollInterval
	defer func() {
		deleteConfigMapTimeout = oldTimeout
		deleteConfigMapPollInterval = oldPollInterval
	}()
	deleteConfigMapTimeout = 100 * time.Millisecond
	deleteConfigMapPollInterval = 5 * time.Millisecond

	t.Run("returns nil when the resource is already gone", func(t *testing.T) {
		calls := 0
		check := func() error {
			calls++
			return newNotFoundError()
		}

		err := waitForDeletion(check)

		require.NoError(t, err)
		assert.Equal(t, 1, calls)
	})

	t.Run("returns nil once the resource becomes gone", func(t *testing.T) {
		calls := 0
		check := func() error {
			calls++
			if calls < 3 {
				return nil
			}
			return newNotFoundError()
		}

		err := waitForDeletion(check)

		require.NoError(t, err)
		assert.Equal(t, 3, calls)
	})

	t.Run("times out when the resource never gets deleted", func(t *testing.T) {
		check := func() error {
			return nil
		}

		err := waitForDeletion(check)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "timeout waiting for deletion")
	})

	t.Run("times out when check keeps returning a non-NotFound error", func(t *testing.T) {
		check := func() error {
			return fmt.Errorf("some other error")
		}

		err := waitForDeletion(check)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "timeout waiting for deletion")
	})

	t.Run("returns nil for a NotFound status error variant", func(t *testing.T) {
		check := func() error {
			return &apierrors.StatusError{ErrStatus: metav1.Status{
				Reason: metav1.StatusReasonNotFound,
			}}
		}

		err := waitForDeletion(check)

		require.NoError(t, err)
	})

	t.Run("returns nil for a cloudogu config-lib NotFound error", func(t *testing.T) {
		calls := 0
		check := func() error {
			calls++
			return cloudoguerrors.NewNotFoundError(fmt.Errorf("could not find a configmap with the given name: global-config"))
		}

		err := waitForDeletion(check)

		require.NoError(t, err)
		assert.Equal(t, 1, calls)
	})
}

func TestWaitForDeletion_isWaitForDeletion(t *testing.T) {
	// The exported var should point at the unexported implementation by default.
	assert.NotNil(t, WaitForDeletion)
}
