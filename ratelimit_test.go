package ratelimit_test

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vcraescu/go-ratelimit"
	"sync/atomic"
	"testing"
	"time"
)

func TestLimiter_Run(t *testing.T) {
	t.Parallel()

	t.Run("total time", func(t *testing.T) {
		t.Parallel()

		const (
			want         = 20
			rate         = 2
			wantDuration = want/rate - 1
			per          = time.Second / 5
		)

		limiter := ratelimit.NewLimiter(rate, ratelimit.WithPer(per))
		start := time.Now()

		var got int32

		for i := 0; i < want; i++ {
			limiter.Run(func() {
				atomic.AddInt32(&got, 1)
			})
		}

		err := limiter.Wait()
		require.NoError(t, err)

		gotDuration := int(time.Now().Sub(start) / per)

		require.EqualValues(t, want, got)
		require.Equal(t, wantDuration, gotDuration)
	})

	t.Run("throughput", func(t *testing.T) {
		t.Parallel()

		const (
			want = 20
			rate = 2
			per  = time.Second / 5
		)

		limiter := ratelimit.NewLimiter(rate, ratelimit.WithPer(per))
		sink := make(chan struct{}, want/rate)

		defer close(sink)

		go func() {
			i := 0
			start := time.Now()

			for range sink {
				if i%rate == 0 && i >= rate {
					dur := time.Now().Sub(start)
					start = time.Now()
					assert.InDelta(t, per.Seconds(), dur.Seconds(), 0.01)
				}

				i++
			}
		}()

		for i := 0; i < want; i++ {
			limiter.Run(func() {
				sink <- struct{}{}
			})
		}

		err := limiter.Wait()
		require.NoError(t, err)
	})
}

func TestLimiter_Wait(t *testing.T) {
	t.Parallel()

	t.Run("wait is not blocking if there is no worker", func(t *testing.T) {
		t.Parallel()

		limiter := ratelimit.NewLimiter(2)
		got := limiter.Wait()

		require.ErrorIs(t, ratelimit.ErrNoWorkers, got)
	})
}
