package discovery

import (
	"context"
	"net"
	"strings"
	"sync"
	"time"

	"tinygo.org/x/bluetooth"

	"github.com/tj-smith47/shelly-go/internal/serial"
	"github.com/tj-smith47/shelly-go/types"
)

// BLE Service UUIDs for Shelly devices.
const (
	// ShellyBLEServiceUUID is the primary service UUID for Shelly BLE provisioning.
	ShellyBLEServiceUUID = "5f6d4f53-5f52-5043-5f53-56435f49445f"

	// ShellyBLECharacteristicRPC is the characteristic for RPC communication.
	ShellyBLECharacteristicRPC = "5f6d4f53-5f52-5043-5f72-7063" //nolint:gosec // Not a credential

	// ShellyBLECharacteristicData is the characteristic for data transfer.
	ShellyBLECharacteristicData = "5f6d4f53-5f52-5043-5f64-6174"

	// ShellyBLEAdvertisementPrefix is the prefix for Shelly BLE advertisements.
	ShellyBLEAdvertisementPrefix = "SHELLY-"

	// BTHomeServiceUUID is the BTHome v2 service UUID used by Shelly BLU devices.
	BTHomeServiceUUID = "fcd2"
)

// BLEDiscoveredDevice represents a Shelly device discovered via BLE.
type BLEDiscoveredDevice struct {
	BTHomeData  *BTHomeData `json:"bthome_data,omitempty"`
	ServiceUUID string      `json:"service_uuid,omitempty"`
	LocalName   string      `json:"local_name,omitempty"`
	DiscoveredDevice
	RSSI        int  `json:"rssi"`
	Connectable bool `json:"connectable"`
}

// BTHomeData represents decoded BTHome sensor data.
type BTHomeData struct {
	Temperature *float64 `json:"temperature,omitempty"`
	Humidity    *float64 `json:"humidity,omitempty"`
	Battery     *uint8   `json:"battery,omitempty"`
	Illuminance *uint32  `json:"illuminance,omitempty"`
	Motion      *bool    `json:"motion,omitempty"`
	Button      *uint8   `json:"button,omitempty"`
	WindowOpen  *bool    `json:"window_open,omitempty"`
	Rotation    *float64 `json:"rotation,omitempty"`
	PacketID    uint8    `json:"packet_id"`
}

// BLEDiscoverer discovers Shelly devices via Bluetooth Low Energy.
//
// This discoverer finds:
//   - Gen2+ devices in provisioning mode (advertising Shelly service)
//   - BLU devices broadcasting BTHome sensor data
//   - Any device advertising with "SHELLY-" prefix
//
// Note: This implementation provides the interface and data structures.
// Actual BLE scanning requires platform-specific implementations or
// external BLE libraries (e.g., tinygo-org/bluetooth, go-ble/ble).
type BLEDiscoverer struct {
	Scanner       BLEScanner
	devices       map[string]*BLEDiscoveredDevice
	devicesCh     chan DiscoveredDevice
	cancel        context.CancelFunc
	OnDeviceFound func(*BLEDiscoveredDevice)
	FilterPrefix  string
	found         serial.Queue[*BLEDiscoveredDevice]
	ScanDuration  time.Duration
	rescanDelay   time.Duration
	mu            sync.RWMutex
	running       bool
	IncludeBTHome bool
}

// defaultBLEScanDuration is the length of one scan when ScanDuration is unset.
const defaultBLEScanDuration = 10 * time.Second

// defaultBLERescanDelay is how long continuous discovery waits before the
// next scan when a scan failed or returned before its time was up.
const defaultBLERescanDelay = time.Second

// BLEScanner is the interface for platform-specific BLE scanning.
// The default implementation uses tinygo.org/x/bluetooth.
type BLEScanner interface {
	// Start begins BLE scanning. Found devices are sent to the callback.
	Start(ctx context.Context, callback func(*BLEAdvertisement)) error

	// Stop stops BLE scanning.
	Stop() error
}

// bleScanAdapter is the part of *bluetooth.Adapter the scanner needs. Tests
// supply a fake so no real Bluetooth hardware is touched.
type bleScanAdapter interface {
	Scan(callback func(*bluetooth.Adapter, bluetooth.ScanResult)) error
	StopScan() error
}

