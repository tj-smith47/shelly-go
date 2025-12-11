// Example: custom_component demonstrates creating custom component types.
//
// This example shows how to:
//   - Extend the base component to create custom components
//   - Access raw RPC methods for advanced operations
//   - Create type-safe wrappers for device-specific features
//   - Work with RawFields for forward compatibility
//
// This is useful when:
//   - You need to access device-specific methods not in the standard API
//   - You want to create type-safe wrappers for specific device types
//   - You need to access new firmware features before library updates
//
// Usage:
//
//	go run main.go -host 192.168.1.100 [-user admin] [-pass password]
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/tj-smith47/shelly-go/gen2"
	"github.com/tj-smith47/shelly-go/rpc"
	"github.com/tj-smith47/shelly-go/transport"
)

func main() {
	// Parse command line flags
	host := flag.String("host", "", "Device IP address or hostname (required)")
	user := flag.String("user", "", "Username for authentication (optional)")
	pass := flag.String("pass", "", "Password for authentication (optional)")
	flag.Parse()

	// Check for required host flag or environment variable
	if *host == "" {
		*host = os.Getenv("SHELLY_HOST")
	}
	if *host == "" {
		fmt.Println("Error: -host flag or SHELLY_HOST environment variable is required")
		flag.Usage()
		os.Exit(1)
	}

	// Check for auth from environment if not provided via flags
	if *user == "" {
		*user = os.Getenv("SHELLY_USER")
	}
	if *pass == "" {
		*pass = os.Getenv("SHELLY_PASS")
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create HTTP transport
	transportURL := "http://" + *host
	var transportOpts []transport.Option
	transportOpts = append(transportOpts, transport.WithTimeout(10*time.Second))

	if *user != "" && *pass != "" {
		transportOpts = append(transportOpts, transport.WithAuth(*user, *pass))
	}

	httpTransport := transport.NewHTTP(transportURL, transportOpts...)

	// Create RPC client
	var client *rpc.Client
	if *user != "" && *pass != "" {
		auth := &rpc.AuthData{
			Realm:    "shelly",
			Username: *user,
			Password: *pass,
		}
		client = rpc.NewClientWithAuth(httpTransport, auth)
	} else {
		client = rpc.NewClient(httpTransport)
	}
	defer client.Close()

	// Create Gen2 device
	device := gen2.NewDevice(client)

	fmt.Println("Custom Component Example")
	fmt.Println("========================")
	fmt.Println()

	// Get device info
	fmt.Println("Connecting to device...")
	info, err := device.Shelly().GetDeviceInfo(ctx)
	if err != nil {
		log.Printf("Failed to get device info: %v", err)
		return
	}
	fmt.Printf("Connected to: %s (%s)\n", info.Name, info.Model)
	fmt.Println()

	// Demonstrate custom component patterns
	demonstrateCustomComponent(ctx, client)
	demonstrateRawRPCCalls(ctx, client)
	demonstrateKVS(ctx, client)

	fmt.Println("\nCustom component example completed!")
}

// CustomThermostat is an example custom component for thermostat devices.
// This demonstrates how to create type-safe wrappers for device-specific features.
type CustomThermostat struct {
	*gen2.BaseComponent
}

// ThermostatConfig represents thermostat configuration.
type ThermostatConfig struct {
	Enable         *bool    `json:"enable,omitempty"`
	TargetC        *float64 `json:"target_C,omitempty"`
	TargetF        *float64 `json:"target_F,omitempty"`
	CurrentC       *float64 `json:"current_C,omitempty"`
	CurrentF       *float64 `json:"current_F,omitempty"`
	OutputEnabled  *bool    `json:"output_enabled,omitempty"`
	Schedule       *bool    `json:"schedule,omitempty"`
	ScheduleHidden *bool    `json:"schedule_hidden,omitempty"`
	ID             int      `json:"id"`
}

// ThermostatStatus represents thermostat status.
type ThermostatStatus struct {
	Errors        []string `json:"errors,omitempty"`
	ID            int      `json:"id"`
	TargetC       float64  `json:"target_C"`
	CurrentC      float64  `json:"current_C"`
	Enable        bool     `json:"enable"`
	OutputEnabled bool     `json:"output_enabled"`
}

// NewCustomThermostat creates a new custom thermostat component.
func NewCustomThermostat(client *rpc.Client, id int) *CustomThermostat {
	return &CustomThermostat{
		BaseComponent: gen2.NewBaseComponent(client, "thermostat", id),
	}
}

// SetTargetTemperature sets the target temperature in Celsius.
func (t *CustomThermostat) SetTargetTemperature(ctx context.Context, tempC float64) error {
	params := map[string]any{
		"id":       t.ID(),
		"target_C": tempC,
	}
	_, err := t.Client().Call(ctx, "Thermostat.SetConfig", params)
	return err
}

// Enable enables or disables the thermostat.
func (t *CustomThermostat) Enable(ctx context.Context, enable bool) error {
	params := map[string]any{
		"id":     t.ID(),
		"enable": enable,
	}
	_, err := t.Client().Call(ctx, "Thermostat.SetConfig", params)
	return err
}

// GetStatus returns the thermostat status.
func (t *CustomThermostat) GetStatus(ctx context.Context) (*ThermostatStatus, error) {
	params := map[string]any{
		"id": t.ID(),
	}
	result, err := t.Client().Call(ctx, "Thermostat.GetStatus", params)
	if err != nil {
		return nil, err
	}

	var status ThermostatStatus
	if err := json.Unmarshal(result, &status); err != nil {
		return nil, err
	}
	return &status, nil
}

// demonstrateCustomComponent shows how to use custom components.
func demonstrateCustomComponent(ctx context.Context, client *rpc.Client) {
	fmt.Println("--- Custom Component Pattern ---")
	fmt.Println()

	fmt.Println("Creating custom components allows you to:")
	fmt.Println("  - Add type-safe methods for device-specific features")
	fmt.Println("  - Create domain-specific abstractions")
	fmt.Println("  - Support new firmware features before library updates")
	fmt.Println()

	fmt.Println("Example: Custom Thermostat Component")
	fmt.Println()
	fmt.Println("```go")
	fmt.Println("type CustomThermostat struct {")
	fmt.Println("    *gen2.BaseComponent")
	fmt.Println("}")
	fmt.Println()
	fmt.Println("func NewCustomThermostat(client *rpc.Client, id int) *CustomThermostat {")
	fmt.Println("    return &CustomThermostat{")
	fmt.Println("        BaseComponent: gen2.NewBaseComponent(client, \"thermostat\", id),")
	fmt.Println("    }")
	fmt.Println("}")
	fmt.Println()
	fmt.Println("func (t *CustomThermostat) SetTargetTemperature(ctx context.Context, tempC float64) error {")
	fmt.Println("    params := map[string]any{")
	fmt.Println("        \"id\": t.ID(),")
	fmt.Println("        \"target_C\": tempC,")
	fmt.Println("    }")
	fmt.Println("    _, err := t.Client().Call(ctx, \"Thermostat.SetConfig\", params)")
	fmt.Println("    return err")
	fmt.Println("}")
	fmt.Println("```")
	fmt.Println()

	// Try to get thermostat status (will fail if device doesn't have one)
	thermostat := NewCustomThermostat(client, 0)
	status, err := thermostat.GetStatus(ctx)
	if err != nil {
		fmt.Printf("Note: This device doesn't have a thermostat component\n")
		fmt.Printf("      (Error: %v)\n", err)
	} else {
		fmt.Printf("Thermostat status: enabled=%v, target=%.1fC, current=%.1fC\n",
			status.Enable, status.TargetC, status.CurrentC)
	}
	fmt.Println()
}

// demonstrateRawRPCCalls shows how to make raw RPC calls.
func demonstrateRawRPCCalls(ctx context.Context, client *rpc.Client) {
	fmt.Println("--- Raw RPC Calls ---")
	fmt.Println()

	fmt.Println("For advanced operations, you can make raw RPC calls:")
	fmt.Println()

	// Get system status (works on any Gen2+ device)
	fmt.Println("1. Getting system status with raw RPC call...")
	result, err := client.Call(ctx, "Sys.GetStatus", nil)
	if err != nil {
		fmt.Printf("   Error: %v\n", err)
	} else {
		var sysStatus map[string]any
		if unmarshalErr := json.Unmarshal(result, &sysStatus); unmarshalErr != nil {
			fmt.Printf("   Error unmarshaling result: %v\n", unmarshalErr)
		} else {
			fmt.Println("   System Status:")
			if mac, ok := sysStatus["mac"].(string); ok {
				fmt.Printf("     MAC: %s\n", mac)
			}
			if uptime, ok := sysStatus["uptime"].(float64); ok {
				fmt.Printf("     Uptime: %.0f seconds\n", uptime)
			}
			if ramFree, ok := sysStatus["ram_free"].(float64); ok {
				fmt.Printf("     RAM Free: %.0f bytes\n", ramFree)
			}
			if wakeupReason, ok := sysStatus["wakeup_reason"].(map[string]any); ok {
				if reason, ok := wakeupReason["cause"].(string); ok {
					fmt.Printf("     Wake Reason: %s\n", reason)
				}
			}
		}
	}
	fmt.Println()

	// List all methods
	fmt.Println("2. Listing available RPC methods...")
	result, err = client.Call(ctx, "Shelly.ListMethods", nil)
	if err != nil {
		fmt.Printf("   Error: %v\n", err)
	} else {
		var methodsResp struct {
			Methods []string `json:"methods"`
		}
		if unmarshalErr := json.Unmarshal(result, &methodsResp); unmarshalErr != nil {
			fmt.Printf("   Error unmarshaling result: %v\n", unmarshalErr)
		} else {
			fmt.Printf("   Available methods (%d):\n", len(methodsResp.Methods))
			// Show first 10
			for i, method := range methodsResp.Methods {
				if i >= 10 {
					fmt.Printf("     ... and %d more\n", len(methodsResp.Methods)-10)
					break
				}
				fmt.Printf("     - %s\n", method)
			}
		}
	}
	fmt.Println()

	// Get full component status
	fmt.Println("3. Getting full device status...")
	result, err = client.Call(ctx, "Shelly.GetStatus", nil)
	if err != nil {
		fmt.Printf("   Error: %v\n", err)
	} else {
		var fullStatus map[string]any
		if err := json.Unmarshal(result, &fullStatus); err != nil {
			fmt.Printf("   Error unmarshaling result: %v\n", err)
		} else {
			fmt.Println("   Components present:")
			for key := range fullStatus {
				fmt.Printf("     - %s\n", key)
			}
		}
	}
	fmt.Println()
}

// demonstrateKVS shows how to use Key-Value Storage.
func demonstrateKVS(ctx context.Context, client *rpc.Client) {
	fmt.Println("--- Key-Value Storage (KVS) ---")
	fmt.Println()

	fmt.Println("Gen2+ devices have built-in key-value storage:")
	fmt.Println()

	// Set a value
	testKey := "shelly_go_example"
	testValue := map[string]any{
		"timestamp": time.Now().Unix(),
		"message":   "Hello from shelly-go!",
		"version":   "1.0.0",
	}

	fmt.Printf("1. Setting KVS value for key '%s'...\n", testKey)
	valueJSON, err := json.Marshal(testValue)
	if err != nil {
		fmt.Printf("   Error marshaling value: %v\n", err)
		return
	}
	params := map[string]any{
		"key":   testKey,
		"value": string(valueJSON),
	}
	_, err = client.Call(ctx, "KVS.Set", params)
	if err != nil {
		fmt.Printf("   Error: %v\n", err)
	} else {
		fmt.Println("   Value set successfully")
	}

	// Get the value back
	fmt.Printf("\n2. Getting KVS value for key '%s'...\n", testKey)
	params = map[string]any{
		"key": testKey,
	}
	result, err := client.Call(ctx, "KVS.Get", params)
	if err != nil {
		fmt.Printf("   Error: %v\n", err)
	} else {
		var kvsResp struct {
			Value string `json:"value"`
		}
		if unmarshalErr := json.Unmarshal(result, &kvsResp); unmarshalErr != nil {
			fmt.Printf("   Error unmarshaling result: %v\n", unmarshalErr)
		} else {
			fmt.Printf("   Value: %s\n", kvsResp.Value)
		}
	}

	// List all keys
	fmt.Println("\n3. Listing all KVS keys...")
	result, err = client.Call(ctx, "KVS.GetMany", nil)
	if err != nil {
		fmt.Printf("   Error: %v\n", err)
	} else {
		var kvsResp struct {
			Items map[string]string `json:"items"`
		}
		if unmarshalErr := json.Unmarshal(result, &kvsResp); unmarshalErr != nil {
			fmt.Printf("   Error unmarshaling result: %v\n", unmarshalErr)
		} else {
			if len(kvsResp.Items) == 0 {
				fmt.Println("   No keys stored")
			} else {
				fmt.Printf("   Keys: %d\n", len(kvsResp.Items))
				for key := range kvsResp.Items {
					fmt.Printf("     - %s\n", key)
				}
			}
		}
	}

	// Delete the test key
	fmt.Printf("\n4. Deleting key '%s'...\n", testKey)
	params = map[string]any{
		"key": testKey,
	}
	_, err = client.Call(ctx, "KVS.Delete", params)
	if err != nil {
		fmt.Printf("   Error: %v\n", err)
	} else {
		fmt.Println("   Key deleted")
	}
	fmt.Println()
}
