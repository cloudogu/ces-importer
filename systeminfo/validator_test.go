package systeminfo

import (
	"context"
	"fmt"
	"github.com/cloudogu/ces-importer/configuration"
	v2 "github.com/cloudogu/k8s-dogu-operator/v3/api/v2"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	kubv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func TestNewValidator(t *testing.T) {
	t.Run("should return new validator", func(t *testing.T) {
		p := newMockSystemInfoProvider(t)
		n := "namespace"
		v, err := NewValidator(configuration.Configuration{}, n, p)
		require.NoError(t, err)
		require.Equal(t, v.conf, configuration.Configuration{})
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

		v := Validator{
			conf:               configuration.Configuration{},
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

		v := Validator{
			conf:               configuration.Configuration{},
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

		v := Validator{
			conf:               configuration.Configuration{},
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

		v := Validator{
			conf:               configuration.Configuration{},
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

		v := Validator{
			conf:               configuration.Configuration{},
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
			conf:               configuration.Configuration{},
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
			conf:               configuration.Configuration{},
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
			conf:      configuration.Configuration{},
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
		c := make(chan error)
		go v.updatePVC(exDogu, imDogu, context.Background(), c)
		err := <-c
		if err != nil {
			require.NoError(t, err)
		}
	})
	t.Run("importing dogu pvc size is not large enough", func(t *testing.T) {
		doguClient := newMockDoguClient(t)
		pvcClient := newMockPvcClient(t)
		pvClient := newMockPvClient(t)
		v := Validator{
			conf:       configuration.Configuration{},
			namespace:  "",
			doguClient: doguClient,
			pvcClient:  pvcClient,
			pvClient:   pvClient,
		}
		exDogu := dogu{
			Name:    "",
			Version: "",
			Volume: volume{
				SizeInBytes: 2147483648,
			},
		}
		imDogu := dogu{
			Name:    "",
			Version: "",
			Volume: volume{
				SizeInBytes: 1073741824,
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
		dogu := v2.Dogu{
			TypeMeta:   metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{},
			Spec: v2.DoguSpec{
				Resources: v2.DoguResources{},
			},
			Status: v2.DoguStatus{},
		}
		pv := &kubv1.PersistentVolume{
			TypeMeta:   metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{},
			Spec: kubv1.PersistentVolumeSpec{
				Capacity: kubv1.ResourceList{
					kubv1.ResourceStorage: resource.MustParse("2Gi"),
				},
			},
			Status: kubv1.PersistentVolumeStatus{},
		}
		pvcClient.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).Return(pvc, nil)
		doguClient.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).Return(&dogu, nil)
		doguClient.EXPECT().Update(mock.Anything, mock.Anything, mock.Anything).Return(nil, nil)
		pvClient.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).Return(pv, nil)

		c := make(chan error)
		go v.updatePVC(exDogu, imDogu, context.Background(), c)
		err := <-c
		if err != nil {
			require.NoError(t, err)
		}
	})

	t.Run("can not find dogus volume", func(t *testing.T) {
		doguClient := newMockDoguClient(t)
		v := Validator{
			conf:       configuration.Configuration{},
			namespace:  "",
			doguClient: doguClient,
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
		doguClient.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).Return(nil, fmt.Errorf("testerror"))
		c := make(chan error)
		go v.updatePVC(exDogu, imDogu, context.Background(), c)
		err := <-c
		if err != nil {
			require.ErrorContains(t, err, "dogu testDogu volume could not be found")
		}
	})

	t.Run("can not update dogus volume", func(t *testing.T) {
		doguClient := newMockDoguClient(t)
		v := Validator{
			conf:       configuration.Configuration{},
			namespace:  "",
			doguClient: doguClient,
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
		dogu := v2.Dogu{
			TypeMeta:   metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{},
			Spec: v2.DoguSpec{
				Resources: v2.DoguResources{},
			},
			Status: v2.DoguStatus{},
		}
		doguClient.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).Return(&dogu, nil)
		doguClient.EXPECT().Update(mock.Anything, mock.Anything, mock.Anything).Return(nil, fmt.Errorf("testerror"))
		c := make(chan error)
		go v.updatePVC(exDogu, imDogu, context.Background(), c)
		err := <-c
		if err != nil {
			require.ErrorContains(t, err, "dogu testDogu does not have enough volume capacity and the volume could not be resized")
		}
	})
}
