package cron

import (
	"context"
	"fmt"

	"github.com/adhocore/gronx"
	"github.com/adhocore/gronx/pkg/tasker"
)

type newTaskerFunc func(opt tasker.Option) taskRunner

type taskRunner interface {
	Task(expr string, task tasker.TaskFunc, concurrent ...bool) *tasker.Tasker
	Run()
	Stop()
	Running() bool
}

type MainLooper struct {
	expr        string
	taskFactory newTaskerFunc
	currentTask taskRunner
}

func NewMainLooper(expr string) *MainLooper {
	return &MainLooper{
		expr: expr,
		taskFactory: func(opt tasker.Option) taskRunner {
			return tasker.New(opt)
		},
	}
}

func (cj *MainLooper) Run(jobClosure func(ctx context.Context) error) error {
	if !gronx.IsValid(cj.expr) {
		return fmt.Errorf("cron expression %q is invalid", cj.expr)
	}
	cj.currentTask = cj.taskFactory(tasker.Option{})

	cj.currentTask.Task(cj.expr, func(ctx context.Context) (int, error) {
		return 0, jobClosure(ctx)
	})

	cj.currentTask.Run()

	return nil
}

func (cj *MainLooper) Stop() {
	if cj.currentTask != nil && cj.currentTask.Running() {
		cj.currentTask.Stop()
	}

	cj.currentTask = nil
}
