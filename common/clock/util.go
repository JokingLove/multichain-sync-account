package clock

import "time"

type RWClock interface {
	Now() time.Time
}

func MinCheckedTimestamp(clock RWClock, duration time.Duration) uint64 {
	if duration.Seconds() == 0 {
		return 0
	}
	if clock.Now().Unix() > int64(duration.Seconds()) {
		return uint64(clock.Now().Add(-duration).Unix())
	}
	return 0
}
