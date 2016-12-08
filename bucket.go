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
func (tb *TokenBucket) Capability() int64 {
	return tb.cap
}

// Availible returns how many tokens are availible in the bucket.
func (tb *TokenBucket) Availible() int64 {
	tb.tokenMutex.Lock()
	defer tb.tokenMutex.Unlock()

	return tb.avail
}

// TryTake trys to task specified count tokens from the bucket. if there are
// not enough tokens in the bucket, it will return false.
func (tb *TokenBucket) TryTake(count int64) bool {
	return tb.tryTake(count, count)
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

func (tb *TokenBucket) tryTake(need, use int64) bool {
	tb.checkCount(use)

	tb.tokenMutex.Lock()
	defer tb.tokenMutex.Unlock()

	if need <= tb.avail {
		tb.avail -= use

		return true
	}

	return false
}

func (tb *TokenBucket) waitAndTake(need, use int64) {
	if ok := tb.tryTake(need, use); ok {
		return
	}

	w := &waitingJob{
		ch:   make(chan struct{}),
		use:  use,
		need: need,
	}

	tb.addWaitingJob(w)

	<-w.ch
	tb.avail -= use
	w.ch <- struct{}{}

	close(w.ch)
}

func (tb *TokenBucket) waitAndTakeMaxDuration(need, use int64, max time.Duration) bool {
	if ok := tb.tryTake(need, use); ok {
		return true
	}

	w := &waitingJob{
		ch:   make(chan struct{}),
		use:  use,
		need: need,
	}

	defer close(w.ch)

	tb.addWaitingJob(w)

	select {
	case <-w.ch:
		tb.avail -= use
		w.ch <- struct{}{}
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

func (tb *TokenBucket) adjustDaemon() {
	var waitingJobNow *waitingJob

	for now := range tb.ticker.C {
		var _ = now

		tb.tokenMutex.Lock()

		if tb.avail < tb.cap {
			tb.avail++
		}

		element := tb.getFrontWaitingJob()

		if element != nil {
			if waitingJobNow == nil || waitingJobNow.abandoned {
				waitingJobNow = element.Value.(*waitingJob)

				tb.removeWaitingJob(element)
			}

			if tb.avail >= waitingJobNow.need && !waitingJobNow.abandoned {
				waitingJobNow.ch <- struct{}{}
				<-waitingJobNow.ch

				waitingJobNow = nil
			}
		}

		tb.tokenMutex.Unlock()
	}
}

func (tb *TokenBucket) addWaitingJob(w *waitingJob) {
	tb.waitingQuqueMutex.Lock()
	tb.waitingQuque.PushBack(w)
	tb.waitingQuqueMutex.Unlock()
}

func (tb *TokenBucket) getFrontWaitingJob() *list.Element {
	tb.waitingQuqueMutex.Lock()
	e := tb.waitingQuque.Front()
	tb.waitingQuqueMutex.Unlock()

	return e
}

func (tb *TokenBucket) removeWaitingJob(e *list.Element) {
	tb.waitingQuqueMutex.Lock()
	tb.waitingQuque.Remove(e)
	tb.waitingQuqueMutex.Unlock()
}

func (tb *TokenBucket) checkCount(count int64) {
	if count < 0 || count > tb.cap {
		panic(fmt.Sprintf("token-bucket: count %v should be less than bucket's"+
			" capablity %v", count, tb.cap))
	}
}
