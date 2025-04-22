package cron

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Run("should return an param validation error", func(t *testing.T) {
		_, err := New("blubb")

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, `cron expression "blubb" is invalid`)
	})
}

func Test_mainLooper(t *testing.T) {
	t.Run("should return without error", func(t *testing.T) {
		// Given
		sut, err := New("* * * * * *") // exec every second
		require.NoError(t, err)
		require.NotNil(t, sut)

		var calledCounter *int
		calledCounter = new(int)

		go func() {
			// When
			sut.Run(func(ctx context.Context) error {
				println("Test function was calledCounter")
				*calledCounter++
				time.Sleep(500 * time.Millisecond) // run less than a second

				return nil
			})
			require.NoError(t, err)
		}()

		time.Sleep(2 * time.Second)
		sut.Stop()

		// Then
		require.NoError(t, err)
		assert.GreaterOrEqual(t, 1, *calledCounter)
	})
}
