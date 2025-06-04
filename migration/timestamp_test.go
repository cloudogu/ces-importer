package migration

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestFinalTimestamp_String(t *testing.T) {
	now := time.Now()
	nowString := now.Format(time.RFC3339)

	tests := []struct {
		name     string
		time     time.Time
		expValue string
	}{
		{
			name:     "Zero value",
			time:     time.Time{},
			expValue: "0001-01-01T00:00:00Z",
		},
		{
			name:     "Valid value",
			time:     now,
			expValue: nowString,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := FinalTimestamp(tt.time)
			assert.Equal(t, tt.expValue, ts.String())
		})
	}
}

func TestFinalTimestamp_Expired(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		time     time.Time
		expValue bool
	}{
		{
			name:     "Zero value",
			time:     time.Time{},
			expValue: true,
		},
		{
			name:     "valid value expired",
			time:     now,
			expValue: true,
		},
		{
			name:     "valid value expired",
			time:     now.Add(1 * time.Hour),
			expValue: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := FinalTimestamp(tt.time)
			assert.Equal(t, tt.expValue, ts.Expired())
		})
	}
}

func TestFinalTimestamp_IsZero(t *testing.T) {
	ft := FinalTimestamp{}
	assert.True(t, ft.IsZero())
}

func TestFinalTimestamp_WaitUntil(t *testing.T) {
	t.Run("final timestamp is not zero", func(t *testing.T) {
		tests := []struct {
			name string
			wait time.Duration
		}{
			{
				name: "Wait for 10 ms",
				wait: 10 * time.Millisecond,
			},
			{
				name: "Wait for 0 ms",
				wait: 0 * time.Millisecond,
			},
			{
				name: "Wait for -10 ms",
				wait: -10 * time.Millisecond,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				ft := FinalTimestamp(time.Now().Add(tt.wait))
				ft.WaitUntil(context.TODO())

				assert.True(t, ft.Expired())
			})
		}
	})

	t.Run("final timestamp is zero", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		done := make(chan struct{})
		defer close(done)

		ft := FinalTimestamp{}

		go func() {
			ft.WaitUntil(context.TODO())
			done <- struct{}{}
		}()

		select {
		case <-ctx.Done():
			assert.Fail(t, "context timeout reached")
		case <-done:
		}
	})

	t.Run("stop waiting when context is canceled", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		cancel()

		ft := FinalTimestamp(time.Now().Add(1 * time.Hour))
		ft.WaitUntil(ctx)

		require.ErrorIs(t, ctx.Err(), context.Canceled)
		assert.False(t, ft.Expired())
	})

	t.Run("stop waiting when context reached timeout", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		ft := FinalTimestamp(time.Now().Add(1 * time.Hour))
		ft.WaitUntil(ctx)

		require.ErrorIs(t, ctx.Err(), context.DeadlineExceeded)
		assert.False(t, ft.Expired())
	})
}

func TestFinalTimestamp_WaitUntilReady(t *testing.T) {
	oldTickDuration := tickDuration
	defer func() {
		tickDuration = oldTickDuration
	}()

	tickDuration = 10 * time.Millisecond

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	done := make(chan struct{})
	defer close(done)

	ft := Now()

	loopCounter := 0
	ready := func() bool {
		if loopCounter > 0 {
			return true
		}

		loopCounter++
		return false
	}

	go func() {
		ft.WaitUntilReady(ctx, ready)
		done <- struct{}{}
	}()

	select {
	case <-ctx.Done():
		assert.Fail(t, "context timeout reached")
	case <-done:
	}

	assert.True(t, ft.Expired())
	assert.Equal(t, 1, loopCounter)
}

func TestParseFinalTimestamp(t *testing.T) {
	valid := time.Now().Add(1 * time.Hour)

	tests := []struct {
		name          string
		value         string
		expValue      string
		expError      bool
		errorContains string
	}{
		{
			name:     "Parse valid timestamp",
			value:    valid.Format(time.RFC3339),
			expValue: FinalTimestamp(valid).String(),
		},
		{
			name:     "Parse valid timestamp - Zero Value",
			value:    "",
			expValue: FinalTimestamp(time.Time{}).String(),
		},
		{
			name:          "Error - invalid format",
			value:         "invalid",
			expError:      true,
			errorContains: "timestamp is not in RFC3339 format",
		},
		{
			name:          "Error - final timestamp is in the past",
			value:         time.Now().Format(time.RFC3339),
			expError:      true,
			errorContains: "final migration timestamp is in the past",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ft, err := ParseFinalTimestamp(tt.value)

			if tt.expError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorContains)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expValue, ft.String())
			}
		})
	}
}

func TestNow(t *testing.T) {
	now := Now()

	assert.False(t, now.IsZero())
	assert.True(t, now.Expired())
	assert.True(t, now.time().Before(time.Now()))

}
