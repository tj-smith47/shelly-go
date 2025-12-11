// Package helpers provides convenience utilities for working with Shelly devices.
//
// The helpers package offers high-level operations that span multiple devices,
// including batch operations, device groups, scene management, and scheduling.
//
// Batch Operations:
//
//	// Turn off multiple switches
//	results := helpers.BatchSet(ctx, devices, false)
//
//	// Toggle all switches
//	results := helpers.BatchToggle(ctx, devices)
//
//	// Turn off all devices in a group
//	group := helpers.NewGroup("Living Room", device1, device2, device3)
//	helpers.AllOff(ctx, group)
//
// Device Groups:
//
//	// Create a group of devices
//	group := helpers.NewGroup("Kitchen Lights",
//	    helpers.WithDevice(light1),
//	    helpers.WithDevice(light2),
//	)
//
//	// Operate on all devices in the group
//	group.AllOn(ctx)
//	group.AllOff(ctx)
//	group.SetBrightness(ctx, 75)
//
// Scene Management:
//
//	// Define a scene
//	scene := helpers.NewScene("Movie Night")
//	scene.AddAction(livingRoomLight, helpers.ActionSet(false))
//	scene.AddAction(tvBacklight, helpers.ActionSetBrightness(30))
//
//	// Activate the scene
//	scene.Activate(ctx)
//
//	// Save/load scenes to JSON
//	data, _ := scene.ToJSON()
//	scene, _ := helpers.SceneFromJSON(data)
package helpers
