package migration

import (
	"fmt"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

const (
	deleteConfigMapTimeoutSeconds     = 10
	deleteConfigMapPollIntervalMillis = 200
)

var WaitForDeletion = waitForDeletion

func waitForDeletion(check func() error) error {
	timeout := time.After(deleteConfigMapTimeoutSeconds * time.Second)

	for {
		if apierrors.IsNotFound(check()) {
			break
		}

		select {
		case <-timeout:
			return fmt.Errorf("timeout waiting for deletion")
		case <-time.After(deleteConfigMapPollIntervalMillis * time.Millisecond):
		}
	}
	return nil
}
