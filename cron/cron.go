package cron

import (
	"context"
	"fmt"

	"github.com/adhocore/gronx"
	"github.com/adhocore/gronx/pkg/tasker"
)

type newTaskerFunc func(opt tasker.Option) taskRunner

type taskRunner interface {
	// Task registers the provided task with the CRON-like expression and returns the task. This task should be
	// considered to be saved into a form of state allowing to call Stop() and thus end the task.
	Task(expr string, task tasker.TaskFunc, concurrent ...bool) *tasker.Tasker
	// Run starts the provided task. No errors will be returned during the execution since this would end the main
	// loop. Implementors should either provide an error channel to the task function or simply log errors to
	// indicate error situations.
	Run()
	// Stop interrupts the provided task.
	Stop()
	// Running returns true if the provided task is currently running, otherwise false.
	Running() bool
}

// mainLooper allows executing functions in recurring points in time, depending on the system time. Considering
// container restarts or pod kills, this behavior is more flexible than a regular time ticker.
type mainLooper struct {
	expr        string
	taskFactory newTaskerFunc
	currentTask taskRunner
}

// New creates a new instance for executing the same task. The task must be provided to its Run() function.
func New(expr string) (*mainLooper, error) {
	if !gronx.IsValid(expr) {
		return nil, fmt.Errorf("cron expression %q is invalid", expr)
	}

	return &mainLooper{
		expr: expr,
		taskFactory: func(opt tasker.Option) taskRunner {
			return tasker.New(opt)
		},
	}, nil
}

// Run executes the given function. It can be stopped with Stop(). Please note that Run() does not return an error.
func (cj *mainLooper) Run(jobClosure func(ctx context.Context) error) {
	cj.currentTask = cj.taskFactory(tasker.Option{})

	cj.currentTask.Task(cj.expr, func(ctx context.Context) (int, error) {
		return 0, jobClosure(ctx)
	})

	cj.currentTask.Run()
}

// Stop stops the looping over the provided function given to Run().
func (cj *mainLooper) Stop() {
	if cj.currentTask != nil && cj.currentTask.Running() {
		cj.currentTask.Stop()
	}

	cj.currentTask = nil
}
