package systeminfo

import (
	"context"
	"errors"
	"fmt"
	"github.com/cloudogu/ces-importer/api/exporter"
	doguv2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	kubv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"log/slog"
	"slices"
)

const (
	doguNginx        = "official/nginx"
	doguNginxStatic  = "k8s/nginx-static"
	doguNginxIngress = "k8s/nginx-ingress"
)

var (
	excludedDogus = []string{
		"official/monitoring",
		"premium/backup",
		"official/registrator",
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
	doguVolumeResizer  doguVolumeResizer
}

func NewValidator(p systemInfoProvider, doguClient doguClient, pvcClient pvcClient) (*Validator, error) {
	return &Validator{
		systemInfoProvider: p,
		doguVolumeResizer: &defaultDoguVolumeResizer{
			doguClient:    doguClient,
			pvcClient:     pvcClient,
			excludedDogus: append(excludedDogus, doguNginx),
		},
	}, nil
}

// Validate validates that the importing system has the same configuration as the exporting system
// - dogus exist in correct version
// - components exist in correct version
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

func (v *Validator) doValidateSystemInfo(exInfo exporter.SystemInfo, imInfo exporter.SystemInfo, ctx context.Context) error {
	//validate dogus
	result := validateDogus(imInfo, exInfo)

	// validate components
	result = errors.Join(result, validateComponents(imInfo, exInfo))

	// validate dogu-volume-sizes
	result = errors.Join(result, v.doguVolumeResizer.ResizeDogusIfNeeded(ctx, exInfo.Dogus, imInfo.Dogus))

	return result
}

// validateDogus validates that the importing system has the same configuration as the exporting system
func validateDogus(imInfo exporter.SystemInfo, exInfo exporter.SystemInfo) (result error) {
	// Create a map of importing dogus for quick lookup
	imDoguMap := make(map[string]exporter.Dogu)
	for _, d := range imInfo.Dogus {
		imDoguMap[d.Name] = d
	}

	// Validate each exporting dogu
	for _, exDogu := range exInfo.Dogus {
		// Skip excluded dogus
		if slices.Contains(excludedDogus, exDogu.Name) {
			continue
		}

		// Handle nginx as a special case
		if exDogu.Name == doguNginx {
			result = validateNginxDogus(imDoguMap, result)
			continue
		}

		// Validate regular dogus
		result = validateRegularDogu(exDogu, imDoguMap, result)
	}

	// Check for extra dogus in the importing system
	for name := range imDoguMap {
		result = errors.Join(result, fmt.Errorf("dogu %s is installed in the importing system but not present in the exporting system \n", name))
	}

	return result
}

// validateRegularDogu validates a single non-nginx dogu and removes it from the map if valid
func validateRegularDogu(exDogu exporter.Dogu, imDoguMap map[string]exporter.Dogu, result error) error {
	imDogu, exists := imDoguMap[exDogu.Name]
	if !exists {
		return errors.Join(result, fmt.Errorf("dogu %s is not installed (needed version: %s) \n", exDogu.Name, exDogu.Version))
	}

	// Validate version
	if imDogu.Version != exDogu.Version {
		result = errors.Join(result, fmt.Errorf("dogu %s is installed in version %s but needs to have version %s \n", exDogu.Name, imDogu.Version, exDogu.Version))
	}

	// Remove validated dogu from map
	delete(imDoguMap, exDogu.Name)
	return result
}

// validateNginxDogus validates the special case of nginx-related dogus
func validateNginxDogus(imDoguMap map[string]exporter.Dogu, result error) error {
	nginxMnDogus := []string{doguNginxStatic, doguNginxIngress}
	for _, name := range nginxMnDogus {
		imDogu := imDoguMap[name]
		if imDogu.Name == "" {
			result = errors.Join(result, fmt.Errorf("dogu %s is not installed \n", name))
		}
		delete(imDoguMap, name)
	}
	return result
}

func validateComponents(imInfo exporter.SystemInfo, exInfo exporter.SystemInfo) (result error) {
	imComponentsMap := make(map[string]exporter.Component)
	for _, c := range imInfo.Components {
		imComponentsMap[c.Name] = c
	}
	for _, c := range exInfo.Components {
		// validate that the component exists
		imComponent := imComponentsMap[c.Name]
		if imComponent.Name == "" {
			result = errors.Join(result, fmt.Errorf("component %s is not installed (needed version: %s) \n", c.Name, c.Version))
			continue
		}

		// validate that the version is correct
		if !(imComponent.Version == c.Version) {
			result = errors.Join(result, fmt.Errorf("component %s is installed in version %s but needs to have version %s \n", c.Name, imComponent.Version, c.Version))
		}
	}
	return result
}
