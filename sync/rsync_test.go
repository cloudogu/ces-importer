package sync

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/cloudogu/ces-importer/api/exporter"
	"github.com/cloudogu/ces-importer/configuration"
	"github.com/stretchr/testify/require"
	"io"
	"strings"
	"testing"
)

func TestSyncData(t *testing.T) {
	t.Run("should return with no error", func(t *testing.T) {
		command := NewMockCommand(t)
		command.EXPECT().String().Return("command")
		l := strings.NewReader("log")
		lc := io.NopCloser(l)
		command.EXPECT().StdoutPipe().Return(lc, nil)
		e := strings.NewReader("error")
		ec := io.NopCloser(e)
		command.EXPECT().StderrPipe().Return(ec, nil)
		command.EXPECT().Start().Return(nil)
		command.EXPECT().Wait().Return(nil)
		commandMaker := func(name string, arg ...string) Command {
			return command
		}
		syncer := NewRsyncSyncer("localhost", "user", "secret/private.key", commandMaker)

		client := NewMockApiCli(t)
		// system info request
		systemInfo := exporter.SystemInfo{
			FQDN:        "",
			IsMultinode: false,
			Dogus: []exporter.Dogu{
				{
					Name:    "test",
					Version: "",
					Volume:  exporter.DoguVolume{},
				},
			},
			Components: nil,
		}
		sIbytes, err := json.Marshal(systemInfo)
		require.NoError(t, err)
		client.EXPECT().DoGetRequest(context.Background(), "https:///system-info").Return(sIbytes, nil)

		// set export dogu request
		export := exporter.DoguExport{
			Dogu:         "test",
			VolumePath:   "/a/b",
			ExporterPort: 1234,
		}
		eBytes, err := json.Marshal(export)
		client.EXPECT().DoPostRequest(context.Background(), "https:///export/dogu", nil, []string{"test"}).Return(eBytes, nil)

		config := configuration.Job{
			JobConfig: configuration.JobConfig{
				Exclude: []configuration.Exclude{
					{DoguName: "testDogu", Pattern: "*.file"},
				},
			},
		}
		err = syncer.SyncData(context.Background(), client, config)
		require.NoError(t, err)
	})

	t.Run("should fail to fetch system info", func(t *testing.T) {
		command := NewMockCommand(t)
		commandMaker := func(name string, arg ...string) Command {
			return command
		}
		syncer := NewRsyncSyncer("localhost", "user", "secret/private.key", commandMaker)

		client := NewMockApiCli(t)
		// system info request
		client.EXPECT().DoGetRequest(context.Background(), "https:///system-info").Return(nil, fmt.Errorf("testerror"))

		config := configuration.Job{}
		err := syncer.SyncData(context.Background(), client, config)
		require.EqualError(t, err, "failed to fetch exporter system info: testerror")
	})

	t.Run("should fail to set export dogu", func(t *testing.T) {
		command := NewMockCommand(t)
		commandMaker := func(name string, arg ...string) Command {
			return command
		}
		syncer := NewRsyncSyncer("localhost", "user", "secret/private.key", commandMaker)

		client := NewMockApiCli(t)
		// system info request
		systemInfo := exporter.SystemInfo{
			FQDN:        "",
			IsMultinode: false,
			Dogus: []exporter.Dogu{
				{
					Name:    "test",
					Version: "",
					Volume:  exporter.DoguVolume{},
				},
			},
			Components: nil,
		}
		sIbytes, err := json.Marshal(systemInfo)
		require.NoError(t, err)
		client.EXPECT().DoGetRequest(context.Background(), "https:///system-info").Return(sIbytes, nil)

		client.EXPECT().DoPostRequest(context.Background(), "https:///export/dogu", nil, []string{"test"}).Return(nil, fmt.Errorf("testerror"))

		config := configuration.Job{}
		err = syncer.SyncData(context.Background(), client, config)
		require.EqualError(t, err, "failed to set dogu test as export dogu: testerror")
	})

	t.Run("should catch sync dogu error", func(t *testing.T) {
		command := NewMockCommand(t)
		command.EXPECT().String().Return("command")
		l := strings.NewReader("log")
		lc := io.NopCloser(l)
		command.EXPECT().StdoutPipe().Return(lc, nil)
		e := strings.NewReader("error")
		ec := io.NopCloser(e)
		command.EXPECT().StderrPipe().Return(ec, nil)
		command.EXPECT().Start().Return(nil)
		command.EXPECT().Wait().Return(fmt.Errorf("testerror"))
		commandMaker := func(name string, arg ...string) Command {
			return command
		}
		syncer := NewRsyncSyncer("localhost", "user", "secret/private.key", commandMaker)

		client := NewMockApiCli(t)
		// system info request
		systemInfo := exporter.SystemInfo{
			FQDN:        "",
			IsMultinode: false,
			Dogus: []exporter.Dogu{
				{
					Name:    "test",
					Version: "",
					Volume:  exporter.DoguVolume{},
				},
			},
			Components: nil,
		}
		sIbytes, err := json.Marshal(systemInfo)
		require.NoError(t, err)
		client.EXPECT().DoGetRequest(context.Background(), "https:///system-info").Return(sIbytes, nil)

		// set export dogu request
		export := exporter.DoguExport{
			Dogu:         "test",
			VolumePath:   "/a/b",
			ExporterPort: 1234,
		}
		eBytes, err := json.Marshal(export)
		client.EXPECT().DoPostRequest(context.Background(), "https:///export/dogu", nil, []string{"test"}).Return(eBytes, nil)

		config := configuration.Job{
			JobConfig: configuration.JobConfig{
				Exclude: []configuration.Exclude{
					{DoguName: "testDogu", Pattern: "*.file"},
				},
			},
		}
		err = syncer.SyncData(context.Background(), client, config)
		require.EqualError(t, err, "failed to sync source /a/b to destination test: rsync exited with error: testerror")
	})

	t.Run("should fail to read export dogu response", func(t *testing.T) {
		command := NewMockCommand(t)
		commandMaker := func(name string, arg ...string) Command {
			return command
		}
		syncer := NewRsyncSyncer("localhost", "user", "secret/private.key", commandMaker)

		client := NewMockApiCli(t)
		// system info request
		systemInfo := exporter.SystemInfo{
			FQDN:        "",
			IsMultinode: false,
			Dogus: []exporter.Dogu{
				{
					Name:    "test",
					Version: "",
					Volume:  exporter.DoguVolume{},
				},
			},
			Components: nil,
		}
		sIbytes, err := json.Marshal(systemInfo)
		require.NoError(t, err)
		client.EXPECT().DoGetRequest(context.Background(), "https:///system-info").Return(sIbytes, nil)

		eBytes := []byte("bad response")
		client.EXPECT().DoPostRequest(context.Background(), "https:///export/dogu", nil, []string{"test"}).Return(eBytes, nil)

		config := configuration.Job{
			JobConfig: configuration.JobConfig{
				Exclude: []configuration.Exclude{
					{DoguName: "testDogu", Pattern: "*.file"},
				},
			},
		}
		err = syncer.SyncData(context.Background(), client, config)
		require.EqualError(t, err, "failed to parse dogu export response: \"bad response\": invalid character 'b' looking for beginning of value")
	})
}