// tinyGoBLEScanner implements BLEScanner using tinygo.org/x/bluetooth.
// This is the default scanner used by BLEDiscoverer.
type tinyGoBLEScanner struct {
	adapter bleScanAdapter
	stopCh  chan struct{}
	mu      sync.Mutex
	running bool
}

// newTinyGoBLEScanner creates a new BLE scanner using the TinyGo bluetooth library.
func newTinyGoBLEScanner() (*tinyGoBLEScanner, error) {
	adapter := bluetooth.DefaultAdapter
	if err := adapter.Enable(); err != nil {
		return nil, &BLEError{Message: "failed to enable bluetooth adapter", Err: err}
	}

	return &tinyGoBLEScanner{
		adapter: adapter,
	}, nil
}

// scanStartTimeout is the maximum time to wait for the BLE scan to start.
// This is separate from the discovery timeout - it detects if the scan itself
// is failing to initialize (e.g., due to system issues).
const scanStartTimeout = 5 * time.Second

// Start begins BLE scanning. Found devices are sent to the callback.
func (s *tinyGoBLEScanner) Start(ctx context.Context, callback func(*BLEAdvertisement)) error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return nil
	}
	// Everything below works on this scan's own stop channel. A later Start
	// replaces s.stopCh, and this call must not wait on or end that scan.
	stop := make(chan struct{})
	s.stopCh = stop
	s.running = true
	s.mu.Unlock()

	// Channel to signal that scanning has actually started
	scanStarted := make(chan struct{}, 1)
	errCh := make(chan error, 1)

	// Run scan in goroutine
	go s.runScan(stop, callback, scanStarted, errCh)

	// Wait for scan to start
	if err := s.waitForScanStart(ctx, stop, scanStarted, errCh); err != nil {
		return err
	}

	// Scan started, now wait for completion
	return s.waitForCompletion(ctx, stop, errCh)
}

// runScan runs the BLE scan in a goroutine. Advertisements go to callback
// until stop is closed.
func (s *tinyGoBLEScanner) runScan(
	stop <-chan struct{}, callback func(*BLEAdvertisement), scanStarted chan<- struct{}, errCh chan<- error,
) {
	// Signal that we're about to call Scan
	select {
	case scanStarted <- struct{}{}:
	default:
	}

	err := s.adapter.Scan(func(adapter *bluetooth.Adapter, device bluetooth.ScanResult) {
		// Signal that scan is working (we got a callback)
		select {
		case scanStarted <- struct{}{}:
		default:
		}

		if callback == nil {
			return
		}

		// The adapter may still report results for a moment after this scan
		// was stopped; the caller no longer expects them.
		select {
		case <-stop:
			return
		default:
		}

		callback(s.convertAdvertisement(device))
	})
	errCh <- err
}

// waitForScanStart waits for the scan to start or returns an error.
func (s *tinyGoBLEScanner) waitForScanStart(
	ctx context.Context, stop chan struct{}, scanStarted <-chan struct{}, errCh <-chan error,
) error {
	startTimer := time.NewTimer(scanStartTimeout)
	defer startTimer.Stop()

	select {
	case <-scanStarted:
		return nil // Scan started successfully
	case <-startTimer.C:
		s.stopScan(stop) //nolint:errcheck // Best effort
		return &BLEError{Message: "BLE scan failed to start - Bluetooth may be busy or unavailable"}
	case <-ctx.Done():
		s.stopScan(stop) //nolint:errcheck // Best effort
		return ctx.Err()
	case err := <-errCh:
		s.scanEnded(stop)
		return err
	}
}

// waitForCompletion waits for the scan to complete via context or stop signal.
func (s *tinyGoBLEScanner) waitForCompletion(ctx context.Context, stop chan struct{}, errCh <-chan error) error {
	select {
	case <-ctx.Done():
		s.stopScan(stop) //nolint:errcheck // Best effort
		return nil
	case <-stop:
		return nil
	case err := <-errCh:
		s.scanEnded(stop)
		if err != nil {
			return &BLEError{Message: "BLE scan stopped unexpectedly", Err: err}
		}
		return nil
	}
}

