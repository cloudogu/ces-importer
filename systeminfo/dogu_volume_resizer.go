package systeminfo

import (
	"context"
	"errors"
	"fmt"
	cescommons "github.com/cloudogu/ces-commons-lib/dogu"
	"github.com/cloudogu/ces-importer/migration"
	doguv2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"log/slog"
	"slices"
	"sync"
	"time"
)

const (
	defaultWaitSecondsBetweenRetries = 10
	defaultMaxWaitMinutes            = 10
)

var (
	waitSecondsBetweenRetries = defaultWaitSecondsBetweenRetries
	maxWaitMinutes            = defaultMaxWaitMinutes
	maxRetries                = (maxWaitMinutes * 60) / waitSecondsBetweenRetries
)

// client used for interacting with dogus
type doguClient interface {
	Get(ctx context.Context, name string, opts metav1.GetOptions) (*doguv2.Dogu, error)
	Update(ctx context.Context, dogu *doguv2.Dogu, opts metav1.UpdateOptions) (*doguv2.Dogu, error)
}

// client used for interacting with persistent volume claims
type pvcClient interface {
	Get(ctx context.Context, name string, opts metav1.GetOptions) (*corev1.PersistentVolumeClaim, error)
}

type DoguVolumeResizer struct {
	doguClient    doguClient
	pvcClient     pvcClient
	excludedDogus []string
}

func NewDoguVolumeResizer(doguClient doguClient, pvcCLient pvcClient, excludedDogus []string) *DoguVolumeResizer {
	return &DoguVolumeResizer{
		doguClient:    doguClient,
		pvcClient:     pvcCLient,
		excludedDogus: append(excludedDogus, doguNginx),
	}
}

func (d *DoguVolumeResizer) ResizeDogusIfNeeded(ctx context.Context, exporterDogus []migration.Dogu, importerDogus []migration.Dogu) error {
	var wg sync.WaitGroup

	var err error
	errorsChan := make(chan error)
	defer close(errorsChan)

	for _, exporterDogu := range exporterDogus {
		if slices.Contains(d.excludedDogus, exporterDogu.Name) {
			continue
		}

		importerDoguIndex := slices.IndexFunc(importerDogus, func(dogu migration.Dogu) bool { return dogu.Name == exporterDogu.Name })
		if importerDoguIndex < 0 {
			err = errors.Join(err, fmt.Errorf("failed to find dogu %s in the importing system", exporterDogu.Name))
			continue
		}

		importerDogu := importerDogus[importerDoguIndex]

		if exporterDogu.Volume.SizeInBytes > importerDogu.Volume.SizeInBytes {
			wg.Add(1)
			go func() {
				defer wg.Done()
				if resizeErr := d.resize(ctx, importerDogu.Name, exporterDogu.Volume.SizeInBytes); resizeErr != nil {
					errorsChan <- fmt.Errorf("failed to resize dogu %s: %w", exporterDogu.Name, resizeErr)
				}
			}()
		}
	}

	go func() {
		for resizeErr := range errorsChan {
			err = errors.Join(err, resizeErr)
		}
	}()

	wg.Wait()

	return err
}

func (d *DoguVolumeResizer) resize(ctx context.Context, fullDoguName string, newSizeInBytes int64) error {
	fullImportDoguName, err := cescommons.QualifiedNameFromString(fullDoguName)
	if err != nil {
		return fmt.Errorf("dogu %s name is not a qualified dogu name: %w", fullDoguName, err)
	}
	doguName := fullImportDoguName.SimpleName.String()

	dogu, err := d.doguClient.Get(ctx, doguName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("dogu %q could not be found: %w", doguName, err)
	}

	// convert sizeInBytes to a quantitiy
	minDataVolumeSize := resource.NewQuantity(newSizeInBytes, resource.BinarySI)

	slog.Info(fmt.Sprintf("Resizing dogu %s volume to %s", fullDoguName, minDataVolumeSize.String()))

	dogu.Spec.Resources.MinDataVolumeSize = *minDataVolumeSize
	_, err = d.doguClient.Update(ctx, dogu, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("dogu %q does not have enough volume capacity and the volume could not be resized: %w", doguName, err)
	}

	err = d.waitForPVCResize(ctx, doguName, minDataVolumeSize)
	if err != nil {
		return fmt.Errorf("error waiting for pvc of dogu %s to be resized: %w", doguName, err)
	}

	return nil
}

// waitForPVCResize waits until the pvc of the dogu has the expected size
func (d *DoguVolumeResizer) waitForPVCResize(ctx context.Context, doguName string, requestedMinDataVolumeSize *resource.Quantity) error {
	retries := 0
	for {
		retries++
		if retries > maxRetries {
			return fmt.Errorf("maximum amount of retries reached for the resize of dogu %q volume", doguName)
		}
		// repeat every 10 seconds
		time.Sleep(time.Duration(waitSecondsBetweenRetries) * time.Second)

		pvc, err := d.pvcClient.Get(ctx, doguName, metav1.GetOptions{})
		if err != nil {
			slog.Warn("could not get pvc for dogu %q: %w", doguName, err)
			continue
		}

		actualStorage := pvc.Status.Capacity.Storage()

		if actualStorage.Cmp(*requestedMinDataVolumeSize) >= 0 {
			slog.Info(fmt.Sprintf("Dogu %s volume resized to %s", doguName, actualStorage.String()))
			return nil
		}

		slog.Info(fmt.Sprintf("Dogu %s: current size: %s, expected size: %s", doguName, actualStorage.String(), requestedMinDataVolumeSize.String()))
	}
}
