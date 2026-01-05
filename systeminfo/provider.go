package systeminfo

import (
	"context"
	"fmt"

	"github.com/cloudogu/ces-importer/migration"
	componentv1 "github.com/cloudogu/k8s-component-lib/api/v1"
	doguv2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type doguLister interface {
	List(ctx context.Context, opts metav1.ListOptions) (*doguv2.DoguList, error)
}

type componentLister interface {
	List(ctx context.Context, opts metav1.ListOptions) (*componentv1.ComponentList, error)
}

type systemInfoApiClient interface {
	GetSystemInfo(ctx context.Context) (*migration.SystemInfo, error)
}

type Provider struct {
	componentLister     componentLister
	doguLister          doguLister
	systemInfoApiClient systemInfoApiClient
}

func NewSystemInfoProvider(componentLister componentLister, doguLister doguLister, systemInfoApiClient systemInfoApiClient) (*Provider, error) {
	return &Provider{
		componentLister:     componentLister,
		doguLister:          doguLister,
		systemInfoApiClient: systemInfoApiClient,
	}, nil
}

// GetImporterSystemInfo gets the current systems system info about dogus and components
func (s *Provider) GetImporterSystemInfo(ctx context.Context) (*migration.SystemInfo, error) {
	// collect Dogus
	dogus, err := s.doguLister.List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("could not get systems dogus: %w", err)
	}
	var systemInfoDogus []migration.Dogu
	for _, d := range dogus.Items {
		vol, err := d.GetMinDataVolumeSize()
		if err != nil {
			return nil, fmt.Errorf("could not get minDataVolumeSize for dogu: %w", err)
		}

		systemInfoDogus = append(systemInfoDogus, migration.Dogu{
			Name:    d.Spec.Name,
			Version: d.Status.InstalledVersion,
			Volume:  migration.DoguVolume{SizeInBytes: vol.Value()},
		})
	}

	// collect components
	components, err := s.componentLister.List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("could not get systems components: %s", err)
	}
	var systemInfoComponents []migration.Component
	for _, c := range components.Items {
		systemInfoComponents = append(systemInfoComponents, migration.Component{
			Name:    c.Name,
			Version: c.Status.InstalledVersion,
		})
	}

	return &migration.SystemInfo{Dogus: systemInfoDogus, Components: systemInfoComponents}, nil
}

// GetExporterSystemInfo gets the exporters system info via get request
func (s *Provider) GetExporterSystemInfo(ctx context.Context) (*migration.SystemInfo, error) {
	return s.systemInfoApiClient.GetSystemInfo(ctx)
}
