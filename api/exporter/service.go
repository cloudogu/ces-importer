package exporter

type Service struct {
	*MaintenanceModeService
	*ConfigService
}

func NewService(client apiClient) *Service {
	return &Service{
		MaintenanceModeService: NewMaintenanceModeService(client),
		ConfigService:          NewConfigService(client),
	}
}
