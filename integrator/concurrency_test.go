package integrator

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"
)

// reentryTimeout bounds how long a test waits for a call that deadlocks when a
// callback is run with a lock held.
const reentryTimeout = 5 * time.Second

// runOrFail runs fn on its own goroutine and fails the test if it has not
// returned within reentryTimeout.
func runOrFail(t *testing.T, what string, fn func()) {
	t.Helper()

	done := make(chan struct{})
	go func() {
		defer close(done)
		fn()
	}()

	select {
	case <-done:
	case <-time.After(reentryTimeout):
		t.Fatalf("%s did not return: a callback that calls back in deadlocked", what)
	}
}

func TestAccountManager_CallbacksMayCallBackIn(t *testing.T) {
	am := NewAccountManager()

	var calls int
	readBack := func() {
		calls++
		am.GetAccount("user1")
		am.ListDevices()
		am.GetStats()
		if _, err := am.ToJSON(); err != nil {
			t.Errorf("ToJSON() error = %v", err)
		}
	}

	am.OnAccountAdded(func(account *Account) {
		readBack()
		if _, ok := am.GetAccount(account.UserID); !ok {
			t.Errorf("account %s not visible in OnAccountAdded", account.UserID)
		}
		// A change made from a callback is announced after this one returns.
		if err := am.AddDevice(account.UserID, &AccountDevice{DeviceID: "dev-from-callback"}); err != nil {
			t.Errorf("AddDevice() error = %v", err)
		}
	})
	am.OnDeviceAdded(func(string, *AccountDevice) { readBack() })
	am.OnDeviceRemoved(func(string, string) {
		readBack()
		am.OnDeviceRemoved(nil)
	})
	am.OnAccountRemoved(func(string) { readBack() })

	runOrFail(t, "AccountManager change", func() {
		am.AddAccount(&Account{UserID: "user1"})
		if err := am.ProcessCallback(&DeviceCallback{UserID: "user1", DeviceID: "dev1", Action: ActionAdd}); err != nil {
			t.Errorf("ProcessCallback(add) error = %v", err)
		}
		if err := am.ProcessCallback(&DeviceCallback{UserID: "user1", DeviceID: "dev1", Action: ActionRemove}); err != nil {
			t.Errorf("ProcessCallback(remove) error = %v", err)
		}
		am.RemoveAccount("user1")
	})

	// account added, device added from the callback, dev1 added, dev1 removed,
	// account removed.
	if calls != 5 {
		t.Errorf("callbacks ran %d times, want 5", calls)
	}
}

func TestAccountManager_CallbacksNeverOverlapAndKeepOrder(t *testing.T) {
	const workers = 32

	am := NewAccountManager()

	// Deliberately unsynchronized: the race detector reports any two callbacks
	// that run at the same time.
	var accountsAdded, accountsRemoved, devicesAdded, devicesRemoved int
	present := make(map[string]bool)

	am.OnAccountAdded(func(account *Account) {
		accountsAdded++
		present[account.UserID] = true
	})
	am.OnDeviceAdded(func(userID string, device *AccountDevice) {
		devicesAdded++
		if !present[userID] {
			t.Errorf("device %s announced before its account %s", device.DeviceID, userID)
		}
		present[device.DeviceID] = true
	})
	am.OnDeviceRemoved(func(_, deviceID string) {
		devicesRemoved++
		if !present[deviceID] {
			t.Errorf("device %s removal announced before its addition", deviceID)
		}
		delete(present, deviceID)
	})
	am.OnAccountRemoved(func(userID string) {
		accountsRemoved++
		if !present[userID] {
			t.Errorf("account %s removal announced before its addition", userID)
		}
		delete(present, userID)
	})

	var wg sync.WaitGroup
	for i := range workers {
		wg.Go(func() {
			userID := fmt.Sprintf("user%d", i)
			deviceID := fmt.Sprintf("dev%d", i)

			am.AddAccount(&Account{UserID: userID})
			if err := am.AddDevice(userID, &AccountDevice{DeviceID: deviceID}); err != nil {
				t.Errorf("AddDevice() error = %v", err)
			}
			if !am.RemoveDevice(userID, deviceID) {
				t.Errorf("RemoveDevice(%s) = false", deviceID)
			}
			if !am.RemoveAccount(userID) {
				t.Errorf("RemoveAccount(%s) = false", userID)
			}
		})
	}
	wg.Wait()

	for name, got := range map[string]int{
		"OnAccountAdded":   accountsAdded,
		"OnAccountRemoved": accountsRemoved,
		"OnDeviceAdded":    devicesAdded,
		"OnDeviceRemoved":  devicesRemoved,
	} {
		if got != workers {
			t.Errorf("%s ran %d times, want %d", name, got, workers)
		}
	}
	if len(present) != 0 {
		t.Errorf("%d entries left after every removal was announced", len(present))
	}
}

