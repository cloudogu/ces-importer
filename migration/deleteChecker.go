package migration

import (
	"fmt"
	"time"

	cloudoguerrors "github.com/cloudogu/ces-commons-lib/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

var (
	deleteConfigMapTimeout      = 10 * time.Second
	deleteConfigMapPollInterval = 200 * time.Millisecond
)

var WaitForDeletion = waitForDeletion

func isNotFound(err error) bool {
	return apierrors.IsNotFound(err) || cloudoguerrors.IsNotFoundError(err)
}

func waitForDeletion(check func() error) error {
	timeout := time.After(deleteConfigMapTimeout)

	for !isNotFound(check()) {

		select {
		case <-timeout:
			return fmt.Errorf("timeout waiting for deletion")
		case <-time.After(deleteConfigMapPollInterval):
		}
	}
	return nil
}
