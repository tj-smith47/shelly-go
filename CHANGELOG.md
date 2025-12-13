# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.5] - 2025-12-13

### Added
- **EMData component** for 3-phase historical energy data retrieval
  - Compatible with Shelly Pro 3EM and Pro EM-50 devices
  - `GetStatus()` - Retrieve data collection status and available records
  - `GetRecords()` - List available time intervals containing stored data
  - `GetData()` - Retrieve historical measurements for specified timestamp range
  - `DeleteAllData()` - Clear all stored historical data
  - `GetDataCSVURL()` - Generate HTTP URL for CSV data export
  - Stores up to 60 days of 1-minute interval measurements in non-volatile memory
  - Per-phase measurements: voltage, current, active/apparent power, power factor, frequency
  - Total measurements: current, power, neutral current, energy counters
- **EM1Data component** for single-phase historical energy data retrieval
  - Compatible with Shelly Pro EM and Pro EM-50 devices
  - `GetConfig()` / `SetConfig()` - Configure data collection settings (period, retention)
  - `GetStatus()` - Retrieve data collection status and available records
  - `GetRecords()` - List available time intervals containing stored data
  - `GetData()` - Retrieve historical measurements for specified timestamp range
  - `DeleteAllData()` - Clear all stored historical data
  - `GetDataCSVURL()` - Generate HTTP URL for CSV data export
  - Configurable collection interval (data_period) and retention period (data_storage_days)
  - Single-phase measurements: voltage, current, active/apparent power, power factor, frequency, energy counters
- **Example**: `examples/energy/historical_data.go`
  - Demonstrates historical energy data retrieval and analysis
  - Energy consumption calculation from power measurements
  - Peak power detection and timestamp tracking
  - Cost estimation based on consumption
  - CSV export URL generation
  - Supports both 3-phase (Pro 3EM) and single-phase (Pro EM) devices

### Tests
- Comprehensive test coverage (≥90%) for EMData and EM1Data components
- Table-driven tests for all methods (GetStatus, GetRecords, GetData, DeleteAllData, GetDataCSVURL)
- Error handling validation (network errors, invalid JSON, RPC failures)
- Parameter validation tests (timestamp ranges, optional parameters)
- CSV URL generation tests with various parameter combinations

### Fixed
- Added componentTypeNames entries for "emdata" and "em1data" to ensure proper RPC method name capitalization

## [0.1.0] - 2025-12-11

Initial public release of shelly-go, a comprehensive Go library for Shelly smart home devices.

### Features

#### Device Support
- **Gen1 devices**: Full support via HTTP API and CoIoT (CoAP)
  - Relays, lights, RGB/RGBW, dimmers, rollers/covers
  - Power metering (meters, emeters)
  - Input handling and actions
- **Gen2/Gen3/Gen4 devices**: Full support via JSON-RPC 2.0
  - All component types (Switch, Light, Cover, Input, etc.)
  - Energy monitoring (PM, EM, EM1)
  - Environmental sensors (Temperature, Humidity, Illuminance)
  - Thermostat control
  - UI components and virtual components
- **BLU devices**: BTHome sensor data parsing
  - Button, H&T, Motion, Door/Window sensors
  - Battery, temperature, humidity, illuminance readings
- **Wave devices**: Z-Wave integration via Shelly hub

#### Transport Layer
- **HTTP**: Synchronous RPC with retry and timeout
- **WebSocket**: Real-time bidirectional communication
- **MQTT**: Pub/sub messaging support
- **CoAP**: Gen1 CoIoT multicast discovery and status

#### Authentication
- Basic authentication for Gen1 devices
- Digest authentication (MD5/SHA-256) for Gen2+ devices
- OAuth 2.0 for Shelly Cloud API

#### Discovery
- **mDNS**: Zero-config service discovery (`_shelly._tcp.local`)
- **CoIoT**: Gen1 multicast announcements
- **BLE**: Bluetooth Low Energy scanning with BTHome parsing
- **WiFi AP**: Connect to device AP for provisioning (Linux/macOS/Windows)
- **Network scanning**: Probe IP ranges for Shelly devices

#### Cloud Integration
- Shelly Cloud API client
- OAuth authentication flow
- Device listing and control
- Real-time WebSocket events

#### Integrator API
- Fleet management
- Device provisioning
- Analytics and telemetry

#### Utilities
- **Backup/Restore**: Full device configuration backup
- **Migration**: Move configurations between devices
- **Firmware**: Check, download, and update firmware
- **Scenes**: Create and manage device scenes
- **Schedules**: Schedule automation rules
- **Groups**: Control multiple devices together
- **Batch operations**: Concurrent device control

#### Events
- Real-time event bus with filtering
- Typed event handlers per component
- Notification parsing and routing

### Architecture
- Clean separation of concerns (transport, rpc, types)
- Interface-based design for testability
- Context-aware cancellation throughout
- Comprehensive error handling with wrapped errors

### Quality
- 71%+ overall test coverage (90%+ for core packages)
- golangci-lint clean with strict configuration
- Benchmark tests for hot paths
- Comprehensive godoc documentation
- Working examples for all major features

### Documentation
- Package-level documentation in doc.go files
- Inline godoc comments for all exports
- Examples directory with runnable code
- ARCHITECTURE.md explaining design decisions
- DEVICES.md listing supported devices
- CONTRIBUTING.md for contributors

### Developer Experience
- GitHub issue templates (bug reports, feature requests)
- CI/CD with GitHub Actions
- Automated testing and linting
- Module documentation at pkg.go.dev

[0.1.5]: https://github.com/tj-smith47/shelly-go/releases/tag/v0.1.5
[0.1.0]: https://github.com/tj-smith47/shelly-go/releases/tag/v0.1.0
