package exporter

type Service struct {
	*MaintenanceModeService
}

func NewService(client apiClient) *Service {
	return &Service{
		MaintenanceModeService: NewMaintenanceModeService(client),
	}
}