func TestSyncDogu(t *testing.T) {
	t.Run("should return with no error", func(t *testing.T) {
		command := NewMockCommand(t)
		command.EXPECT().String().Return("command")
		l := strings.NewReader("log")
		lc := io.NopCloser(l)
		command.EXPECT().StdoutPipe().Return(lc, nil)
		e := strings.NewReader("error")
		ec := io.NopCloser(e)
		command.EXPECT().StderrPipe().Return(ec, nil)
		command.EXPECT().Start().Return(nil)
		command.EXPECT().Wait().Return(nil)
		commandMaker := func(name string, arg ...string) Command {
			return command
		}
		syncer := NewRsyncSyncer("localhost", "user", "secret/private.key", commandMaker)

		exclude := configuration.Exclude{
			DoguName: "test",
			Pattern:  "*.test",
		}
		err := syncer.SyncDogu(context.Background(), 1234, "data/dogu", "data/dogu", exclude, true)
		require.NoError(t, err)
	})
	t.Run("should fail to create std out pipe", func(t *testing.T) {
		command := NewMockCommand(t)
		command.EXPECT().String().Return("command")
		command.EXPECT().StdoutPipe().Return(nil, fmt.Errorf("testerror"))
		commandMaker := func(name string, arg ...string) Command {
			return command
		}
		syncer := NewRsyncSyncer("localhost", "user", "secret/private.key", commandMaker)

		exclude := configuration.Exclude{}
		err := syncer.SyncDogu(context.Background(), 1234, "data/dogu", "data/dogu", exclude, false)
		require.EqualError(t, err, "error creating stdout pipe: testerror")
	})

	t.Run("should fail to create std err pipe", func(t *testing.T) {
		command := NewMockCommand(t)
		command.EXPECT().String().Return("command")
		command.EXPECT().String().Return("command")
		l := strings.NewReader("log")
		lc := io.NopCloser(l)
		command.EXPECT().StdoutPipe().Return(lc, nil)
		command.EXPECT().StderrPipe().Return(nil, fmt.Errorf("testerror"))
		commandMaker := func(name string, arg ...string) Command {
			return command
		}
		syncer := NewRsyncSyncer("localhost", "user", "secret/private.key", commandMaker)

		exclude := configuration.Exclude{}
		err := syncer.SyncDogu(context.Background(), 1234, "data/dogu", "data/dogu", exclude, true)
		require.EqualError(t, err, "error creating stderr pipe: testerror")
	})

	t.Run("should fail to start command", func(t *testing.T) {
		command := NewMockCommand(t)
		command.EXPECT().String().Return("command")
		l := strings.NewReader("log")
		lc := io.NopCloser(l)
		command.EXPECT().StdoutPipe().Return(lc, nil)
		e := strings.NewReader("error")
		ec := io.NopCloser(e)
		command.EXPECT().StderrPipe().Return(ec, nil)
		command.EXPECT().Start().Return(fmt.Errorf("testerror"))
		commandMaker := func(name string, arg ...string) Command {
			return command
		}
		syncer := NewRsyncSyncer("localhost", "user", "secret/private.key", commandMaker)

		exclude := configuration.Exclude{}
		err := syncer.SyncDogu(context.Background(), 1234, "data/dogu", "data/dogu", exclude, false)
		require.EqualError(t, err, "error starting rsync: testerror")
	})

	t.Run("rsync should throw error", func(t *testing.T) {
		command := NewMockCommand(t)
		command.EXPECT().String().Return("command")
		l := strings.NewReader("log")
		lc := io.NopCloser(l)
		command.EXPECT().StdoutPipe().Return(lc, nil)
		e := strings.NewReader("error")
		ec := io.NopCloser(e)
		command.EXPECT().StderrPipe().Return(ec, nil)
		command.EXPECT().Start().Return(nil)
		command.EXPECT().Wait().Return(fmt.Errorf("testerror"))
		commandMaker := func(name string, arg ...string) Command {
			return command
		}
		syncer := NewRsyncSyncer("localhost", "user", "secret/private.key", commandMaker)

		exclude := configuration.Exclude{}
		err := syncer.SyncDogu(context.Background(), 1234, "data/dogu", "data/dogu", exclude, true)
		require.EqualError(t, err, "rsync exited with error: testerror")
	})
}

