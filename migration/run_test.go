package migration

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
	"time"
)

func TestRun(t *testing.T) {
	type inFields struct {
		ctxTimeout  time.Duration
		timestamp   func() string
		regularCron string
		setupMock   func(*mockMigrationRunner)
	}

	tests := []struct {
		name        string
		in          inFields
		expErr      bool
		errContains string
	}{
		{
			name: "Run delta and final migration",
			in: inFields{
				ctxTimeout: 5 * time.Second,
				timestamp: func() string {
					return time.Now().Add(2 * time.Second).Format(time.RFC3339)
				},
				regularCron: "* * * * * *",
				setupMock: func(m *mockMigrationRunner) {
					m.EXPECT().RunMigration(mock.Anything).Return(nil)
				},
			},
			expErr:      false,
			errContains: "",
		},
		{
			name: "Delta migrations fails but final migration succeeds",
			in: inFields{
				ctxTimeout: 5 * time.Second,
				timestamp: func() string {
					return time.Now().Add(2 * time.Second).Format(time.RFC3339)
				},
				regularCron: "* * * * * *",
				setupMock: func(m *mockMigrationRunner) {
					m.EXPECT().RunMigration(mock.Anything).RunAndReturn(func(ctx context.Context) error {
						if !IsFinalMigration(ctx) {
							return assert.AnError
						}

						return nil
					})
				},
			},
			expErr:      false,
			errContains: "",
		},
		{
			name: "Dont run any migration if final timestamp is in the past",
			in: inFields{
				ctxTimeout: 5 * time.Second,
				timestamp: func() string {
					return time.Now().Format(time.RFC3339)
				},
				regularCron: "* * * * * *",
				setupMock:   func(m *mockMigrationRunner) {},
			},
			expErr:      false,
			errContains: "",
		},
		{
			name: "Dont run final migration if final timestamp is empty",
			in: inFields{
				ctxTimeout: 0 * time.Second,
				timestamp: func() string {
					return ""
				},
				regularCron: "* * * * * *",
				setupMock: func(m *mockMigrationRunner) {
					m.On("RunMigration", mock.Anything).Maybe().Return(nil)
				},
			},
			expErr:      false,
			errContains: "",
		},
		{
			name: "Fallback to empty final timestamp if final timestamp is invalid",
			in: inFields{
				ctxTimeout: 0 * time.Second,
				timestamp: func() string {
					return "invalid"
				},
				regularCron: "* * * * * *",
				setupMock: func(m *mockMigrationRunner) {
					m.On("RunMigration", mock.Anything).Maybe().Return(nil)
				},
			},
			expErr:      false,
			errContains: "",
		},
		{
			name: "Error: Migration fails if cron is invalid",
			in: inFields{
				ctxTimeout: 5 * time.Second,
				timestamp: func() string {
					return time.Now().Add(2 * time.Second).Format(time.RFC3339)
				},
				regularCron: "invalid",
				setupMock:   func(m *mockMigrationRunner) {},
			},
			expErr:      true,
			errContains: "failed to create cron looper for expression",
		},
		{
			name: "Error: Migration fails if context is cancelled before final migration is started",
			in: inFields{
				ctxTimeout: 0 * time.Second,
				timestamp: func() string {
					return time.Now().Add(10 * time.Second).Format(time.RFC3339)
				},
				regularCron: "* * * * * *",
				setupMock: func(m *mockMigrationRunner) {
					m.On("RunMigration", mock.Anything).Maybe().Return(nil)
				},
			},
			expErr:      true,
			errContains: "received shutdown signal before final migration has been completed",
		},
		{
			name: "Error: Final migrations fails",
			in: inFields{
				ctxTimeout: 5 * time.Second,
				timestamp: func() string {
					return time.Now().Add(1 * time.Second).Format(time.RFC3339)
				},
				regularCron: "* * 1 * * *",
				setupMock: func(m *mockMigrationRunner) {
					m.EXPECT().RunMigration(mock.Anything).RunAndReturn(func(ctx context.Context) error {
						if IsFinalMigration(ctx) {
							return assert.AnError
						}

						return nil
					})
				},
			},
			expErr:      true,
			errContains: "failed to run final migration",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			migrationRunnerMock := newMockMigrationRunner(t)
			tt.in.setupMock(migrationRunnerMock)

			ctx, cancel := context.WithTimeout(context.Background(), tt.in.ctxTimeout)
			defer cancel()

			err := Run(ctx, tt.in.timestamp(), tt.in.regularCron, migrationRunnerMock)

			assert.Equal(t, tt.expErr, err != nil)
			if err != nil {
				assert.Contains(t, err.Error(), tt.errContains)
			}
		})
	}
}
