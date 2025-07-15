package main

import (
	"context"
	"github.com/cloudogu/ces-importer/configuration"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestNewPreflightExecuter(t *testing.T) {
	t.Run("should create PreflightExecuter without errors", func(t *testing.T) {
		hc := newMockHealthClient(t)
		sig := newMockSystemInfoGetter(t)
		sc := newMockSecretClient(t)
		edc := newMockExportDoguClient(t)

		pe := newPreflightExecuter(hc, edc, sig, sc)

		require.Equal(t, hc, pe.healthClient)
		require.Equal(t, sig, pe.systemInfoGetter)
		require.Equal(t, sc, pe.secretClient)
	})
}

func TestRunPreflightCheck(t *testing.T) {
	t.Run("should return no errors", func(t *testing.T) {
		hc := newMockHealthClient(t)
		hc.EXPECT().GetIsHealthy(mock.Anything).Return(true, nil)

		sig := newMockSystemInfoGetter(t)
		sig.EXPECT().GetImporterSystemInfo(mock.Anything).Return(nil, nil)

		sshct := newMockTestSSHConnection(t)
		sshct.EXPECT().Execute(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

		pe := PreflightExecuter{
			healthClient:     hc,
			systemInfoGetter: sig,
			secretClient:     nil,

			testSSHConnection: sshct.Execute,
		}
		cfg := configuration.Coordinator{
			Logging:      configuration.Logging{},
			API:          configuration.API{},
			Migration:    configuration.Migration{},
			SSH:          configuration.SSH{},
			JobConfig:    configuration.JobConfig{},
			JobContainer: configuration.JobContainer{},
			Smtp:         configuration.Smtp{},
			General:      configuration.General{},
		}
		err := pe.runPreflightCheck(context.Background(), cfg)
		require.NoError(t, err)
	})
	t.Run("should error on getting health status", func(t *testing.T) {
		hc := newMockHealthClient(t)
		hc.EXPECT().GetIsHealthy(mock.Anything).Return(false, assert.AnError)

		sig := newMockSystemInfoGetter(t)
		sig.EXPECT().GetImporterSystemInfo(mock.Anything).Return(nil, nil)

		sshct := newMockTestSSHConnection(t)
		sshct.EXPECT().Execute(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

		pe := PreflightExecuter{
			healthClient:     hc,
			systemInfoGetter: sig,
			secretClient:     nil,

			testSSHConnection: sshct.Execute,
		}
		cfg := configuration.Coordinator{
			Logging:      configuration.Logging{},
			API:          configuration.API{},
			Migration:    configuration.Migration{},
			SSH:          configuration.SSH{},
			JobConfig:    configuration.JobConfig{},
			JobContainer: configuration.JobContainer{},
			Smtp:         configuration.Smtp{},
			General:      configuration.General{},
		}
		err := pe.runPreflightCheck(context.Background(), cfg)
		require.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "unable to determine exporter health status")
	})

	t.Run("should error on unhealthy exporter", func(t *testing.T) {
		hc := newMockHealthClient(t)
		hc.EXPECT().GetIsHealthy(mock.Anything).Return(false, nil)

		sig := newMockSystemInfoGetter(t)
		sig.EXPECT().GetImporterSystemInfo(mock.Anything).Return(nil, nil)

		sshct := newMockTestSSHConnection(t)
		sshct.EXPECT().Execute(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

		pe := PreflightExecuter{
			healthClient:     hc,
			systemInfoGetter: sig,
			secretClient:     nil,

			testSSHConnection: sshct.Execute,
		}
		cfg := configuration.Coordinator{
			Logging:      configuration.Logging{},
			API:          configuration.API{},
			Migration:    configuration.Migration{},
			SSH:          configuration.SSH{},
			JobConfig:    configuration.JobConfig{},
			JobContainer: configuration.JobContainer{},
			Smtp:         configuration.Smtp{},
			General:      configuration.General{},
		}
		err := pe.runPreflightCheck(context.Background(), cfg)
		assert.ErrorContains(t, err, "exporter health status is unhealthy")
	})

	t.Run("should error on getting system info", func(t *testing.T) {
		hc := newMockHealthClient(t)
		hc.EXPECT().GetIsHealthy(mock.Anything).Return(true, nil)

		sig := newMockSystemInfoGetter(t)
		sig.EXPECT().GetImporterSystemInfo(mock.Anything).Return(nil, assert.AnError)

		sshct := newMockTestSSHConnection(t)
		sshct.EXPECT().Execute(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

		pe := PreflightExecuter{
			healthClient:      hc,
			systemInfoGetter:  sig,
			secretClient:      nil,
			testSSHConnection: sshct.Execute,
		}
		cfg := configuration.Coordinator{
			Logging:      configuration.Logging{},
			API:          configuration.API{},
			Migration:    configuration.Migration{},
			SSH:          configuration.SSH{},
			JobConfig:    configuration.JobConfig{},
			JobContainer: configuration.JobContainer{},
			Smtp:         configuration.Smtp{},
			General:      configuration.General{},
		}
		err := pe.runPreflightCheck(context.Background(), cfg)
		require.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "unable to retrieve current systems system info")
	})

	t.Run("should error on testing ssh connection", func(t *testing.T) {
		hc := newMockHealthClient(t)
		hc.EXPECT().GetIsHealthy(mock.Anything).Return(true, nil)

		sig := newMockSystemInfoGetter(t)
		sig.EXPECT().GetImporterSystemInfo(mock.Anything).Return(nil, nil)

		sshct := newMockTestSSHConnection(t)
		sshct.EXPECT().Execute(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(assert.AnError)

		pe := PreflightExecuter{
			healthClient:      hc,
			systemInfoGetter:  sig,
			secretClient:      nil,
			testSSHConnection: sshct.Execute,
		}
		cfg := configuration.Coordinator{
			Logging:      configuration.Logging{},
			API:          configuration.API{},
			Migration:    configuration.Migration{},
			SSH:          configuration.SSH{},
			JobConfig:    configuration.JobConfig{},
			JobContainer: configuration.JobContainer{},
			Smtp:         configuration.Smtp{},
			General:      configuration.General{},
		}
		err := pe.runPreflightCheck(context.Background(), cfg)
		require.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "unable to test ssh connection")
	})
}
