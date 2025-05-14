package systeminfo

import (
	"context"
	"fmt"
	"github.com/cloudogu/ces-importer/api/exporter"
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
		dc := newMockDoguClient(t)
		pc := newMockPvcClient(t)
		v, err := NewValidator(p, dc, pc)
		require.NoError(t, err)
		require.Equal(t, v.systemInfoProvider, p)
		require.Equal(t, v.doguClient, dc)
		require.Equal(t, v.pvcClient, pc)
	})
}

func TestValidateSystemInfo(t *testing.T) {
	t.Run("should return with no error", func(t *testing.T) {
		sysInfo := exporter.SystemInfo{
			Dogus: []exporter.Dogu{
				{
					Name:    "testdogu",
					Version: "1.2.3",
					Volume: exporter.DoguVolume{
						SizeInBytes: 10,
					},
				},
			},
			Components: []exporter.Component{
				{
					Name:    "testcomponent",
					Version: "1.2.3",
				},
			},
		}
		s := newMockSystemInfoProvider(t)
		s.EXPECT().getImporterSystemInfo(context.Background()).Return(&sysInfo, nil)
		s.EXPECT().getExporterSystemInfo(mock.Anything).Return(&sysInfo, nil)

		v := Validator{
			systemInfoProvider: s,
		}
		err := v.Validate(context.Background())
		require.Nil(t, err)
	})

	t.Run("should return error mismatching dogu versions", func(t *testing.T) {
		exsysInfo := exporter.SystemInfo{
			Dogus: []exporter.Dogu{
				{
					Name:    "testdogu",
					Version: "1.2.3",
					Volume: exporter.DoguVolume{
						SizeInBytes: 10,
					},
				},
			},
			Components: []exporter.Component{
				{
					Name:    "testcomponent",
					Version: "1.2.3",
				},
			},
		}
		imSysInfo := exporter.SystemInfo{
			Dogus: []exporter.Dogu{
				{
					Name:    "testdogu",
					Version: "9.9.9",
					Volume: exporter.DoguVolume{
						SizeInBytes: 10,
					},
				},
			},
			Components: []exporter.Component{
				{
					Name:    "testcomponent",
					Version: "1.2.3",
				},
			},
		}
		s := newMockSystemInfoProvider(t)
		s.EXPECT().getImporterSystemInfo(context.Background()).Return(&imSysInfo, nil)
		s.EXPECT().getExporterSystemInfo(mock.Anything).Return(&exsysInfo, nil)

		v := Validator{
			systemInfoProvider: s,
		}
		err := v.Validate(context.Background())
		require.ErrorContains(t, err, "dogu testdogu is installed in version 9.9.9 but needs to have version 1.2.3")
	})

	t.Run("should return error dogu not installed", func(t *testing.T) {
		exsysInfo := exporter.SystemInfo{
			Dogus: []exporter.Dogu{
				{
					Name:    "testdogu",
					Version: "1.2.3",
					Volume: exporter.DoguVolume{
						SizeInBytes: 10,
					},
				},
			},
			Components: []exporter.Component{
				{
					Name:    "testcomponent",
					Version: "1.2.3",
				},
			},
		}
		imSysInfo := exporter.SystemInfo{
			Dogus: []exporter.Dogu{},
			Components: []exporter.Component{
				{
					Name:    "testcomponent",
					Version: "1.2.3",
				},
			},
		}
		s := newMockSystemInfoProvider(t)
		s.EXPECT().getImporterSystemInfo(context.Background()).Return(&imSysInfo, nil)
		s.EXPECT().getExporterSystemInfo(mock.Anything).Return(&exsysInfo, nil)

		v := Validator{
			systemInfoProvider: s,
		}
		err := v.Validate(context.Background())
		require.ErrorContains(t, err, "dogu testdogu is not installed (needed version: 1.2.3)")
	})

	t.Run("should return error component not installed", func(t *testing.T) {
		exsysInfo := exporter.SystemInfo{
			Dogus: []exporter.Dogu{
				{
					Name:    "testdogu",
					Version: "1.2.3",
					Volume: exporter.DoguVolume{
						SizeInBytes: 10,
					},
				},
			},
			Components: []exporter.Component{
				{
					Name:    "testcomponent",
					Version: "1.2.3",
				},
			},
		}
		imSysInfo := exporter.SystemInfo{
			Dogus: []exporter.Dogu{
				{
					Name:    "testdogu",
					Version: "1.2.3",
					Volume: exporter.DoguVolume{
						SizeInBytes: 10,
					},
				},
			},
			Components: []exporter.Component{},
		}
		s := newMockSystemInfoProvider(t)
		s.EXPECT().getImporterSystemInfo(context.Background()).Return(&imSysInfo, nil)
		s.EXPECT().getExporterSystemInfo(mock.Anything).Return(&exsysInfo, nil)

		v := Validator{
			systemInfoProvider: s,
		}
		err := v.Validate(context.Background())
		require.ErrorContains(t, err, "component testcomponent is not installed (needed version: 1.2.3)")
	})

	t.Run("should return error component mismatching component version", func(t *testing.T) {
		exsysInfo := exporter.SystemInfo{
			Dogus: []exporter.Dogu{
				{
					Name:    "testdogu",
					Version: "1.2.3",
					Volume: exporter.DoguVolume{
						SizeInBytes: 10,
					},
				},
			},
			Components: []exporter.Component{
				{
					Name:    "testcomponent",
					Version: "1.2.3",
				},
			},
		}
		imSysInfo := exporter.SystemInfo{
			Dogus: []exporter.Dogu{
				{
					Name:    "testdogu",
					Version: "1.2.3",
					Volume: exporter.DoguVolume{
						SizeInBytes: 10,
					},
				},
			},
			Components: []exporter.Component{
				{
					Name:    "testcomponent",
					Version: "9.9.9",
				},
			},
		}
		s := newMockSystemInfoProvider(t)
		s.EXPECT().getImporterSystemInfo(context.Background()).Return(&imSysInfo, nil)
		s.EXPECT().getExporterSystemInfo(mock.Anything).Return(&exsysInfo, nil)

		v := Validator{
			systemInfoProvider: s,
		}
		err := v.Validate(context.Background())
		require.ErrorContains(t, err, "component testcomponent is installed in version 9.9.9 but needs to have version 1.2.3")
	})

	t.Run("should return error getting importer system info", func(t *testing.T) {

		s := newMockSystemInfoProvider(t)
		s.EXPECT().getImporterSystemInfo(context.Background()).Return(nil, fmt.Errorf("testerror"))

		v := Validator{
			systemInfoProvider: s,
		}
		err := v.Validate(context.Background())
		require.ErrorContains(t, err, "could not get importer system info: testerror")
	})

	t.Run("should return error getting exporter system info", func(t *testing.T) {

		s := newMockSystemInfoProvider(t)
		s.EXPECT().getImporterSystemInfo(context.Background()).Return(&exporter.SystemInfo{}, nil)
		s.EXPECT().getExporterSystemInfo(mock.Anything).Return(nil, fmt.Errorf("testerror"))

		v := Validator{
			systemInfoProvider: s,
		}
		err := v.Validate(context.Background())
		require.ErrorContains(t, err, "could not get exporter system info: testerror")
	})
}