// scanEnded records that the scan owning stop finished by itself. A scan that
// was already replaced by a newer one leaves the newer one marked as running.
func (s *tinyGoBLEScanner) scanEnded(stop chan struct{}) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running && s.stopCh == stop {
		s.running = false
		close(stop)
	}
}

// Stop stops BLE scanning.
func (s *tinyGoBLEScanner) Stop() error {
	return s.stopScan(nil)
}

// stopScan stops the scan that owns stop, or whichever scan is running when
// stop is nil. It does nothing if that scan is no longer the current one.
func (s *tinyGoBLEScanner) stopScan(stop chan struct{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running || (stop != nil && s.stopCh != stop) {
		return nil
	}

	s.running = false
	close(s.stopCh)
	if s.adapter == nil {
		return nil
	}
	return s.adapter.StopScan()
}

// convertAdvertisement converts a TinyGo ScanResult to our BLEAdvertisement.
func (s *tinyGoBLEScanner) convertAdvertisement(device bluetooth.ScanResult) *BLEAdvertisement {
	adv := &BLEAdvertisement{
		Address:     device.Address.String(),
		LocalName:   device.LocalName(),
		RSSI:        int(device.RSSI),
		Connectable: true, // TinyGo doesn't expose this, assume connectable
		ServiceData: make(map[string][]byte),
	}

	// Convert manufacturer data
	mfgData := device.ManufacturerData()
	if len(mfgData) > 0 {
		// Use first manufacturer data entry
		adv.ManufacturerID = mfgData[0].CompanyID
		adv.ManufacturerData = mfgData[0].Data
	}

	return adv
}

// BLEAdvertisement represents a raw BLE advertisement.
type BLEAdvertisement struct {
	ServiceData      map[string][]byte
	Address          string
	LocalName        string
	ServiceUUIDs     []string
	ManufacturerData []byte
	RSSI             int
	ManufacturerID   uint16
	Connectable      bool
}

// ErrBLENotSupported indicates BLE is not available on this platform.
var ErrBLENotSupported = &BLEError{Message: "BLE scanning not supported on this platform - set Scanner field"}

// BLEError represents a BLE-related error.
type BLEError struct {
	Err     error
	Message string
}

func (e *BLEError) Error() string {
	if e.Err != nil {
		return e.Message + ": " + e.Err.Error()
	}
	return e.Message
}

func (e *BLEError) Unwrap() error {
	return e.Err
}

// NewBLEDiscoverer creates a new BLE discoverer with the default TinyGo scanner.
// Returns an error if the bluetooth adapter cannot be enabled.
func NewBLEDiscoverer() (*BLEDiscoverer, error) {
	scanner, err := newTinyGoBLEScanner()
	if err != nil {
		return nil, err
	}

	return &BLEDiscoverer{
		Scanner:       scanner,
		devices:       make(map[string]*BLEDiscoveredDevice),
		ScanDuration:  defaultBLEScanDuration,
		FilterPrefix:  ShellyBLEAdvertisementPrefix,
		IncludeBTHome: true,
	}, nil
}

// NewBLEDiscovererWithScanner creates a BLE discoverer with a custom scanner.
// Use this if you want to provide your own BLEScanner implementation.
func NewBLEDiscovererWithScanner(scanner BLEScanner) *BLEDiscoverer {
	return &BLEDiscoverer{
		Scanner:       scanner,
		devices:       make(map[string]*BLEDiscoveredDevice),
		ScanDuration:  defaultBLEScanDuration,
		FilterPrefix:  ShellyBLEAdvertisementPrefix,
		IncludeBTHome: true,
	}
}

// Discover scans for BLE devices for the specified duration.
func (b *BLEDiscoverer) Discover(timeout time.Duration) ([]DiscoveredDevice, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	return b.DiscoverWithContext(ctx)
}

// DiscoverWithContext scans for BLE devices until the context is canceled.
func (b *BLEDiscoverer) DiscoverWithContext(ctx context.Context) ([]DiscoveredDevice, error) {
	if b.Scanner == nil {
		return nil, ErrBLENotSupported
	}

	b.mu.Lock()
	b.devices = make(map[string]*BLEDiscoveredDevice)
	b.mu.Unlock()

	err := b.Scanner.Start(ctx, b.handleAdvertisement)
	if err != nil {
		return nil, &BLEError{Message: "failed to start BLE scan", Err: err}
	}
	defer b.Scanner.Stop() //nolint:errcheck // Best-effort cleanup

	// Wait for context to complete
	<-ctx.Done()

	// Collect results
	b.mu.RLock()
	defer b.mu.RUnlock()

	result := make([]DiscoveredDevice, 0, len(b.devices))
	for _, d := range b.devices {
		result = append(result, d.DiscoveredDevice)
	}

	return result, nil
}

// handleAdvertisement processes a BLE advertisement.
func (b *BLEDiscoverer) handleAdvertisement(adv *BLEAdvertisement) {
	// Check if this is a Shelly device
	if !b.isShellyDevice(adv) {
		return
	}

	device := b.parseAdvertisement(adv)
	if device == nil {
		return
	}

	// devicesCh is replaced by StartDiscovery under the same lock.
	b.mu.Lock()
	b.devices[device.ID] = device
	devicesCh := b.devicesCh
	b.mu.Unlock()

	// A scanner may report advertisements from several goroutines; the queue
	// keeps OnDeviceFound from running concurrently with itself.
	b.found.Push(device, func(d *BLEDiscoveredDevice) {
		if b.OnDeviceFound != nil {
			b.OnDeviceFound(d)
		}
	})

	if devicesCh != nil {
		select {
		case devicesCh <- device.DiscoveredDevice:
		default:
		}
	}
}

// isShellyDevice checks if an advertisement is from a Shelly device.
func (b *BLEDiscoverer) isShellyDevice(adv *BLEAdvertisement) bool {
	// Check local name prefix
	if strings.HasPrefix(strings.ToUpper(adv.LocalName), b.FilterPrefix) {
		return true
	}

	// Check for Shelly service UUID
	for _, uuid := range adv.ServiceUUIDs {
		if strings.EqualFold(uuid, ShellyBLEServiceUUID) {
			return true
		}
	}

	// Check for BTHome service data
	if b.IncludeBTHome {
		if _, ok := adv.ServiceData[BTHomeServiceUUID]; ok {
			return true
		}
	}

	return false
}

// parseAdvertisement parses a BLE advertisement into a device.
func (b *BLEDiscoverer) parseAdvertisement(adv *BLEAdvertisement) *BLEDiscoveredDevice {
	device := &BLEDiscoveredDevice{
		DiscoveredDevice: DiscoveredDevice{
			ID:         adv.Address,
			Name:       adv.LocalName,
			MACAddress: adv.Address,
			Protocol:   ProtocolBLE,
			Address:    nil,        // BLE devices don't have IP until provisioned
			Generation: types.Gen2, // BLE provisioning is Gen2+
			LastSeen:   time.Now(),
		},
		RSSI:        adv.RSSI,
		LocalName:   adv.LocalName,
		Connectable: adv.Connectable,
	}

	// Extract service UUID
	for _, uuid := range adv.ServiceUUIDs {
		device.ServiceUUID = uuid
		break
	}

	// Parse model from local name (format: SHELLY-MODEL-XXXX)
	if strings.HasPrefix(strings.ToUpper(adv.LocalName), ShellyBLEAdvertisementPrefix) {
		parts := strings.Split(adv.LocalName, "-")
		if len(parts) >= 2 {
			device.Model = parts[1]
		}
	}

	// Parse BTHome data if present
	if bthomeData, ok := adv.ServiceData[BTHomeServiceUUID]; ok {
		device.BTHomeData = parseBTHomeData(bthomeData)
		device.Generation = types.Gen2 // BLU devices are considered Gen2+
	}

	return device
}

// BTHome object type sizes.
var bthomeObjectSizes = map[uint8]int{
	0x00: 1, // Packet ID
	0x01: 1, // Battery
	0x02: 2, // Temperature
	0x03: 2, // Humidity
	0x05: 3, // Illuminance
	0x21: 1, // Motion
	0x2D: 1, // Window
	0x3A: 1, // Button
	0x3F: 2, // Rotation
}

// parseBTHomeData parses BTHome v2 service data.
func parseBTHomeData(data []byte) *BTHomeData {
	if len(data) < 1 {
		return nil
	}

	// First byte is device info: encrypted (1 bit) | trigger (1 bit) | version (5 bits)
	// We only parse unencrypted BTHome v2 data
	if data[0]&0x01 != 0 {
		return nil // Encrypted, can't parse
	}

	result := &BTHomeData{}
	offset := 1

	for offset < len(data) {
		objectID := data[offset]
		offset++

		size, known := bthomeObjectSizes[objectID]
		if !known {
			size = 1 // Skip unknown objects
		}

		if offset+size-1 >= len(data) {
			break // Not enough data
		}

		parseBTHomeObject(result, objectID, data[offset:offset+size])
		offset += size
	}

	return result
}

// parseBTHomeObject parses a single BTHome object into the result.
func parseBTHomeObject(result *BTHomeData, objectID uint8, data []byte) {
	// Safety check: caller should ensure proper slice length
	if len(data) == 0 {
		return
	}

	switch objectID {
	case 0x00: // Packet ID
		result.PacketID = data[0]
	case 0x01: // Battery (uint8, %)
		val := data[0]
		result.Battery = &val
	case 0x02: // Temperature (int16, 0.01°C)
		if len(data) < 2 {
			return
		}
		temp := float64(int16(data[0])|int16(data[1])<<8) * 0.01
		result.Temperature = &temp
	case 0x03: // Humidity (uint16, 0.01%)
		if len(data) < 2 {
			return
		}
		hum := float64(uint16(data[0])|uint16(data[1])<<8) * 0.01
		result.Humidity = &hum
	case 0x05: // Illuminance (uint24, 0.01 lux)
		if len(data) < 3 {
			return
		}
		lux := uint32(data[0]) | uint32(data[1])<<8 | uint32(data[2])<<16
		result.Illuminance = &lux
	case 0x21: // Motion (uint8, 0=clear, 1=detected)
		motion := data[0] != 0
		result.Motion = &motion
	case 0x2D: // Window (uint8, 0=closed, 1=open)
		open := data[0] != 0
		result.WindowOpen = &open
	case 0x3A: // Button (uint8, event type)
		val := data[0]
		result.Button = &val
	case 0x3F: // Rotation (int16, 0.1°)
		if len(data) < 2 {
			return
		}
		rot := float64(int16(data[0])|int16(data[1])<<8) * 0.1
		result.Rotation = &rot
	}
}

// StartDiscovery begins continuous BLE discovery.
func (b *BLEDiscoverer) StartDiscovery() (<-chan DiscoveredDevice, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.running {
		return b.devicesCh, nil
	}

	if b.Scanner == nil {
		return nil, ErrBLENotSupported
	}

	scanDuration := b.ScanDuration
	if scanDuration <= 0 {
		scanDuration = defaultBLEScanDuration
	}
	rescanDelay := b.rescanDelay
	if rescanDelay <= 0 {
		rescanDelay = defaultBLERescanDelay
	}

	ctx, cancel := context.WithCancel(context.Background())
	b.devicesCh = make(chan DiscoveredDevice, 100)
	b.cancel = cancel
	b.running = true

	// The loop gets everything it needs as arguments: the fields are replaced
	// by the next StartDiscovery and may be changed by the caller meanwhile.
	go b.continuousDiscovery(ctx, b.Scanner, scanDuration, rescanDelay)

	return b.devicesCh, nil
}

// continuousDiscovery scans repeatedly until ctx is canceled.
func (b *BLEDiscoverer) continuousDiscovery(
	ctx context.Context, scanner BLEScanner, scanDuration, rescanDelay time.Duration,
) {
	for ctx.Err() == nil {
		// Deriving from ctx ends a scan in progress when discovery stops, even
		// if the scanner's Stop was called just before this scan began.
		scanCtx, cancel := context.WithTimeout(ctx, scanDuration)
		err := scanner.Start(scanCtx, b.handleAdvertisement)
		endedEarly := scanCtx.Err() == nil
		cancel()

		// A failed scan does not end discovery, but a scanner that fails or
		// returns at once must not be restarted in a tight loop.
		if err == nil && !endedEarly {
			continue
		}

		timer := time.NewTimer(rescanDelay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
	}
}

// StopDiscovery stops continuous BLE discovery.
func (b *BLEDiscoverer) StopDiscovery() error {
	b.mu.Lock()
	if !b.running {
		b.mu.Unlock()
		return nil
	}
	b.running = false
	b.cancel()
	scanner := b.Scanner
	b.mu.Unlock()

	// The scanner reports advertisements through handleAdvertisement, which
	// takes b.mu. A Stop that waits for those callbacks to return would never
	// finish if it were called with the lock held.
	if scanner != nil {
		//nolint:errcheck // Best-effort stop
		scanner.Stop()
	}

	return nil
}

// Stop stops the discoverer and releases resources.
func (b *BLEDiscoverer) Stop() error {
	return b.StopDiscovery()
}

// GetDiscoveredDevices returns all currently discovered BLE devices.
func (b *BLEDiscoverer) GetDiscoveredDevices() []BLEDiscoveredDevice {
	b.mu.RLock()
	defer b.mu.RUnlock()

	result := make([]BLEDiscoveredDevice, 0, len(b.devices))
	for _, d := range b.devices {
		result = append(result, *d)
	}
	return result
}

// DeviceByAddress returns a discovered device by its BLE address.
func (b *BLEDiscoverer) DeviceByAddress(address string) *BLEDiscoveredDevice {
	b.mu.RLock()
	defer b.mu.RUnlock()

	return b.devices[address]
}

// Clear clears all discovered devices.
func (b *BLEDiscoverer) Clear() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.devices = make(map[string]*BLEDiscoveredDevice)
}

// IsDeviceProvisioned checks if a BLE device has an IP address (is provisioned).
func IsDeviceProvisioned(device *BLEDiscoveredDevice) bool {
	return device.Address != nil && !device.Address.Equal(net.IPv4zero)
}

// BLEConnector provides methods for connecting to BLE devices.
// This interface allows platform-specific implementations for actual connection testing.
type BLEConnector interface {
	// Connect attempts to connect to the device at the given address.
	Connect(ctx context.Context, address string) error

	// Disconnect disconnects from the currently connected device.
	Disconnect() error

	// IsConnected returns true if currently connected to a device.
	IsConnected() bool
}

// connectabilityCache caches IsConnectable results to avoid repeated connection attempts.
type connectabilityCache struct {
	results map[string]bool
	mu      sync.RWMutex
}

var bleConnectabilityCache = &connectabilityCache{
	results: make(map[string]bool),
}

// IsConnectable checks if a BLE device is connectable.
//
// This method first checks the cached result. If not cached, it uses the provided
// connector to attempt a connection. If no connector is provided, it returns
// the value from the BLE advertisement (which may be inaccurate for some platforms).
//
// The result is cached to avoid repeated connection attempts to the same device.
//
// Parameters:
//   - ctx: Context for connection timeout/cancellation
//   - device: The BLE device to check
//   - connector: Optional BLEConnector for actual connection testing (nil to use advertisement value)
//
// Returns:
//   - bool: true if the device is connectable
//   - error: any error from the connection attempt (nil if using cached or advertisement value)
func IsConnectable(ctx context.Context, device *BLEDiscoveredDevice, connector BLEConnector) (bool, error) {
	if device == nil {
		return false, &BLEError{Message: "device is nil"}
	}

	// Check cache first
	bleConnectabilityCache.mu.RLock()
	if cached, ok := bleConnectabilityCache.results[device.MACAddress]; ok {
		bleConnectabilityCache.mu.RUnlock()
		return cached, nil
	}
	bleConnectabilityCache.mu.RUnlock()

	// If no connector provided, use the advertisement value
	if connector == nil {
		return device.Connectable, nil
	}

	// Attempt actual connection
	connectErr := connector.Connect(ctx, device.MACAddress)

	var connectable bool
	if connectErr == nil {
		// Successfully connected - disconnect and mark as connectable
		connectable = true
		connector.Disconnect() //nolint:errcheck // Best effort
	} else {
		// Connection failed - not connectable
		connectable = false
	}

	// Cache the result
	bleConnectabilityCache.mu.Lock()
	bleConnectabilityCache.results[device.MACAddress] = connectable
	bleConnectabilityCache.mu.Unlock()

	return connectable, nil
}

// ClearConnectabilityCache clears the connectability cache.
// Call this if you want to retest device connectability.
func ClearConnectabilityCache() {
	bleConnectabilityCache.mu.Lock()
	defer bleConnectabilityCache.mu.Unlock()
	bleConnectabilityCache.results = make(map[string]bool)
}

// tinyGoBLEConnector implements BLEConnector using tinygo.org/x/bluetooth.
// Note: This is a basic implementation. For production use, consider
// handling connection state more robustly.
type tinyGoBLEConnector struct {
	adapter bleDialer
	// dropDevice replaces device.Disconnect in tests, where a bluetooth.Device
	// has no real connection behind it.
	dropDevice func(bluetooth.Device) error
	device     bluetooth.Device
	connected  bool
	mu         sync.Mutex
}

// bleDialer is the part of *bluetooth.Adapter the connector needs. Tests
// supply a fake so no real Bluetooth hardware is touched.
type bleDialer interface {
	Connect(addr bluetooth.Address, params bluetooth.ConnectionParams) (bluetooth.Device, error)
}

// bleDialResult is what a connection attempt hands back to Connect.
type bleDialResult struct {
	err    error
	device bluetooth.Device
}

// NewTinyGoBLEConnector creates a new BLE connector using the TinyGo bluetooth library.
// Returns an error if the bluetooth adapter cannot be enabled.
func NewTinyGoBLEConnector() (*tinyGoBLEConnector, error) {
	adapter := bluetooth.DefaultAdapter
	if err := adapter.Enable(); err != nil {
		return nil, &BLEError{Message: "failed to enable bluetooth adapter", Err: err}
	}

	return &tinyGoBLEConnector{
		adapter: adapter,
	}, nil
}

// Connect attempts to connect to the device at the given address.
func (c *tinyGoBLEConnector) Connect(ctx context.Context, address string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.connected {
		return &BLEError{Message: "already connected to a device"}
	}

	// Create address using platform-agnostic Set method
	var addr bluetooth.Address
	addr.Set(address)

	// Create connection parameters
	params := bluetooth.ConnectionParams{}

	// The goroutine only reports its result. It may outlive this call, so it
	// must not touch the connector's fields, which belong to whoever holds
	// c.mu.
	adapter := c.adapter
	done := make(chan bleDialResult, 1)
	go func() {
		device, err := adapter.Connect(addr, params)
		done <- bleDialResult{device: device, err: err}
	}()

	select {
	case <-ctx.Done():
		go c.dropLate(done)
		return ctx.Err()
	case res := <-done:
		if res.err != nil {
			return &BLEError{Message: "failed to connect", Err: res.err}
		}
		c.device = res.device
		c.connected = true
		return nil
	}
}

// dropLate closes a connection that is made after Connect stopped waiting for
// it. The adapter call cannot be interrupted, and nobody owns its result.
func (c *tinyGoBLEConnector) dropLate(done <-chan bleDialResult) {
	res := <-done
	if res.err != nil {
		return
	}
	// Connect has already returned, so a failed disconnect has no one to go to.
	if err := c.drop(res.device); err != nil {
		return
	}
}

// drop closes the connection to device.
func (c *tinyGoBLEConnector) drop(device bluetooth.Device) error {
	if c.dropDevice != nil {
		return c.dropDevice(device)
	}
	return device.Disconnect()
}

// Disconnect disconnects from the currently connected device.
func (c *tinyGoBLEConnector) Disconnect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected {
		return nil
	}

	err := c.drop(c.device)
	c.connected = false
	return err
}

// IsConnected returns true if currently connected to a device.
func (c *tinyGoBLEConnector) IsConnected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.connected
}
