package sync

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/require"
	"io"
	"strings"
	"testing"
)

func TestGetExporterSystemInfo(t *testing.T) {
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

		err := syncer.SyncDogu(context.Background(), 1234, "data/dogu", "data/dogu", "*.file", true)
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

		err := syncer.SyncDogu(context.Background(), 1234, "data/dogu", "data/dogu", "*.file", false)
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

		err := syncer.SyncDogu(context.Background(), 1234, "data/dogu", "data/dogu", "*.file", true)
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

		err := syncer.SyncDogu(context.Background(), 1234, "data/dogu", "data/dogu", "*.file", false)
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

		err := syncer.SyncDogu(context.Background(), 1234, "data/dogu", "data/dogu", "*.file", true)
		require.EqualError(t, err, "rsync exited with error: testerror")
	})
}
