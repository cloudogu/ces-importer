package exporter

import "github.com/cloudogu/ces-importer/configuration"

type Service struct {
	*MaintenanceModeService
	*ConfigService
	*ExportDoguService
	*ExportModeService
	*SystemInfoService
	*HealthService
}

func NewService(client apiClient) *Service {
	return &Service{
		MaintenanceModeService: NewMaintenanceModeService(client),
		ConfigService:          NewConfigService(client),
		ExportDoguService:      NewExportDoguService(client),
		ExportModeService:      NewExportModeService(client),
		SystemInfoService:      NewSystemInfoService(client),
		HealthService:          NewHealthService(client),
	}
}

type APIHost string
type APIKey string
type SkipTLSVerification bool

func NewServiceFromConfig(cfg configuration.API) *Service {
	var options []HTTPClientOption

	if cfg.SkipTLSVerify {
		options = append(options, WithInsecure())
	} else {
		options = append(options, WithCustomCAs(cfg))
	}

	return NewService(NewClient(string(cfg.ExporterHost), string(cfg.ExporterApiKey), options...))
}
