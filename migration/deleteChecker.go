package migration

import (
	"fmt"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

var (
	deleteConfigMapTimeout      = 10 * time.Second
	deleteConfigMapPollInterval = 200 * time.Millisecond
)

var WaitForDeletion = waitForDeletion

func waitForDeletion(check func() error) error {
	timeout := time.After(deleteConfigMapTimeout)

	for !apierrors.IsNotFound(check()) {

		select {
		case <-timeout:
			return fmt.Errorf("timeout waiting for deletion")
		case <-time.After(deleteConfigMapPollInterval):
		}
	}
	return nil
}
