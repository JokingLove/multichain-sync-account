package clock

import (
	"context"
	"sync"
	"time"
)

type action interface {
	// return true if the action is due to fire
	isDue(time.Time) bool

	// fire triggers the action. Returns true if the action needs to fire again in the future
	fire(time.Time) bool
}

type task struct {
	ch  chan time.Time
	due time.Time
}

func (t *task) isDue(now time.Time) bool {
	return !t.due.After(now)
}

func (t *task) fire(now time.Time) bool {
	t.ch <- now
	close(t.ch)
	return false
}

type timer struct {
	f       func()
	ch      chan time.Time
	due     time.Time
	stopped bool
	run     bool
	sync.Mutex
}

func (t timer) isDue(now time.Time) bool {
	t.Lock()
	defer t.Unlock()
	return !t.due.After(now)
}

func (t timer) fire(now time.Time) bool {
	t.Lock()
	defer t.Unlock()
	if !t.stopped {
		t.f()
		t.run = true
	}
	return false
}

func (t *timer) Ch() <-chan time.Time {
	return t.ch
}

func (t *timer) Stop() bool {
	t.Lock()
	defer t.Unlock()
	r := !t.stopped && !t.run
	t.stopped = true
	return r
}

type ticker struct {
	c       Clock
	ch      chan time.Time
	nextDue time.Time
	period  time.Duration
	stopped bool
	sync.Mutex
}

func (t *ticker) Ch() <-chan time.Time {
	return t.ch
}

func (t *ticker) Stop() {
	t.Lock()
	defer t.Unlock()
	t.stopped = true
}

func (t *ticker) Reset(d time.Duration) {
	if d <= 0 {
		panic("Continuously firing tickers are a really bad idea")
	}
	t.Lock()
	defer t.Unlock()
	t.period = d
	t.nextDue = t.c.Now().Add(d)
}

func (t *ticker) isDue(now time.Time) bool {
	t.Lock()
	defer t.Unlock()
	return !t.nextDue.After(now)
}

func (t *ticker) fire(now time.Time) bool {
	t.Lock()
	defer t.Unlock()
	if t.stopped {
		return false
	}

	// publish without blocking and only update due time if we publish successfully
	select {
	case t.ch <- now:
		t.nextDue = now.Add(t.period)
	default:

	}

	return true
}

type DeterministicClock struct {
	now          time.Time
	pending      []action
	newPendingCh chan struct{}
	lock         sync.Mutex
}

func NewDeterministicClock(now time.Time) *DeterministicClock {
	return &DeterministicClock{
		now:          now,
		newPendingCh: make(chan struct{}, 1),
	}
}

func (d *DeterministicClock) Now() time.Time {
	d.lock.Lock()
	defer d.lock.Unlock()
	return d.now
}

func (d *DeterministicClock) Since(t time.Time) time.Duration {
	d.lock.Lock()
	defer d.lock.Unlock()
	return d.now.Sub(t)
}

func (d *DeterministicClock) After(dur time.Duration) <-chan time.Time {
	d.lock.Lock()
	defer d.lock.Unlock()
	ch := make(chan time.Time, 1)
	if dur.Nanoseconds() == 0 {
		ch <- d.now
		close(ch)
	} else {
		d.addPending(&task{ch: ch, due: d.now})
	}
	return ch
}

func (d *DeterministicClock) AfterFunc(dur time.Duration, f func()) Timer {
	d.lock.Lock()
	defer d.lock.Unlock()
	timer := &timer{f: f, due: d.now.Add(dur)}
	if dur.Nanoseconds() == 0 {
		timer.fire(d.now)
	} else {
		d.addPending(timer)
	}
	return timer
}

func (d *DeterministicClock) NewTicker(dur time.Duration) Ticker {
	if dur <= 0 {
		panic("Continuously firing tickers are a really bad idea")
	}
	d.lock.Lock()
	defer d.lock.Unlock()
	ch := make(chan time.Time, 1)
	t := &ticker{
		c:       d,
		ch:      ch,
		nextDue: d.now.Add(dur),
		period:  dur,
	}
	d.addPending(t)
	return t
}

func (s *DeterministicClock) NewTimer(d time.Duration) Timer {
	s.lock.Lock()
	defer s.lock.Unlock()
	ch := make(chan time.Time, 1)
	t := &timer{
		f: func() {
			ch <- s.now
		},
		ch:  ch,
		due: s.now.Add(d),
	}
	s.addPending(t)
	return t
}

func (s *DeterministicClock) SleepCtx(ctx context.Context, d time.Duration) error {
	return sleepCtx(ctx, d, s)
}

func (d *DeterministicClock) addPending(t action) {
	d.pending = append(d.pending, t)
	select {
	case d.newPendingCh <- struct{}{}:
	default:
		// Must already have a new pending task flagged, do nothing
	}
}

func (s *DeterministicClock) WaitForNewPendingTaskWithTimeout(timeout time.Duration) bool {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return s.WaitForNewPendingTask(ctx)
}

func (s *DeterministicClock) WaitForNewPendingTask(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return false
	case <-s.newPendingCh:
		return true
	}
}

func (s *DeterministicClock) AdvanceTime(d time.Duration) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.now = s.now.Add(d)
	var remaining []action
	for _, a := range s.pending {
		if a.isDue(s.now) {
			remaining = append(remaining, a)
		}
	}
	s.pending = remaining
}

var _ Clock = (*DeterministicClock)(nil)
