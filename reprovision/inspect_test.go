package reprovision

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/tj-smith47/shelly-go/types"
)

func TestInspect_RequiredOptions(t *testing.T) {
	t.Parallel()
	if _, err := Inspect(context.Background(), &InspectOptions{}); !errors.Is(err, types.ErrInvalidParam) {
		t.Errorf("err = %v, want ErrInvalidParam", err)
	}
}

func TestInspect_ConnectFailureReturnsHome(t *testing.T) {
	t.Parallel()
	s := homeScanner()
	s.connectErr = func(ssid string) error {
		if ssid == fakeAPSSID {
			return errors.New("association failed")
		}
		return nil
	}
	_, err := Inspect(context.Background(), &InspectOptions{APSSID: fakeAPSSID, Scanner: s})
	if err == nil || !strings.Contains(err.Error(), "AP hop for") {
		t.Fatalf("err = %v, want an AP hop failure", err)
	}
	assertReturnedHome(t, s, fakeAPSSID)
}

func TestInspect_Gen1(t *testing.T) {
	t.Parallel()
	d := newFakeDevice(t, 1)
	s := homeScanner()
	insp, err := testRunner(t, s, d.addr()).inspect(context.Background(), fakeAPSSID)
	if err != nil {
		t.Fatalf("inspect: %v", err)
	}
	want := Inspection{
		Model: fakeGen1Model, MAC: fakeMAC, Firmware: d.currentFW(), StaSSID: homeSSID, Ipv4Method: "dhcp",
		StaIP: "192.0.2.7", Generation: 1, StaKeySet: true, StaConnected: true,
	}
	if *insp != want {
		t.Errorf("inspection = %+v, want %+v", *insp, want)
	}
	if w := d.writes(); len(w) != 0 {
		t.Errorf("inspect wrote to the device: %v", w)
	}
	assertReturnedHome(t, s, fakeAPSSID)
}

func TestInspect_Gen2(t *testing.T) {
	t.Parallel()
	d := newFakeDevice(t, 2)
	insp, err := testRunner(t, homeScanner(), d.addr()).inspect(context.Background(), fakeAPSSID)
	if err != nil {
		t.Fatalf("inspect: %v", err)
	}
	want := Inspection{
		Generation: 2, MAC: fakeMAC, Model: "SNSW-001P16EU", Firmware: insp.Firmware,
		StaSSID: "HomeNet", StaKeySet: true, Ipv4Method: "dhcp", StaIP: "192.0.2.77", StaConnected: true,
	}
	if *insp != want {
		t.Errorf("inspection = %+v, want %+v", *insp, want)
	}
}

func TestInspect_Gen1WiFiReadFailureKeepsIdentity(t *testing.T) {
	t.Parallel()
	d := newFakeDevice(t, 1)
	d.settingsErr = true
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	insp, err := testRunner(t, homeScanner(), d.addr()).inspect(ctx, fakeAPSSID)
	if err == nil || !strings.Contains(err.Error(), "read Gen1 WiFi config") {
		t.Fatalf("err = %v, want the WiFi read failure", err)
	}
	if insp == nil || insp.MAC != fakeMAC {
		t.Errorf("inspection = %+v, want the identity kept", insp)
	}
}

func TestInspect_UnreachableDevice(t *testing.T) {
	t.Parallel()
	_, err := testRunner(t, homeScanner(), refusingAddr(t)).inspect(context.Background(), fakeAPSSID)
	if err == nil || !strings.Contains(err.Error(), "identify device") {
		t.Errorf("err = %v, want an identify failure", err)
	}
}

func TestInspect_Gen2WiFiReadFailureKeepsIdentity(t *testing.T) {
	t.Parallel()
	d := newFakeDevice(t, 2)
	d.failMethod = "WiFi.GetConfig"
	insp, err := testRunner(t, homeScanner(), d.addr()).inspect(context.Background(), fakeAPSSID)
	if err == nil || insp == nil || insp.MAC != fakeMAC || insp.StaSSID != "" {
		t.Errorf("inspection = %+v err = %v, want the identity with an error", insp, err)
	}
}
