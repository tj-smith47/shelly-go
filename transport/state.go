package transport

import (
	"sync"

	"github.com/tj-smith47/shelly-go/internal/serial"
)

// connState holds a transport's connection state and its listeners.
//
// Listeners are told about every change in the order the changes happened and
// never concurrently, even when the state moves on several goroutines at once
// (Connect, Close, a background reconnect). A listener may call back into the
// transport, including Connect and Close; the change that causes is reported
// after the listener returns.
type connState struct {
	callbacks []func(ConnectionState)
	changes   serial.Queue[ConnectionState]
	mu        sync.RWMutex
	current   ConnectionState
}

// State returns the current connection state.
func (s *connState) State() ConnectionState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.current
}

// OnStateChange registers a callback for connection state changes.
//
// Callbacks receive every change in order and are never run concurrently, so
// they need no locking of their own. A callback may call Connect or Close on
// the transport; the resulting change is delivered after it returns.
func (s *connState) OnStateChange(callback func(ConnectionState)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.callbacks = append(s.callbacks, callback)
}

// setState records the new state and notifies the listeners. Callers holding
// a lock that a listener could need use queueState, then flushState once the
// lock is released.
func (s *connState) setState(state ConnectionState) {
	s.queueState(state)
	s.flushState()
}

// queueState records the new state without notifying anyone. Setting the
// state it already has is not a change and is not reported, and nothing
// follows StateClosed.
func (s *connState) queueState(state ConnectionState) {
	// Queued under the same lock that orders the writes, so listeners cannot
	// see two changes in the opposite order to the one they were applied in.
	s.mu.Lock()
	defer s.mu.Unlock()
	// Closed is final: a reconnect or dial that was still running when Close
	// returned must not report anything after it.
	if s.current == state || s.current == StateClosed {
		return
	}
	s.current = state
	s.changes.Add(state)
}

// flushState reports queued changes to the listeners.
func (s *connState) flushState() {
	s.changes.Drain(s.notify)
}

func (s *connState) notify(state ConnectionState) {
	s.mu.RLock()
	callbacks := make([]func(ConnectionState), len(s.callbacks))
	copy(callbacks, s.callbacks)
	s.mu.RUnlock()

	for _, cb := range callbacks {
		cb(state)
	}
}
