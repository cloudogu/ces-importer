package configuration

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestConfigImporter_SyncConfig(t *testing.T) {
	testCtx := context.Background()

	t.Run("should import the configuration", func(t *testing.T) {
		cfg := &configuration{
			GlobalConfig: []keyValue{
				{Key: "key1", Value: "value1"},
				{Key: "key2", Value: "value2"},
			},
			DoguConfigs: []doguConfig{
				{
					Name: "dogu1",
					NormalConfig: []keyValue{
						{Key: "key1", Value: "value1"},
					},
				},
			},
			BackupSchedules: []backupSchedule{
				{Name: "schedule 1", Schedule: "* * * *"},
			},
		}

		mGetter := newMockConfigGetter(t)
		mGetter.EXPECT().GetConfig(testCtx).Return(cfg, nil)

		mockGci := newMockGlobalConfigImporter(t)
		mockGci.EXPECT().importGlobalConfig(testCtx, cfg.GlobalConfig).Return(nil)

		mockDci := newMockDoguConfigImporter(t)
		mockDci.EXPECT().importDoguConfigs(testCtx, cfg.DoguConfigs).Return(nil)

		mockBsi := newMockBackupScheduleImporter(t)
		mockBsi.EXPECT().importBackupSchedules(testCtx, cfg.BackupSchedules).Return(nil)

		ci := &ConfigImporter{
			getter:                 mGetter,
			globalConfigImporter:   mockGci,
			doguConfigImporter:     mockDci,
			backupScheduleImporter: mockBsi,
		}

		err := ci.SyncConfig(testCtx)

		require.NoError(t, err)
	})

	t.Run("should fail to import the configuration on error in getter", func(t *testing.T) {
		cfg := &configuration{
			GlobalConfig: []keyValue{
				{Key: "key1", Value: "value1"},
				{Key: "key2", Value: "value2"},
			},
			DoguConfigs: []doguConfig{
				{
					Name: "dogu1",
					NormalConfig: []keyValue{
						{Key: "key1", Value: "value1"},
					},
				},
			},
			BackupSchedules: []backupSchedule{
				{Name: "schedule 1", Schedule: "* * * *"},
			},
		}

		mGetter := newMockConfigGetter(t)
		mGetter.EXPECT().GetConfig(testCtx).Return(cfg, assert.AnError)

		mockGci := newMockGlobalConfigImporter(t)

		mockDci := newMockDoguConfigImporter(t)

		mockBsi := newMockBackupScheduleImporter(t)

		ci := &ConfigImporter{
			getter:                 mGetter,
			globalConfigImporter:   mockGci,
			doguConfigImporter:     mockDci,
			backupScheduleImporter: mockBsi,
		}

		err := ci.SyncConfig(testCtx)

		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to get configuration from exporter:")
	})

	t.Run("should fail to import the configuration on error in global config", func(t *testing.T) {
		cfg := &configuration{
			GlobalConfig: []keyValue{
				{Key: "key1", Value: "value1"},
				{Key: "key2", Value: "value2"},
			},
			DoguConfigs: []doguConfig{
				{
					Name: "dogu1",
					NormalConfig: []keyValue{
						{Key: "key1", Value: "value1"},
					},
				},
			},
			BackupSchedules: []backupSchedule{
				{Name: "schedule 1", Schedule: "* * * *"},
			},
		}

		mGetter := newMockConfigGetter(t)
		mGetter.EXPECT().GetConfig(testCtx).Return(cfg, nil)

		mockGci := newMockGlobalConfigImporter(t)
		mockGci.EXPECT().importGlobalConfig(testCtx, cfg.GlobalConfig).Return(assert.AnError)

		mockDci := newMockDoguConfigImporter(t)

		mockBsi := newMockBackupScheduleImporter(t)

		ci := &ConfigImporter{
			getter:                 mGetter,
			globalConfigImporter:   mockGci,
			doguConfigImporter:     mockDci,
			backupScheduleImporter: mockBsi,
		}

		err := ci.SyncConfig(testCtx)

		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to import global configuration:")
	})

	t.Run("should fail to import the configuration on error in dogu config", func(t *testing.T) {
		cfg := &configuration{
			GlobalConfig: []keyValue{
				{Key: "key1", Value: "value1"},
				{Key: "key2", Value: "value2"},
			},
			DoguConfigs: []doguConfig{
				{
					Name: "dogu1",
					NormalConfig: []keyValue{
						{Key: "key1", Value: "value1"},
					},
				},
			},
			BackupSchedules: []backupSchedule{
				{Name: "schedule 1", Schedule: "* * * *"},
			},
		}

		mGetter := newMockConfigGetter(t)
		mGetter.EXPECT().GetConfig(testCtx).Return(cfg, nil)

		mockGci := newMockGlobalConfigImporter(t)
		mockGci.EXPECT().importGlobalConfig(testCtx, cfg.GlobalConfig).Return(nil)

		mockDci := newMockDoguConfigImporter(t)
		mockDci.EXPECT().importDoguConfigs(testCtx, cfg.DoguConfigs).Return(assert.AnError)

		mockBsi := newMockBackupScheduleImporter(t)

		ci := &ConfigImporter{
			getter:                 mGetter,
			globalConfigImporter:   mockGci,
			doguConfigImporter:     mockDci,
			backupScheduleImporter: mockBsi,
		}

		err := ci.SyncConfig(testCtx)

		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to import dogu configuration:")
	})

	t.Run("should fail to import the configuration on error in backup schedules", func(t *testing.T) {
		cfg := &configuration{
			GlobalConfig: []keyValue{
				{Key: "key1", Value: "value1"},
				{Key: "key2", Value: "value2"},
			},
			DoguConfigs: []doguConfig{
				{
					Name: "dogu1",
					NormalConfig: []keyValue{
						{Key: "key1", Value: "value1"},
					},
				},
			},
			BackupSchedules: []backupSchedule{
				{Name: "schedule 1", Schedule: "* * * *"},
			},
		}

		mGetter := newMockConfigGetter(t)
		mGetter.EXPECT().GetConfig(testCtx).Return(cfg, nil)

		mockGci := newMockGlobalConfigImporter(t)
		mockGci.EXPECT().importGlobalConfig(testCtx, cfg.GlobalConfig).Return(nil)

		mockDci := newMockDoguConfigImporter(t)
		mockDci.EXPECT().importDoguConfigs(testCtx, cfg.DoguConfigs).Return(nil)

		mockBsi := newMockBackupScheduleImporter(t)
		mockBsi.EXPECT().importBackupSchedules(testCtx, cfg.BackupSchedules).Return(assert.AnError)

		ci := &ConfigImporter{
			getter:                 mGetter,
			globalConfigImporter:   mockGci,
			doguConfigImporter:     mockDci,
			backupScheduleImporter: mockBsi,
		}

		err := ci.SyncConfig(testCtx)

		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to import backup schedules:")
	})
}

