package main

import (
	"context"
	"fmt"
	"github.com/cloudogu/ces-importer/api/exporter"
	"github.com/cloudogu/ces-importer/configuration"
	"github.com/cloudogu/ces-importer/sync"
	"net/http"
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
	apiConfig, err := configuration.ReadAPIConfiguration()
	if err != nil {
		return nil, fmt.Errorf("failed to read API configuration: %w", err)
	}

	sshConfig, err := configuration.ReadSSHConfiguration()
	if err != nil {
		return nil, fmt.Errorf("failed to read SSH configuration: %w", err)
	}

	_ = exporter.NewClient(apiConfig.ExporterApiKey, http.DefaultClient)

	_ = sync.NewRsyncSyncer(apiConfig.ExporterHost, sshConfig.ExporterSSHUser, sshConfig.ImporterPrivateSSHKeyPath)

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
