package firmware

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/tj-smith47/shelly-go/rpc"
	"github.com/tj-smith47/shelly-go/transport"
)

func rolloutDevices(n int) []Device {
	client := rpc.NewClient(&mockTransport{
		callFunc: func(context.Context, transport.RPCRequest) (json.RawMessage, error) {
			return jsonrpcResponse(nil)
		},
	})

	devices := make([]Device, n)
	for i := range devices {
		devices[i] = &mockDevice{address: "test", client: client}
	}
	return devices
}

// TestStagedRollout_CancelEndsBatchDelay cancels during a long wait between
// batches. Start must return instead of sitting out the delay.
func TestStagedRollout_CancelEndsBatchDelay(t *testing.T) {
	rollout := NewStagedRollout(rolloutDevices(3), 100, nil)
	rollout.BatchSize = 1
	rollout.DelayBetweenBatches = time.Hour

	firstBatch := make(chan struct{})
	rollout.OnProgress = func(_ Device, _ *UpdateResult, completed, _ int) {
		if completed == 1 {
			close(firstBatch)
		}
	}

	type outcome struct {
		err     error
		results []UpdateResult
	}
	finished := make(chan outcome, 1)
	go func() {
		results, err := rollout.Start(context.Background())
		finished <- outcome{err: err, results: results}
	}()

	select {
	case <-firstBatch:
	case <-time.After(5 * time.Second):
		t.Fatal("first batch never completed")
	}

	rollout.Cancel()
	rollout.Cancel()

	select {
	case got := <-finished:
		if got.err != nil {
			t.Errorf("Start() error = %v, want nil", got.err)
		}
		if len(got.results) != 1 {
			t.Errorf("Start() results = %d, want 1", len(got.results))
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Start still blocked in the batch delay after Cancel")
	}

	if rollout.IsInProgress() {
		t.Error("IsInProgress() = true after a canceled Start returned")
	}
}

// TestStagedRollout_CancelBeforeStart checks that an early Cancel neither
// panics nor stops the rollout that follows.
func TestStagedRollout_CancelBeforeStart(t *testing.T) {
	rollout := NewStagedRollout(rolloutDevices(2), 100, nil)
	rollout.DelayBetweenBatches = 0

	rollout.Cancel()
	rollout.Cancel()

	results, err := rollout.Start(context.Background())
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if len(results) != 2 {
		t.Errorf("Start() results = %d, want 2", len(results))
	}

	// The finished rollout's channel is still open; closing it twice or
	// carrying it into the next Start would break the second run.
	rollout.Cancel()
	results, err = rollout.Start(context.Background())
	if err != nil {
		t.Fatalf("second Start() error = %v", err)
	}
	if len(results) != 2 {
		t.Errorf("second Start() results = %d, want 2", len(results))
	}
}

// TestStagedRollout_NonPositiveBatchSize checks that a zero batch size still
// advances through the devices.
func TestStagedRollout_NonPositiveBatchSize(t *testing.T) {
	rollout := NewStagedRollout(rolloutDevices(2), 100, nil)
	rollout.BatchSize = 0
	rollout.DelayBetweenBatches = 0

	finished := make(chan int, 1)
	go func() {
		results, err := rollout.Start(context.Background())
		if err != nil {
			t.Errorf("Start() error = %v", err)
		}
		finished <- len(results)
	}()

	select {
	case got := <-finished:
		if got != 2 {
			t.Errorf("Start() results = %d, want 2", got)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Start never finished with a batch size of zero")
	}
}
