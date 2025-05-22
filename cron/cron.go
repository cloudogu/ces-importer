package cron

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/adhocore/gronx"
	"github.com/adhocore/gronx/pkg/tasker"
)

// JobFunc contains the payload function that will be run from Task. The int return value may or may not reflect on any error situations.
type JobFunc func(ctx context.Context) (int, error)

type taskRunner interface {
	// Run starts the provided task. No errors will be returned during the execution since this would end the main
	// loop. Implementors should either provide an error channel to the task function or simply log errors to
	// indicate error situations.
	Run()
	// Stop interrupts the provided task.
	Stop()
}

// Task allows executing functions in recurring points in time, depending on the system time. Considering
// container restarts or pod kills, this behavior is more flexible than a regular time ticker.
type Task struct {
	taskExecuter taskRunner
}

// New creates a new instance for executing the same task. The task must be provided to its Run() function.
func New(ctx context.Context, expr string, jobClosure JobFunc) (*Task, error) {
	if !gronx.IsValid(expr) {
		return nil, fmt.Errorf("cron expression %q is invalid", expr)
	}

	taskManager := tasker.New(tasker.Option{}).WithContext(ctx)

	return &Task{
		taskExecuter: taskManager.Task(expr, tasker.TaskFunc(jobClosure), false),
	}, nil
}

// Run executes the given function. It can be stopped with Stop(). Please note that Run() does not return an error.
func (t *Task) Run() {
	slog.Debug("starting a new run")
	t.taskExecuter.Run()
}

// Stop stops the looping over the provided function given to Run().
func (t *Task) Stop() {
	t.taskExecuter.Stop()
}
