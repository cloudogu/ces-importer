package systeminfo

import (
	"context"
	"errors"
	"fmt"
	"github.com/cloudogu/ces-importer/configuration"
	ecoSystemV2 "github.com/cloudogu/k8s-dogu-operator/v3/api/ecoSystem"
	doguv2 "github.com/cloudogu/k8s-dogu-operator/v3/api/v2"
	kubv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"log/slog"
	"math"
	ctrl "sigs.k8s.io/controller-runtime"
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

// client used for interacting with persistent volume claims
type doguClient interface {
	Get(ctx context.Context, name string, opts metav1.GetOptions) (*doguv2.Dogu, error)
	Update(ctx context.Context, dogu *doguv2.Dogu, opts metav1.UpdateOptions) (*doguv2.Dogu, error)
}

type pvcClient interface {
	Get(ctx context.Context, name string, opts metav1.GetOptions) (*kubv1.PersistentVolumeClaim, error)
}

type systemInfoProvider interface {
	getSystemInfo(ctx context.Context) (*systemInfo, error)
	getExporterSystemInfo(conf configuration.Configuration, ctx context.Context) (*systemInfo, error)
}

type Validator struct {
	conf               configuration.Configuration
	namespace          string
	systemInfoProvider systemInfoProvider
	doguClient         doguClient
	pvcClient          pvcClient
}

func NewValidator(conf configuration.Configuration, namespace string, p systemInfoProvider) (*Validator, error) {
	clusterConfig, err := ctrl.GetConfig()
	if err != nil {
		return nil, err
	}

	ecoSystemV2Client, err := ecoSystemV2.NewForConfig(clusterConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create dogu client: %s", err)
	}
	doguClient := ecoSystemV2Client.Dogus(namespace)

	kubernetesClient, err := kubernetes.NewForConfig(clusterConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create k8s kubernetesClient: %s", err)
	}
	pvcClient := kubernetesClient.CoreV1().PersistentVolumeClaims(namespace)

	return &Validator{
		conf:               conf,
		namespace:          namespace,
		systemInfoProvider: p,
		doguClient:         doguClient,
		pvcClient:          pvcClient,
	}, nil
}

// ValidateSystemInfo
// validate that the importing system has the same configuration as the exporting system
//
// validates:
//
// - dogus exist in correct version
//
// - components exist in correct version
//
// - pvcs are large enough (a resize is attempted)
func (v *Validator) ValidateSystemInfo(ctx context.Context) error {
	slog.Info("Starting validation of system configuration")
	imSystemInfo, err := v.systemInfoProvider.getSystemInfo(ctx)
	if err != nil {
		return fmt.Errorf("could not get importer system info: %s", err)
	}
	exSystemInfo, err := v.systemInfoProvider.getExporterSystemInfo(v.conf, ctx)
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
func (v *Validator) doValidateSystemInfo(exInfo systemInfo, imInfo systemInfo, ctx context.Context) error {
	//validate dogus
	var result error
	doguResizesStartedCounter := 0
	c := make(chan error)

	imDoguMap := make(map[string]dogu)
	for _, d := range imInfo.Dogus {
		imDoguMap[d.Name] = d
	}
	for _, exDogu := range exInfo.Dogus {
		// validate that the dogu exists
		imDogu := imDoguMap[exDogu.Name]
		if imDogu.Name == "" {
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
	}

	// check every started resize for errors
	for range doguResizesStartedCounter {
		err := <-c
		if err != nil {
			result = errors.Join(result, err)
		}
	}

	// validate components
	imComponentsMap := make(map[string]component)
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
func (v *Validator) updatePVC(exDogu dogu, imDogu dogu, ctx context.Context, c chan error) {
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
