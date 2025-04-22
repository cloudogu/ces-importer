package validate

import (
	"context"
	"fmt"
	"github.com/cloudogu/ces-importer/configuration"
	"github.com/cloudogu/ces-importer/systeminfo"
	"github.com/hashicorp/go-multierror"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"log/slog"
)

type Validator struct {
	conf                  configuration.Configuration
	ctx                   context.Context
	namespace             string
	getExporterSystemInfo func(conf configuration.Configuration) (systeminfo.SystemInfo, error)
}

func NewValidator(conf configuration.Configuration, ctx context.Context, namespace string) *Validator {
	return &Validator{
		conf:                  conf,
		ctx:                   ctx,
		namespace:             namespace,
		getExporterSystemInfo: systeminfo.GetExporterSystemInfo,
	}
}

// ValidateSystemInfo
// validate that the importing system has the same configuration as the exporting system
// validates:
// - dogus exist in correct version
// - components exist in correct version
// - pvcs are large enough (a resize is attempted)
func (v *Validator) ValidateSystemInfo() error {
	slog.Info("Starting validation of system configuration")
	exSystemInfo, err := v.getExporterSystemInfo(v.conf)
	if err != nil {
		return fmt.Errorf("could not get exporter system info: %s", err)
	}
	systemInfoProvider, err := systeminfo.NewSystemInfoProvider()
	if err != nil {
		return fmt.Errorf("could not get importer system info: %s", err)
	}
	imSystemInfo, err := systemInfoProvider.GetSystemInfo(v.namespace)
	if err != nil {
		return fmt.Errorf("could not get importer system info: %s", err)
	}
	err = v.validateSystemInfo(exSystemInfo, *imSystemInfo, *systemInfoProvider)
	if err != nil {
		return fmt.Errorf("could not validate system info: %s", err)
	}
	slog.Info("System configuration validated")
	return nil
}

// returns a formatted multierror if any error occured
func (v *Validator) validateSystemInfo(exInfo systeminfo.SystemInfo, imInfo systeminfo.SystemInfo, provider systeminfo.Provider) error {
	// collect persistent volume claims
	pvcs := provider.KubernetesClient.CoreV1().PersistentVolumeClaims(v.namespace)

	var result *multierror.Error
	//validate dogus
	imDoguMap := make(map[string]systeminfo.Dogu)
	for _, d := range imInfo.Dogus {
		imDoguMap[d.Name] = d
	}
	for _, exDogu := range exInfo.Dogus {
		// validate that the Dogu exists
		imDogu := imDoguMap[exDogu.Name]
		if imDogu.Name == "" {
			result = multierror.Append(result, fmt.Errorf("dogu %s is not installed (needed version: %s) \n", exDogu.Name, exDogu.Version))
		} else {
			// validate that the version is correct
			if !(imDogu.Version == exDogu.Version) {
				result = multierror.Append(result, fmt.Errorf("dogu %s is installed in version %s but needs to have version %s) \n", exDogu.Name, imDogu.Version, exDogu.Version))

				// validate and update the size of the dogus pvc
				v.updatePVC(exDogu, imDogu, pvcs, result)
			}
		}
	}

	// validate components
	for _, c := range exInfo.Components {
		// validate that the component exists
		imDogu := imDoguMap[c.Name]
		if imDogu.Name == "" {
			result = multierror.Append(result, fmt.Errorf("component %s is not installed (needed version: %s) \n", c.Name, c.Version))
		} else {
			// validate that the version is correct
			if !(imDogu.Version == c.Version) {
				result = multierror.Append(result, fmt.Errorf("component %s is installed in version %s but needs to have version %s \n", c.Name, imDogu, c.Version))
			}
		}
	}

	return result
}

// resize the dogus pvc if is it not large enough
func (v *Validator) updatePVC(exDogu systeminfo.Dogu, imDogu systeminfo.Dogu, pvcs v1.PersistentVolumeClaimInterface, result *multierror.Error) {
	// validate that the volume size fits the exported data
	if exDogu.Volume.SizeInBytes > imDogu.Volume.SizeInBytes {
		// try to resize the volume
		pvc, err := pvcs.Get(v.ctx, imDogu.Name, metav1.GetOptions{})
		if err != nil {
			result = multierror.Append(result, fmt.Errorf("dogu %s volume could not be found: %s \n", imDogu.Name, err.Error()))
		} else {
			pvc.Spec.Resources.Requests.Storage().Set(exDogu.Volume.SizeInBytes)
			_, err = pvcs.Update(v.ctx, pvc, metav1.UpdateOptions{})
			if err != nil {
				result = multierror.Append(result, fmt.Errorf("dogu %s does not have enough volume capacity and the volume could not be resized: %s \n"))
			}
		}
	}
}
