package ratelimit

import (
	"sync"
	"time"
)

// TokenBucket is a token bucket (https://en.wikipedia.org/wiki/Token_bucket)
// that fills itself in a specified rate.
type TokenBucket struct {
	start    time.Time
	interval time.Duration
	m        *sync.Mutex
	cap      int64
	avail    int64
}

// NewTokenBucket returns a new token bucket with specified fill interval and
// capability.
func NewTokenBucket(interval time.Duration, cap int64) *TokenBucket {
	tb := &TokenBucket{
		start:    time.Now(),
		interval: interval,
		m:        &sync.Mutex{},
		cap:      cap,
		avail:    cap,
	}

	go tb.adjustDaemon()

	return tb
}

// Capability returns the capability of this token bucket.
func (tb TokenBucket) Capability() int64 {
	return tb.cap
}

// Take tasks specified count tokens from the bucket, the return value
// indicates whether this attempt is successful.
func (tb *TokenBucket) Take(count int64) bool {
	defer tb.m.Unlock()
	tb.m.Lock()

	if count < tb.avail {
		tb.avail -= count

		return true
	}

	return false
}

func (tb *TokenBucket) adjustDaemon() {
	tick := time.Tick(tb.interval)

	for now := range tick {
		var _ = now

		tb.m.Lock()

		if tb.avail < tb.cap {
			tb.avail++
		}

		tb.m.Unlock()
	}
}
