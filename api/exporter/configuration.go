package exporter

import (
	"context"
	"encoding/json"
	"fmt"
)

const pathConfiguration = "/configuration"

// KeyValue contains a key-value pair.
// This is used to represent configuration values.
type KeyValue struct {
	// Key is the name of the configuration value.
	Key string `json:"key"`
	// Value is the value of the configuration value.
	Value string `json:"value"`
}

// GlobalConfig contains the global configuration of the exporter system.
type GlobalConfig []KeyValue

// DoguConfig contains the configuration of a single dogu.
type DoguConfig struct {
	// Name of the dogu.
	Name string `json:"name"`
	// NormalConfig normal dogu configuration as KeyValue pairs
	NormalConfig []KeyValue `json:"normal"`
	// LocalConfig local dogu configuration as KeyValue pairs
	LocalConfig []KeyValue `json:"local"`
	// SensitiveConfig sensitive dogu configuration as KeyValue pairs
	SensitiveConfig []KeyValue `json:"sensitive"`
}

// BackupSchedule contains the configuration of a single backup schedule.
type BackupSchedule struct {
	// Name of the backup schedule.
	Name string `json:"name"`
	// Schedule is defined as a cron expression.
	Schedule string `json:"schedule"`
}

// Configuration contains the configuration of the exporter system.
type Configuration struct {
	// GlobalConfig is the global configuration of the exporter system.
	GlobalConfig GlobalConfig `json:"global"`
	// DoguConfigs is the configuration of all installed dogus in the exporter system.
	DoguConfigs []DoguConfig `json:"dogus"`
	// BackupSchedules is the configuration of all backup schedules in the exporter system.
	BackupSchedules []BackupSchedule `json:"backupSchedules"`
}

// NewConfigService creates a new ConfigService.
func NewConfigService(apiClient apiClient) *ConfigService {
	return &ConfigService{
		apiClient: apiClient,
	}
}

// ConfigService is a client for the exporter configuration API.
type ConfigService struct {
	apiClient apiClient
}

// GetConfig returns the configuration of the exporter system.
func (cs *ConfigService) GetConfig(ctx context.Context) (*Configuration, error) {
	var config Configuration
	res, err := cs.apiClient.DoGetRequest(ctx, pathConfiguration)
	if err != nil {
		return nil, fmt.Errorf("error getting configuration from exporter: %w", err)
	}

	err = json.Unmarshal(res, &config)
	if err != nil {
		return nil, fmt.Errorf("could not unmarshal configuration from exporter: %w", err)
	}

	return &config, nil
}