func TestConnection_HandlersMayCallBackIn(t *testing.T) {
	tests := []struct {
		name     string
		message  string
		register func(c *Connection, called chan<- struct{})
	}{
		{
			name:    "OnError",
			message: "not json",
			register: func(c *Connection, called chan<- struct{}) {
				c.OnError(func(error) {
					c.OnError(nil)
					if err := c.Close(); err != nil {
						t.Errorf("Close() error = %v", err)
					}
					close(called)
				})
			},
		},
		{
			name:    "OnRawMessage",
			message: `{"event":"Shelly:Online","deviceId":"dev1","online":1}`,
			register: func(c *Connection, called chan<- struct{}) {
				c.OnRawMessage(func(*WSMessage) {
					c.OnRawMessage(nil)
					if err := c.Close(); err != nil {
						t.Errorf("Close() error = %v", err)
					}
					close(called)
				})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conn := &Connection{ws: &mockWSConnector{}, closeCh: make(chan struct{})}
			called := make(chan struct{})
			tt.register(conn, called)

			runOrFail(t, "handleMessage", func() { conn.handleMessage([]byte(tt.message)) })

			select {
			case <-called:
			default:
				t.Fatal("handler was not called")
			}
			if !conn.IsClosed() {
				t.Error("connection not closed by the handler")
			}
		})
	}
}

func TestFleetManager_SetAccountManagerRacesReaders(t *testing.T) {
	const rounds = 200

	fm := NewFleetManager(New("tag", "token"))
	ctx := context.Background()

	newManager := func() *AccountManager {
		am := NewAccountManager()
		am.AddAccount(&Account{UserID: "user1", Devices: []AccountDevice{
			{DeviceID: "dev1", DeviceType: deviceTypeSHSW1, AccessGroups: "01", Host: "host1"},
		}})
		return am
	}

	readers := []func(){
		func() { fm.getUniqueHosts() },
		func() { fm.handleOnlineStatus(&OnlineStatusEvent{DeviceID: "dev1", Online: true}) },
		func() {
			if err := fm.SendCommand(ctx, "dev1", actionRelay, nil); err == nil {
				t.Error("SendCommand() succeeded without a connection")
			}
		},
		func() { fm.AllRelaysOn(ctx) },
		func() { fm.AllRelaysOff(ctx) },
		func() { fm.GetStats() },
	}

	var wg sync.WaitGroup
	wg.Go(func() {
		for range rounds {
			fm.SetAccountManager(newManager())
		}
	})
	for _, read := range readers {
		wg.Go(func() {
			for range rounds {
				read()
			}
		})
	}
	wg.Wait()
}

func TestFleetManager_ToJSONRacesWriters(t *testing.T) {
	const rounds = 500

	fm := NewFleetManager(New("tag", "token"))

	runOrFail(t, "ToJSON alongside writers", func() {
		var wg sync.WaitGroup
		wg.Go(func() {
			for i := range rounds {
				fm.CreateGroup(fmt.Sprintf("g%d", i%4), "group", nil)
			}
		})
		wg.Go(func() {
			for range rounds {
				fm.handleStatusChange(&StatusChangeEvent{DeviceID: "dev1"})
				fm.handleOnlineStatus(&OnlineStatusEvent{DeviceID: "dev1", Online: true})
			}
		})
		wg.Go(func() {
			for range rounds {
				if _, err := fm.ToJSON(); err != nil {
					t.Errorf("ToJSON() error = %v", err)
				}
			}
		})
		wg.Wait()
	})
}

