package main

import (
	"context"
	"fmt"
	"github.com/cloudogu/ces-importer/configuration"
	"github.com/cloudogu/ces-importer/sync"
)

type dataSyncer interface {
	SyncData(ctx context.Context) error
}

type configSyncer interface {
	SyncConfig(ctx context.Context) error
}

type ImportExecutor struct {
	configSyncer
	dataSyncer
}

func NewImportExecutor() (*ImportExecutor, error) {
	jobConfig, err := configuration.ReadJobConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to read job configuration: %w", err)
	}

	//_ = exporter.NewClient(jobConfig.API.ExporterApiKey, http.DefaultClient)

	_ = sync.NewRsyncSyncer(jobConfig.API.ExporterHost, jobConfig.SSH.User, jobConfig.SSH.PrivateSSHKeyPath)

	return &ImportExecutor{}, nil
}

func (j ImportExecutor) Start(ctx context.Context) error {
	err := j.configSyncer.SyncConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to sync configuration: %w", err)
	}

	err = j.dataSyncer.SyncData(ctx)
	if err != nil {
		return fmt.Errorf("failed to sync data: %w", err)
	}

	return nil
}
