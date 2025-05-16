package exporter

type Service struct {
	*MaintenanceModeService
	*ConfigApiClient
}

func NewService(client apiClient) *Service {
	return &Service{
		MaintenanceModeService: NewMaintenanceModeService(client),
		ConfigApiClient:        NewConfigApiClient(client),
	}
}
