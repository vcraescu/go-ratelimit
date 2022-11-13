package ratelimit

import (
	"sync"
	"time"
)

type config struct {
	per time.Duration
}

type Option interface {
	apply(cfg *config)
}

var _ Option = optionFunc(nil)

type optionFunc func(cfg *config)

func (fn optionFunc) apply(cfg *config) {
	fn(cfg)
}

type Limiter struct {
	bucket chan struct{}
	rate   int
	once   sync.Once
	ticker *time.Ticker
	wg     sync.WaitGroup
	init   chan struct{}
	done   chan struct{}
	per    time.Duration
}

func WithPer(per time.Duration) Option {
	return optionFunc(func(cfg *config) {
		cfg.per = per
	})
}

func NewLimiter(rate int, opts ...Option) *Limiter {
	cfg := &config{
		per: time.Second,
	}

	for _, opt := range opts {
		opt.apply(cfg)
	}

	return &Limiter{
		per:  cfg.per,
		rate: rate,
		wg:   sync.WaitGroup{},
		init: make(chan struct{}, 1),
		done: make(chan struct{}, 1),
	}
}

func (l *Limiter) refillBucket() {
	n := l.rate - len(l.bucket)

	for i := 0; i < n; i++ {
		l.tryChPush(l.bucket)
	}
}

func (l *Limiter) tryChPush(ch chan struct{}) {
	select {
	case ch <- struct{}{}:
	default:
	}
}

func (l *Limiter) startBucketRefill() {
	l.bucket = make(chan struct{}, l.rate)
	l.ticker = time.NewTicker(l.per)

	l.refillBucket()

	go func() {
		for {
			select {
			case <-l.ticker.C:
				l.refillBucket()
			case <-l.done:
				return
			}
		}
	}()
}

func (l *Limiter) Run(worker func()) {
	l.once.Do(l.startBucketRefill)
	l.wg.Add(1)

	l.tryChPush(l.init)

	go func() {
		defer l.wg.Done()

		<-l.bucket
		worker()
	}()
}

func (l *Limiter) Wait() {
	<-l.init
	l.wg.Wait()

	l.once = sync.Once{}
	l.ticker.Stop()
	l.done <- struct{}{}
}
