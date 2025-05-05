package systeminfo

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/cloudogu/ces-importer/configuration"
	componentv1 "github.com/cloudogu/k8s-component-operator/pkg/api/v1"
	doguv2 "github.com/cloudogu/k8s-dogu-operator/v3/api/v2"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"testing"
)

func TestGetSystemInfo(t *testing.T) {
	t.Run("should return with no error", func(t *testing.T) {
		// dogus
		dogus := newMockDoguLister(t)
		doguList := doguv2.DoguList{
			TypeMeta: metav1.TypeMeta{},
			ListMeta: metav1.ListMeta{},
			Items: []doguv2.Dogu{{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name: "testDogu",
				},
				Spec: doguv2.DoguSpec{
					Name:    "",
					Version: "",
					Resources: doguv2.DoguResources{
						DataVolumeSize: "10",
					},
				},
				Status: doguv2.DoguStatus{},
			}},
		}
		dogus.EXPECT().List(mock.Anything, mock.Anything).Return(&doguList, nil)

		// components
		components := newMockComponentLister(t)
		componentList := componentv1.ComponentList{
			TypeMeta: metav1.TypeMeta{},
			ListMeta: metav1.ListMeta{},
			Items: []componentv1.Component{
				{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name: "testComponent",
					},
					Spec: componentv1.ComponentSpec{
						Version: "1.1.1",
					},
					Status: componentv1.ComponentStatus{},
				},
			},
		}
		components.EXPECT().List(mock.Anything, mock.Anything).Return(&componentList, nil)

		mockProvider := Provider{
			componentLister: components,
			doguLister:      dogus,
			pvcClient:       nil,
		}
		_, err := mockProvider.getSystemInfo(context.Background())

		require.NoError(t, err)
	})
	t.Run("should not be able to get dogus", func(t *testing.T) {
		// dogus
		dogus := newMockDoguLister(t)
		dogus.EXPECT().List(mock.Anything, mock.Anything).Return(nil, fmt.Errorf("error"))

		mockProvider := Provider{
			componentLister: nil,
			doguLister:      dogus,
			pvcClient:       nil,
		}
		_, err := mockProvider.getSystemInfo(context.Background())

		require.EqualError(t, err, "could not get systems dogus: error")
	})
	t.Run("should not be able to get components", func(t *testing.T) {
		// dogus
		dogus := newMockDoguLister(t)
		doguList := doguv2.DoguList{
			TypeMeta: metav1.TypeMeta{},
			ListMeta: metav1.ListMeta{},
			Items: []doguv2.Dogu{{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name: "testDogu",
				},
				Spec: doguv2.DoguSpec{
					Name:    "",
					Version: "",
					Resources: doguv2.DoguResources{
						DataVolumeSize: "10",
					},
				},
				Status: doguv2.DoguStatus{},
			}},
		}
		dogus.EXPECT().List(mock.Anything, mock.Anything).Return(&doguList, nil)

		// components
		components := newMockComponentLister(t)
		components.EXPECT().List(mock.Anything, mock.Anything).Return(nil, fmt.Errorf("error"))

		mockProvider := Provider{
			componentLister: components,
			doguLister:      dogus,
			pvcClient:       nil,
		}
		_, err := mockProvider.getSystemInfo(context.Background())

		require.EqualError(t, err, "could not get systems components: error")
	})
}

func TestNewSystemInfoProvider(t *testing.T) {
	t.Run("should instantiate system info provider", func(t *testing.T) {
		// override default controller method to retrieve a kube config
		oldGetConfigDelegate := ctrl.GetConfig
		defer func() {
			ctrl.GetConfig = oldGetConfigDelegate
		}()
		ctrl.GetConfig = func() (*rest.Config, error) {
			return &rest.Config{}, nil
		}

		_, err := NewSystemInfoProvider("test")
		require.NoError(t, err)
	})
}

func TestGetExporterSystemInfo(t *testing.T) {
	t.Run("should get exporter system info", func(t *testing.T) {
		sInfo := systemInfo{
			Dogus: []dogu{
				{
					Name:    "testD",
					Version: "1.1.1",
					Volume:  volume{},
				},
			},
			Components: []component{
				{
					Name:    "testC",
					Version: "1.2.3",
				},
			},
		}
		apiCli := newMockExporterApiClient(t)
		apiCli.EXPECT().DoGetRequest(mock.Anything, mock.Anything).Return(json.Marshal(sInfo))

		p := Provider{
			componentLister: nil,
			doguLister:      nil,
			pvcClient:       nil,
			apiClient:       apiCli,
		}

		apiSystemInfo, err := p.getExporterSystemInfo(configuration.Configuration{
			ExporterHost: "",
		}, context.Background())
		require.NoError(t, err)
		require.Equal(t, sInfo.Dogus, apiSystemInfo.Dogus)
		require.Equal(t, sInfo.Components, apiSystemInfo.Components)
	})

	t.Run("should get an error on http request", func(t *testing.T) {
		apiCli := newMockExporterApiClient(t)
		apiCli.EXPECT().DoGetRequest(mock.Anything, mock.Anything).Return(nil, fmt.Errorf("testerror"))

		p := Provider{
			componentLister: nil,
			doguLister:      nil,
			pvcClient:       nil,
			apiClient:       apiCli,
		}

		_, err := p.getExporterSystemInfo(configuration.Configuration{
			ExporterHost: "",
		}, context.Background())
		require.EqualError(t, err, "error performing http request: testerror")
	})

	t.Run("should get an error on json unmarshall", func(t *testing.T) {
		b := []byte("{")
		apiCli := newMockExporterApiClient(t)
		apiCli.EXPECT().DoGetRequest(mock.Anything, mock.Anything).Return(b, nil)

		p := Provider{
			componentLister: nil,
			doguLister:      nil,
			pvcClient:       nil,
			apiClient:       apiCli,
		}

		_, err := p.getExporterSystemInfo(configuration.Configuration{
			ExporterHost: "",
		}, context.Background())
		require.EqualError(t, err, "could not read exporter response: unexpected end of JSON input")
	})
}
