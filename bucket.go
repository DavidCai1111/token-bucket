package bucket

import (
	"container/list"
	"fmt"
	"sync"
	"time"
)

// TokenBucket is a token bucket (https://en.wikipedia.org/wiki/Token_bucket)
// that fills itself at a specified rate.
type TokenBucket struct {
	start             time.Time
	interval          time.Duration
	ticker            *time.Ticker
	tokenMutex        *sync.Mutex
	waitingQuqueMutex *sync.Mutex
	waitingQuque      *list.List
	cap               int64
	avail             int64
}

type waitingJob struct {
	ch    chan struct{}
	count int64
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
		start:             time.Now(),
		interval:          interval,
		tokenMutex:        &sync.Mutex{},
		waitingQuqueMutex: &sync.Mutex{},
		waitingQuque:      list.New(),
		cap:               cap,
		avail:             cap,
		ticker:            time.NewTicker(interval),
	}

	go tb.adjustDaemon()

	return tb
}

// Capability returns the capability of this token bucket.
func (tb TokenBucket) Capability() int64 {
	return tb.cap
}

// TryTake tasks specified count tokens from the bucket, the return value
// indicates whether this attempt is successful.
func (tb *TokenBucket) TryTake(count int64) bool {
	defer tb.tokenMutex.Unlock()
	tb.tokenMutex.Lock()

	if count <= tb.avail {
		tb.avail -= count

		return true
	}

	return false
}

// Take tasks specified count tokens from the bucket, if there is no
// more availible token in the bucket, it will wait until the toekns
// are arrived.
func (tb *TokenBucket) Take(count int64) {
	w := &waitingJob{
		ch:    make(chan struct{}),
		count: count,
	}

	tb.appendToWaitingQueue(w)

	<-w.ch
}

// TakeMaxDuration tasks specified count tokens from the bucket, if there is no
// more availible token in the bucket, it will wait until the toekns
// are arrived unless up to the duration.
func (tb *TokenBucket) TakeMaxDuration(count int64) bool {
	return false
}

func (tb *TokenBucket) Wait(count int64) {

}

func (tb *TokenBucket) WaitMaxDuration(count int64) {

}

// Destory destorys the token bucket and stop the inner channels.
func (tb *TokenBucket) Destory() {
	tb.ticker.Stop()
}

func (tb *TokenBucket) appendToWaitingQueue(w *waitingJob) {
	tb.waitingQuqueMutex.Lock()
	tb.waitingQuque.PushBack(w)
	tb.waitingQuqueMutex.Unlock()
}

func (tb *TokenBucket) adjustDaemon() {
	var waitingJobNow *waitingJob

	for now := range tb.ticker.C {
		var _ = now

		tb.tokenMutex.Lock()

		if tb.avail < tb.cap {
			tb.avail++
		}

		element := tb.waitingQuque.Front()

		if element != nil {
			if waitingJobNow == nil {
				waitingJobNow = element.Value.(*waitingJob)

				tb.waitingQuqueMutex.Lock()
				tb.waitingQuque.Remove(element)
				tb.waitingQuqueMutex.Unlock()

				if tb.avail >= waitingJobNow.count {
					tb.avail -= waitingJobNow.count

					waitingJobNow.ch <- struct{}{}
					waitingJobNow = nil
				}
			}
		}

		tb.tokenMutex.Unlock()
	}
}
