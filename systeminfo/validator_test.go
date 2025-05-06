package systeminfo

import (
	"context"
	"fmt"
	"github.com/cloudogu/ces-importer/configuration"
	"github.com/hashicorp/go-multierror"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	kubv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func TestNewValidator(t *testing.T) {
	t.Run("should return new validator", func(t *testing.T) {
		p := newMockSystemInfoProvider(t)
		n := "namespace"
		v := NewValidator(configuration.Coordinator{}, n, p)
		require.Equal(t, v.conf, configuration.Coordinator{})
		require.Equal(t, v.namespace, n)
		require.Equal(t, v.systemInfoProvider, p)
	})
}

func TestValidateSystemInfo(t *testing.T) {
	t.Run("should return with no error", func(t *testing.T) {
		sysInfo := systemInfo{
			Dogus: []dogu{
				{
					Name:    "testdogu",
					Version: "1.2.3",
					Volume: volume{
						SizeInBytes: 10,
					},
				},
			},
			Components: []component{
				{
					Name:    "testcomponent",
					Version: "1.2.3",
				},
			},
		}
		s := newMockSystemInfoProvider(t)
		s.EXPECT().getSystemInfo(context.Background()).Return(&sysInfo, nil)
		s.EXPECT().getExporterSystemInfo(mock.Anything, mock.Anything).Return(&sysInfo, nil)

		client := newMockKubernetesClient(t)
		s.EXPECT().getPvcClient().Return(client)

		v := Validator{
			conf:               configuration.Coordinator{},
			namespace:          "",
			systemInfoProvider: s,
		}
		err := v.ValidateSystemInfo(context.Background())
		require.Nil(t, err)
	})

	t.Run("should return error mismatching dogu versions", func(t *testing.T) {
		exsysInfo := systemInfo{
			Dogus: []dogu{
				{
					Name:    "testdogu",
					Version: "1.2.3",
					Volume: volume{
						SizeInBytes: 10,
					},
				},
			},
			Components: []component{
				{
					Name:    "testcomponent",
					Version: "1.2.3",
				},
			},
		}
		imSysInfo := systemInfo{
			Dogus: []dogu{
				{
					Name:    "testdogu",
					Version: "9.9.9",
					Volume: volume{
						SizeInBytes: 10,
					},
				},
			},
			Components: []component{
				{
					Name:    "testcomponent",
					Version: "1.2.3",
				},
			},
		}
		s := newMockSystemInfoProvider(t)
		s.EXPECT().getSystemInfo(context.Background()).Return(&imSysInfo, nil)
		s.EXPECT().getExporterSystemInfo(mock.Anything, mock.Anything).Return(&exsysInfo, nil)
		client := newMockKubernetesClient(t)
		s.EXPECT().getPvcClient().Return(client)

		v := Validator{
			conf:               configuration.Coordinator{},
			namespace:          "",
			systemInfoProvider: s,
		}
		err := v.ValidateSystemInfo(context.Background())
		require.ErrorContains(t, err, "dogu testdogu is installed in version 9.9.9 but needs to have version 1.2.3")
	})

	t.Run("should return error dogu not installed", func(t *testing.T) {
		exsysInfo := systemInfo{
			Dogus: []dogu{
				{
					Name:    "testdogu",
					Version: "1.2.3",
					Volume: volume{
						SizeInBytes: 10,
					},
				},
			},
			Components: []component{
				{
					Name:    "testcomponent",
					Version: "1.2.3",
				},
			},
		}
		imSysInfo := systemInfo{
			Dogus: []dogu{},
			Components: []component{
				{
					Name:    "testcomponent",
					Version: "1.2.3",
				},
			},
		}
		s := newMockSystemInfoProvider(t)
		s.EXPECT().getSystemInfo(context.Background()).Return(&imSysInfo, nil)
		s.EXPECT().getExporterSystemInfo(mock.Anything, mock.Anything).Return(&exsysInfo, nil)
		client := newMockKubernetesClient(t)
		s.EXPECT().getPvcClient().Return(client)

		v := Validator{
			conf:               configuration.Coordinator{},
			namespace:          "",
			systemInfoProvider: s,
		}
		err := v.ValidateSystemInfo(context.Background())
		require.ErrorContains(t, err, "dogu testdogu is not installed (needed version: 1.2.3)")
	})

	t.Run("should return error component not installed", func(t *testing.T) {
		exsysInfo := systemInfo{
			Dogus: []dogu{
				{
					Name:    "testdogu",
					Version: "1.2.3",
					Volume: volume{
						SizeInBytes: 10,
					},
				},
			},
			Components: []component{
				{
					Name:    "testcomponent",
					Version: "1.2.3",
				},
			},
		}
		imSysInfo := systemInfo{
			Dogus: []dogu{
				{
					Name:    "testdogu",
					Version: "1.2.3",
					Volume: volume{
						SizeInBytes: 10,
					},
				},
			},
			Components: []component{},
		}
		s := newMockSystemInfoProvider(t)
		s.EXPECT().getSystemInfo(context.Background()).Return(&imSysInfo, nil)
		s.EXPECT().getExporterSystemInfo(mock.Anything, mock.Anything).Return(&exsysInfo, nil)
		client := newMockKubernetesClient(t)
		s.EXPECT().getPvcClient().Return(client)

		v := Validator{
			conf:               configuration.Coordinator{},
			namespace:          "",
			systemInfoProvider: s,
		}
		err := v.ValidateSystemInfo(context.Background())
		require.ErrorContains(t, err, "component testcomponent is not installed (needed version: 1.2.3)")
	})

	t.Run("should return error component mismatching component version", func(t *testing.T) {
		exsysInfo := systemInfo{
			Dogus: []dogu{
				{
					Name:    "testdogu",
					Version: "1.2.3",
					Volume: volume{
						SizeInBytes: 10,
					},
				},
			},
			Components: []component{
				{
					Name:    "testcomponent",
					Version: "1.2.3",
				},
			},
		}
		imSysInfo := systemInfo{
			Dogus: []dogu{
				{
					Name:    "testdogu",
					Version: "1.2.3",
					Volume: volume{
						SizeInBytes: 10,
					},
				},
			},
			Components: []component{
				{
					Name:    "testcomponent",
					Version: "9.9.9",
				},
			},
		}
		s := newMockSystemInfoProvider(t)
		s.EXPECT().getSystemInfo(context.Background()).Return(&imSysInfo, nil)
		s.EXPECT().getExporterSystemInfo(mock.Anything, mock.Anything).Return(&exsysInfo, nil)
		client := newMockKubernetesClient(t)
		s.EXPECT().getPvcClient().Return(client)

		v := Validator{
			conf:               configuration.Coordinator{},
			namespace:          "",
			systemInfoProvider: s,
		}
		err := v.ValidateSystemInfo(context.Background())
		require.ErrorContains(t, err, "component testcomponent is installed in version 9.9.9 but needs to have version 1.2.3")
	})

	t.Run("should return error getting importer system info", func(t *testing.T) {

		s := newMockSystemInfoProvider(t)
		s.EXPECT().getSystemInfo(context.Background()).Return(nil, fmt.Errorf("testerror"))

		v := Validator{
			conf:               configuration.Coordinator{},
			namespace:          "",
			systemInfoProvider: s,
		}
		err := v.ValidateSystemInfo(context.Background())
		require.ErrorContains(t, err, "could not get importer system info: testerror")
	})

	t.Run("should return error getting exporter system info", func(t *testing.T) {

		s := newMockSystemInfoProvider(t)
		s.EXPECT().getSystemInfo(context.Background()).Return(&systemInfo{}, nil)
		s.EXPECT().getExporterSystemInfo(mock.Anything, mock.Anything).Return(nil, fmt.Errorf("testerror"))

		v := Validator{
			conf:               configuration.Coordinator{},
			namespace:          "",
			systemInfoProvider: s,
		}
		err := v.ValidateSystemInfo(context.Background())
		require.ErrorContains(t, err, "could not get exporter system info: testerror")
	})
}

func TestUpdatePVC(t *testing.T) {
	t.Run("importing dogu pvc size is large enough", func(t *testing.T) {
		v := Validator{
			conf:      configuration.Coordinator{},
			namespace: "",
		}
		exDogu := dogu{
			Name:    "",
			Version: "",
			Volume: volume{
				SizeInBytes: 10,
			},
		}
		imDogu := dogu{
			Name:    "",
			Version: "",
			Volume: volume{
				SizeInBytes: 10,
			},
		}
		var result *multierror.Error
		result = v.updatePVC(exDogu, imDogu, nil, result, context.Background())
		require.Nil(t, result)
	})
	t.Run("importing dogu pvc size is not large enough", func(t *testing.T) {
		v := Validator{
			conf:      configuration.Coordinator{},
			namespace: "",
		}
		exDogu := dogu{
			Name:    "",
			Version: "",
			Volume: volume{
				SizeInBytes: 10,
			},
		}
		imDogu := dogu{
			Name:    "",
			Version: "",
			Volume: volume{
				SizeInBytes: 3,
			},
		}
		pvc := &kubv1.PersistentVolumeClaim{
			TypeMeta:   metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{},
			Spec: kubv1.PersistentVolumeClaimSpec{
				Resources: kubv1.VolumeResourceRequirements{
					Requests: kubv1.ResourceList{},
				},
			},
			Status: kubv1.PersistentVolumeClaimStatus{},
		}
		pvcClient := newMockPvcClient(t)
		pvcClient.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).Return(pvc, nil)
		pvcClient.EXPECT().Update(mock.Anything, mock.Anything, mock.Anything).Return(nil, nil)
		var result *multierror.Error
		result = v.updatePVC(exDogu, imDogu, pvcClient, result, context.Background())
		require.Nil(t, result)
	})

	t.Run("can not find dogus volume", func(t *testing.T) {
		v := Validator{
			conf:      configuration.Coordinator{},
			namespace: "",
		}
		exDogu := dogu{
			Name:    "",
			Version: "",
			Volume: volume{
				SizeInBytes: 10,
			},
		}
		imDogu := dogu{
			Name:    "testDogu",
			Version: "",
			Volume: volume{
				SizeInBytes: 3,
			},
		}
		pvcClient := newMockPvcClient(t)
		pvcClient.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).Return(nil, fmt.Errorf("testerror"))
		var result *multierror.Error
		result = v.updatePVC(exDogu, imDogu, pvcClient, result, context.Background())
		require.ErrorContains(t, result, "dogu testDogu volume could not be found")
	})

	t.Run("can not update dogus volume", func(t *testing.T) {
		v := Validator{
			conf:      configuration.Coordinator{},
			namespace: "",
		}
		exDogu := dogu{
			Name:    "",
			Version: "",
			Volume: volume{
				SizeInBytes: 10,
			},
		}
		imDogu := dogu{
			Name:    "testDogu",
			Version: "",
			Volume: volume{
				SizeInBytes: 3,
			},
		}
		pvcClient := newMockPvcClient(t)
		pvc := &kubv1.PersistentVolumeClaim{
			TypeMeta:   metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{},
			Spec: kubv1.PersistentVolumeClaimSpec{
				Resources: kubv1.VolumeResourceRequirements{
					Requests: kubv1.ResourceList{},
				},
			},
			Status: kubv1.PersistentVolumeClaimStatus{},
		}
		pvcClient.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).Return(pvc, nil)
		pvcClient.EXPECT().Update(mock.Anything, mock.Anything, mock.Anything).Return(nil, fmt.Errorf("testerror"))
		var result *multierror.Error
		result = v.updatePVC(exDogu, imDogu, pvcClient, result, context.Background())
		require.ErrorContains(t, result, "dogu testDogu does not have enough volume capacity and the volume could not be resized")
	})
}
