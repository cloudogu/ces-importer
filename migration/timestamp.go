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

type FinalTimestamp time.Time

func (ft FinalTimestamp) time() time.Time {
	return time.Time(ft)
}

func (ft FinalTimestamp) String() string {
	return time.Time(ft).Format(timeFormat)
}

func (ft FinalTimestamp) Expired() bool {
	if ft.time().IsZero() {
		return true
	}

	return time.Now().After(ft.time())
}

func (ft FinalTimestamp) IsZero() bool {
	return ft.time().IsZero()
}

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

func (ft FinalTimestamp) WaitUntilReady(ctx context.Context, ready func() bool) {
	ft.WaitUntil(ctx)

	for !ready() {
		time.Sleep(tickDuration)
	}
}

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

func Now() FinalTimestamp {
	return FinalTimestamp(time.Now())
}
