package exporter

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/cloudogu/ces-importer/migration"
)

// pathSystemInfo contains the endpoint which returns data which describe the exporter system, f. i. installed dogus etc.
const pathSystemInfo = "/system-info"

type SystemInfoService struct {
	apiClient apiClient
}

func NewSystemInfoService(apiClient apiClient) *SystemInfoService {
	return &SystemInfoService{
		apiClient: apiClient,
	}
}

func (emc *SystemInfoService) GetSystemInfo(ctx context.Context) (*migration.SystemInfo, error) {
	result, err := emc.apiClient.DoGetRequest(ctx, pathSystemInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to get system info: %w", err)
	}

	systemInfo := &SystemInfo{}
	err = json.Unmarshal(result, systemInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to parse system info response: %q: %w", result, err)
	}

	return toMigrationSystemInfo(systemInfo), nil
}

func toMigrationSystemInfo(systemInfo *SystemInfo) *migration.SystemInfo {
	return &migration.SystemInfo{
		FQDN:        systemInfo.FQDN,
		IsMultinode: systemInfo.IsMultinode,
		Dogus: mapSlice(systemInfo.Dogus, func(dogu Dogu) migration.Dogu {
			return migration.Dogu{
				Name:    dogu.Name,
				Version: dogu.Version,
				Volume: migration.DoguVolume{
					SizeInBytes: dogu.Volume.SizeInBytes,
				},
			}
		}),
		Components: mapSlice(systemInfo.Components, func(comp Component) migration.Component {
			return migration.Component{
				Name:    comp.Name,
				Version: comp.Version,
			}
		}),
	}
}
