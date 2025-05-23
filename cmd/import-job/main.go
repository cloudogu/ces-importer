package main

import (
	"context"
	"log/slog"
	"os"
)

func main() {
	os.Exit(run())
}

func run() int {
	slog.Info("New import job started.")

	ctx := context.Background()

	importJob, err := NewImportExecuter()
	if err != nil {
		slog.Error("failed to create executer for import", "cause", err)
		return 1
	}

	err = importJob.Start(ctx)
	if err != nil {
		slog.Error("Import job failed", "cause", err)
		return 1
	}

	slog.Info("Import job finished.")
	return 0

}
