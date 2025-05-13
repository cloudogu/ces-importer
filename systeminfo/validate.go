package systeminfo

import (
	"context"
	"errors"
	"fmt"
	"github.com/cloudogu/ces-importer/api/exporter"
	doguv2 "github.com/cloudogu/k8s-dogu-operator/v3/api/v2"
	kubv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"log/slog"
	"math"
	"slices"
	"time"
)

const (
	defaultWaitSecondsBetweenRetries = 10
	defaultMaxWaitMinutes            = 10
	doguNginx                        = "nginx"
	doguNginxStatic                  = "nginx-static"
	doguNginxIngress                 = "nginx-ingress"
)

var (
	waitSecondsBetweenRetries = defaultWaitSecondsBetweenRetries
	maxWaitMinutes            = defaultMaxWaitMinutes
	maxRetries                = (maxWaitMinutes * 60) / waitSecondsBetweenRetries
	excludedDogus             = []string{
		"monitoring",
		"backup",
		"registrator",
	}
)

// client used for interacting with persistent volume claims
type doguClient interface {
	Get(ctx context.Context, name string, opts metav1.GetOptions) (*doguv2.Dogu, error)
	Update(ctx context.Context, dogu *doguv2.Dogu, opts metav1.UpdateOptions) (*doguv2.Dogu, error)
}

type pvcClient interface {
	Get(ctx context.Context, name string, opts metav1.GetOptions) (*kubv1.PersistentVolumeClaim, error)
}

type systemInfoProvider interface {
	getImporterSystemInfo(ctx context.Context) (*exporter.SystemInfo, error)
	getExporterSystemInfo(ctx context.Context) (*exporter.SystemInfo, error)
}

type Validator struct {
	systemInfoProvider systemInfoProvider
	doguClient         doguClient
	pvcClient          pvcClient
}

func NewValidator(p systemInfoProvider, doguClient doguClient, pvcClient pvcClient) (*Validator, error) {
	return &Validator{
		systemInfoProvider: p,
		doguClient:         doguClient,
		pvcClient:          pvcClient,
	}, nil
}

// Validate
// validates that the importing system has the same configuration as the exporting system
//
// validates:
//
// - dogus exist in correct version
//
// - components exist in correct version
//
// - pvcs are large enough (a resize is attempted)
func (v *Validator) Validate(ctx context.Context) error {
	slog.Info("Starting validation of system configuration")
	imSystemInfo, err := v.systemInfoProvider.getImporterSystemInfo(ctx)
	if err != nil {
		return fmt.Errorf("could not get importer system info: %s", err)
	}
	exSystemInfo, err := v.systemInfoProvider.getExporterSystemInfo(ctx)
	if err != nil {
		return fmt.Errorf("could not get exporter system info: %s", err)
	}
	err = v.doValidateSystemInfo(*exSystemInfo, *imSystemInfo, ctx)
	if err != nil {
		return fmt.Errorf("could not validate system info: %w", err)
	}
	slog.Info("System configuration validated")
	return nil
}

