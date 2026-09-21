package transport

import (
	"sync"
	"testing"
)

// The callbacks below touch unsynchronized state on purpose: the race
// detector fails these tests if a callback ever runs concurrently.

// assertChangeLog fails unless every reported state differs from the one
// before it and the last one matches what State() returns.
func assertChangeLog(t *testing.T, name string, seen []ConnectionState, current ConnectionState) {
	t.Helper()
	if len(seen) == 0 {
		t.Fatalf("%s: no state change was reported", name)
	}
	for i := 1; i < len(seen); i++ {
		if seen[i] == seen[i-1] {
			t.Fatalf("%s: state %v reported twice in a row: %v", name, seen[i], seen)
		}
	}
	if last := seen[len(seen)-1]; last != current {
		t.Errorf("%s: last reported state = %v, but State() = %v", name, last, current)
	}
}

func toggleStates(set func(ConnectionState)) {
	var wg sync.WaitGroup
	for i := range 100 {
		wg.Go(func() {
			if i%2 == 0 {
				set(StateConnected)
			} else {
				set(StateReconnecting)
			}
		})
	}
	wg.Wait()
}

func TestConnState_CallbacksSerializedAndOrdered(t *testing.T) {
	var s connState
	var seen []ConnectionState
	s.OnStateChange(func(st ConnectionState) { seen = append(seen, st) })

	toggleStates(s.setState)

	assertChangeLog(t, "connState", seen, s.State())
}

func TestConnState_UnchangedStateNotReported(t *testing.T) {
	var s connState
	var calls int
	s.OnStateChange(func(ConnectionState) { calls++ })

	s.setState(StateConnected)
	s.setState(StateConnected)

	if calls != 1 {
		t.Errorf("callback ran %d times, want 1", calls)
	}
}

func TestConnState_CallbackMayChangeState(t *testing.T) {
	var s connState
	var seen []ConnectionState
	s.OnStateChange(func(st ConnectionState) {
		seen = append(seen, st)
		if st == StateConnected {
			s.setState(StateClosed)
		}
	})

	s.setState(StateConnected)

	if len(seen) != 2 || seen[0] != StateConnected || seen[1] != StateClosed {
		t.Fatalf("seen = %v, want [connected closed]", seen)
	}
	if s.State() != StateClosed {
		t.Errorf("State() = %v, want closed", s.State())
	}
}

// Every stateful transport must share the ordered implementation.
func TestStatefulTransportsEmbedConnState(t *testing.T) {
	type stateful interface {
		setState(ConnectionState)
		OnStateChange(func(ConnectionState))
		State() ConnectionState
	}
	for name, tr := range map[string]stateful{
		"websocket": &WebSocket{},
		"mqtt":      &MQTT{},
		"coap":      &CoAP{},
	} {
		var seen []ConnectionState
		tr.OnStateChange(func(st ConnectionState) { seen = append(seen, st) })

		toggleStates(tr.setState)

		assertChangeLog(t, name, seen, tr.State())
	}
}

func TestConnState_NothingFollowsClosed(t *testing.T) {
	var s connState
	var log []ConnectionState
	s.OnStateChange(func(state ConnectionState) { log = append(log, state) })

	s.setState(StateConnected)
	s.setState(StateClosed)
	s.setState(StateReconnecting)
	s.setState(StateDisconnected)

	if got := s.State(); got != StateClosed {
		t.Errorf("State() = %v, want closed", got)
	}
	if len(log) != 2 || log[1] != StateClosed {
		t.Errorf("reported %v, want [connected closed]", log)
	}
}
