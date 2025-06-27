package exporter

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

func NewServiceFromConfig(host APIHost, key APIKey, skipTLSVerification SkipTLSVerification) *Service {
	var options []HTTPClientOption

	if skipTLSVerification {
		options = append(options, WithInsecure())
	}

	return NewService(NewClient(string(host), string(key), options...))
}