func TestTokenManager_RestartAutoRefresh(t *testing.T) {
	tm := NewTokenManager(nil)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	for range 200 {
		tm.StartAutoRefresh(ctx)
		tm.StopAutoRefresh()
	}
}

// TestTokenManager_RestartAfterContextEnds covers a refresh loop ended by its
// context instead of StopAutoRefresh: a later start must run a new loop.
func TestTokenManager_RestartAfterContextEnds(t *testing.T) {
	tm := NewTokenManager(nil)

	ctx, cancel := context.WithCancel(context.Background())
	tm.StartAutoRefresh(ctx)
	tm.mu.RLock()
	first := tm.refreshDone
	tm.mu.RUnlock()
	cancel()

	ctx2, cancel2 := context.WithCancel(context.Background())
	defer cancel2()
	tm.StartAutoRefresh(ctx2)
	defer tm.StopAutoRefresh()

	tm.mu.RLock()
	second, running := tm.refreshDone, tm.refreshRunning
	tm.mu.RUnlock()
	if second == first || !running {
		t.Fatalf("StartAutoRefresh after the context ended did not start a new loop (running = %v)", running)
	}

	// The first loop, ending late, must leave the second loop's mark alone.
	tm.autoRefreshLoop(ctx, first)
	tm.mu.RLock()
	running = tm.refreshRunning
	tm.mu.RUnlock()
	if !running {
		t.Error("an old loop cleared the running mark of the loop that replaced it")
	}
}

// TestTokenManager_LoopExitClearsRunning checks that a loop ended by its
// context marks the manager as not running.
func TestTokenManager_LoopExitClearsRunning(t *testing.T) {
	tm := NewTokenManager(nil)
	done := make(chan struct{})
	tm.refreshDone = done
	tm.refreshRunning = true

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	tm.autoRefreshLoop(ctx, done)

	tm.mu.RLock()
	running := tm.refreshRunning
	tm.mu.RUnlock()
	if running {
		t.Error("refreshRunning stayed true after the loop ended")
	}
}

// TestConnection_SendCommandWritesOneAtATime relies on the race detector: the
// connection's write methods mutate unlocked state.
func TestConnection_SendCommandWritesOneAtATime(t *testing.T) {
	var writes int
	mock := &mockWSConnector{
		setWriteDeadline: func(time.Time) error { writes++; return nil },
		writeFunc:        func(int, []byte) error { writes++; return nil },
	}
	conn := &Connection{ws: mock, closeCh: make(chan struct{})}

	const senders = 20
	var wg sync.WaitGroup
	for range senders {
		wg.Go(func() {
			if err := conn.SendCommand(context.Background(), "dev1", actionRelay, nil); err != nil {
				t.Errorf("SendCommand() error = %v", err)
			}
		})
	}
	wg.Wait()

	if writes != 2*senders {
		t.Errorf("write calls = %d, want %d", writes, 2*senders)
	}
}

// TestProvisioningManager_DelayEndsWithContext cancels the context while a
// template action's delay is running; the task must not sit out the delay.
func TestProvisioningManager_DelayEndsWithContext(t *testing.T) {
	fm := NewFleetManager(New("tag", "token"))
	pm := NewProvisioningManager(fm)

	if err := fm.accounts.AddDevice("user1", &AccountDevice{
		DeviceID: "dev1", DeviceType: "SHSW-1", AccessGroups: "01", Host: "host1",
	}); err != nil {
		t.Fatalf("AddDevice() error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	fm.connections["host1"] = &Connection{
		ws:      &mockWSConnector{writeFunc: func(int, []byte) error { cancel(); return nil }},
		closeCh: make(chan struct{}),
		host:    "host1",
	}

	template := pm.CreateTemplate("t1", "T", nil)
	template.DeviceTypes = []string{"SHSW-1"}
	template.Actions = []TemplateAction{{Type: actionRelay, DelayAfter: time.Hour}}
	if _, err := pm.CreateTask("task1", "T1", "t1", []string{"dev1"}); err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}

	runOrFail(t, "ExecuteTask with a canceled context", func() {
		// The outcome of the task is not the point; returning is.
		if err := pm.ExecuteTask(ctx, "task1"); err != nil && !errors.Is(err, context.Canceled) {
			t.Errorf("ExecuteTask() error = %v", err)
		}
	})
}
