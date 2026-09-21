package events

import (
	"sync"
	"sync/atomic"

	"github.com/tj-smith47/shelly-go/internal/serial"
)

// Handler is a function that handles events.
type Handler func(Event)

// Subscription represents an active event subscription.
type Subscription struct {
	handler Handler
	filter  Filter
	queue   serial.Queue[Event]
	id      uint64
}

// EventBus is a thread-safe event dispatcher.
type EventBus struct {
	subscriptions []*Subscription
	history       []Event
	nextID        atomic.Uint64
	historySize   int
	mu            sync.RWMutex
	historyMu     sync.RWMutex
	closed        atomic.Bool
}

// EventBusOption configures the event bus.
type EventBusOption func(*EventBus)

// WithHistorySize sets the event history size.
// Setting to 0 disables history (default).
func WithHistorySize(size int) EventBusOption {
	return func(bus *EventBus) {
		bus.historySize = size
		if size > 0 {
			bus.history = make([]Event, 0, size)
		}
	}
}

// NewEventBus creates a new event bus.
func NewEventBus(opts ...EventBusOption) *EventBus {
	bus := &EventBus{
		subscriptions: make([]*Subscription, 0),
	}
	for _, opt := range opts {
		opt(bus)
	}
	return bus
}

// Subscribe registers a handler for all events.
// Returns a subscription ID that can be used to unsubscribe.
func (bus *EventBus) Subscribe(handler Handler) uint64 {
	return bus.SubscribeFiltered(nil, handler)
}

// SubscribeFiltered registers a handler with a filter.
// The handler will only receive events that match the filter.
// Returns a subscription ID that can be used to unsubscribe.
func (bus *EventBus) SubscribeFiltered(filter Filter, handler Handler) uint64 {
	if bus.closed.Load() {
		return 0
	}

	id := bus.nextID.Add(1)
	sub := &Subscription{
		id:      id,
		handler: handler,
		filter:  filter,
	}

	bus.mu.Lock()
	bus.subscriptions = append(bus.subscriptions, sub)
	bus.mu.Unlock()

	return id
}

// Unsubscribe removes a subscription by ID.
// Returns true if the subscription was found and removed.
func (bus *EventBus) Unsubscribe(id uint64) bool {
	bus.mu.Lock()
	defer bus.mu.Unlock()

	for i, sub := range bus.subscriptions {
		if sub.id == id {
			// Remove subscription by swapping with last element
			bus.subscriptions[i] = bus.subscriptions[len(bus.subscriptions)-1]
			bus.subscriptions = bus.subscriptions[:len(bus.subscriptions)-1]
			return true
		}
	}
	return false
}

// Publish dispatches an event to all matching subscribers.
// Events are delivered synchronously in subscription order.
//
// A subscription's filter and handler never run concurrently with themselves,
// whichever mix of Publish and PublishAsync callers is active, so they need no
// locking of their own. When another goroutine is already delivering to a
// subscription, or a handler publishes from inside itself, the event is queued
// behind the one in progress and delivered in order.
func (bus *EventBus) Publish(event Event) {
	for _, sub := range bus.record(event) {
		sub.deliver(event)
	}
}

// PublishAsync dispatches an event asynchronously, one goroutine per
// subscriber, and returns once every subscriber has been handed the event.
// The serialization described on Publish applies here too.
func (bus *EventBus) PublishAsync(event Event) {
	var wg sync.WaitGroup
	for _, sub := range bus.record(event) {
		wg.Go(func() { sub.deliver(event) })
	}
	wg.Wait()
}

// record adds the event to the history and returns the subscriptions to
// offer it to. It returns nil once the bus is closed.
func (bus *EventBus) record(event Event) []*Subscription {
	if bus.closed.Load() {
		return nil
	}

	if bus.historySize > 0 {
		bus.historyMu.Lock()
		if len(bus.history) >= bus.historySize {
			copy(bus.history, bus.history[1:])
			bus.history = bus.history[:len(bus.history)-1]
		}
		bus.history = append(bus.history, event)
		bus.historyMu.Unlock()
	}

	bus.mu.RLock()
	defer bus.mu.RUnlock()
	subs := make([]*Subscription, len(bus.subscriptions))
	copy(subs, bus.subscriptions)
	return subs
}

// deliver offers the event to the subscription, in order and one at a time.
func (s *Subscription) deliver(event Event) {
	s.queue.Push(event, func(e Event) {
		if s.filter == nil || s.filter(e) {
			s.handler(e)
		}
	})
}

// SubscriberCount returns the number of active subscriptions.
func (bus *EventBus) SubscriberCount() int {
	bus.mu.RLock()
	defer bus.mu.RUnlock()
	return len(bus.subscriptions)
}

// History returns the event history.
// Returns nil if history is disabled.
func (bus *EventBus) History() []Event {
	if bus.historySize == 0 {
		return nil
	}

	bus.historyMu.RLock()
	defer bus.historyMu.RUnlock()

	result := make([]Event, len(bus.history))
	copy(result, bus.history)
	return result
}

// ClearHistory clears the event history.
func (bus *EventBus) ClearHistory() {
	if bus.historySize == 0 {
		return
	}

	bus.historyMu.Lock()
	bus.history = bus.history[:0]
	bus.historyMu.Unlock()
}

// Close closes the event bus and removes all subscriptions.
// After closing, Publish and Subscribe operations are no-ops.
func (bus *EventBus) Close() {
	if bus.closed.Swap(true) {
		return // Already closed
	}

	bus.mu.Lock()
	bus.subscriptions = nil
	bus.mu.Unlock()

	bus.historyMu.Lock()
	bus.history = nil
	bus.historyMu.Unlock()
}

// IsClosed returns true if the bus has been closed.
func (bus *EventBus) IsClosed() bool {
	return bus.closed.Load()
}