func TestMatchesAnyKeyByPattern(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		patterns []string
		want     bool
	}{
		{
			name:     "Exact match",
			key:      "my-key",
			patterns: []string{"my-key"},
			want:     true,
		},
		{
			name:     "Pattern match",
			key:      "config1/key",
			patterns: []string{"config1/*"},
			want:     true,
		},
		{
			name:     "No pattern match",
			key:      "config2/other",
			patterns: []string{"config1/*"},
			want:     false,
		},
		{
			name:     "Empty patterns",
			key:      "my-key",
			patterns: []string{},
			want:     false,
		},
		{
			name:     "Invalid pattern",
			key:      "key-to-test",
			patterns: []string{"["},
			want:     false,
		},
		{
			name:     "Multiple patterns, one match",
			key:      "key1",
			patterns: []string{"key2", "key1"},
			want:     true,
		},
		{
			name:     "No match in multiple patterns",
			key:      "key3",
			patterns: []string{"key1", "key2"},
			want:     false,
		},
		{
			name:     "Wildcard pattern",
			key:      "any-key",
			patterns: []string{"*"},
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchesAnyKeyByPattern(tt.key, tt.patterns)

			if got != tt.want {
				t.Errorf("matchesAnyKeyByPattern() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfigImporter_SyncCertificates(t *testing.T) {
	testCtx := context.Background()

	t.Run("should import the certificates", func(t *testing.T) {
		cfg := &configuration{
			GlobalConfig: []keyValue{
				{Key: "ignoredkey", Value: "ignoredvalue"},
				{Key: "fqdn", Value: "https://test.ces.importer.de"},
				{Key: "certificate/server.crt", Value: "https://test.ces.importer.de"},
			},
		}

		mGetter := newMockConfigGetter(t)
		mGetter.EXPECT().GetConfig(testCtx).Return(cfg, nil)

		mockGci := newMockGlobalConfigImporter(t)
		mockGci.EXPECT().importGlobalCertificates(testCtx, cfg.GlobalConfig).Return(nil)

		ci := &ConfigImporter{
			getter:               mGetter,
			globalConfigImporter: mockGci,
		}

		err := ci.SyncCertificates(testCtx)

		require.NoError(t, err)
	})

	t.Run("should fail to import the certificates on error in getter", func(t *testing.T) {
		cfg := &configuration{
			GlobalConfig: []keyValue{
				{Key: "key1", Value: "value1"},
				{Key: "key2", Value: "value2"},
			},
		}

		mGetter := newMockConfigGetter(t)
		mGetter.EXPECT().GetConfig(testCtx).Return(cfg, assert.AnError)

		mockGci := newMockGlobalConfigImporter(t)

		ci := &ConfigImporter{
			getter:               mGetter,
			globalConfigImporter: mockGci,
		}

		err := ci.SyncCertificates(testCtx)

		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to get configuration from exporter:")
	})

	t.Run("should fail to import the certificates on error in global config", func(t *testing.T) {
		cfg := &configuration{
			GlobalConfig: []keyValue{
				{Key: "key1", Value: "value1"},
				{Key: "key2", Value: "value2"},
			},
		}

		mGetter := newMockConfigGetter(t)
		mGetter.EXPECT().GetConfig(testCtx).Return(cfg, nil)

		mockGci := newMockGlobalConfigImporter(t)
		mockGci.EXPECT().importGlobalCertificates(testCtx, cfg.GlobalConfig).Return(assert.AnError)

		ci := &ConfigImporter{
			getter:               mGetter,
			globalConfigImporter: mockGci,
		}

		err := ci.SyncCertificates(testCtx)

		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to import global certificates:")
	})
}
