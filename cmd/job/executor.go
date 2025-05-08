package main

import (
	"context"
	"fmt"
	"github.com/cloudogu/ces-importer/api/exporter"
	"github.com/cloudogu/ces-importer/configuration"
	"github.com/cloudogu/ces-importer/sync"
	"net/http"
	"os/exec"
)

type dataSyncer interface {
	SyncData(ctx context.Context, apiCli sync.ApiCli, config configuration.Job) error
}

type configSyncer interface {
	SyncConfig(ctx context.Context) error
}

type ImportExecutor struct {
	configSyncer
	dataSyncer
	apiClient sync.ApiCli
	config    configuration.Job
}

func NewImportExecutor() (*ImportExecutor, error) {
	jobConfig, err := configuration.ReadJobConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to read job configuration: %w", err)
	}

	client := exporter.NewClient(jobConfig.API.ExporterApiKey, http.DefaultClient)

	cmdFunc := func(name string, args ...string) sync.Command {
		return exec.Command(name, args...)
	}
	dataSyncer := sync.NewRsyncSyncer(jobConfig.API.ExporterHost, jobConfig.SSH.User, jobConfig.SSH.PrivateSSHKeyPath, cmdFunc)

	return &ImportExecutor{apiClient: client, dataSyncer: dataSyncer}, nil
}

func (j ImportExecutor) Start(ctx context.Context) error {
	err := j.configSyncer.SyncConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to sync configuration: %w", err)
	}

	err = j.dataSyncer.SyncData(ctx, j.apiClient, j.config)
	if err != nil {
		return fmt.Errorf("failed to sync data: %w", err)
	}

	return nil
}
