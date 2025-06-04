package migration

import (
	"context"
	"fmt"
	"strings"
	"time"
)

const timeFormat = time.RFC3339

var (
	tickDuration = 1 * time.Minute
)

// FinalTimestamp is the timestamp of the final migration.
type FinalTimestamp time.Time

func (ft FinalTimestamp) time() time.Time {
	return time.Time(ft)
}

// String converts FinalTimestamp into a string in RFC3339 format.
func (ft FinalTimestamp) String() string {
	return time.Time(ft).Format(timeFormat)
}

// Expired return the status whether the final timestamp is expired meaning the time now is bigger than the time of
// the final timestamp. When FinalTimeStamp is the zero value the timestamp expires immediately.
func (ft FinalTimestamp) Expired() bool {
	if ft.time().IsZero() {
		return true
	}

	return time.Now().After(ft.time())
}

// IsZero reports whether the final timestamp is the zero value.
func (ft FinalTimestamp) IsZero() bool {
	return ft.time().IsZero()
}

// WaitUntil waits until the final timestamp is reached. This method blocks until the timestamp is reached or the
// provided context is done.
func (ft FinalTimestamp) WaitUntil(ctx context.Context) {
	if ft.IsZero() {
		return
	}

	waitDuration := time.Until(ft.time())
	if waitDuration <= 0 {
		return
	}

	select {
	case <-ctx.Done():
	case <-time.After(waitDuration):
	}
}

// WaitUntilReady waits until the final timestamp is reached and the provided ready function returns true.
// This method blocks until the timestamp is reached and ready or the provided context is done.
func (ft FinalTimestamp) WaitUntilReady(ctx context.Context, ready func() bool) {
	ft.WaitUntil(ctx)

	for !ready() {
		time.Sleep(tickDuration)
	}
}

// ParseFinalTimestamp parses the provided string into a FinalTimestamp. The function expects a string in the RFC3339
// format. An error is returned when parsing fails or the timestamp lies in the past. An empty string results in a
// FinalTimestamp with a zero value.
func ParseFinalTimestamp(timestamp string) (FinalTimestamp, error) {
	if strings.TrimSpace(timestamp) == "" {
		return FinalTimestamp{}, nil
	}

	finalTimestamp, err := time.Parse(timeFormat, timestamp)
	if err != nil {
		return FinalTimestamp{}, fmt.Errorf("timestamp is not in RFC3339 format: %w", err)
	}

	if time.Now().After(finalTimestamp) {
		return FinalTimestamp{}, fmt.Errorf("final migration timestamp is in the past")
	}

	return FinalTimestamp(finalTimestamp), nil
}

// Now return a FinalTimestamp with the current time.
func Now() FinalTimestamp {
	return FinalTimestamp(time.Now())
}
