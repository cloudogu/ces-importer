package sync

import (
	"context"
	"fmt"
	"github.com/cloudogu/ces-importer/api/exporter"
	"github.com/cloudogu/ces-importer/configuration"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"io"
	"strings"
	"testing"
)

func TestSyncData(t *testing.T) {
	t.Run("should return with no error", func(t *testing.T) {
		cmd := newMockCommand(t)
		cmd.EXPECT().String().Return("cmd")
		l := strings.NewReader("log")
		lc := io.NopCloser(l)
		cmd.EXPECT().StdoutPipe().Return(lc, nil)
		e := strings.NewReader("error")
		ec := io.NopCloser(e)
		cmd.EXPECT().StderrPipe().Return(ec, nil)
		cmd.EXPECT().Start().Return(nil)
		cmd.EXPECT().Wait().Return(nil)
		commandMaker := func(name string, arg ...string) command {
			return cmd
		}

		systemInfoProvider := newMockSystemInfoProvider(t)
		exportDoguApiClient := newMockExportDoguApiClient(t)
		syncer := &RsyncSyncer{
			host:                "localhost",
			user:                "user",
			privateKeyPath:      "secret/private.key",
			makeCommand:         commandMaker,
			exportModeApiClient: exportDoguApiClient,
			systemInfoProvider:  systemInfoProvider,
			excludePattern: []configuration.ExcludePattern{
				{DoguName: "testDogu", Pattern: "*.file"},
			},
		}

		// system info request
		systemInfo := exporter.SystemInfo{
			FQDN:        "",
			IsMultinode: false,
			Dogus: []exporter.Dogu{
				{
					Name:    "official/test",
					Version: "",
					Volume:  exporter.DoguVolume{},
				},
			},
			Components: nil,
		}

		systemInfoProvider.EXPECT().GetSystemInfo(mock.Anything).Return(&systemInfo, nil)

		// set export dogu request
		export := exporter.DoguExport{
			Dogu:         "test",
			VolumePath:   "/a/b",
			ExporterPort: 1234,
		}
		exportDoguApiClient.EXPECT().SetExportDogu(mock.Anything, mock.Anything).Return(&export, nil)

		err := syncer.SyncData(context.Background())
		require.NoError(t, err)
	})

	t.Run("should fail to fetch system info", func(t *testing.T) {
		cmd := newMockCommand(t)
		commandMaker := func(name string, arg ...string) command {
			return cmd
		}

		systemInfoProvider := newMockSystemInfoProvider(t)
		exportDoguApiClient := newMockExportDoguApiClient(t)
		syncer := &RsyncSyncer{
			host:                "localhost",
			user:                "user",
			privateKeyPath:      "secret/private.key",
			makeCommand:         commandMaker,
			exportModeApiClient: exportDoguApiClient,
			systemInfoProvider:  systemInfoProvider,
		}

		systemInfoProvider.EXPECT().GetSystemInfo(mock.Anything).Return(nil, fmt.Errorf("testerror"))

		err := syncer.SyncData(context.Background())
		require.EqualError(t, err, "failed to fetch exporter system info: testerror")
	})

	t.Run("should fail to set export dogu", func(t *testing.T) {
		cmd := newMockCommand(t)
		commandMaker := func(name string, arg ...string) command {
			return cmd
		}
		systemInfoProvider := newMockSystemInfoProvider(t)
		exportDoguApiClient := newMockExportDoguApiClient(t)
		syncer := &RsyncSyncer{
			host:                "localhost",
			user:                "user",
			privateKeyPath:      "secret/private.key",
			makeCommand:         commandMaker,
			exportModeApiClient: exportDoguApiClient,
			systemInfoProvider:  systemInfoProvider,
		}

		// system info request
		systemInfo := exporter.SystemInfo{
			FQDN:        "",
			IsMultinode: false,
			Dogus: []exporter.Dogu{
				{
					Name:    "official/test",
					Version: "",
					Volume:  exporter.DoguVolume{},
				},
			},
			Components: nil,
		}
		systemInfoProvider.EXPECT().GetSystemInfo(mock.Anything).Return(&systemInfo, nil)
		exportDoguApiClient.EXPECT().SetExportDogu(mock.Anything, mock.Anything).Return(nil, fmt.Errorf("testerror"))

		err := syncer.SyncData(context.Background())
		require.EqualError(t, err, "failed to set dogu official/test as export dogu: testerror")
	})

	t.Run("should catch sync dogu error", func(t *testing.T) {
		cmd := newMockCommand(t)
		cmd.EXPECT().String().Return("cmd")
		l := strings.NewReader("log")
		lc := io.NopCloser(l)
		cmd.EXPECT().StdoutPipe().Return(lc, nil)
		e := strings.NewReader("error")
		ec := io.NopCloser(e)
		cmd.EXPECT().StderrPipe().Return(ec, nil)
		cmd.EXPECT().Start().Return(nil)
		cmd.EXPECT().Wait().Return(fmt.Errorf("testerror"))
		commandMaker := func(name string, arg ...string) command {
			return cmd
		}
		systemInfoProvider := newMockSystemInfoProvider(t)
		exportDoguApiClient := newMockExportDoguApiClient(t)
		syncer := &RsyncSyncer{
			host:                "localhost",
			user:                "user",
			privateKeyPath:      "secret/private.key",
			makeCommand:         commandMaker,
			exportModeApiClient: exportDoguApiClient,
			systemInfoProvider:  systemInfoProvider,
			excludePattern: []configuration.ExcludePattern{
				{DoguName: "testDogu", Pattern: "*.file"},
			},
		}

		// system info request
		systemInfo := exporter.SystemInfo{
			FQDN:        "",
			IsMultinode: false,
			Dogus: []exporter.Dogu{
				{
					Name:    "official/test",
					Version: "",
					Volume:  exporter.DoguVolume{},
				},
			},
			Components: nil,
		}
		systemInfoProvider.EXPECT().GetSystemInfo(mock.Anything).Return(&systemInfo, nil)

		// set export dogu request
		export := exporter.DoguExport{
			Dogu:         "test",
			VolumePath:   "/a/b",
			ExporterPort: 1234,
		}
		exportDoguApiClient.EXPECT().SetExportDogu(mock.Anything, mock.Anything).Return(&export, nil)

		err := syncer.SyncData(context.Background())
		require.EqualError(t, err, "failed to sync source /a/b to destination test: rsync exited with error: testerror")
	})
}

