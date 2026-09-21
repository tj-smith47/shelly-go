package gen1

import (
	"fmt"
	"net"
	"testing"
	"time"
)

// handleMessage parses one message and delivers it on the calling goroutine.
func (l *CoIoTListener) handleMessage(data []byte, sourceAddr string) {
	if l.enqueue(data, sourceAddr) {
		l.deliver()
	}
}

// TestReceiveLoop_EndsAfterRestart restarts the listener inside the old
// loop's read deadline. The old loop must end even though the listener's
// conn and stop channel now belong to the new run.
func TestReceiveLoop_EndsAfterRestart(t *testing.T) {
	oldConn, _ := openLocalUDPPair(t)
	newConn, _ := openLocalUDPPair(t)

	listener := NewCoIoTListener()
	stop := make(chan struct{})
	ended := make(chan struct{})
	go func() {
		defer close(ended)
		listener.receiveLoop(oldConn, stop, 1500)
	}()

	listener.listenFn = func(*net.UDPAddr) (*net.UDPConn, error) { return newConn, nil }
	if err := listener.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() {
		if err := listener.Stop(); err != nil {
			t.Errorf("Stop: %v", err)
		}
	})

	close(stop)
	if err := oldConn.Close(); err != nil {
		t.Fatalf("close old conn: %v", err)
	}

	select {
	case <-ended:
	case <-time.After(5 * time.Second):
		t.Fatal("old receive loop still running after a restart")
	}
}

// TestStartStop_RepeatedRestart relies on the race detector: a loop that read
// the listener's conn or stop channel from the struct would race with Start.
func TestStartStop_RepeatedRestart(t *testing.T) {
	listener := NewCoIoTListener()
	listener.listenFn = func(*net.UDPAddr) (*net.UDPConn, error) {
		conn, _ := openLocalUDPPair(t)
		return conn, nil
	}

	for range 20 {
		if err := listener.Start(); err != nil {
			t.Fatalf("Start: %v", err)
		}
		if err := listener.Stop(); err != nil {
			t.Fatalf("Stop: %v", err)
		}
	}
}

// TestReceiveLoop_ClosedConnEndsLoop covers a socket closed underneath the
// loop without Stop: the loop must end instead of retrying forever.
func TestReceiveLoop_ClosedConnEndsLoop(t *testing.T) {
	conn, _ := openLocalUDPPair(t)
	listener := NewCoIoTListener()

	ended := make(chan struct{})
	go func() {
		defer close(ended)
		listener.receiveLoop(conn, make(chan struct{}), 1500)
	}()

	if err := conn.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	select {
	case <-ended:
	case <-time.After(5 * time.Second):
		t.Fatal("receive loop kept running on a closed socket")
	}
}

// TestReceiveLoop_PreservesArrivalOrder sends numbered packets and checks the
// handler sees them in the order they arrived. The first delivery is slow so
// that later packets are already waiting behind it.
func TestReceiveLoop_PreservesArrivalOrder(t *testing.T) {
	const packets = 300

	srvConn, srvAddr := openLocalUDPPair(t)
	// A large socket buffer keeps the burst from being dropped by the kernel.
	listener := NewCoIoTListener(WithCoIoTBufferSize(1 << 20))
	listener.listenFn = func(*net.UDPAddr) (*net.UDPConn, error) { return srvConn, nil }

	var got []string
	done := make(chan struct{})
	listener.OnStatus(func(deviceID string, _ *CoIoTStatus) {
		if len(got) == 0 {
			time.Sleep(50 * time.Millisecond)
		}
		got = append(got, deviceID)
		if len(got) == packets {
			close(done)
		}
	})

	if err := listener.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() {
		if err := listener.Stop(); err != nil {
			t.Errorf("Stop: %v", err)
		}
	})

	sender, err := net.DialUDP("udp4", nil, srvAddr)
	if err != nil {
		t.Fatalf("dial sender: %v", err)
	}
	defer sender.Close()

	for i := range packets {
		options := map[int][]byte{optionGlobalDevID: fmt.Appendf(nil, "DEV#%06d#2", i)}
		msg := buildCoAPMessage(0, codeStatus, options, []byte(`{"G":[[0,1101,1]]}`))
		if _, err := sender.Write(msg); err != nil {
			t.Fatalf("send %d: %v", i, err)
		}
	}

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for all packets")
	}

	for i, id := range got {
		if want := fmt.Sprintf("dev-%06d", i); id != want {
			t.Fatalf("delivery %d = %q, want %q", i, id, want)
		}
	}
}
