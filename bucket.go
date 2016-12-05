package ratelimit

import (
	"fmt"
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
	ticker   *time.Ticker
}

// NewTokenBucket returns a new token bucket with specified fill interval and
// capability.
func NewTokenBucket(interval time.Duration, cap int64) *TokenBucket {
	if interval < 0 {
		panic(fmt.Sprintf("ratelimit: interval %v should > 0", interval))
	}

	if cap < 0 {
		panic(fmt.Sprintf("ratelimit: capability %v should > 0", cap))
	}

	tb := &TokenBucket{
		start:    time.Now(),
		interval: interval,
		m:        &sync.Mutex{},
		cap:      cap,
		avail:    cap,
		ticker:   time.NewTicker(interval),
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

// Destory destorys the token bucket and stop the inner channels.
func (tb *TokenBucket) Destory() {
	tb.ticker.Stop()
}

func (tb *TokenBucket) adjustDaemon() {
	for now := range tb.ticker.C {
		var _ = now

		tb.m.Lock()

		if tb.avail < tb.cap {
			tb.avail++
		}

		tb.m.Unlock()
	}
}
