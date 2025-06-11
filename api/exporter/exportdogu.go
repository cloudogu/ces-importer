package exporter

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/cloudogu/ces-importer/migration"
	"path"
)

const exportModeEndpoint = "/export/dogu"

type ExportDoguService struct {
	apiClient apiClient
}

func NewExportDoguService(apiClient apiClient) *ExportDoguService {
	return &ExportDoguService{
		apiClient: apiClient,
	}
}

func (emc *ExportDoguService) GetExportDogu(ctx context.Context) (export *migration.DoguExport, err error) {
	result, err := emc.apiClient.DoGetRequest(ctx, exportModeEndpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to get export dogu: %w", err)
	}

	var doguExport DoguExport
	err = json.Unmarshal(result, &doguExport)
	if err != nil {
		return nil, fmt.Errorf("failed to parse export dogu response: %q: %w", result, err)
	}

	return toMigrationDoguExport(&doguExport), nil
}

func (emc *ExportDoguService) SetExportDogu(ctx context.Context, doguName string) (export *migration.DoguExport, err error) {
	apiPath := path.Join(exportModeEndpoint, doguName)
	result, err := emc.apiClient.DoPostRequest(ctx, apiPath, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to set export dogu: %w", err)
	}

	var doguExport DoguExport
	err = json.Unmarshal(result, &doguExport)
	if err != nil {
		return nil, fmt.Errorf("failed to parse export dogu response: %q: %w", result, err)
	}

	return toMigrationDoguExport(&doguExport), nil
}

func toMigrationDoguExport(doguExport *DoguExport) *migration.DoguExport {
	return &migration.DoguExport{
		Dogu:         doguExport.Dogu,
		VolumePath:   doguExport.VolumePath,
		ExporterPort: doguExport.ExporterPort,
	}
}
