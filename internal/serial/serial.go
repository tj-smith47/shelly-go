// Package serial delivers values to a callback one at a time, in order.
package serial

import "sync"

// Queue hands queued values to a callback so that the callback never runs
// concurrently with itself and sees values in the order they were added.
//
// No goroutine is started. The caller that finds the queue idle delivers its
// own value and any added meanwhile; callers that find a delivery in progress
// return at once and leave their value to it. A callback may therefore add to
// the same queue without deadlocking: the value is delivered after it returns.
//
// The zero value is ready to use.
type Queue[T any] struct {
	pending  []T
	mu       sync.Mutex
	draining bool
}

// Add queues v without delivering it. Call it while holding whatever lock
// defines the order of values, then call Drain after releasing that lock.
func (q *Queue[T]) Add(v T) {
	q.mu.Lock()
	q.pending = append(q.pending, v)
	q.mu.Unlock()
}

// Drain delivers queued values until none remain, unless another caller is
// already doing so.
func (q *Queue[T]) Drain(deliver func(T)) {
	q.mu.Lock()
	if q.draining {
		q.mu.Unlock()
		return
	}
	q.draining = true

	// A panicking callback must not leave the queue marked busy forever. The
	// lock is not held while deliver runs, so the panic path has to retake it.
	delivering := false
	defer func() {
		if delivering {
			q.mu.Lock()
		}
		q.draining = false
		q.mu.Unlock()
	}()

	for len(q.pending) > 0 {
		next := q.pending[0]
		q.pending = q.pending[1:]
		q.mu.Unlock()
		delivering = true
		deliver(next)
		q.mu.Lock()
		delivering = false
	}
}

// Push is Add followed by Drain.
func (q *Queue[T]) Push(v T, deliver func(T)) {
	q.Add(v)
	q.Drain(deliver)
}
