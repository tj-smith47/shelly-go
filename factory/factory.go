package factory

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/tj-smith47/shelly-go/discovery"
	"github.com/tj-smith47/shelly-go/gen1"
	"github.com/tj-smith47/shelly-go/gen2"
	"github.com/tj-smith47/shelly-go/rpc"
	"github.com/tj-smith47/shelly-go/transport"
	"github.com/tj-smith47/shelly-go/types"
)

// Device is the common interface for all Shelly device generations.
type Device interface {
	// Address returns the device address.
	Address() string

	// Generation returns the device generation.
	Generation() types.Generation
}

// Gen1Device wraps a Gen1 device.
type Gen1Device struct {
	*gen1.Device
	addr       string
	generation types.Generation
}

// Address returns the device address.
func (d *Gen1Device) Address() string {
	return d.addr
}

// Generation returns the device generation.
func (d *Gen1Device) Generation() types.Generation {
	return d.generation
}

// Gen2Device wraps a Gen2+ device.
type Gen2Device struct {
	*gen2.Device
	addr       string
	generation types.Generation
}

// Address returns the device address.
func (d *Gen2Device) Address() string {
	return d.addr
}

// Generation returns the device generation.
func (d *Gen2Device) Generation() types.Generation {
	return d.generation
}

// Options configures device creation.
type Options struct {
	Context    context.Context
	HTTPClient *http.Client
	Username   string
	Password   string
	Timeout    time.Duration
	Generation types.Generation
}

// Option configures device creation.
type Option func(*Options)

// WithAuth sets authentication credentials.
func WithAuth(username, password string) Option {
	return func(o *Options) {
		o.Username = username
		o.Password = password
	}
}

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(client *http.Client) Option {
	return func(o *Options) {
		o.HTTPClient = client
	}
}

// WithTimeout sets the request timeout.
func WithTimeout(timeout time.Duration) Option {
	return func(o *Options) {
		o.Timeout = timeout
	}
}

// WithContext sets the context for operations.
func WithContext(ctx context.Context) Option {
	return func(o *Options) {
		o.Context = ctx
	}
}

// WithGeneration forces a specific device generation.
func WithGeneration(gen types.Generation) Option {
	return func(o *Options) {
		o.Generation = gen
	}
}

// Errors returned by the factory.
var (
	ErrUnknownGeneration = errors.New("unknown device generation")
	ErrDetectionFailed   = errors.New("failed to detect device generation")
)

// FromAddress creates a device from an address string.
//
// The address can be an IP address, hostname, or URL.
// If no generation is specified via WithGeneration, the factory
// will probe the device to auto-detect its generation.
func FromAddress(address string, opts ...Option) (Device, error) {
	options := &Options{
		Context: context.Background(),
		Timeout: 5 * time.Second,
	}
	for _, opt := range opts {
		opt(options)
	}

	// Normalize address
	if !strings.HasPrefix(address, "http://") && !strings.HasPrefix(address, "https://") {
		address = "http://" + address
	}
	address = strings.TrimSuffix(address, "/")

	// If generation is specified, create device directly
	if options.Generation != 0 {
		return createDevice(address, options.Generation, options)
	}

	// Auto-detect generation
	info, err := discovery.IdentifyWithTimeout(options.Context, address, options.Timeout)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDetectionFailed, err)
	}

	return createDevice(address, info.Generation, options)
}

// FromDiscovery creates a device from a discovery result.
func FromDiscovery(d *discovery.DiscoveredDevice, opts ...Option) (Device, error) {
	options := &Options{
		Context: context.Background(),
		Timeout: 5 * time.Second,
	}
	for _, opt := range opts {
		opt(options)
	}

	address := d.URL()

	return createDevice(address, d.Generation, options)
}

// FromInfo creates a device from device info.
func FromInfo(info *discovery.DeviceInfo, address string, opts ...Option) (Device, error) {
	options := &Options{
		Context: context.Background(),
		Timeout: 5 * time.Second,
	}
	for _, opt := range opts {
		opt(options)
	}

	// Normalize address
	if !strings.HasPrefix(address, "http://") && !strings.HasPrefix(address, "https://") {
		address = "http://" + address
	}

	return createDevice(address, info.Generation, options)
}

// createDevice creates the appropriate device type.
func createDevice(address string, generation types.Generation, options *Options) (Device, error) {
	switch generation {
	case types.Gen1:
		return createGen1Device(address, options), nil
	case types.Gen2, types.Gen3, types.Gen4:
		return createGen2Device(address, generation, options), nil
	default:
		return nil, ErrUnknownGeneration
	}
}

// createGen1Device creates a Gen1 device.
func createGen1Device(address string, options *Options) *Gen1Device {
	var transportOpts []transport.Option

	if options.Timeout > 0 {
		transportOpts = append(transportOpts, transport.WithTimeout(options.Timeout))
	}

	if options.Username != "" && options.Password != "" {
		transportOpts = append(transportOpts, transport.WithAuth(options.Username, options.Password))
	}

	t := transport.NewHTTP(address, transportOpts...)
	device := gen1.NewDevice(t)

	return &Gen1Device{
		Device:     device,
		addr:       address,
		generation: types.Gen1,
	}
}

// createGen2Device creates a Gen2+ device.
func createGen2Device(address string, generation types.Generation, options *Options) *Gen2Device {
	var transportOpts []transport.Option

	if options.Timeout > 0 {
		transportOpts = append(transportOpts, transport.WithTimeout(options.Timeout))
	}

	t := transport.NewHTTP(address, transportOpts...)

	var client *rpc.Client
	if options.Username != "" && options.Password != "" {
		auth := &rpc.AuthData{
			Realm:    "shelly",
			Username: options.Username,
			Password: options.Password,
		}
		client = rpc.NewClientWithAuth(t, auth)
	} else {
		client = rpc.NewClient(t)
	}

	device := gen2.NewDevice(client)

	return &Gen2Device{
		Device:     device,
		addr:       address,
		generation: generation,
	}
}

// MustFromAddress creates a device from an address and panics on error.
func MustFromAddress(address string, opts ...Option) Device {
	device, err := FromAddress(address, opts...)
	if err != nil {
		panic(err)
	}
	return device
}

// MustFromDiscovery creates a device from discovery result and panics on error.
func MustFromDiscovery(d *discovery.DiscoveredDevice, opts ...Option) Device {
	device, err := FromDiscovery(d, opts...)
	if err != nil {
		panic(err)
	}
	return device
}

// BatchFromAddresses creates devices from multiple addresses concurrently.
func BatchFromAddresses(addresses []string, opts ...Option) ([]Device, []error) {
	devices := make([]Device, len(addresses))
	errs := make([]error, len(addresses))

	type result struct {
		device Device
		err    error
		index  int
	}

	results := make(chan result, len(addresses))

	for i, addr := range addresses {
		go func(index int, address string) {
			device, err := FromAddress(address, opts...)
			results <- result{index: index, device: device, err: err}
		}(i, addr)
	}

	for range addresses {
		r := <-results
		devices[r.index] = r.device
		errs[r.index] = r.err
	}

	return devices, errs
}

// BatchFromDiscovery creates devices from multiple discovery results.
func BatchFromDiscovery(discovered []discovery.DiscoveredDevice, opts ...Option) ([]Device, []error) {
	devices := make([]Device, len(discovered))
	errs := make([]error, len(discovered))

	for i := range discovered {
		device, err := FromDiscovery(&discovered[i], opts...)
		devices[i] = device
		errs[i] = err
	}

	return devices, errs
}
