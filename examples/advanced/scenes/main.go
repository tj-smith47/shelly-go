// Example: scenes demonstrates scene management for coordinated device control.
//
// This example shows how to:
//   - Create and configure scenes
//   - Add devices and actions to scenes
//   - Activate scenes to control multiple devices
//   - Serialize/deserialize scenes to JSON
//   - Link devices after loading scenes
//
// Usage:
//
//	go run main.go -addresses 192.168.1.100,192.168.1.101,192.168.1.102
//	go run main.go -addresses 192.168.1.100 -demo  # Run demonstration
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/tj-smith47/shelly-go/factory"
	"github.com/tj-smith47/shelly-go/helpers"
)

func main() {
	// Parse command line flags
	addresses := flag.String("addresses", "", "Comma-separated list of device IP addresses")
	user := flag.String("user", "", "Username for authentication (optional)")
	pass := flag.String("pass", "", "Password for authentication (optional)")
	demoScenes := flag.Bool("demo", false, "Run scene demonstration")
	saveScene := flag.String("save", "", "Save scene to JSON file")
	loadScene := flag.String("load", "", "Load and activate scene from JSON file")
	flag.Parse()

	// Check for credentials from environment
	if *user == "" {
		*user = os.Getenv("SHELLY_USER")
	}
	if *pass == "" {
		*pass = os.Getenv("SHELLY_PASS")
	}

	// Need addresses
	if *addresses == "" && *loadScene == "" {
		fmt.Println("Error: -addresses is required (or -load to load a scene)")
		fmt.Println()
		fmt.Println("Usage:")
		fmt.Println("  go run main.go -addresses 192.168.1.100,192.168.1.101")
		fmt.Println("  go run main.go -addresses 192.168.1.100 -demo")
		fmt.Println("  go run main.go -addresses 192.168.1.100,192.168.1.101 -save movie-night.json")
		fmt.Println("  go run main.go -addresses 192.168.1.100,192.168.1.101 -load movie-night.json")
		os.Exit(1)
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	fmt.Println("Shelly Scene Management Example")
	fmt.Println("================================")
	fmt.Println()

	// Get devices
	var devices []factory.Device

	if *addresses != "" {
		addrs := strings.Split(*addresses, ",")
		for i := range addrs {
			addrs[i] = strings.TrimSpace(addrs[i])
		}

		fmt.Printf("Creating devices from %d address(es)...\n", len(addrs))

		var opts []factory.Option
		if *user != "" && *pass != "" {
			opts = append(opts, factory.WithAuth(*user, *pass))
		}
		opts = append(opts, factory.WithTimeout(10*time.Second))

		var errs []error
		devices, errs = factory.BatchFromAddresses(addrs, opts...)

		for i, err := range errs {
			if err != nil {
				fmt.Printf("Warning: Failed to create device for %s: %v\n", addrs[i], err)
			}
		}

		// Filter out nil devices
		var validDevices []factory.Device
		for _, dev := range devices {
			if dev != nil {
				validDevices = append(validDevices, dev)
			}
		}
		devices = validDevices
	}

	if len(devices) == 0 && *loadScene == "" {
		fmt.Println("No devices available.")
		return
	}

	fmt.Printf("Found %d device(s)\n", len(devices))
	for i, dev := range devices {
		fmt.Printf("  %d. %s\n", i+1, dev.Address())
	}
	fmt.Println()

	// Handle different modes
	switch {
	case *loadScene != "":
		loadAndActivateScene(ctx, *loadScene, devices)
	case *saveScene != "":
		createAndSaveScene(*saveScene, devices)
	case *demoScenes:
		demonstrateScenes(ctx, devices)
	default:
		showSceneExamples(devices)
	}

	fmt.Println("\nScene management example completed!")
}

// demonstrateScenes demonstrates scene creation and activation.
func demonstrateScenes(ctx context.Context, devices []factory.Device) {
	if len(devices) == 0 {
		fmt.Println("No devices available for demonstration")
		return
	}

	fmt.Println("--- Demonstrating Scene Operations ---")
	fmt.Println()

	// Create "All Off" scene
	allOffScene := helpers.NewScene("All Off")
	for _, dev := range devices {
		allOffScene.AddAction(dev, helpers.ActionSet(false))
	}
	fmt.Printf("Created scene: '%s' with %d action(s)\n", allOffScene.Name(), allOffScene.Len())

	// Create "All On" scene
	allOnScene := helpers.NewScene("All On")
	for _, dev := range devices {
		allOnScene.AddAction(dev, helpers.ActionSet(true))
	}
	fmt.Printf("Created scene: '%s' with %d action(s)\n", allOnScene.Name(), allOnScene.Len())

	// Create "Dim" scene (for lights, will only work on light devices)
	dimScene := helpers.NewScene("Dim Lights")
	for _, dev := range devices {
		dimScene.AddAction(dev, helpers.ActionSetBrightness(30))
	}
	fmt.Printf("Created scene: '%s' with %d action(s)\n", dimScene.Name(), dimScene.Len())

	fmt.Println()

	// Activate "All Off"
	fmt.Println("1. Activating 'All Off' scene...")
	results := allOffScene.Activate(ctx)
	printSceneResults(results)

	time.Sleep(2 * time.Second)

	// Activate "All On"
	fmt.Println("\n2. Activating 'All On' scene...")
	results = allOnScene.Activate(ctx)
	printSceneResults(results)

	time.Sleep(2 * time.Second)

	// Activate "All Off" to restore
	fmt.Println("\n3. Activating 'All Off' scene (restore)...")
	results = allOffScene.Activate(ctx)
	printSceneResults(results)

	// Show JSON serialization
	fmt.Println("\n--- Scene Serialization ---")
	fmt.Println()

	jsonData, err := allOnScene.ToJSON()
	if err != nil {
		fmt.Printf("Failed to serialize scene: %v\n", err)
	} else {
		var prettyJSON map[string]any
		if err := json.Unmarshal(jsonData, &prettyJSON); err != nil {
			fmt.Printf("Failed to unmarshal JSON: %v\n", err)
		} else {
			pretty, err := json.MarshalIndent(prettyJSON, "", "  ")
			if err != nil {
				fmt.Printf("Failed to marshal pretty JSON: %v\n", err)
			} else {
				fmt.Println("Scene JSON:")
				fmt.Println(string(pretty))
			}
		}
	}
}

// showSceneExamples shows examples of scene operations without executing.
func showSceneExamples(devices []factory.Device) {
	fmt.Println("--- Scene API Examples ---")
	fmt.Println()

	fmt.Println("Creating a scene:")
	fmt.Println("   scene := helpers.NewScene(\"Movie Night\")")
	fmt.Println()

	fmt.Println("Adding actions:")
	fmt.Println("   // Turn off main lights")
	fmt.Println("   scene.AddAction(mainLight, helpers.ActionSet(false))")
	fmt.Println()
	fmt.Println("   // Dim ambient light")
	fmt.Println("   scene.AddAction(ambientLight, helpers.ActionSetBrightness(30))")
	fmt.Println()
	fmt.Println("   // Toggle backlight")
	fmt.Println("   scene.AddAction(backlight, helpers.ActionToggle())")
	fmt.Println()

	fmt.Println("Available action types:")
	fmt.Println("   helpers.ActionSet(true)          // Turn on")
	fmt.Println("   helpers.ActionSet(false)         // Turn off")
	fmt.Println("   helpers.ActionToggle()           // Toggle state")
	fmt.Println("   helpers.ActionSetBrightness(75)  // Set brightness 0-100")
	fmt.Println()

	fmt.Println("Activating a scene:")
	fmt.Println("   results := scene.Activate(ctx)")
	fmt.Println("   if results.AllSuccessful() {")
	fmt.Println("       fmt.Println(\"Scene activated successfully\")")
	fmt.Println("   }")
	fmt.Println()

	fmt.Println("Scene management:")
	fmt.Println("   scene.Name()                  // Get scene name")
	fmt.Println("   scene.SetName(\"New Name\")     // Set scene name")
	fmt.Println("   scene.Len()                   // Number of actions")
	fmt.Println("   scene.Actions()               // Get all actions")
	fmt.Println("   scene.RemoveAction(address)   // Remove action by device address")
	fmt.Println("   scene.Clear()                 // Remove all actions")
	fmt.Println()

	fmt.Println("Serialization:")
	fmt.Println("   // Save to JSON")
	fmt.Println("   jsonData, _ := scene.ToJSON()")
	fmt.Println()
	fmt.Println("   // Load from JSON")
	fmt.Println("   scene, _ := helpers.SceneFromJSON(jsonData)")
	fmt.Println()
	fmt.Println("   // Link devices after loading")
	fmt.Println("   scene.LinkDevices(devices)")
	fmt.Println()
	fmt.Println("   // Check for unlinked devices")
	fmt.Println("   unlinked := scene.UnlinkedAddresses()")
	fmt.Println()

	// Example scene for the current devices
	if len(devices) > 0 {
		fmt.Println("--- Example Scene for Current Devices ---")
		fmt.Println()

		scene := helpers.NewScene("Example Scene")
		for i, dev := range devices {
			if i%2 == 0 {
				scene.AddAction(dev, helpers.ActionSet(true))
			} else {
				scene.AddAction(dev, helpers.ActionSet(false))
			}
		}

		jsonData, err := scene.ToJSON()
		if err != nil {
			fmt.Printf("Failed to serialize scene: %v\n", err)
		} else {
			var prettyJSON map[string]any
			if err := json.Unmarshal(jsonData, &prettyJSON); err != nil {
				fmt.Printf("Failed to unmarshal JSON: %v\n", err)
			} else {
				pretty, err := json.MarshalIndent(prettyJSON, "", "  ")
				if err != nil {
					fmt.Printf("Failed to marshal pretty JSON: %v\n", err)
				} else {
					fmt.Println(string(pretty))
				}
			}
		}
	}

	fmt.Println()
	fmt.Println("Run with -demo flag to execute scene operations.")
	fmt.Println("Run with -save filename.json to save a scene.")
	fmt.Println("Run with -load filename.json to load and activate a scene.")
}

// createAndSaveScene creates a sample scene and saves it to a file.
func createAndSaveScene(filename string, devices []factory.Device) {
	fmt.Printf("Creating and saving scene to '%s'...\n", filename)
	fmt.Println()

	// Create a sample scene
	scene := helpers.NewScene("Saved Scene")

	// Add alternating on/off actions
	for i, dev := range devices {
		if i%2 == 0 {
			scene.AddAction(dev, helpers.ActionSet(true))
		} else {
			scene.AddAction(dev, helpers.ActionSet(false))
		}
	}

	// Serialize to JSON
	jsonData, err := scene.ToJSON()
	if err != nil {
		log.Fatalf("Failed to serialize scene: %v", err)
	}

	// Pretty print
	var prettyJSON map[string]any
	if unmarshalErr := json.Unmarshal(jsonData, &prettyJSON); unmarshalErr != nil {
		log.Fatalf("Failed to unmarshal JSON: %v", unmarshalErr)
	}
	pretty, err := json.MarshalIndent(prettyJSON, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal pretty JSON: %v", err)
	}

	// Save to file (0o600 for user-only access as recommended by security best practices)
	err = os.WriteFile(filename, pretty, 0o600)
	if err != nil {
		log.Fatalf("Failed to save scene: %v", err)
	}

	fmt.Printf("Scene saved to '%s'\n", filename)
	fmt.Println()
	fmt.Println("Contents:")
	fmt.Println(string(pretty))
}

// loadAndActivateScene loads a scene from file and activates it.
func loadAndActivateScene(ctx context.Context, filename string, devices []factory.Device) {
	fmt.Printf("Loading scene from '%s'...\n", filename)
	fmt.Println()

	// Read file
	jsonData, err := os.ReadFile(filename)
	if err != nil {
		log.Fatalf("Failed to read scene file: %v", err)
	}

	// Parse scene
	scene, err := helpers.SceneFromJSON(jsonData)
	if err != nil {
		log.Fatalf("Failed to parse scene: %v", err)
	}

	fmt.Printf("Loaded scene: '%s' with %d action(s)\n", scene.Name(), scene.Len())

	// Link devices
	scene.LinkDevices(devices)

	// Check for unlinked devices
	unlinked := scene.UnlinkedAddresses()
	if len(unlinked) > 0 {
		fmt.Println()
		fmt.Printf("Warning: %d device(s) not found:\n", len(unlinked))
		for _, addr := range unlinked {
			fmt.Printf("  - %s\n", addr)
		}
		fmt.Println("These actions will fail.")
	}

	// Activate scene
	fmt.Println()
	fmt.Println("Activating scene...")
	results := scene.Activate(ctx)
	printSceneResults(results)
}

// printSceneResults prints the results of a scene activation.
func printSceneResults(results helpers.SceneResults) {
	if results.AllSuccessful() {
		fmt.Printf("   All %d action(s) succeeded\n", len(results))
	} else {
		successes := 0
		for _, r := range results {
			if r.Success {
				successes++
			}
		}
		failures := results.Failures()
		fmt.Printf("   %d succeeded, %d failed\n", successes, len(failures))
		for _, failure := range failures {
			fmt.Printf("     - %s (%s): %v\n",
				failure.DeviceAddress,
				failure.Action.Type,
				failure.Error)
		}
	}
}
