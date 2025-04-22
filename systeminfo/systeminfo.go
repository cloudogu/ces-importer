package systeminfo

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/cloudogu/ces-importer/configuration"
	componentEcoClient "github.com/cloudogu/k8s-component-operator/pkg/api/ecosystem"
	ecoSystemV2 "github.com/cloudogu/k8s-dogu-operator/v3/api/ecoSystem"
	"io"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"net/http"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	Namespace = "ecosystem"
)

type ecosystemComponentClient interface {
	componentEcoClient.ComponentV1Alpha1Interface
}

type ecosystemDogusClient interface {
	ecoSystemV2.EcoSystemV2Interface
}

type kubernetesClient interface {
	kubernetes.Interface
}

// type ProviderInterface interface {
// 	GetSystemInfo(namespace string) (*SystemInfo, error)
// }

type Provider struct {
	componentClient  ecosystemComponentClient
	doguClient       ecosystemDogusClient
	KubernetesClient kubernetesClient
	// Provider         ProviderInterface
}

func NewSystemInfoProvider() (*Provider, error) {
	clusterConfig, err := ctrl.GetConfig()
	if err != nil {
		return nil, err
	}

	componentClient, err := componentEcoClient.NewForConfig(clusterConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create component component client: %s", err)
	}

	doguClient, err := ecoSystemV2.NewForConfig(clusterConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create doguc lient: %s", err)
	}

	kubernetesClient, err := kubernetes.NewForConfig(clusterConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create k8s kubernetesClient: %s", err)
	}

	return &Provider{
		componentClient:  componentClient,
		doguClient:       doguClient,
		KubernetesClient: kubernetesClient,
	}, nil
}

func (s *Provider) GetSystemInfo(namespace string) (*SystemInfo, error) {
	// collect Dogus
	dogus, err := s.doguClient.Dogus(namespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("could not get systems dogus: %s", err)
	}
	var systemInfoDogus []Dogu
	for _, d := range dogus.Items {
		vol := d.GetDataVolumeSize()
		systemInfoDogus = append(systemInfoDogus, Dogu{
			Name:    d.Name,
			Version: d.Spec.Version,
			Volume:  volume{SizeInBytes: vol.Value()},
		})
	}

	// collect components
	components, err := s.componentClient.Components(namespace).List(context.Background(), metav1.ListOptions{})
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

	return &SystemInfo{Dogus: systemInfoDogus, Components: systemInfoComponents}, nil
}

// TODO use api.go from boris pr
// GetExporterSystemInfo gets the exporters system info via get request
func GetExporterSystemInfo(conf configuration.Configuration) (SystemInfo, error) {
	var sInfo SystemInfo
	res, err := http.Get(conf.ExporterHost + "system-info")
	if err != nil {
		return sInfo, fmt.Errorf("error performing http request: %s", err)
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return sInfo, fmt.Errorf("could not read exporter response: %s", err)
	}
	err = json.Unmarshal(body, &sInfo)
	if err != nil {
		return sInfo, fmt.Errorf("could not read exporter response: %s", err)
	}
	return sInfo, nil
}
