package systeminfo

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"

	doguCommons "github.com/cloudogu/ces-commons-lib/dogu"
	"github.com/cloudogu/ces-importer/migration"
)

const (
	doguNginx = "official/nginx"
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
		imDoguMap[getDoguNameWithoutNamespace(d.Name)] = d
	}

	// Dogu names in excludedDogus may be names with or without namespace information
	excludedDoguNames := getExcludedDoguNames(v.excludedDogus)

	// Validate each exporting dogu
	for _, exDogu := range exInfo.Dogus {
		doguName := getDoguNameWithoutNamespace(exDogu.Name)

		// Skip excluded dogus
		if slices.Contains(excludedDoguNames, doguName) {
			slog.Info(fmt.Sprintf("skipping validation for excluded dogu %s", doguName))
			continue
		}

		// Handle nginx as a special case
		if exDogu.Name == doguNginx {
			continue
		}

		// Validate regular dogus
		result = validateRegularDogu(exDogu, imDoguMap, result)
	}

	// remove excluded dogus from map so they don't get flagged as missing on the exporting system
	for _, excludedDoguName := range excludedDoguNames {
		delete(imDoguMap, excludedDoguName)
	}

	// Check for extra dogus in the importing system
	for name := range imDoguMap {
		result = errors.Join(result, fmt.Errorf("dogu %s is installed in the importing system but not present in the exporting system \n", name))
	}

	return result
}

// getDoguNameWithoutNamespace gets the simple name of a dogu without the namespace
func getDoguNameWithoutNamespace(doguName string) string {
	qualifiedDoguName, err := doguCommons.QualifiedNameFromString(doguName)
	// fall back on name that was passed in
	if err == nil {
		doguName = qualifiedDoguName.SimpleName.String()
	}
	return doguName
}

// getExcludedDoguNames gets the list of excluded dogu names without the namespace
func getExcludedDoguNames(excludedDogus []string) []string {
	excludedDoguNames := make([]string, len(excludedDogus))
	for i, doguName := range excludedDogus {
		excludedDoguNames[i] = getDoguNameWithoutNamespace(doguName)
	}
	return excludedDoguNames
}

// validateRegularDogu validates a single non-nginx dogu and removes it from the map if valid
func validateRegularDogu(exDogu migration.Dogu, imDoguMap map[string]migration.Dogu, result error) error {
	exDoguName := getDoguNameWithoutNamespace(exDogu.Name)
	imDogu, exists := imDoguMap[exDoguName]
	if !exists {
		return errors.Join(result, fmt.Errorf("dogu %s is not installed (required version: %s) \n", exDogu.Name, exDogu.Version))
	}

	// Validate version
	if imDogu.Version != exDogu.Version {
		result = errors.Join(result, fmt.Errorf("version discrepancy for dogu %s. Source instance version: %s, Target instance version: %s", exDogu.Name, exDogu.Version, imDogu.Version))
	}

	// Remove validated dogu from map
	delete(imDoguMap, exDoguName)
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
			result = errors.Join(result, fmt.Errorf("version discrepancy for component %s. Source instance version: %s, Target instance version: %s \n", c.Name, c.Version, imComponent.Version))
		}
	}
	return result
}
