package migration

import (
	"context"
	"fmt"
	"github.com/cloudogu/ces-importer/api/exporter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"
)

func TestMigrator_RunMigration(t *testing.T) {
	exporterInfo := &exporter.SystemInfo{
		Dogus: []exporter.Dogu{{Name: "exporter/dogu"}},
	}
	importerInfo := &exporter.SystemInfo{
		Dogus: []exporter.Dogu{{Name: "importer/dogu"}},
	}

	t.Run("should run delta migration", func(t *testing.T) {
		testCtx := context.Background()

		mExportModeValidator := NewMockExportModeValidator(t)
		mExportModeValidator.EXPECT().Validate(testCtx).Return(nil)

		mSystemInfoProvider := NewMockSystemInfoProvider(t)
		mSystemInfoProvider.EXPECT().GetExporterSystemInfo(testCtx).Return(exporterInfo, nil)
		mSystemInfoProvider.EXPECT().GetImporterSystemInfo(testCtx).Return(importerInfo, nil)

		mSystemInfoValidator := NewMockSystemInfoValidator(t)
		mSystemInfoValidator.EXPECT().Validate(testCtx, exporterInfo, importerInfo).Return(nil)

		mDoguVolumeResizer := NewMockDoguVolumeResizer(t)
		mDoguVolumeResizer.EXPECT().ResizeDogusIfNeeded(testCtx, exporterInfo.Dogus, importerInfo.Dogus).Return(nil)

		mMaintenanceModeHandler := NewMockMaintenanceModeHandler(t)

		mMailSender := NewMockMailSender(t)
		mMailSender.EXPECT().Send(testCtx, false, nil, mock.Anything, mock.Anything).Return(nil)

		jobLogs := io.NopCloser(strings.NewReader("test"))

		mLogWriter := NewMockLogWriter(t)
		mLogWriter.EXPECT().Write(jobLogs).Return(nil)

		mlogIntializer := NewMockLogInitializer(t)
		mlogIntializer.EXPECT().InitializeWithLogFile().Return(nil)

		mJobRunner := NewMockJobRunner(t)
		mJobRunner.EXPECT().Run(testCtx).Return(jobLogs, nil)

		mDoguStopper := NewMockDoguStopper(t)
		mDoguStopper.EXPECT().StopAll(testCtx).Return(nil)

		mDoguStarter := NewMockDoguStarter(t)
		mDoguStarter.EXPECT().StartAll(testCtx).Return(nil)

		m := &Migrator{
			exportModeValidator:    mExportModeValidator,
			systemInfoProvider:     mSystemInfoProvider,
			systemInfoValidator:    mSystemInfoValidator,
			doguVolumeResizer:      mDoguVolumeResizer,
			maintenanceModeHandler: mMaintenanceModeHandler,
			mailSender:             mMailSender,
			logWriter:              mLogWriter,
			jobRunner:              mJobRunner,
			doguStopper:            mDoguStopper,
			doguStarter:            mDoguStarter,
			logInitializer:         mlogIntializer,
		}

		err := m.RunMigration(testCtx)

		require.NoError(t, err)
	})

	t.Run("should run final migration", func(t *testing.T) {
		testCtx := context.Background()
		testCtx = SetFinalMigration(testCtx)

		mExportModeValidator := NewMockExportModeValidator(t)
		mExportModeValidator.EXPECT().Validate(testCtx).Return(nil)

		mSystemInfoProvider := NewMockSystemInfoProvider(t)
		mSystemInfoProvider.EXPECT().GetExporterSystemInfo(testCtx).Return(exporterInfo, nil)
		mSystemInfoProvider.EXPECT().GetImporterSystemInfo(testCtx).Return(importerInfo, nil)

		mSystemInfoValidator := NewMockSystemInfoValidator(t)
		mSystemInfoValidator.EXPECT().Validate(testCtx, exporterInfo, importerInfo).Return(nil)

		mDoguVolumeResizer := NewMockDoguVolumeResizer(t)
		mDoguVolumeResizer.EXPECT().ResizeDogusIfNeeded(testCtx, exporterInfo.Dogus, importerInfo.Dogus).Return(nil)

		mMaintenanceModeHandler := NewMockMaintenanceModeHandler(t)
		mMaintenanceModeHandler.EXPECT().Enable(testCtx).Return(nil)

		mMailSender := NewMockMailSender(t)
		mMailSender.EXPECT().Send(testCtx, true, nil, mock.Anything, mock.Anything).Return(nil)

		jobLogs := io.NopCloser(strings.NewReader("test"))

		mLogWriter := NewMockLogWriter(t)
		mLogWriter.EXPECT().Write(jobLogs).Return(nil)

		mlogIntializer := NewMockLogInitializer(t)
		mlogIntializer.EXPECT().InitializeWithLogFile().Return(nil)

		mJobRunner := NewMockJobRunner(t)
		mJobRunner.EXPECT().Run(testCtx).Return(jobLogs, nil)

		mDoguStopper := NewMockDoguStopper(t)
		mDoguStopper.EXPECT().StopAll(testCtx).Return(nil)

		mDoguStarter := NewMockDoguStarter(t)
		mDoguStarter.EXPECT().StartAll(testCtx).Return(nil)

		m := &Migrator{
			exportModeValidator:    mExportModeValidator,
			systemInfoProvider:     mSystemInfoProvider,
			systemInfoValidator:    mSystemInfoValidator,
			doguVolumeResizer:      mDoguVolumeResizer,
			maintenanceModeHandler: mMaintenanceModeHandler,
			mailSender:             mMailSender,
			logWriter:              mLogWriter,
			jobRunner:              mJobRunner,
			doguStopper:            mDoguStopper,
			doguStarter:            mDoguStarter,
			logInitializer:         mlogIntializer,
		}

		err := m.RunMigration(testCtx)

		require.NoError(t, err)
	})

	t.Run("should fail to run delta migration for error initializing log", func(t *testing.T) {
		testCtx := context.Background()

		mExportModeValidator := NewMockExportModeValidator(t)

		mSystemInfoProvider := NewMockSystemInfoProvider(t)

		mSystemInfoValidator := NewMockSystemInfoValidator(t)

		mDoguVolumeResizer := NewMockDoguVolumeResizer(t)

		mMaintenanceModeHandler := NewMockMaintenanceModeHandler(t)

		mMailSender := NewMockMailSender(t)

		mLogWriter := NewMockLogWriter(t)

		mlogIntializer := NewMockLogInitializer(t)
		mlogIntializer.EXPECT().InitializeWithLogFile().Return(assert.AnError)

		mJobRunner := NewMockJobRunner(t)

		mDoguStopper := NewMockDoguStopper(t)

		mDoguStarter := NewMockDoguStarter(t)

		m := &Migrator{
			exportModeValidator:    mExportModeValidator,
			systemInfoProvider:     mSystemInfoProvider,
			systemInfoValidator:    mSystemInfoValidator,
			doguVolumeResizer:      mDoguVolumeResizer,
			maintenanceModeHandler: mMaintenanceModeHandler,
			mailSender:             mMailSender,
			logWriter:              mLogWriter,
			jobRunner:              mJobRunner,
			doguStopper:            mDoguStopper,
			doguStarter:            mDoguStarter,
			logInitializer:         mlogIntializer,
		}

		err := m.RunMigration(testCtx)

		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to reinitialize logger:")
	})

	t.Run("should fail to run delta migration for error validating export mode", func(t *testing.T) {
		testCtx := context.Background()

		mExportModeValidator := NewMockExportModeValidator(t)
		mExportModeValidator.EXPECT().Validate(testCtx).Return(assert.AnError)

		mSystemInfoProvider := NewMockSystemInfoProvider(t)

		mSystemInfoValidator := NewMockSystemInfoValidator(t)

		mDoguVolumeResizer := NewMockDoguVolumeResizer(t)

		mMaintenanceModeHandler := NewMockMaintenanceModeHandler(t)

		mMailSender := NewMockMailSender(t)
		mMailSender.EXPECT().Send(testCtx, false, mock.Anything, mock.Anything, mock.Anything).Return(nil)

		mLogWriter := NewMockLogWriter(t)

		mlogIntializer := NewMockLogInitializer(t)
		mlogIntializer.EXPECT().InitializeWithLogFile().Return(nil)

		mJobRunner := NewMockJobRunner(t)

		mDoguStopper := NewMockDoguStopper(t)

		mDoguStarter := NewMockDoguStarter(t)
		mDoguStarter.EXPECT().StartAll(testCtx).Return(nil)

		m := &Migrator{
			exportModeValidator:    mExportModeValidator,
			systemInfoProvider:     mSystemInfoProvider,
			systemInfoValidator:    mSystemInfoValidator,
			doguVolumeResizer:      mDoguVolumeResizer,
			maintenanceModeHandler: mMaintenanceModeHandler,
			mailSender:             mMailSender,
			logWriter:              mLogWriter,
			jobRunner:              mJobRunner,
			doguStopper:            mDoguStopper,
			doguStarter:            mDoguStarter,
			logInitializer:         mlogIntializer,
		}

		err := m.RunMigration(testCtx)

		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to validate export mode:")
	})

	t.Run("should fail to run delta migration for error in getting exporter system-info", func(t *testing.T) {
		testCtx := context.Background()

		mExportModeValidator := NewMockExportModeValidator(t)
		mExportModeValidator.EXPECT().Validate(testCtx).Return(nil)

		mSystemInfoProvider := NewMockSystemInfoProvider(t)
		mSystemInfoProvider.EXPECT().GetExporterSystemInfo(testCtx).Return(exporterInfo, assert.AnError)

		mSystemInfoValidator := NewMockSystemInfoValidator(t)

		mDoguVolumeResizer := NewMockDoguVolumeResizer(t)

		mMaintenanceModeHandler := NewMockMaintenanceModeHandler(t)

		mMailSender := NewMockMailSender(t)
		mMailSender.EXPECT().Send(testCtx, false, mock.Anything, mock.Anything, mock.Anything).Return(nil)

		mLogWriter := NewMockLogWriter(t)

		mlogIntializer := NewMockLogInitializer(t)
		mlogIntializer.EXPECT().InitializeWithLogFile().Return(nil)

		mJobRunner := NewMockJobRunner(t)

		mDoguStopper := NewMockDoguStopper(t)

		mDoguStarter := NewMockDoguStarter(t)
		mDoguStarter.EXPECT().StartAll(testCtx).Return(nil)

		m := &Migrator{
			exportModeValidator:    mExportModeValidator,
			systemInfoProvider:     mSystemInfoProvider,
			systemInfoValidator:    mSystemInfoValidator,
			doguVolumeResizer:      mDoguVolumeResizer,
			maintenanceModeHandler: mMaintenanceModeHandler,
			mailSender:             mMailSender,
			logWriter:              mLogWriter,
			jobRunner:              mJobRunner,
			doguStopper:            mDoguStopper,
			doguStarter:            mDoguStarter,
			logInitializer:         mlogIntializer,
		}

		err := m.RunMigration(testCtx)

		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to get system-info from exporter:")
	})

	t.Run("should fail to run delta migration for error getting importer system-info", func(t *testing.T) {
		testCtx := context.Background()

		mExportModeValidator := NewMockExportModeValidator(t)
		mExportModeValidator.EXPECT().Validate(testCtx).Return(nil)

		mSystemInfoProvider := NewMockSystemInfoProvider(t)
		mSystemInfoProvider.EXPECT().GetExporterSystemInfo(testCtx).Return(exporterInfo, nil)
		mSystemInfoProvider.EXPECT().GetImporterSystemInfo(testCtx).Return(importerInfo, assert.AnError)

		mSystemInfoValidator := NewMockSystemInfoValidator(t)

		mDoguVolumeResizer := NewMockDoguVolumeResizer(t)

		mMaintenanceModeHandler := NewMockMaintenanceModeHandler(t)

		mMailSender := NewMockMailSender(t)
		mMailSender.EXPECT().Send(testCtx, false, mock.Anything, mock.Anything, mock.Anything).Return(nil)

		mLogWriter := NewMockLogWriter(t)

		mlogIntializer := NewMockLogInitializer(t)
		mlogIntializer.EXPECT().InitializeWithLogFile().Return(nil)

		mJobRunner := NewMockJobRunner(t)

		mDoguStopper := NewMockDoguStopper(t)

		mDoguStarter := NewMockDoguStarter(t)
		mDoguStarter.EXPECT().StartAll(testCtx).Return(nil)

		m := &Migrator{
			exportModeValidator:    mExportModeValidator,
			systemInfoProvider:     mSystemInfoProvider,
			systemInfoValidator:    mSystemInfoValidator,
			doguVolumeResizer:      mDoguVolumeResizer,
			maintenanceModeHandler: mMaintenanceModeHandler,
			mailSender:             mMailSender,
			logWriter:              mLogWriter,
			jobRunner:              mJobRunner,
			doguStopper:            mDoguStopper,
			doguStarter:            mDoguStarter,
			logInitializer:         mlogIntializer,
		}

		err := m.RunMigration(testCtx)

		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to get system-info from importer:")
	})

	t.Run("should fail to run delta migration for error in systemInfoValidator", func(t *testing.T) {
		testCtx := context.Background()

		mExportModeValidator := NewMockExportModeValidator(t)
		mExportModeValidator.EXPECT().Validate(testCtx).Return(nil)

		mSystemInfoProvider := NewMockSystemInfoProvider(t)
		mSystemInfoProvider.EXPECT().GetExporterSystemInfo(testCtx).Return(exporterInfo, nil)
		mSystemInfoProvider.EXPECT().GetImporterSystemInfo(testCtx).Return(importerInfo, nil)

		mSystemInfoValidator := NewMockSystemInfoValidator(t)
		mSystemInfoValidator.EXPECT().Validate(testCtx, exporterInfo, importerInfo).Return(assert.AnError)

		mDoguVolumeResizer := NewMockDoguVolumeResizer(t)

		mMaintenanceModeHandler := NewMockMaintenanceModeHandler(t)

		mMailSender := NewMockMailSender(t)
		mMailSender.EXPECT().Send(testCtx, false, mock.Anything, mock.Anything, mock.Anything).Return(nil)

		mLogWriter := NewMockLogWriter(t)

		mlogIntializer := NewMockLogInitializer(t)
		mlogIntializer.EXPECT().InitializeWithLogFile().Return(nil)

		mJobRunner := NewMockJobRunner(t)

		mDoguStopper := NewMockDoguStopper(t)

		mDoguStarter := NewMockDoguStarter(t)
		mDoguStarter.EXPECT().StartAll(testCtx).Return(nil)

		m := &Migrator{
			exportModeValidator:    mExportModeValidator,
			systemInfoProvider:     mSystemInfoProvider,
			systemInfoValidator:    mSystemInfoValidator,
			doguVolumeResizer:      mDoguVolumeResizer,
			maintenanceModeHandler: mMaintenanceModeHandler,
			mailSender:             mMailSender,
			logWriter:              mLogWriter,
			jobRunner:              mJobRunner,
			doguStopper:            mDoguStopper,
			doguStarter:            mDoguStarter,
			logInitializer:         mlogIntializer,
		}

		err := m.RunMigration(testCtx)

		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to validate system info:")
	})

	t.Run("should fail to run delta migration for error in resizing dogu volumes", func(t *testing.T) {
		testCtx := context.Background()

		mExportModeValidator := NewMockExportModeValidator(t)
		mExportModeValidator.EXPECT().Validate(testCtx).Return(nil)

		mSystemInfoProvider := NewMockSystemInfoProvider(t)
		mSystemInfoProvider.EXPECT().GetExporterSystemInfo(testCtx).Return(exporterInfo, nil)
		mSystemInfoProvider.EXPECT().GetImporterSystemInfo(testCtx).Return(importerInfo, nil)

		mSystemInfoValidator := NewMockSystemInfoValidator(t)
		mSystemInfoValidator.EXPECT().Validate(testCtx, exporterInfo, importerInfo).Return(nil)

		mDoguVolumeResizer := NewMockDoguVolumeResizer(t)
		mDoguVolumeResizer.EXPECT().ResizeDogusIfNeeded(testCtx, exporterInfo.Dogus, importerInfo.Dogus).Return(assert.AnError)

		mMaintenanceModeHandler := NewMockMaintenanceModeHandler(t)

		mMailSender := NewMockMailSender(t)
		mMailSender.EXPECT().Send(testCtx, false, mock.Anything, mock.Anything, mock.Anything).Return(nil)

		mLogWriter := NewMockLogWriter(t)

		mlogIntializer := NewMockLogInitializer(t)
		mlogIntializer.EXPECT().InitializeWithLogFile().Return(nil)

		mJobRunner := NewMockJobRunner(t)

		mDoguStopper := NewMockDoguStopper(t)

		mDoguStarter := NewMockDoguStarter(t)
		mDoguStarter.EXPECT().StartAll(testCtx).Return(nil)

		m := &Migrator{
			exportModeValidator:    mExportModeValidator,
			systemInfoProvider:     mSystemInfoProvider,
			systemInfoValidator:    mSystemInfoValidator,
			doguVolumeResizer:      mDoguVolumeResizer,
			maintenanceModeHandler: mMaintenanceModeHandler,
			mailSender:             mMailSender,
			logWriter:              mLogWriter,
			jobRunner:              mJobRunner,
			doguStopper:            mDoguStopper,
			doguStarter:            mDoguStarter,
			logInitializer:         mlogIntializer,
		}

		err := m.RunMigration(testCtx)

		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to resize dogu-volumes:")
	})

	t.Run("should fail to run delta migration for error stopping dogus", func(t *testing.T) {
		testCtx := context.Background()

		mExportModeValidator := NewMockExportModeValidator(t)
		mExportModeValidator.EXPECT().Validate(testCtx).Return(nil)

		mSystemInfoProvider := NewMockSystemInfoProvider(t)
		mSystemInfoProvider.EXPECT().GetExporterSystemInfo(testCtx).Return(exporterInfo, nil)
		mSystemInfoProvider.EXPECT().GetImporterSystemInfo(testCtx).Return(importerInfo, nil)

		mSystemInfoValidator := NewMockSystemInfoValidator(t)
		mSystemInfoValidator.EXPECT().Validate(testCtx, exporterInfo, importerInfo).Return(nil)

		mDoguVolumeResizer := NewMockDoguVolumeResizer(t)
		mDoguVolumeResizer.EXPECT().ResizeDogusIfNeeded(testCtx, exporterInfo.Dogus, importerInfo.Dogus).Return(nil)

		mMaintenanceModeHandler := NewMockMaintenanceModeHandler(t)

		mMailSender := NewMockMailSender(t)
		mMailSender.EXPECT().Send(testCtx, false, mock.Anything, mock.Anything, mock.Anything).Return(nil)

		mLogWriter := NewMockLogWriter(t)

		mlogIntializer := NewMockLogInitializer(t)
		mlogIntializer.EXPECT().InitializeWithLogFile().Return(nil)

		mJobRunner := NewMockJobRunner(t)

		mDoguStopper := NewMockDoguStopper(t)
		mDoguStopper.EXPECT().StopAll(testCtx).Return(assert.AnError)

		mDoguStarter := NewMockDoguStarter(t)
		mDoguStarter.EXPECT().StartAll(testCtx).Return(nil)

		m := &Migrator{
			exportModeValidator:    mExportModeValidator,
			systemInfoProvider:     mSystemInfoProvider,
			systemInfoValidator:    mSystemInfoValidator,
			doguVolumeResizer:      mDoguVolumeResizer,
			maintenanceModeHandler: mMaintenanceModeHandler,
			mailSender:             mMailSender,
			logWriter:              mLogWriter,
			jobRunner:              mJobRunner,
			doguStopper:            mDoguStopper,
			doguStarter:            mDoguStarter,
			logInitializer:         mlogIntializer,
		}

		err := m.RunMigration(testCtx)

		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to stop all dogus:")
	})

	t.Run("should fail to run delta migration for error running job", func(t *testing.T) {
		testCtx := context.Background()

		mExportModeValidator := NewMockExportModeValidator(t)
		mExportModeValidator.EXPECT().Validate(testCtx).Return(nil)

		mSystemInfoProvider := NewMockSystemInfoProvider(t)
		mSystemInfoProvider.EXPECT().GetExporterSystemInfo(testCtx).Return(exporterInfo, nil)
		mSystemInfoProvider.EXPECT().GetImporterSystemInfo(testCtx).Return(importerInfo, nil)

		mSystemInfoValidator := NewMockSystemInfoValidator(t)
		mSystemInfoValidator.EXPECT().Validate(testCtx, exporterInfo, importerInfo).Return(nil)

		mDoguVolumeResizer := NewMockDoguVolumeResizer(t)
		mDoguVolumeResizer.EXPECT().ResizeDogusIfNeeded(testCtx, exporterInfo.Dogus, importerInfo.Dogus).Return(nil)

		mMaintenanceModeHandler := NewMockMaintenanceModeHandler(t)

		mMailSender := NewMockMailSender(t)
		mMailSender.EXPECT().Send(testCtx, false, mock.Anything, mock.Anything, mock.Anything).Return(nil)

		mLogWriter := NewMockLogWriter(t)

		mlogIntializer := NewMockLogInitializer(t)
		mlogIntializer.EXPECT().InitializeWithLogFile().Return(nil)

		mJobRunner := NewMockJobRunner(t)
		mJobRunner.EXPECT().Run(testCtx).Return(nil, assert.AnError)

		mDoguStopper := NewMockDoguStopper(t)
		mDoguStopper.EXPECT().StopAll(testCtx).Return(nil)

		mDoguStarter := NewMockDoguStarter(t)
		mDoguStarter.EXPECT().StartAll(testCtx).Return(nil)

		m := &Migrator{
			exportModeValidator:    mExportModeValidator,
			systemInfoProvider:     mSystemInfoProvider,
			systemInfoValidator:    mSystemInfoValidator,
			doguVolumeResizer:      mDoguVolumeResizer,
			maintenanceModeHandler: mMaintenanceModeHandler,
			mailSender:             mMailSender,
			logWriter:              mLogWriter,
			jobRunner:              mJobRunner,
			doguStopper:            mDoguStopper,
			doguStarter:            mDoguStarter,
			logInitializer:         mlogIntializer,
		}

		err := m.RunMigration(testCtx)

		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to run migration job:")
	})

	t.Run("should log error when running delta migration with error in logWriter", func(t *testing.T) {
		testCtx := context.Background()

		originalLogger := slog.Default()
		defer func() {
			slog.SetDefault(originalLogger)
		}()
		sb := new(strings.Builder)
		testLogger := slog.New(slog.NewTextHandler(sb, nil))
		slog.SetDefault(testLogger)

		mExportModeValidator := NewMockExportModeValidator(t)
		mExportModeValidator.EXPECT().Validate(testCtx).Return(nil)

		mSystemInfoProvider := NewMockSystemInfoProvider(t)
		mSystemInfoProvider.EXPECT().GetExporterSystemInfo(testCtx).Return(exporterInfo, nil)
		mSystemInfoProvider.EXPECT().GetImporterSystemInfo(testCtx).Return(importerInfo, nil)

		mSystemInfoValidator := NewMockSystemInfoValidator(t)
		mSystemInfoValidator.EXPECT().Validate(testCtx, exporterInfo, importerInfo).Return(nil)

		mDoguVolumeResizer := NewMockDoguVolumeResizer(t)
		mDoguVolumeResizer.EXPECT().ResizeDogusIfNeeded(testCtx, exporterInfo.Dogus, importerInfo.Dogus).Return(nil)

		mMaintenanceModeHandler := NewMockMaintenanceModeHandler(t)

		mMailSender := NewMockMailSender(t)
		mMailSender.EXPECT().Send(testCtx, false, mock.Anything, mock.Anything, mock.Anything).Return(nil)

		jobLogs := io.NopCloser(strings.NewReader("test"))

		mLogWriter := NewMockLogWriter(t)
		mLogWriter.EXPECT().Write(jobLogs).Return(assert.AnError)

		mlogIntializer := NewMockLogInitializer(t)
		mlogIntializer.EXPECT().InitializeWithLogFile().Return(nil)

		mJobRunner := NewMockJobRunner(t)
		mJobRunner.EXPECT().Run(testCtx).Return(jobLogs, nil)

		mDoguStopper := NewMockDoguStopper(t)
		mDoguStopper.EXPECT().StopAll(testCtx).Return(nil)

		mDoguStarter := NewMockDoguStarter(t)
		mDoguStarter.EXPECT().StartAll(testCtx).Return(nil)

		m := &Migrator{
			exportModeValidator:    mExportModeValidator,
			systemInfoProvider:     mSystemInfoProvider,
			systemInfoValidator:    mSystemInfoValidator,
			doguVolumeResizer:      mDoguVolumeResizer,
			maintenanceModeHandler: mMaintenanceModeHandler,
			mailSender:             mMailSender,
			logWriter:              mLogWriter,
			jobRunner:              mJobRunner,
			doguStopper:            mDoguStopper,
			doguStarter:            mDoguStarter,
			logInitializer:         mlogIntializer,
		}

		err := m.RunMigration(testCtx)

		require.NoError(t, err)
		assert.Contains(t, sb.String(), "failed to write job log file")
	})

	t.Run("should fail to run final migration for error enabling maintenance-mode", func(t *testing.T) {
		testCtx := context.Background()
		testCtx = SetFinalMigration(testCtx)

		mExportModeValidator := NewMockExportModeValidator(t)
		mExportModeValidator.EXPECT().Validate(testCtx).Return(nil)

		mSystemInfoProvider := NewMockSystemInfoProvider(t)
		mSystemInfoProvider.EXPECT().GetExporterSystemInfo(testCtx).Return(exporterInfo, nil)
		mSystemInfoProvider.EXPECT().GetImporterSystemInfo(testCtx).Return(importerInfo, nil)

		mSystemInfoValidator := NewMockSystemInfoValidator(t)
		mSystemInfoValidator.EXPECT().Validate(testCtx, exporterInfo, importerInfo).Return(nil)

		mDoguVolumeResizer := NewMockDoguVolumeResizer(t)
		mDoguVolumeResizer.EXPECT().ResizeDogusIfNeeded(testCtx, exporterInfo.Dogus, importerInfo.Dogus).Return(nil)

		mMaintenanceModeHandler := NewMockMaintenanceModeHandler(t)
		mMaintenanceModeHandler.EXPECT().Enable(testCtx).Return(assert.AnError)
		mMaintenanceModeHandler.EXPECT().Disable(testCtx).Return(nil)

		mMailSender := NewMockMailSender(t)
		mMailSender.EXPECT().Send(testCtx, true, mock.Anything, mock.Anything, mock.Anything).Return(nil)

		mLogWriter := NewMockLogWriter(t)

		mlogIntializer := NewMockLogInitializer(t)
		mlogIntializer.EXPECT().InitializeWithLogFile().Return(nil)

		mJobRunner := NewMockJobRunner(t)

		mDoguStopper := NewMockDoguStopper(t)
		mDoguStopper.EXPECT().StopAll(testCtx).Return(nil)

		mDoguStarter := NewMockDoguStarter(t)
		mDoguStarter.EXPECT().StartAll(testCtx).Return(nil)

		m := &Migrator{
			exportModeValidator:    mExportModeValidator,
			systemInfoProvider:     mSystemInfoProvider,
			systemInfoValidator:    mSystemInfoValidator,
			doguVolumeResizer:      mDoguVolumeResizer,
			maintenanceModeHandler: mMaintenanceModeHandler,
			mailSender:             mMailSender,
			logWriter:              mLogWriter,
			jobRunner:              mJobRunner,
			doguStopper:            mDoguStopper,
			doguStarter:            mDoguStarter,
			logInitializer:         mlogIntializer,
		}

		err := m.RunMigration(testCtx)

		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to enable maintenance mode:")
	})
}

