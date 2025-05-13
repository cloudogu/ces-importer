package exporter

type Service struct {
	*MaintenanceModeService
}

func NewService(baseURL string, client apiClient) *Service {
	return &Service{
		MaintenanceModeService: NewMaintenanceModeService(baseURL, client),
	}
}
