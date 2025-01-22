package clock

import (
	"context"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestSystemClock_SleepCtx(t *testing.T) {
	t.Run("ReturnWhenContextDone", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		start := time.Now()
		err := SystemClock.SleepCtx(ctx, 5*time.Second)
		end := time.Now()
		require.ErrorIs(t, err, context.Canceled)

		require.Less(t, end.Sub(start), time.Minute)
	})
}
