package migration

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"time"
)

const (
	finalMigrationKey = "TBD"
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
	Enable(ctx context.Context, title string, message string) error
	Disable(ctx context.Context) error
}

type MailSender interface {
	Send(isFinal bool, migrationResult error, source string, target string, startTime time.Time, endTime time.Time) error
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
}

type MigratorDependencies struct {
	ExportModeValidator
	SystemInfoValidator
	MaintenanceModeHandler
	MailSender
	LogWriter
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
	}
}

func (m Migrator) RunMigration(ctx context.Context) (err error) {
	isFinalMigration := ctx.Value(finalMigrationKey).(bool)
	startTime := time.Now()
	defer m.cleanup(ctx, startTime, isFinalMigration, err)

	err = m.exportModeValidator.Validate(ctx)
	if err != nil {
		return fmt.Errorf("failed to validate export mode: %w", err)
	}

	err = m.doguStopper.StopAll(ctx)
	if err != nil {
		return fmt.Errorf("failed to stop all dogus: %w", err)
	}

	// TODO: Do not resize inside validate function, create a new interface
	err = m.systemInfoValidator.Validate(ctx)
	if err != nil {
		return fmt.Errorf("failed to validate system info: %w", err)
	}

	if isFinalMigration {
		err = m.maintenanceModeHandler.Enable(ctx, "", "")
		if err != nil {
			return fmt.Errorf("failed to enable maintenance mode: %w", err)
		}
	}

	logs, err := m.jobRunner.Run(ctx)
	if err != nil {
		return fmt.Errorf("failed to run migration job: %w", err)
	}

	err = m.logWriter.Write(logs)
	if err != nil {
		return fmt.Errorf("failed to write job log file: %w", err)
	}

	return
}

func (m Migrator) cleanup(ctx context.Context, startTime time.Time, isFinalMigration bool, runError error) {
	if runError != nil && isFinalMigration {
		if err := m.maintenanceModeHandler.Disable(ctx); err != nil {
			slog.Error(fmt.Sprintf("failed to disabled maintenance mode: %v", err))
		}
	}

	if err := m.doguStarter.StartAll(ctx); err != nil {
		slog.Error(fmt.Sprintf("failed to start all dogus: %s", err.Error()))
	}

	endTime := time.Now()
	if err := m.mailSender.Send(isFinalMigration, runError, "", "", startTime, endTime); err != nil {
		slog.Error(fmt.Sprintf("failed to send mail: %s", err.Error()))
	}
}
