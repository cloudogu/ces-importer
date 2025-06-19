package systeminfo

import (
	"context"
	"errors"
	"fmt"
	"github.com/cloudogu/ces-importer/migration"
	"log/slog"
	"slices"
)

const (
	doguNginx        = "official/nginx"
	doguNginxStatic  = "k8s/nginx-static"
	doguNginxIngress = "k8s/nginx-ingress"
)

type Validator struct {
	excludedDogus []string
}

func NewValidator(excludedDogus []string) *Validator {
	return &Validator{
		excludedDogus: excludedDogus,
	}
}

// Validate validates that the importing system has the same configuration as the exporting system
// - dogus exist in correct version
// - components exist in correct version
// - pvcs are large enough (a resize is attempted)
func (v *Validator) Validate(_ context.Context, exporterInfo *migration.SystemInfo, importerInfo *migration.SystemInfo) error {
	slog.Info("Starting validation of system configuration")

	err := v.doValidateSystemInfo(exporterInfo, importerInfo)
	if err != nil {
		return fmt.Errorf("could not validate system info: %w", err)
	}

	slog.Info("System configuration validated successfully")

	return nil
}

func (v *Validator) doValidateSystemInfo(exInfo *migration.SystemInfo, imInfo *migration.SystemInfo) error {
	//validate dogus
	result := v.validateDogus(imInfo, exInfo)

	// validate components
	result = errors.Join(result, validateComponents(imInfo, exInfo))

	return result
}

// validateDogus validates that the importing system has the same configuration as the exporting system
func (v *Validator) validateDogus(imInfo *migration.SystemInfo, exInfo *migration.SystemInfo) (result error) {
	// Create a map of importing dogus for quick lookup
	imDoguMap := make(map[string]migration.Dogu)
	for _, d := range imInfo.Dogus {
		imDoguMap[d.Name] = d
	}

	// Validate each exporting dogu
	for _, exDogu := range exInfo.Dogus {
		// Skip excluded dogus
		if slices.Contains(v.excludedDogus, exDogu.Name) {
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
func validateRegularDogu(exDogu migration.Dogu, imDoguMap map[string]migration.Dogu, result error) error {
	imDogu, exists := imDoguMap[exDogu.Name]
	if !exists {
		return errors.Join(result, fmt.Errorf("dogu %s is not installed (required version: %s) \n", exDogu.Name, exDogu.Version))
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
func validateNginxDogus(imDoguMap map[string]migration.Dogu, result error) error {
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

func validateComponents(imInfo *migration.SystemInfo, exInfo *migration.SystemInfo) (result error) {
	imComponentsMap := make(map[string]migration.Component)
	for _, c := range imInfo.Components {
		imComponentsMap[c.Name] = c
	}
	for _, c := range exInfo.Components {
		// validate that the component exists
		imComponent := imComponentsMap[c.Name]
		if imComponent.Name == "" {
			result = errors.Join(result, fmt.Errorf("component %s is not installed (required version: %s) \n", c.Name, c.Version))
			continue
		}

		// validate that the version is correct
		if !(imComponent.Version == c.Version) {
			result = errors.Join(result, fmt.Errorf("component %s is installed in version %s but needs to have version %s \n", c.Name, imComponent.Version, c.Version))
		}
	}
	return result
}
