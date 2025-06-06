package sync

import (
	"context"
	"fmt"
	"github.com/cloudogu/ces-importer/configuration"
	"github.com/cloudogu/ces-importer/migration"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"io"
	"os/exec"
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

		iterator := 0
		commandMaker := func(name string, arg ...string) command {
			subDir := ""
			switch iterator {
			case 0:
				subDir = "db"
			case 1:
				subDir = "localConfig"
			default:
				t.Error("unexpected call of make command")
			}
			iterator++

			assert.Equal(t, "rsync", name)
			assert.Len(t, arg, 8)
			assert.Equal(t, "-avhz", arg[0])
			assert.Equal(t, "--delete", arg[1])
			assert.Equal(t, "--sparse", arg[2])
			assert.Equal(t, "--stats", arg[3])
			assert.Equal(t, "-e", arg[4])
			assert.Equal(t, "ssh -p 1234 -l user -i secret/private.key -o StrictHostKeyChecking=no -o BatchMode=yes", arg[5])
			assert.Equal(t, fmt.Sprintf("localhost:/a/b/%s/", subDir), arg[6])
			assert.Equal(t, fmt.Sprintf("../../testdata/sync/test/%s", subDir), arg[7])

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
			doguVolumeBasePath:  "../../testdata/sync",
			excludePattern: []configuration.ExcludePattern{
				{DoguName: "testDogu", Pattern: "*.file"},
			},
		}

		// system info request
		systemInfo := migration.SystemInfo{
			FQDN:        "",
			IsMultinode: false,
			Dogus: []migration.Dogu{
				{
					Name:    "official/test",
					Version: "",
					Volume:  migration.DoguVolume{},
				},
			},
			Components: nil,
		}

		systemInfoProvider.EXPECT().GetSystemInfo(mock.Anything).Return(&systemInfo, nil)

		// set export dogu request
		export := migration.DoguExport{
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
		systemInfo := migration.SystemInfo{
			FQDN:        "",
			IsMultinode: false,
			Dogus: []migration.Dogu{
				{
					Name:    "official/test",
					Version: "",
					Volume:  migration.DoguVolume{},
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
		cmd.EXPECT().Wait().Return(assert.AnError)
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
			doguVolumeBasePath:  "../../testdata/sync",
			excludePattern: []configuration.ExcludePattern{
				{DoguName: "testDogu", Pattern: "*.file"},
			},
		}

		// system info request
		systemInfo := migration.SystemInfo{
			FQDN:        "",
			IsMultinode: false,
			Dogus: []migration.Dogu{
				{
					Name:    "official/test",
					Version: "",
					Volume:  migration.DoguVolume{},
				},
			},
			Components: nil,
		}
		systemInfoProvider.EXPECT().GetSystemInfo(mock.Anything).Return(&systemInfo, nil)

		// set export dogu request
		export := migration.DoguExport{
			Dogu:         "test",
			VolumePath:   "/a/b",
			ExporterPort: 1234,
		}
		exportDoguApiClient.EXPECT().SetExportDogu(mock.Anything, mock.Anything).Return(&export, nil)

		err := syncer.SyncData(context.Background())
		require.ErrorIs(t, err, assert.AnError)
		require.ErrorContains(t, err, "failed to sync source /a/b/db/ to destination ../../testdata/sync/test/db: rsync exited with error:")
	})

	t.Run("should not sync excluded dogu", func(t *testing.T) {
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
			doguVolumeBasePath:  "../../testdata/sync",
			excludedDogus:       []string{"official/test"},
		}

		// system info request
		systemInfo := migration.SystemInfo{
			FQDN:        "",
			IsMultinode: false,
			Dogus: []migration.Dogu{
				{
					Name:    "official/test",
					Version: "",
					Volume:  migration.DoguVolume{},
				},
			},
			Components: nil,
		}
		systemInfoProvider.EXPECT().GetSystemInfo(mock.Anything).Return(&systemInfo, nil)

		err := syncer.SyncData(context.Background())
		// this test is verified by there being no error and no calls to the cmd
		require.NoError(t, err)
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

func Test_getSubDirs(t *testing.T) {
	t.Run("should return with no error if importDir does not exists", func(t *testing.T) {
		dirs, err := getSubDirs("/does/not/exist")

		require.NoError(t, err)
		assert.Len(t, dirs, 0)
	})

	t.Run("should ignore subDirs", func(t *testing.T) {
		dirs, err := getSubDirs("../../testdata/sync/test")

		require.NoError(t, err)
		assert.Len(t, dirs, 2)
		assert.Equal(t, "db", dirs[0])
		assert.Equal(t, "localConfig", dirs[1])
	})
}

func TestNewRsyncSyncer(t *testing.T) {
	t.Run("should create new RsyncSyncer", func(t *testing.T) {
		systemInfoProvider := newMockSystemInfoProvider(t)
		exportDoguApiClient := newMockExportDoguApiClient(t)

		host := "host"
		user := "user"
		privateKeyPath := "/.ssh/private"
		doguVolumeBasePath := "/dogu/volume"
		excludePattern := []configuration.ExcludePattern{{DoguName: "test", Pattern: "*.test"}}
		excludedDogus := make([]string, 0)

		syncer := NewRsyncSyncer(host, user, privateKeyPath, exportDoguApiClient, systemInfoProvider, excludePattern, doguVolumeBasePath, excludedDogus)

		require.NotNil(t, syncer)
		assert.Equal(t, host, syncer.host)
		assert.Equal(t, user, syncer.user)
		assert.Equal(t, privateKeyPath, syncer.privateKeyPath)
		assert.Equal(t, exportDoguApiClient, syncer.exportModeApiClient)
		assert.Equal(t, systemInfoProvider, syncer.systemInfoProvider)
		assert.Equal(t, doguVolumeBasePath, syncer.doguVolumeBasePath)
		assert.Equal(t, excludePattern, syncer.excludePattern)
		assert.NotNil(t, syncer.makeCommand)
		assert.Equal(t, exec.Command("test", "a", "b"), syncer.makeCommand("test", "a", "b"))
	})
}