func TestMigrator_cleanup(t *testing.T) {
	t.Run("should log error on final migration with error in disable maintenanceMode", func(t *testing.T) {
		testCtx := context.Background()
		startTime := time.Now()
		runErr := fmt.Errorf("test")

		originalLogger := slog.Default()
		sb := new(strings.Builder)
		testLogger := slog.New(slog.NewTextHandler(sb, nil))
		slog.SetDefault(testLogger)

		defer func() {
			slog.SetDefault(originalLogger)
		}()

		mMaintenanceModeHandler := NewMockMaintenanceModeHandler(t)
		mMaintenanceModeHandler.EXPECT().Disable(testCtx).Return(assert.AnError)

		mMailSender := NewMockMailSender(t)
		mMailSender.EXPECT().Send(testCtx, true, runErr, startTime, mock.Anything).Return(nil)

		mDoguStarter := NewMockDoguStarter(t)
		mDoguStarter.EXPECT().StartAll(testCtx).Return(nil)

		m := &Migrator{
			maintenanceModeHandler: mMaintenanceModeHandler,
			mailSender:             mMailSender,
			doguStarter:            mDoguStarter,
		}

		m.cleanup(testCtx, startTime, true, runErr, nil)

		assert.Contains(t, sb.String(), "failed to disabled maintenance mode:")
	})

	t.Run("should log error on delta migration with error in starting dogus", func(t *testing.T) {
		testCtx := context.Background()
		startTime := time.Now()
		runErr := fmt.Errorf("test")

		originalLogger := slog.Default()
		sb := new(strings.Builder)
		testLogger := slog.New(slog.NewTextHandler(sb, nil))
		slog.SetDefault(testLogger)

		defer func() {
			slog.SetDefault(originalLogger)
		}()

		mMaintenanceModeHandler := NewMockMaintenanceModeHandler(t)

		mMailSender := NewMockMailSender(t)
		mMailSender.EXPECT().Send(testCtx, false, runErr, startTime, mock.Anything).Return(nil)

		mDoguStarter := NewMockDoguStarter(t)
		mDoguStarter.EXPECT().StartAll(testCtx).Return(assert.AnError)

		m := &Migrator{
			maintenanceModeHandler: mMaintenanceModeHandler,
			mailSender:             mMailSender,
			doguStarter:            mDoguStarter,
		}

		m.cleanup(testCtx, startTime, false, runErr, nil)

		assert.Contains(t, sb.String(), "failed to start all dogus:")
	})

	t.Run("should log error on delta migration with error in sending mail", func(t *testing.T) {
		testCtx := context.Background()
		startTime := time.Now()
		runErr := fmt.Errorf("test")

		originalLogger := slog.Default()
		sb := new(strings.Builder)
		testLogger := slog.New(slog.NewTextHandler(sb, nil))
		slog.SetDefault(testLogger)

		defer func() {
			slog.SetDefault(originalLogger)
		}()

		mMaintenanceModeHandler := NewMockMaintenanceModeHandler(t)

		mMailSender := NewMockMailSender(t)
		mMailSender.EXPECT().Send(testCtx, false, runErr, startTime, mock.Anything).Return(assert.AnError)

		mDoguStarter := NewMockDoguStarter(t)
		mDoguStarter.EXPECT().StartAll(testCtx).Return(nil)

		m := &Migrator{
			maintenanceModeHandler: mMaintenanceModeHandler,
			mailSender:             mMailSender,
			doguStarter:            mDoguStarter,
		}

		m.cleanup(testCtx, startTime, false, runErr, nil)

		assert.Contains(t, sb.String(), "failed to send mail:")
	})
}

