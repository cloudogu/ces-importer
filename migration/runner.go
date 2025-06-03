package migration

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"time"
)

type ExportModeValidator interface {
	Validate(ctx context.Context) error
}

type SystemInfoValidator interface {
	Validate(ctx context.Context) error
}

type DoguStopper interface {
	StopAll(ctx context.Context) error
}

type DoguStarter interface {
	StartAll(ctx context.Context) error
}

type JobRunner interface {
	Run(ctx context.Context) (io.ReadCloser, error)
}

type LogWriter interface {
	Write(io.ReadCloser) error
}

type MaintenanceModeHandler interface {
	Enable(ctx context.Context) error
	Disable(ctx context.Context) error
}

type MailSender interface {
	Send(ctx context.Context, isFinal bool, migrationResult error, startTime time.Time, endTime time.Time) error
}

type LogInitializer interface {
	InitializeWithLogFile() error
}

type Migrator struct {
	exportModeValidator    ExportModeValidator
	systemInfoValidator    SystemInfoValidator
	maintenanceModeHandler MaintenanceModeHandler
	mailSender             MailSender
	logWriter              LogWriter
	jobRunner              JobRunner
	doguStopper            DoguStopper
	doguStarter            DoguStarter
	logInitializer         LogInitializer
}

type MigratorDependencies struct {
	ExportModeValidator
	SystemInfoValidator
	MaintenanceModeHandler
	MailSender
	LogWriter
	LogInitializer
	JobRunner
	DoguStopper
	DoguStarter
}

func NewMigrator(dependencies MigratorDependencies) *Migrator {
	return &Migrator{
		exportModeValidator:    dependencies.ExportModeValidator,
		systemInfoValidator:    dependencies.SystemInfoValidator,
		maintenanceModeHandler: dependencies.MaintenanceModeHandler,
		mailSender:             dependencies.MailSender,
		logWriter:              dependencies.LogWriter,
		jobRunner:              dependencies.JobRunner,
		doguStopper:            dependencies.DoguStopper,
		doguStarter:            dependencies.DoguStarter,
		logInitializer:         dependencies.LogInitializer,
	}
}

func (m Migrator) RunMigration(ctx context.Context) (err error) {
	err = m.logInitializer.InitializeWithLogFile()
	if err != nil {
		return fmt.Errorf("failed to reinitialize logger: %w", err)
	}

	isFinalMigration := IsFinalMigration(ctx)
	slog.Debug("Starting migration", "finalMigration", isFinalMigration)

	startTime := time.Now()
	defer func() {
		err = m.cleanup(ctx, startTime, isFinalMigration, err, recover())
	}()

	err = m.exportModeValidator.Validate(ctx)
	if err != nil {
		return fmt.Errorf("failed to validate export mode: %w", err)
	}

	// TODO: Do not resize inside validate function, create a new interface
	err = m.systemInfoValidator.Validate(ctx)
	if err != nil {
		return fmt.Errorf("failed to validate system info: %w", err)
	}

	err = m.doguStopper.StopAll(ctx)
	if err != nil {
		return fmt.Errorf("failed to stop all dogus: %w", err)
	}

	if isFinalMigration {
		err = m.maintenanceModeHandler.Enable(ctx)
		if err != nil {
			return fmt.Errorf("failed to enable maintenance mode: %w", err)
		}
	}

	logs, err := m.jobRunner.Run(ctx)
	if logs != nil {
		lerr := m.logWriter.Write(logs)
		if lerr != nil {
			slog.Error(fmt.Sprintf("failed to write job log file: %s", lerr.Error()))
		}
	}

	if err != nil {
		return fmt.Errorf("failed to run migration job: %w", err)
	}

	return
}

func (m Migrator) cleanup(ctx context.Context, startTime time.Time, isFinalMigration bool, runError error, runPanic any) error {
	retError := runError
	if runError != nil {
		slog.Error(fmt.Sprintf("migration failed: %s", runError.Error()))
	}
	if runPanic != nil {
		slog.Error(fmt.Sprintf("migration failed: %s", runPanic))
	}

	if (runError != nil || runPanic != nil) && isFinalMigration {
		if err := m.maintenanceModeHandler.Disable(ctx); err != nil {
			retError = errors.Join(runError, err)
			slog.Error(fmt.Sprintf("failed to disabled maintenance mode: %v", err))
		}
	}

	if err := m.doguStarter.StartAll(ctx); err != nil {
		retError = errors.Join(runError, err)
		slog.Error(fmt.Sprintf("failed to start all dogus: %s", err.Error()))
	}

	endTime := time.Now()
	if err := m.mailSender.Send(ctx, isFinalMigration, runError, startTime, endTime); err != nil {
		retError = errors.Join(runError, err)
		slog.Error(fmt.Sprintf("failed to send mail: %s", err.Error()))
	}

	// because recover() catches the panic it needs to be rethrown
	if runPanic != nil {
		panic(runPanic)
	}

	return retError
}
