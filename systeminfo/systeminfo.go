package systeminfo

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/cloudogu/ces-importer/configuration"
	componentEcoClient "github.com/cloudogu/k8s-component-operator/pkg/api/ecosystem"
	componentv1 "github.com/cloudogu/k8s-component-operator/pkg/api/v1"
	ecoSystemV2 "github.com/cloudogu/k8s-dogu-operator/v3/api/ecoSystem"
	doguv2 "github.com/cloudogu/k8s-dogu-operator/v3/api/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

type kubernetesClient interface {
	v1.PersistentVolumeClaimInterface
}

type doguLister interface {
	List(ctx context.Context, opts metav1.ListOptions) (*doguv2.DoguList, error)
}

type componentLister interface {
	List(ctx context.Context, opts metav1.ListOptions) (*componentv1.ComponentList, error)
}

type exporterApiClient interface {
	// DoGetRequest allows issuing HTTP requests towards the exporter API. The result will be a byte slice that must
	// be parsed by the caller respectively.
	DoGetRequest(ctx context.Context, url string) ([]byte, error)
}

type Provider struct {
	componentLister componentLister
	doguLister      doguLister
	pvcClient       kubernetesClient
	apiClient       exporterApiClient
}

func NewSystemInfoProvider(namespace string) (*Provider, error) {
	clusterConfig, err := ctrl.GetConfig()
	if err != nil {
		return nil, err
	}

	componentClient, err := componentEcoClient.NewForConfig(clusterConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create component component client: %s", err)
	}
	componentLister := componentClient.Components(namespace)

	doguClient, err := ecoSystemV2.NewForConfig(clusterConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create dogu client: %s", err)
	}
	doguLister := doguClient.Dogus(namespace)

	kubernetesClient, err := kubernetes.NewForConfig(clusterConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create k8s kubernetesClient: %s", err)
	}
	pvcClient := kubernetesClient.CoreV1().PersistentVolumeClaims(namespace)

	return &Provider{
		componentLister: componentLister,
		doguLister:      doguLister,
		pvcClient:       pvcClient,
	}, nil
}

// getSystemInfo
//
// gets the current systems system info about dogus and components
func (s *Provider) getSystemInfo(ctx context.Context) (*systemInfo, error) {
	// collect Dogus
	dogus, err := s.doguLister.List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("could not get systems dogus: %s", err)
	}
	var systemInfoDogus []dogu
	for _, d := range dogus.Items {
		vol := d.GetDataVolumeSize()
		systemInfoDogus = append(systemInfoDogus, dogu{
			Name:    d.Name,
			Version: d.Spec.Version,
			Volume:  volume{SizeInBytes: vol.Value()},
		})
	}

	// collect components
	components, err := s.componentLister.List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("could not get systems components: %s", err)
	}
	var systemInfoComponents []component
	for _, c := range components.Items {
		systemInfoComponents = append(systemInfoComponents, component{
			Name:    c.Name,
			Version: c.Spec.Version,
		})
	}

	return &systemInfo{Dogus: systemInfoDogus, Components: systemInfoComponents}, nil
}

// GetExporterSystemInfo gets the exporters system info via get request
func (s *Provider) getExporterSystemInfo(conf configuration.Configuration, ctx context.Context) (*systemInfo, error) {
	var sInfo *systemInfo
	res, err := s.apiClient.DoGetRequest(ctx, "https://"+conf.ExporterHost+"/system-info")
	if err != nil {
		return sInfo, fmt.Errorf("error performing http request: %s", err)
	}
	err = json.Unmarshal(res, &sInfo)
	if err != nil {
		return sInfo, fmt.Errorf("could not read exporter response: %s", err)
	}
	return sInfo, nil
}

func (s *Provider) getPvcClient() kubernetesClient {
	return s.pvcClient
}
