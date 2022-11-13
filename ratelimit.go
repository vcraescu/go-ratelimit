package ratelimit

import (
	"errors"
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

var ErrNoWorkers = errors.New("no workers")

type Limiter struct {
	bucket  chan struct{}
	rate    int
	once    sync.Once
	ticker  *time.Ticker
	wg      sync.WaitGroup
	started chan struct{}
	done    chan struct{}
	per     time.Duration
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
		per:     cfg.per,
		rate:    rate,
		wg:      sync.WaitGroup{},
		started: make(chan struct{}, 1),
		done:    make(chan struct{}, 1),
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

func (l *Limiter) take() {
	<-l.bucket
}

func (l *Limiter) runWorker(worker func()) {
	defer l.wg.Done()

	l.take()
	worker()
}

func (l *Limiter) Run(worker func()) {
	l.once.Do(l.startBucketRefill)
	l.wg.Add(1)

	l.tryChPush(l.started)

	go l.runWorker(worker)
}

func (l *Limiter) isStarted() bool {
	select {
	case <-l.started:
		return true
	default:
		return false
	}
}

func (l *Limiter) reset() {
	l.once = sync.Once{}
	l.started = make(chan struct{}, 1)
	l.ticker.Stop()
}

func (l *Limiter) Wait() error {
	if !l.isStarted() {
		return ErrNoWorkers
	}

	l.wg.Wait()
	l.reset()

	l.done <- struct{}{}

	return nil
}