func TestUpdatePVC(t *testing.T) {
	t.Run("importing dogu pvc size is large enough", func(t *testing.T) {
		v := Validator{}
		exDogu := exporter.Dogu{
			Name:    "",
			Version: "",
			Volume: exporter.DoguVolume{
				SizeInBytes: 10,
			},
		}
		imDogu := exporter.Dogu{
			Name:    "",
			Version: "",
			Volume: exporter.DoguVolume{
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
		v := Validator{
			doguClient: doguClient,
			pvcClient:  pvcClient,
		}
		exDogu := exporter.Dogu{
			Name:    "",
			Version: "",
			Volume: exporter.DoguVolume{
				SizeInBytes: 2147483648,
			},
		}
		imDogu := exporter.Dogu{
			Name:    "",
			Version: "",
			Volume: exporter.DoguVolume{
				SizeInBytes: 1073741824,
			},
		}
		pvc := &kubv1.PersistentVolumeClaim{
			TypeMeta:   metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{},
			Spec: kubv1.PersistentVolumeClaimSpec{
				Resources: kubv1.VolumeResourceRequirements{
					Requests: kubv1.ResourceList{
						kubv1.ResourceStorage: resource.MustParse("2Gi"),
					},
				},
			},
			Status: kubv1.PersistentVolumeClaimStatus{
				Capacity: kubv1.ResourceList{
					kubv1.ResourceStorage: resource.MustParse("2Gi"),
				},
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

		waitSecondsBetweenRetries = 1
		defer func() {
			waitSecondsBetweenRetries = defaultWaitSecondsBetweenRetries
		}()

		pvcClient.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).Return(pvc, nil)
		doguClient.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).Return(&dogu, nil)
		doguClient.EXPECT().Update(mock.Anything, mock.Anything, mock.Anything).Return(nil, nil)

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
			doguClient: doguClient,
		}
		exDogu := exporter.Dogu{
			Name:    "",
			Version: "",
			Volume: exporter.DoguVolume{
				SizeInBytes: 10,
			},
		}
		imDogu := exporter.Dogu{
			Name:    "testDogu",
			Version: "",
			Volume: exporter.DoguVolume{
				SizeInBytes: 3,
			},
		}
		doguClient.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).Return(nil, fmt.Errorf("testerror"))

		waitSecondsBetweenRetries = 1
		defer func() {
			waitSecondsBetweenRetries = defaultWaitSecondsBetweenRetries
		}()

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
			doguClient: doguClient,
		}
		exDogu := exporter.Dogu{
			Name:    "",
			Version: "",
			Volume: exporter.DoguVolume{
				SizeInBytes: 10,
			},
		}
		imDogu := exporter.Dogu{
			Name:    "testDogu",
			Version: "",
			Volume: exporter.DoguVolume{
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

		waitSecondsBetweenRetries = 1
		defer func() {
			waitSecondsBetweenRetries = defaultWaitSecondsBetweenRetries
		}()

		c := make(chan error)
		go v.updatePVC(exDogu, imDogu, context.Background(), c)
		err := <-c
		if err != nil {
			require.ErrorContains(t, err, "dogu testDogu does not have enough volume capacity and the volume could not be resized")
		}
	})
}
