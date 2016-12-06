package bucket

import (
	"container/list"
	"fmt"
	"sync"
	"time"
)

// TokenBucket represents a token bucket
// (https://en.wikipedia.org/wiki/Token_bucket) which based on multi goroutines,
// and is safe to use under concurrency environments.
type TokenBucket struct {
	interval          time.Duration
	ticker            *time.Ticker
	tokenMutex        *sync.Mutex
	waitingQuqueMutex *sync.Mutex
	waitingQuque      *list.List
	cap               int64
	avail             int64
}

type waitingJob struct {
	ch        chan struct{}
	need      int64
	use       int64
	abandoned bool
}

// New returns a new token bucket with specified fill interval and
// capability. The bucket is initially full.
func New(interval time.Duration, cap int64) *TokenBucket {
	if interval < 0 {
		panic(fmt.Sprintf("ratelimit: interval %v should > 0", interval))
	}

	if cap < 0 {
		panic(fmt.Sprintf("ratelimit: capability %v should > 0", cap))
	}

	tb := &TokenBucket{
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

// TryTake trys to task specified count tokens from the bucket. if there are
// not enough tokens in the bucket, it will return false.
func (tb *TokenBucket) TryTake(count int64) bool {
	tb.checkCount(count)

	defer tb.tokenMutex.Unlock()
	tb.tokenMutex.Lock()

	if count <= tb.avail {
		tb.avail -= count

		return true
	}

	return false
}

// Take tasks specified count tokens from the bucket, if there are
// not enough tokens in the bucket, it will keep waiting until count tokens are
// availible and then take them.
func (tb *TokenBucket) Take(count int64) {
	tb.waitAndTake(count, count)
}

// TakeMaxDuration tasks specified count tokens from the bucket, if there are
// not enough tokens in the bucket, it will keep waiting until count tokens are
// availible and then take them or just return false when reach the given max
// duration.
func (tb *TokenBucket) TakeMaxDuration(count int64, max time.Duration) bool {
	return tb.waitAndTakeMaxDuration(count, count, max)
}

// Wait will keep waiting until count tokens are availible in the bucket.
func (tb *TokenBucket) Wait(count int64) {
	tb.waitAndTake(count, 0)
}

// WaitMaxDuration will keep waiting until count tokens are availible in the
// bucket or just return false when reach the given max duration.
func (tb *TokenBucket) WaitMaxDuration(count int64, max time.Duration) bool {
	return tb.waitAndTakeMaxDuration(count, 0, max)
}

func (tb *TokenBucket) waitAndTake(need int64, use int64) {
	tb.checkCount(use)

	w := &waitingJob{
		ch:   make(chan struct{}),
		use:  use,
		need: need,
	}

	defer close(w.ch)

	tb.addWaitingJob(w)

	<-w.ch
}

func (tb *TokenBucket) waitAndTakeMaxDuration(need int64, use int64, max time.Duration) bool {
	tb.checkCount(use)

	w := &waitingJob{
		ch:   make(chan struct{}),
		use:  use,
		need: need,
	}

	defer close(w.ch)

	tb.addWaitingJob(w)

	select {
	case <-w.ch:
		return true
	case <-time.After(max):
		w.abandoned = true
		return false
	}
}

// Destory destorys the token bucket and stop the inner channels.
func (tb *TokenBucket) Destory() {
	tb.ticker.Stop()
}

func (tb *TokenBucket) addWaitingJob(w *waitingJob) {
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
			if waitingJobNow == nil || waitingJobNow.abandoned {
				waitingJobNow = element.Value.(*waitingJob)

				tb.waitingQuqueMutex.Lock()
				tb.waitingQuque.Remove(element)
				tb.waitingQuqueMutex.Unlock()

				if tb.avail >= waitingJobNow.need && !waitingJobNow.abandoned {
					tb.avail -= waitingJobNow.use

					waitingJobNow.ch <- struct{}{}
					waitingJobNow = nil
				}
			}
		}

		tb.tokenMutex.Unlock()
	}
}

func (tb *TokenBucket) checkCount(count int64) {
	if count < 0 || count > tb.cap {
		panic(fmt.Sprintf("token-bucket: count %v should be less than bucket's"+
			" capablity %v", count, tb.cap))
	}
}
