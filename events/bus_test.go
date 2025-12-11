package events

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewEventBus(t *testing.T) {
	bus := NewEventBus()
	defer bus.Close()

	if bus == nil {
		t.Fatal("NewEventBus() returned nil")
	}
	if bus.IsClosed() {
		t.Error("new bus should not be closed")
	}
	if bus.SubscriberCount() != 0 {
		t.Errorf("SubscriberCount() = %v, want 0", bus.SubscriberCount())
	}
}

func TestEventBus_WithHistorySize(t *testing.T) {
	bus := NewEventBus(WithHistorySize(10))
	defer bus.Close()

	if bus.historySize != 10 {
		t.Errorf("historySize = %v, want 10", bus.historySize)
	}
	if bus.history == nil {
		t.Error("history should not be nil when size > 0")
	}
}

func TestEventBus_Subscribe(t *testing.T) {
	bus := NewEventBus()
	defer bus.Close()

	var received Event
	id := bus.Subscribe(func(e Event) {
		received = e
	})

	if id == 0 {
		t.Error("Subscribe() returned 0")
	}
	if bus.SubscriberCount() != 1 {
		t.Errorf("SubscriberCount() = %v, want 1", bus.SubscriberCount())
	}

	event := NewDeviceOnlineEvent("device1")
	bus.Publish(event)

	if received == nil {
		t.Error("handler was not called")
	}
	if received.DeviceID() != "device1" {
		t.Errorf("DeviceID() = %v, want device1", received.DeviceID())
	}
}

func TestEventBus_SubscribeFiltered(t *testing.T) {
	bus := NewEventBus()
	defer bus.Close()

	var count int
	bus.SubscribeFiltered(WithDeviceID("device1"), func(e Event) {
		count++
	})

	bus.Publish(NewDeviceOnlineEvent("device1"))
	bus.Publish(NewDeviceOnlineEvent("device2"))
	bus.Publish(NewDeviceOnlineEvent("device1"))

	if count != 2 {
		t.Errorf("handler called %v times, want 2", count)
	}
}

func TestEventBus_Unsubscribe(t *testing.T) {
	bus := NewEventBus()
	defer bus.Close()

	var count int
	id := bus.Subscribe(func(e Event) {
		count++
	})

	bus.Publish(NewDeviceOnlineEvent("device1"))
	if count != 1 {
		t.Errorf("count = %v, want 1", count)
	}

	if !bus.Unsubscribe(id) {
		t.Error("Unsubscribe() returned false")
	}
	if bus.SubscriberCount() != 0 {
		t.Errorf("SubscriberCount() = %v, want 0", bus.SubscriberCount())
	}

	bus.Publish(NewDeviceOnlineEvent("device1"))
	if count != 1 {
		t.Errorf("count = %v after unsubscribe, want 1", count)
	}
}

func TestEventBus_Unsubscribe_NotFound(t *testing.T) {
	bus := NewEventBus()
	defer bus.Close()

	if bus.Unsubscribe(999) {
		t.Error("Unsubscribe() should return false for non-existent ID")
	}
}

func TestEventBus_MultipleSubscribers(t *testing.T) {
	bus := NewEventBus()
	defer bus.Close()

	var count1, count2 int
	bus.Subscribe(func(e Event) { count1++ })
	bus.Subscribe(func(e Event) { count2++ })

	bus.Publish(NewDeviceOnlineEvent("device1"))

	if count1 != 1 {
		t.Errorf("count1 = %v, want 1", count1)
	}
	if count2 != 1 {
		t.Errorf("count2 = %v, want 1", count2)
	}
}

func TestEventBus_Publish_Closed(t *testing.T) {
	bus := NewEventBus()

	var called bool
	bus.Subscribe(func(e Event) {
		called = true
	})

	bus.Close()
	bus.Publish(NewDeviceOnlineEvent("device1"))

	if called {
		t.Error("handler should not be called after Close()")
	}
}

func TestEventBus_Subscribe_Closed(t *testing.T) {
	bus := NewEventBus()
	bus.Close()

	id := bus.Subscribe(func(e Event) {})
	if id != 0 {
		t.Errorf("Subscribe() on closed bus should return 0, got %v", id)
	}
}

func TestEventBus_PublishAsync(t *testing.T) {
	bus := NewEventBus()
	defer bus.Close()

	var count atomic.Int32
	var wg sync.WaitGroup

	for i := 0; i < 3; i++ {
		bus.Subscribe(func(e Event) {
			time.Sleep(10 * time.Millisecond)
			count.Add(1)
		})
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		bus.PublishAsync(NewDeviceOnlineEvent("device1"))
	}()

	wg.Wait()
	if count.Load() != 3 {
		t.Errorf("count = %v, want 3", count.Load())
	}
}

