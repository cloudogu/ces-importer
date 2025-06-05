package systeminfo

import (
	"context"
	"fmt"
	"github.com/cloudogu/ces-importer/api/exporter"
	componentv1 "github.com/cloudogu/k8s-component-operator/pkg/api/v1"
	doguv2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/stretchr/testify/assert"
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
		}
		_, err := mockProvider.GetImporterSystemInfo(context.Background())

		require.NoError(t, err)
	})
	t.Run("should not be able to get dogus", func(t *testing.T) {
		// dogus
		dogus := newMockDoguLister(t)
		dogus.EXPECT().List(mock.Anything, mock.Anything).Return(nil, fmt.Errorf("error"))

		mockProvider := Provider{
			componentLister: nil,
			doguLister:      dogus,
		}
		_, err := mockProvider.GetImporterSystemInfo(context.Background())

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
		}
		_, err := mockProvider.GetImporterSystemInfo(context.Background())

		require.EqualError(t, err, "could not get systems components: error")
	})
	t.Run("should fail getting minDataVolumeSize", func(t *testing.T) {
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
						DataVolumeSize: "1.2.3Gi",
					},
				},
				Status: doguv2.DoguStatus{},
			}},
		}
		dogus.EXPECT().List(mock.Anything, mock.Anything).Return(&doguList, nil)

		mockProvider := Provider{
			doguLister: dogus,
		}
		_, err := mockProvider.GetImporterSystemInfo(context.Background())

		require.Error(t, err)
		assert.ErrorContains(t, err, "ould not get minDataVolumeSize for dogu: quantities must match the regular expression")
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
		componentLister := newMockComponentLister(t)
		doguLister := newMockDoguLister(t)
		systemInfoApiClient := newMockSystemInfoApiClient(t)
		_, err := NewSystemInfoProvider(componentLister, doguLister, systemInfoApiClient)
		require.NoError(t, err)
	})
}

func TestGetExporterSystemInfo(t *testing.T) {
	t.Run("should get exporter system info", func(t *testing.T) {
		sInfo := exporter.SystemInfo{
			Dogus: []exporter.Dogu{
				{
					Name:    "testD",
					Version: "1.1.1",
					Volume:  exporter.DoguVolume{},
				},
			},
			Components: []exporter.Component{
				{
					Name:    "testC",
					Version: "1.2.3",
				},
			},
		}
		apiCli := newMockSystemInfoApiClient(t)
		apiCli.EXPECT().GetSystemInfo(mock.Anything).Return(&sInfo, nil)

		p := Provider{
			componentLister:     nil,
			doguLister:          nil,
			systemInfoApiClient: apiCli,
		}

		apiSystemInfo, err := p.GetExporterSystemInfo(context.Background())
		require.NoError(t, err)
		require.Equal(t, sInfo.Dogus, apiSystemInfo.Dogus)
		require.Equal(t, sInfo.Components, apiSystemInfo.Components)
	})

	t.Run("should get an error on system info request", func(t *testing.T) {
		apiCli := newMockSystemInfoApiClient(t)
		apiCli.EXPECT().GetSystemInfo(mock.Anything).Return(nil, fmt.Errorf("testerror"))

		p := Provider{
			componentLister:     nil,
			doguLister:          nil,
			systemInfoApiClient: apiCli,
		}

		_, err := p.GetExporterSystemInfo(context.Background())
		require.EqualError(t, err, "testerror")
	})
}