func TestNewMigrator(t *testing.T) {
	mExportModeValidator := NewMockExportModeValidator(t)
	mSystemInfoProvider := NewMockSystemInfoProvider(t)
	mSystemInfoValidator := NewMockSystemInfoValidator(t)
	mDoguVolumeResizer := NewMockDoguVolumeResizer(t)
	mMaintenanceModeHandler := NewMockMaintenanceModeHandler(t)
	mMailSender := NewMockMailSender(t)
	mLogWriter := NewMockLogWriter(t)
	mJobRunner := NewMockJobRunner(t)
	mDoguStopper := NewMockDoguStopper(t)
	mDoguStarter := NewMockDoguStarter(t)

	m := NewMigrator(MigratorDependencies{
		ExportModeValidator:    mExportModeValidator,
		SystemInfoProvider:     mSystemInfoProvider,
		SystemInfoValidator:    mSystemInfoValidator,
		DoguVolumeResizer:      mDoguVolumeResizer,
		MaintenanceModeHandler: mMaintenanceModeHandler,
		MailSender:             mMailSender,
		LogWriter:              mLogWriter,
		JobRunner:              mJobRunner,
		DoguStopper:            mDoguStopper,
		DoguStarter:            mDoguStarter,
	})

	require.NotNil(t, m)
	assert.Equal(t, mExportModeValidator, m.exportModeValidator)
	assert.Equal(t, mSystemInfoProvider, m.systemInfoProvider)
	assert.Equal(t, mSystemInfoValidator, m.systemInfoValidator)
	assert.Equal(t, mDoguVolumeResizer, m.doguVolumeResizer)
	assert.Equal(t, mMaintenanceModeHandler, m.maintenanceModeHandler)
	assert.Equal(t, mMailSender, m.mailSender)
	assert.Equal(t, mLogWriter, m.logWriter)
	assert.Equal(t, mJobRunner, m.jobRunner)
	assert.Equal(t, mDoguStopper, m.doguStopper)
	assert.Equal(t, mDoguStarter, m.doguStarter)
}
