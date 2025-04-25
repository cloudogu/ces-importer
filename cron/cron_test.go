package cron

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testCtx context.Context = context.Background()

func TestNew(t *testing.T) {
	t.Run("should return an param validation error", func(t *testing.T) {
		_, err := New(testCtx, "blubb", nil)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, `cron expression "blubb" is invalid`)
	})
}

func Test_mainLooper(t *testing.T) {
	t.Run("should return without error", func(t *testing.T) {
		// Given
		var calledCounter *int
		calledCounter = new(int)

		sut, err := New(testCtx, "* * * * * *", func(ctx context.Context) (int, error) {
			println("Test function was calledCounter")
			*calledCounter++
			time.Sleep(500 * time.Millisecond) // run less than a second

			return 0, nil
		}) // exec every second
		require.NoError(t, err)
		require.NotNil(t, sut)

		go func() {
			// When
			sut.Run()
			require.NoError(t, err)
		}()

		time.Sleep(2 * time.Second)
		sut.Stop()

		// Then
		assert.GreaterOrEqual(t, 1, *calledCounter)
	})
}
