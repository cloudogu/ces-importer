package systeminfo

import (
	"context"
	"fmt"
	"github.com/cloudogu/ces-importer/api/exporter"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
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
		require.Equal(t, v.doguVolumeResizer.(*defaultDoguVolumeResizer).doguClient, dc)
		require.Equal(t, v.doguVolumeResizer.(*defaultDoguVolumeResizer).pvcClient, pc)
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

		mVolumeResizer := newMockDoguVolumeResizer(t)
		mVolumeResizer.EXPECT().ResizeDogusIfNeeded(mock.Anything, sysInfo.Dogus, sysInfo.Dogus).Return(nil)

		v := Validator{
			systemInfoProvider: s,
			doguVolumeResizer:  mVolumeResizer,
		}
		err := v.Validate(context.Background())
		require.NoError(t, err)
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

		mVolumeResizer := newMockDoguVolumeResizer(t)
		mVolumeResizer.EXPECT().ResizeDogusIfNeeded(mock.Anything, exsysInfo.Dogus, imSysInfo.Dogus).Return(nil)

		v := Validator{
			systemInfoProvider: s,
			doguVolumeResizer:  mVolumeResizer,
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

		mVolumeResizer := newMockDoguVolumeResizer(t)
		mVolumeResizer.EXPECT().ResizeDogusIfNeeded(mock.Anything, exsysInfo.Dogus, imSysInfo.Dogus).Return(nil)

		v := Validator{
			systemInfoProvider: s,
			doguVolumeResizer:  mVolumeResizer,
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

		mVolumeResizer := newMockDoguVolumeResizer(t)
		mVolumeResizer.EXPECT().ResizeDogusIfNeeded(mock.Anything, exsysInfo.Dogus, imSysInfo.Dogus).Return(nil)

		v := Validator{
			systemInfoProvider: s,
			doguVolumeResizer:  mVolumeResizer,
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

		mVolumeResizer := newMockDoguVolumeResizer(t)
		mVolumeResizer.EXPECT().ResizeDogusIfNeeded(mock.Anything, exsysInfo.Dogus, imSysInfo.Dogus).Return(nil)

		v := Validator{
			systemInfoProvider: s,
			doguVolumeResizer:  mVolumeResizer,
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

		mVolumeResizer := newMockDoguVolumeResizer(t)
		//mVolumeResizer.EXPECT().ResizeDogusIfNeeded(mock.Anything, nil, nil).Return(nil)

		v := Validator{
			systemInfoProvider: s,
			doguVolumeResizer:  mVolumeResizer,
		}
		err := v.Validate(context.Background())
		require.ErrorContains(t, err, "could not get exporter system info: testerror")
	})

	t.Run("should error on dogu not installed in exporting system", func(t *testing.T) {
		imSysInfo := exporter.SystemInfo{
			Dogus: []exporter.Dogu{
				{
					Name:    "testdogu",
					Version: "1.2.3",
					Volume: exporter.DoguVolume{
						SizeInBytes: 10,
					},
				},
				{
					Name:    "onlyPresentHere",
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
		exSysInfo := exporter.SystemInfo{
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
		s.EXPECT().getImporterSystemInfo(context.Background()).Return(&imSysInfo, nil)
		s.EXPECT().getExporterSystemInfo(mock.Anything).Return(&exSysInfo, nil)

		mVolumeResizer := newMockDoguVolumeResizer(t)
		mVolumeResizer.EXPECT().ResizeDogusIfNeeded(mock.Anything, exSysInfo.Dogus, imSysInfo.Dogus).Return(nil)

		v := Validator{
			systemInfoProvider: s,
			doguVolumeResizer:  mVolumeResizer,
		}
		err := v.Validate(context.Background())
		require.ErrorContains(t, err, "dogu onlyPresentHere is installed in the importing system but not present in the exporting system")
	})

	t.Run("should validate special nginx case", func(t *testing.T) {
		imSysInfo := exporter.SystemInfo{
			Dogus: []exporter.Dogu{
				{
					Name:    "k8s/nginx-static",
					Version: "1.2.3",
					Volume: exporter.DoguVolume{
						SizeInBytes: 10,
					},
				},
				{
					Name:    "k8s/nginx-ingress",
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
		exSysInfo := exporter.SystemInfo{
			Dogus: []exporter.Dogu{
				{
					Name:    "official/nginx",
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
		s.EXPECT().getImporterSystemInfo(context.Background()).Return(&imSysInfo, nil)
		s.EXPECT().getExporterSystemInfo(mock.Anything).Return(&exSysInfo, nil)

		mVolumeResizer := newMockDoguVolumeResizer(t)
		mVolumeResizer.EXPECT().ResizeDogusIfNeeded(mock.Anything, exSysInfo.Dogus, imSysInfo.Dogus).Return(nil)

		v := Validator{
			systemInfoProvider: s,
			doguVolumeResizer:  mVolumeResizer,
		}
		err := v.Validate(context.Background())
		require.NoError(t, err)
	})

	t.Run("should throw error on nginx-static missing when validating nginx dogu", func(t *testing.T) {
		imSysInfo := exporter.SystemInfo{
			Dogus: []exporter.Dogu{
				{
					Name:    "k8s/nginx-ingress",
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
		exSysInfo := exporter.SystemInfo{
			Dogus: []exporter.Dogu{
				{
					Name:    "official/nginx",
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
		s.EXPECT().getImporterSystemInfo(context.Background()).Return(&imSysInfo, nil)
		s.EXPECT().getExporterSystemInfo(mock.Anything).Return(&exSysInfo, nil)

		mVolumeResizer := newMockDoguVolumeResizer(t)
		mVolumeResizer.EXPECT().ResizeDogusIfNeeded(mock.Anything, exSysInfo.Dogus, imSysInfo.Dogus).Return(nil)

		v := Validator{
			systemInfoProvider: s,
			doguVolumeResizer:  mVolumeResizer,
		}
		err := v.Validate(context.Background())
		require.ErrorContains(t, err, "dogu k8s/nginx-static is not installed")
	})

	t.Run("should throw no error on excluded dogu", func(t *testing.T) {
		imSysInfo := exporter.SystemInfo{
			Dogus: []exporter.Dogu{},
			Components: []exporter.Component{
				{
					Name:    "testcomponent",
					Version: "1.2.3",
				},
			},
		}
		exSysInfo := exporter.SystemInfo{
			Dogus: []exporter.Dogu{
				{
					Name:    "official/registrator",
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
		s.EXPECT().getImporterSystemInfo(context.Background()).Return(&imSysInfo, nil)
		s.EXPECT().getExporterSystemInfo(mock.Anything).Return(&exSysInfo, nil)

		mVolumeResizer := newMockDoguVolumeResizer(t)
		mVolumeResizer.EXPECT().ResizeDogusIfNeeded(mock.Anything, exSysInfo.Dogus, imSysInfo.Dogus).Return(nil)

		v := Validator{
			systemInfoProvider: s,
			doguVolumeResizer:  mVolumeResizer,
		}
		err := v.Validate(context.Background())
		require.NoError(t, err)
	})
}
