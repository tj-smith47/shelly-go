package serial

import (
	"sync"
	"testing"
)

// The callbacks below touch unsynchronized state on purpose: the race
// detector fails these tests if Queue ever runs a callback concurrently.

func TestQueue_SerializesConcurrentPushes(t *testing.T) {
	var q Queue[int]
	var got []int
	deliver := func(v int) { got = append(got, v) }

	const n = 200
	var wg sync.WaitGroup
	for i := range n {
		wg.Go(func() { q.Push(i, deliver) })
	}
	wg.Wait()

	if len(got) != n {
		t.Fatalf("delivered %d values, want %d", len(got), n)
	}
}

func TestQueue_PreservesAddOrder(t *testing.T) {
	var q Queue[int]
	for i := range 10 {
		q.Add(i)
	}
	var got []int
	q.Drain(func(v int) { got = append(got, v) })

	for i, v := range got {
		if v != i {
			t.Fatalf("order = %v, want 0..9", got)
		}
	}
}

func TestQueue_ReentrantPushDeliversAfterCallbackReturns(t *testing.T) {
	var q Queue[string]
	var got []string
	var deliver func(string)
	deliver = func(v string) {
		got = append(got, v+":start")
		if v == "outer" {
			q.Push("inner", deliver)
		}
		got = append(got, v+":end")
	}

	q.Push("outer", deliver)

	want := []string{"outer:start", "outer:end", "inner:start", "inner:end"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}

func TestQueue_RecoversAfterPanic(t *testing.T) {
	var q Queue[int]
	func() {
		defer func() { _ = recover() }()
		q.Push(1, func(int) { panic("boom") })
	}()

	var got []int
	q.Push(2, func(v int) { got = append(got, v) })
	if len(got) != 1 || got[0] != 2 {
		t.Fatalf("after a panic, got %v, want [2]", got)
	}
}
