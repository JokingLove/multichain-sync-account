package clock

import (
	"sync/atomic"
	"time"
)

type AdvancingClock struct {
	*DeterministicClock
	systemTime   Clock
	ticker       time.Ticker
	advanceEvery time.Duration
	quit         chan interface{}
	running      atomic.Value
	lastTick     time.Time
}
