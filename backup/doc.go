// Package backup provides device configuration backup and restore functionality.
//
// This package enables exporting a device's complete configuration to JSON
// format and restoring it later, either to the same device or a replacement
// device. This is useful for:
//   - Disaster recovery
//   - Device replacement/migration
//   - Configuration templating
//   - Fleet management
//
// # Basic Usage
//
// Export a device's configuration:
//
//	client := rpc.NewClient(transport)
//	b := backup.New(client)
//
//	// Export to JSON
//	data, err := b.Export(ctx, nil)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Save to file
//	os.WriteFile("backup.json", data, 0644)
//
// Restore configuration to a device:
//
//	// Load from file
//	data, _ := os.ReadFile("backup.json")
//
//	// Restore to device
//	err := b.Restore(ctx, data, nil)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// # Export Options
//
// Control what gets exported:
//
//	opts := &backup.ExportOptions{
//	    IncludeWiFi:    true,  // Include WiFi credentials
//	    IncludeCloud:   true,  // Include Cloud settings
//	    IncludeAuth:    false, // Exclude authentication (security)
//	    IncludeScripts: true,  // Include script code
//	    IncludeKVS:     true,  // Include KVS data
//	}
//	data, err := b.Export(ctx, opts)
//
// # Restore Options
//
// Control what gets restored:
//
//	opts := &backup.RestoreOptions{
//	    RestoreWiFi:    true,  // Restore WiFi config
//	    RestoreCloud:   true,  // Restore Cloud settings
//	    RestoreAuth:    false, // Skip auth (set separately)
//	    RestoreScripts: true,  // Restore scripts
//	    RestoreKVS:     true,  // Restore KVS data
//	    DryRun:         true,  // Validate only, don't apply
//	}
//	err := b.Restore(ctx, data, opts)
//
// # Device Migration
//
// Migrate configuration from one device to another:
//
//	// Export from old device
//	oldClient := rpc.NewClient(oldTransport)
//	oldBackup := backup.New(oldClient)
//	data, _ := oldBackup.Export(ctx, nil)
//
//	// Restore to new device
//	newClient := rpc.NewClient(newTransport)
//	newBackup := backup.New(newClient)
//	err := newBackup.Restore(ctx, data, nil)
//
// # Security Considerations
//
// Backup files may contain sensitive information:
//   - WiFi passwords (if IncludeWiFi is true)
//   - Cloud credentials
//   - Authentication passwords (if IncludeAuth is true)
//   - API keys in KVS
//
// Handle backup files securely:
//   - Encrypt backup files at rest
//   - Don't commit backups to version control
//   - Use secure channels for transmission
//   - Consider excluding sensitive data from backups
//
// # Backup Format
//
// Backups are stored as JSON with the following structure:
//
//	{
//	    "version": 1,
//	    "created_at": "2024-01-15T10:30:00Z",
//	    "device_info": { ... },
//	    "config": { ... },
//	    "scripts": [ ... ],
//	    "kvs": { ... }
//	}
//
// The format is designed to be forward-compatible. Newer versions
// can read backups from older versions.
package backup