func TestEventBus_PublishAsync_Closed(t *testing.T) {
	bus := NewEventBus()

	var called atomic.Bool
	bus.Subscribe(func(e Event) {
		called.Store(true)
	})

	bus.Close()
	bus.PublishAsync(NewDeviceOnlineEvent("device1"))

	time.Sleep(50 * time.Millisecond)
	if called.Load() {
		t.Error("handler should not be called after Close()")
	}
}

func TestEventBus_History(t *testing.T) {
	bus := NewEventBus(WithHistorySize(3))
	defer bus.Close()

	bus.Publish(NewDeviceOnlineEvent("device1"))
	bus.Publish(NewDeviceOnlineEvent("device2"))
	bus.Publish(NewDeviceOnlineEvent("device3"))

	history := bus.History()
	if len(history) != 3 {
		t.Errorf("len(History()) = %v, want 3", len(history))
	}

	// Add one more, oldest should be dropped
	bus.Publish(NewDeviceOnlineEvent("device4"))

	history = bus.History()
	if len(history) != 3 {
		t.Errorf("len(History()) = %v, want 3", len(history))
	}
	if history[0].DeviceID() != "device2" {
		t.Errorf("oldest event should be device2, got %v", history[0].DeviceID())
	}
	if history[2].DeviceID() != "device4" {
		t.Errorf("newest event should be device4, got %v", history[2].DeviceID())
	}
}

func TestEventBus_History_Disabled(t *testing.T) {
	bus := NewEventBus() // No history
	defer bus.Close()

	bus.Publish(NewDeviceOnlineEvent("device1"))

	history := bus.History()
	if history != nil {
		t.Errorf("History() should be nil when disabled, got %v", history)
	}
}

func TestEventBus_ClearHistory(t *testing.T) {
	bus := NewEventBus(WithHistorySize(10))
	defer bus.Close()

	bus.Publish(NewDeviceOnlineEvent("device1"))
	bus.Publish(NewDeviceOnlineEvent("device2"))

	if len(bus.History()) != 2 {
		t.Errorf("len(History()) = %v, want 2", len(bus.History()))
	}

	bus.ClearHistory()

	if len(bus.History()) != 0 {
		t.Errorf("len(History()) after clear = %v, want 0", len(bus.History()))
	}
}

func TestEventBus_ClearHistory_Disabled(t *testing.T) {
	bus := NewEventBus() // No history
	defer bus.Close()

	// Should not panic
	bus.ClearHistory()
}

func TestEventBus_Close_Idempotent(t *testing.T) {
	bus := NewEventBus()

	// Should not panic
	bus.Close()
	bus.Close()

	if !bus.IsClosed() {
		t.Error("bus should be closed")
	}
}

func TestEventBus_Concurrency(t *testing.T) {
	bus := NewEventBus()
	defer bus.Close()

	var count atomic.Int32

	// Subscribe concurrently
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			bus.Subscribe(func(e Event) {
				count.Add(1)
			})
		}()
	}
	wg.Wait()

	if bus.SubscriberCount() != 10 {
		t.Errorf("SubscriberCount() = %v, want 10", bus.SubscriberCount())
	}

	// Publish concurrently
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			bus.Publish(NewDeviceOnlineEvent("device1"))
		}()
	}
	wg.Wait()

	// Each of 10 subscribers should receive 10 events
	expected := int32(100)
	if count.Load() != expected {
		t.Errorf("count = %v, want %v", count.Load(), expected)
	}
}

func TestEventBus_SubscribeAndUnsubscribeConcurrently(t *testing.T) {
	bus := NewEventBus()
	defer bus.Close()

	var wg sync.WaitGroup
	ids := make(chan uint64, 100)

	// Subscribe many
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			id := bus.Subscribe(func(e Event) {})
			ids <- id
		}()
	}

	// Wait for all subscriptions
	wg.Wait()
	close(ids)

	// Collect IDs
	allIDs := make([]uint64, 0, 50)
	for id := range ids {
		allIDs = append(allIDs, id)
	}

	// Unsubscribe half concurrently
	for i := 0; i < 25; i++ {
		wg.Add(1)
		go func(id uint64) {
			defer wg.Done()
			bus.Unsubscribe(id)
		}(allIDs[i])
	}
	wg.Wait()

	if bus.SubscriberCount() != 25 {
		t.Errorf("SubscriberCount() = %v, want 25", bus.SubscriberCount())
	}
}

func TestEventBus_PublishWithAsyncHistory(t *testing.T) {
	bus := NewEventBus(WithHistorySize(5))
	defer bus.Close()

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			bus.PublishAsync(NewDeviceOnlineEvent("device1"))
		}()
	}
	wg.Wait()

	// History should have at most 5 events
	history := bus.History()
	if len(history) > 5 {
		t.Errorf("len(History()) = %v, want <= 5", len(history))
	}
}
