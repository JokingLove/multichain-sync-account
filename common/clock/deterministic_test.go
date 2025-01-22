package clock

import (
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestNowReturnCurrentTime(t *testing.T) {
	now := time.UnixMilli(23829382)
	clock := NewDeterministicClock(now)
	require.Equal(t, now, clock.Now())
}

func TestAdvanceTime(t *testing.T) {
	start := time.UnixMilli(100)
	clock := NewDeterministicClock(start)
	clock.AdvanceTime(500 * time.Millisecond)
	require.Equal(t, start.Add(500*time.Millisecond), clock.Now())
}