// doValidateSystemInfo
// validate that the importing system has the same configuration as the exporting system
//
// validates:
//
// - dogus exist in correct version
//
// - components exist in correct version
//
// - pvcs are large enough (a resize is attempted)
//
// returns a formatted multierror if any error occurred
func (v *Validator) doValidateSystemInfo(exInfo exporter.SystemInfo, imInfo exporter.SystemInfo, ctx context.Context) error {
	//validate dogus
	var result error
	doguResizesStartedCounter := 0
	c := make(chan error)

	imDoguMap := make(map[string]exporter.Dogu)
	for _, d := range imInfo.Dogus {
		imDoguMap[d.Name] = d
	}
	for _, exDogu := range exInfo.Dogus {
		// special case for excluded dogus
		isExcluded := slices.Contains(excludedDogus, exDogu.Name)
		if isExcluded {
			continue
		}

		// validate that the dogu exists
		imDogu, imDoguExists := imDoguMap[exDogu.Name]

		isNginx := exDogu.Name == doguNginx
		if !isNginx {
			if !imDoguExists {
				result = errors.Join(result, fmt.Errorf("dogu %s is not installed (needed version: %s) \n", exDogu.Name, exDogu.Version))
			} else {
				// validate that the version is correct
				if !(imDogu.Version == exDogu.Version) {
					result = errors.Join(result, fmt.Errorf("dogu %s is installed in version %s but needs to have version %s) \n", exDogu.Name, imDogu.Version, exDogu.Version))
				} else {
					// validate and update the size of the dogus pvc
					//result = errors.Join(result, v.updatePVC(exDogu, imDogu, ctx))
					doguResizesStartedCounter++
					go v.updatePVC(exDogu, imDogu, ctx, c)
				}
			}
		} else {
			// nginx is a special case because it has two corresponding mn dogus
			nginxMnDogus := []string{doguNginxStatic, doguNginxIngress}
			for _, d := range nginxMnDogus {
				imDogu := imDoguMap[d]
				if imDogu.Name == "" {
					result = errors.Join(result, fmt.Errorf("dogu %s is not installed \n", d))
				}
			}
		}

		// delete the validated dogu from the map to later have a map of all dogus installed in the importing system not
		// present in the exporting system
		delete(imDoguMap, exDogu.Name)
	}

	// validate that the importing system does not have dogus installed that are not present in the exporting system
	delete(imDoguMap, doguNginxStatic)
	delete(imDoguMap, doguNginxIngress)
	for key := range imDoguMap {
		result = errors.Join(fmt.Errorf("dogu %s is installed in the importing system but not present in the exporting system  \n", key))
	}

	// check every started resize for errors
	for range doguResizesStartedCounter {
		err := <-c
		if err != nil {
			result = errors.Join(result, err)
		}
	}

	// validate components
	imComponentsMap := make(map[string]exporter.Component)
	for _, c := range imInfo.Components {
		imComponentsMap[c.Name] = c
	}
	for _, c := range exInfo.Components {
		// validate that the component exists
		imComponent := imComponentsMap[c.Name]
		if imComponent.Name == "" {
			result = errors.Join(result, fmt.Errorf("component %s is not installed (needed version: %s) \n", c.Name, c.Version))
		} else {
			// validate that the version is correct
			if !(imComponent.Version == c.Version) {
				result = errors.Join(result, fmt.Errorf("component %s is installed in version %s but needs to have version %s \n", c.Name, imComponent.Version, c.Version))
			}
		}
	}

	return result
}

// resize the dogus pvc if it is not large enough
func (v *Validator) updatePVC(exDogu exporter.Dogu, imDogu exporter.Dogu, ctx context.Context, c chan error) {
	// prevent endless running function when a panic occurs as the result will be awaited
	defer func() {
		if err := recover(); err != nil {
			c <- fmt.Errorf("panic while updating pvc: %v", err)
		}
	}()
	var result error
	// validate that the volume size fits the exported data
	if exDogu.Volume.SizeInBytes > imDogu.Volume.SizeInBytes {
		// try to resize the volume
		dogu, err := v.doguClient.Get(ctx, imDogu.Name, metav1.GetOptions{})
		if err != nil {
			result = errors.Join(result, fmt.Errorf("dogu %s volume could not be found: %s \n", imDogu.Name, err.Error()))
		} else {
			slog.Info(fmt.Sprintf("Resizing dogu %s volume", imDogu.Name))
			// use Gi and round up
			roundedDoguSizeGB := fmt.Sprintf("%.0fGi", math.Ceil(float64(exDogu.Volume.SizeInBytes)/(1024*1024*1024)))
			dogu.Spec.Resources.DataVolumeSize = roundedDoguSizeGB
			_, err = v.doguClient.Update(ctx, dogu, metav1.UpdateOptions{})
			if err != nil {
				result = errors.Join(result, fmt.Errorf("dogu %s does not have enough volume capacity and the volume could not be resized: %s \n", imDogu.Name, err.Error()))
			} else {
				err = v.waitForPVCResize(roundedDoguSizeGB, imDogu.Name, ctx)
				if err != nil {
					result = errors.Join(result, err)
				}
			}
		}
	}
	c <- result
}

// waitForPVCResize waits until the pvc of the dogu has the expected size
func (v *Validator) waitForPVCResize(expectedSize string, doguName string, ctx context.Context) error {
	retries := 0
	for {
		retries++
		if retries > maxRetries {
			return fmt.Errorf("maximum amount of retries reached for the resize of Dogu %s volume", doguName)
		}
		// repeat every 10 seconds
		time.Sleep(time.Duration(waitSecondsBetweenRetries) * time.Second)

		pvc, err := v.pvcClient.Get(ctx, doguName, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("could not get dogu %s pvc: %w", doguName, err)
		}
		requestedStorage := pvc.Spec.Resources.Requests.Storage()
		actualStorage := pvc.Status.Capacity.Storage()

		roundedPVSizeGB := fmt.Sprintf("%.0fGi", math.Ceil(actualStorage.AsApproximateFloat64()/(1024*1024*1024)))
		if requestedStorage.Equal(*actualStorage) {
			slog.Info(fmt.Sprintf("Dogu %s volume resized to %s", doguName, roundedPVSizeGB))
			return nil
		}

		slog.Info(fmt.Sprintf("Dogu %s: current size: %s, expected size: %s", doguName, roundedPVSizeGB, expectedSize))
	}
}
