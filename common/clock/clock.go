package clock

import (
	"context"
	"time"
)

// Clock represents time in a way that can be provided by varying implements.
// Methods are designed to be direct replacements for methods in the time package.
// with some new additions to make common patterns simple.
type Clock interface {
	// Now provides the current local time. Equivalent to time.Now()
	Now() time.Time

	// Since returns the time elapsed since t. It is shorthand for time.Now().Sub(t).
	Since(t time.Time) time.Duration

	// After waits for the duration to elapse and then sends the current time on the returned
	After(d time.Duration) <-chan time.Time

	AfterFunc(d time.Duration, f func()) Timer

	NewTicker(d time.Duration) Ticker

	NewTimer(d time.Duration) Timer

	SleepCtx(ctx context.Context, d time.Duration) error
}

// A ticker holds a channel that delivers "ticks" of a clock at intervals
type Ticker interface {
	Ch() <-chan time.Time

	Stop()

	Reset(d time.Duration)
}

type Timer interface {
	Ch() <-chan time.Time
	Stop() bool
}

var SystemClock Clock = systemClock{}

type systemClock struct {
}

func (s systemClock) Since(t time.Time) time.Duration {
	return time.Since(t)
}

func (s systemClock) Now() time.Time {
	return time.Now()
}

func (s systemClock) After(d time.Duration) <-chan time.Time {
	return time.After(d)
}

type SystemTicker struct {
	*time.Ticker
}

func (t *SystemTicker) Ch() <-chan time.Time {
	return t.C
}

func (s systemClock) NewTicker(d time.Duration) Ticker {
	return &SystemTicker{time.NewTicker(d)}
}

func (s systemClock) NewTimer(d time.Duration) Timer {
	return &SystemTimer{time.NewTimer(d)}
}

type SystemTimer struct {
	*time.Timer
}

func (t *SystemTimer) Ch() <-chan time.Time {
	return t.C
}

func (t systemClock) AfterFunc(d time.Duration, f func()) Timer {
	return &SystemTimer{time.AfterFunc(d, f)}
}

func (s systemClock) SleepCtx(ctx context.Context, d time.Duration) error {
	timer := s.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.Ch():
		return nil
	}
}