func TestSyncDogu(t *testing.T) {
	t.Run("should return with no error", func(t *testing.T) {
		cmd := newMockCommand(t)
		cmd.EXPECT().String().Return("command")
		l := strings.NewReader("log")
		lc := io.NopCloser(l)
		cmd.EXPECT().StdoutPipe().Return(lc, nil)
		e := strings.NewReader("error")
		ec := io.NopCloser(e)
		cmd.EXPECT().StderrPipe().Return(ec, nil)
		cmd.EXPECT().Start().Return(nil)
		cmd.EXPECT().Wait().Return(nil)
		commandMaker := func(name string, arg ...string) command {
			return cmd
		}
		systemInfoProvider := newMockSystemInfoProvider(t)
		exportDoguApiClient := newMockExportDoguApiClient(t)
		syncer := &RsyncSyncer{
			host:                "localhost",
			user:                "user",
			privateKeyPath:      "secret/private.key",
			makeCommand:         commandMaker,
			exportModeApiClient: exportDoguApiClient,
			systemInfoProvider:  systemInfoProvider,
		}

		exclude := configuration.ExcludePattern{
			DoguName: "test",
			Pattern:  "*.test",
		}
		err := syncer.SyncDoguDir(context.Background(), 1234, "data/dogu", "data/dogu", exclude, true)
		require.NoError(t, err)
	})
	t.Run("should fail to create std out pipe", func(t *testing.T) {
		cmd := newMockCommand(t)
		cmd.EXPECT().String().Return("command")
		cmd.EXPECT().StdoutPipe().Return(nil, fmt.Errorf("testerror"))
		commandMaker := func(name string, arg ...string) command {
			return cmd
		}
		systemInfoProvider := newMockSystemInfoProvider(t)
		exportDoguApiClient := newMockExportDoguApiClient(t)
		syncer := &RsyncSyncer{
			host:                "localhost",
			user:                "user",
			privateKeyPath:      "secret/private.key",
			makeCommand:         commandMaker,
			exportModeApiClient: exportDoguApiClient,
			systemInfoProvider:  systemInfoProvider,
		}

		exclude := configuration.ExcludePattern{}
		err := syncer.SyncDoguDir(context.Background(), 1234, "data/dogu", "data/dogu", exclude, false)
		require.EqualError(t, err, "error creating stdout pipe: testerror")
	})

	t.Run("should fail to create std err pipe", func(t *testing.T) {
		cmd := newMockCommand(t)
		cmd.EXPECT().String().Return("command")
		cmd.EXPECT().String().Return("command")
		l := strings.NewReader("log")
		lc := io.NopCloser(l)
		cmd.EXPECT().StdoutPipe().Return(lc, nil)
		cmd.EXPECT().StderrPipe().Return(nil, fmt.Errorf("testerror"))
		commandMaker := func(name string, arg ...string) command {
			return cmd
		}
		systemInfoProvider := newMockSystemInfoProvider(t)
		exportDoguApiClient := newMockExportDoguApiClient(t)
		syncer := &RsyncSyncer{
			host:                "localhost",
			user:                "user",
			privateKeyPath:      "secret/private.key",
			makeCommand:         commandMaker,
			exportModeApiClient: exportDoguApiClient,
			systemInfoProvider:  systemInfoProvider,
		}

		exclude := configuration.ExcludePattern{}
		err := syncer.SyncDoguDir(context.Background(), 1234, "data/dogu", "data/dogu", exclude, true)
		require.EqualError(t, err, "error creating stderr pipe: testerror")
	})

	t.Run("should fail to start command", func(t *testing.T) {
		cmd := newMockCommand(t)
		cmd.EXPECT().String().Return("command")
		l := strings.NewReader("log")
		lc := io.NopCloser(l)
		cmd.EXPECT().StdoutPipe().Return(lc, nil)
		e := strings.NewReader("error")
		ec := io.NopCloser(e)
		cmd.EXPECT().StderrPipe().Return(ec, nil)
		cmd.EXPECT().Start().Return(fmt.Errorf("testerror"))
		commandMaker := func(name string, arg ...string) command {
			return cmd
		}
		systemInfoProvider := newMockSystemInfoProvider(t)
		exportDoguApiClient := newMockExportDoguApiClient(t)
		syncer := &RsyncSyncer{
			host:                "localhost",
			user:                "user",
			privateKeyPath:      "secret/private.key",
			makeCommand:         commandMaker,
			exportModeApiClient: exportDoguApiClient,
			systemInfoProvider:  systemInfoProvider,
		}

		exclude := configuration.ExcludePattern{}
		err := syncer.SyncDoguDir(context.Background(), 1234, "data/dogu", "data/dogu", exclude, false)
		require.EqualError(t, err, "error starting rsync: testerror")
	})

	t.Run("rsync should throw error", func(t *testing.T) {
		cmd := newMockCommand(t)
		cmd.EXPECT().String().Return("command")
		l := strings.NewReader("log")
		lc := io.NopCloser(l)
		cmd.EXPECT().StdoutPipe().Return(lc, nil)
		e := strings.NewReader("error")
		ec := io.NopCloser(e)
		cmd.EXPECT().StderrPipe().Return(ec, nil)
		cmd.EXPECT().Start().Return(nil)
		cmd.EXPECT().Wait().Return(fmt.Errorf("testerror"))
		commandMaker := func(name string, arg ...string) command {
			return cmd
		}
		systemInfoProvider := newMockSystemInfoProvider(t)
		exportDoguApiClient := newMockExportDoguApiClient(t)
		syncer := &RsyncSyncer{
			host:                "localhost",
			user:                "user",
			privateKeyPath:      "secret/private.key",
			makeCommand:         commandMaker,
			exportModeApiClient: exportDoguApiClient,
			systemInfoProvider:  systemInfoProvider,
		}
		exclude := configuration.ExcludePattern{}
		err := syncer.SyncDoguDir(context.Background(), 1234, "data/dogu", "data/dogu", exclude, true)
		require.EqualError(t, err, "rsync exited with error: testerror")
	})
}