func TestFetchSystemInfo(t *testing.T) {
	t.Run("should return with no error", func(t *testing.T) {
		client := NewMockApiCli(t)
		// system info request
		systemInfo := exporter.SystemInfo{
			FQDN:        "",
			IsMultinode: false,
			Dogus: []exporter.Dogu{
				{
					Name:    "test",
					Version: "",
					Volume:  exporter.DoguVolume{},
				},
			},
			Components: nil,
		}
		sIbytes, err := json.Marshal(systemInfo)
		require.NoError(t, err)
		client.EXPECT().DoGetRequest(context.Background(), "https:///system-info").Return(sIbytes, nil)
		exporterSysInfo, err := fetchExporterSystemInfo(context.Background(), "", client)
		require.NoError(t, err)
		require.Equal(t, systemInfo, *exporterSysInfo)
	})

	t.Run("should fail to marshal return value", func(t *testing.T) {
		client := NewMockApiCli(t)
		sIBytes := []byte("no json")
		client.EXPECT().DoGetRequest(context.Background(), "https:///system-info").Return(sIBytes, nil)
		_, err := fetchExporterSystemInfo(context.Background(), "", client)
		require.EqualError(t, err, "failed to parse system info response: \"no json\": invalid character 'o' in literal null (expecting 'u')")
	})
}
