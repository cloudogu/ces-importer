package exporter

type Service struct {
	*maintenanceService
}

func NewService(baseURL string, client apiClient) *Service {
	return &Service{
		maintenanceService: newMaintenanceService(baseURL, client),
	}
}
