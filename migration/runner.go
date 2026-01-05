package migration

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"
)

type ExportModeValidator interface {
	Validate(ctx context.Context) error
}

type SystemInfoProvider interface {
	GetExporterSystemInfo(ctx context.Context) (*SystemInfo, error)
	GetImporterSystemInfo(ctx context.Context) (*SystemInfo, error)
}

type SystemInfoValidator interface {
	Validate(ctx context.Context, exporterInfo *SystemInfo, importerInfo *SystemInfo) error
}

type DoguVolumeResizer interface {
	ResizeDogusIfNeeded(ctx context.Context, exporterDogus []Dogu, importerDogus []Dogu) error
}

type DoguStopper interface {
	StopAll(ctx context.Context) error
}

type DoguStarter interface {
	StartAll(ctx context.Context) error
}

type BlueprintStopper interface {
	StopBlueprint(ctx context.Context) error
}

type BlueprintStarter interface {
	StartBlueprint(ctx context.Context) error
}

type JobRunner interface {
	Run(ctx context.Context) error
}

type MaintenanceModeHandler interface {
	Enable(ctx context.Context) error
	Disable(ctx context.Context) error
}

type MailSender interface {
	Send(ctx context.Context, isFinal bool, migrationResult error, startTime time.Time, endTime time.Time) error
}

type LogInitializerFunc func() error

type Migrator struct {
	exportModeValidator    ExportModeValidator
	systemInfoProvider     SystemInfoProvider
	systemInfoValidator    SystemInfoValidator
	doguVolumeResizer      DoguVolumeResizer
	maintenanceModeHandler MaintenanceModeHandler
	mailSender             MailSender
	jobRunner              JobRunner
	doguStopper            DoguStopper
	doguStarter            DoguStarter
	blueprintStopper       BlueprintStopper
	blueprintStarter       BlueprintStarter
	initializeLogger       LogInitializerFunc
}

type MigratorDependencies struct {
	ExportModeValidator
	SystemInfoProvider
	SystemInfoValidator
	DoguVolumeResizer
	MaintenanceModeHandler
	MailSender
	LogInitializerFunc
	JobRunner
	DoguStopper
	DoguStarter
	BlueprintStopper
	BlueprintStarter
}

func NewMigrator(dependencies MigratorDependencies) *Migrator {
	return &Migrator{
		exportModeValidator:    dependencies.ExportModeValidator,
		systemInfoProvider:     dependencies.SystemInfoProvider,
		systemInfoValidator:    dependencies.SystemInfoValidator,
		doguVolumeResizer:      dependencies.DoguVolumeResizer,
		maintenanceModeHandler: dependencies.MaintenanceModeHandler,
		mailSender:             dependencies.MailSender,
		jobRunner:              dependencies.JobRunner,
		doguStopper:            dependencies.DoguStopper,
		doguStarter:            dependencies.DoguStarter,
		blueprintStopper:       dependencies.BlueprintStopper,
		blueprintStarter:       dependencies.BlueprintStarter,
		initializeLogger:       dependencies.LogInitializerFunc,
	}
}

func (m Migrator) RunMigration(ctx context.Context) (err error) {
	err = m.initializeLogger()
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

	exporterInfo, err := m.systemInfoProvider.GetExporterSystemInfo(ctx)
	if err != nil {
		return fmt.Errorf("failed to get system-info from exporter: %w", err)
	}

	importerInfo, err := m.systemInfoProvider.GetImporterSystemInfo(ctx)
	if err != nil {
		return fmt.Errorf("failed to get system-info from importer: %w", err)
	}

	err = m.systemInfoValidator.Validate(ctx, exporterInfo, importerInfo)
	if err != nil {
		return fmt.Errorf("failed to validate system info: %w", err)
	}

	err = m.blueprintStopper.StopBlueprint(ctx)
	if err != nil {
		return fmt.Errorf("failed to stop blueprint: %w", err)
	}

	err = m.doguVolumeResizer.ResizeDogusIfNeeded(ctx, exporterInfo.Dogus, importerInfo.Dogus)
	if err != nil {
		return fmt.Errorf("failed to resize dogu-volumes: %w", err)
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

	err = m.jobRunner.Run(ctx)
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

	if err := m.blueprintStarter.StartBlueprint(ctx); err != nil {
		retError = errors.Join(runError, err)
		slog.Error(fmt.Sprintf("failed to start blueprint: %s", err.Error()))
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
