// Package main demonstrates how to retrieve and analyze historical energy data
// from Shelly Pro 3EM and Pro EM devices using the EMData and EM1Data components.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/tj-smith47/shelly-go/gen2/components"
	"github.com/tj-smith47/shelly-go/rpc"
)

func main() {
	// Parse command-line flags
	addr := flag.String("addr", "", "Device IP address or hostname (required)")
	deviceType := flag.String("type", "3em", "Device type: '3em' for Pro 3EM (3-phase) or 'em' for Pro EM (single-phase)")
	hours := flag.Int("hours", 24, "Number of hours of historical data to retrieve")
	csvExport := flag.Bool("csv", false, "Print CSV download URL instead of analyzing data")
	flag.Parse()

	if *addr == "" {
		log.Fatal("Error: --addr flag is required (e.g., --addr 192.168.1.100)")
	}

	ctx := context.Background()

	// Connect to device using the convenience constructor
	client, err := rpc.NewHTTPClient(*addr)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Calculate time range
	endTS := time.Now().Unix()
	startTS := endTS - int64(*hours*3600)

	// Process based on device type
	switch *deviceType {
	case "3em":
		process3PhaseData(ctx, client, *addr, startTS, endTS, *csvExport)
	case "em":
		processSinglePhaseData(ctx, client, *addr, startTS, endTS, *csvExport)
	default:
		log.Fatalf("Invalid device type: %s (use '3em' or 'em')", *deviceType)
	}
}

// process3PhaseData handles 3-phase energy data from Shelly Pro 3EM
func process3PhaseData(ctx context.Context, client *rpc.Client, addr string, startTS, endTS int64, csvOnly bool) {
	emdata := components.NewEMData(client, 0)

	// If CSV export requested, just print URL
	if csvOnly {
		csvURL := emdata.GetDataCSVURL(addr, &startTS, &endTS, true)
		fmt.Println("CSV Download URL:")
		fmt.Println(csvURL)
		fmt.Println("\nDownload with curl:")
		fmt.Printf("curl -OJ \"%s\"\n", csvURL)
		return
	}

	// Get status first
	fmt.Println("=== EMData Status ===")
	status, err := emdata.GetStatus(ctx)
	if err != nil {
		log.Fatalf("Failed to get status: %v", err)
	}

	if status.LastRecordID != nil {
		fmt.Printf("Last Record ID: %d\n", *status.LastRecordID)
	}
	if status.AvailableRecords != nil {
		fmt.Printf("Available Records: %d\n", *status.AvailableRecords)
	}
	if len(status.Errors) > 0 {
		fmt.Printf("Errors: %v\n", status.Errors)
	}
	fmt.Println()

	// Get available records
	fmt.Println("=== Available Records ===")
	records, err := emdata.GetRecords(ctx, &startTS)
	if err != nil {
		log.Fatalf("Failed to get records: %v", err)
	}

	for _, record := range records.Records {
		fmt.Printf("Record %d: %d data points from %s (period: %ds)\n",
			record.ID, record.Count,
			time.Unix(record.TS, 0).Format("2006-01-02 15:04:05"),
			record.Period)
	}
	fmt.Println()

	// Get actual data
	fmt.Println("=== Retrieving Historical Data ===")
	data, err := emdata.GetData(ctx, &startTS, &endTS)
	if err != nil {
		log.Fatalf("Failed to get data: %v", err)
	}

	// Analyze data
	var (
		totalSamples     int
		totalEnergyWh    float64
		peakPowerW       float64
		peakPowerTime    time.Time
		phaseAEnergyWh   float64
		phaseBEnergyWh   float64
		phaseCEnergyWh   float64
		avgPhaseAVoltage float64
		avgPhaseBVoltage float64
		avgPhaseCVoltage float64
		voltageCount     int
	)

	for _, block := range data.Data {
		blockTime := time.Unix(block.TS, 0)
		periodSec := float64(block.Period)

		for _, values := range block.Values {
			totalSamples++

			// Calculate energy consumption (Power * Time / 3600 = Wh)
			energyWh := values.TotalActivePower * periodSec / 3600.0
			totalEnergyWh += energyWh

			// Per-phase energy
			phaseAEnergyWh += values.AActivePower * periodSec / 3600.0
			phaseBEnergyWh += values.BActivePower * periodSec / 3600.0
			phaseCEnergyWh += values.CActivePower * periodSec / 3600.0

			// Track peak power
			if values.TotalActivePower > peakPowerW {
				peakPowerW = values.TotalActivePower
				peakPowerTime = blockTime
			}

			// Average voltage
			avgPhaseAVoltage += values.AVoltage
			avgPhaseBVoltage += values.BVoltage
			avgPhaseCVoltage += values.CVoltage
			voltageCount++
		}
	}

	// Calculate averages
	if voltageCount > 0 {
		avgPhaseAVoltage /= float64(voltageCount)
		avgPhaseBVoltage /= float64(voltageCount)
		avgPhaseCVoltage /= float64(voltageCount)
	}

	// Display results
	fmt.Printf("=== Energy Analysis (Last %d hours) ===\n", (endTS-startTS)/3600)
	fmt.Printf("Total Samples: %d\n", totalSamples)
	fmt.Printf("Total Energy Consumption: %.2f Wh (%.3f kWh)\n", totalEnergyWh, totalEnergyWh/1000.0)
	fmt.Printf("  Phase A: %.2f Wh (%.1f%%)\n", phaseAEnergyWh, phaseAEnergyWh/totalEnergyWh*100)
	fmt.Printf("  Phase B: %.2f Wh (%.1f%%)\n", phaseBEnergyWh, phaseBEnergyWh/totalEnergyWh*100)
	fmt.Printf("  Phase C: %.2f Wh (%.1f%%)\n", phaseCEnergyWh, phaseCEnergyWh/totalEnergyWh*100)
	fmt.Printf("Peak Power: %.2f W at %s\n", peakPowerW, peakPowerTime.Format("2006-01-02 15:04:05"))
	fmt.Printf("Average Power: %.2f W\n", totalEnergyWh/float64((endTS-startTS)/3600))
	fmt.Printf("Average Voltages: A=%.1fV, B=%.1fV, C=%.1fV\n", avgPhaseAVoltage, avgPhaseBVoltage, avgPhaseCVoltage)
	fmt.Println()

	// Show cost estimate (example rate: $0.15/kWh)
	costPerKWh := 0.15
	estimatedCost := (totalEnergyWh / 1000.0) * costPerKWh
	fmt.Printf("Estimated Cost (at $%.2f/kWh): $%.2f\n", costPerKWh, estimatedCost)
}

// processSinglePhaseData handles single-phase energy data from Shelly Pro EM
func processSinglePhaseData(ctx context.Context, client *rpc.Client, addr string, startTS, endTS int64, csvOnly bool) {
	em1data := components.NewEM1Data(client, 0)

	// If CSV export requested, just print URL
	if csvOnly {
		csvURL := em1data.GetDataCSVURL(addr, &startTS, &endTS, true)
		fmt.Println("CSV Download URL:")
		fmt.Println(csvURL)
		fmt.Println("\nDownload with curl:")
		fmt.Printf("curl -OJ \"%s\"\n", csvURL)
		return
	}

	// Get configuration
	fmt.Println("=== EM1Data Configuration ===")
	config, err := em1data.GetConfig(ctx)
	if err != nil {
		log.Fatalf("Failed to get config: %v", err)
	}

	if config.Name != nil {
		fmt.Printf("Name: %s\n", *config.Name)
	}
	if config.DataPeriod != nil {
		fmt.Printf("Data Period: %d seconds\n", *config.DataPeriod)
	}
	if config.DataStorageDays != nil {
		fmt.Printf("Storage Days: %d\n", *config.DataStorageDays)
	}
	fmt.Println()

	// Get status
	fmt.Println("=== EM1Data Status ===")
	status, err := em1data.GetStatus(ctx)
	if err != nil {
		log.Fatalf("Failed to get status: %v", err)
	}

	if status.LastRecordID != nil {
		fmt.Printf("Last Record ID: %d\n", *status.LastRecordID)
	}
	if status.AvailableRecords != nil {
		fmt.Printf("Available Records: %d\n", *status.AvailableRecords)
	}
	if len(status.Errors) > 0 {
		fmt.Printf("Errors: %v\n", status.Errors)
	}
	fmt.Println()

	// Get available records
	fmt.Println("=== Available Records ===")
	records, err := em1data.GetRecords(ctx, &startTS)
	if err != nil {
		log.Fatalf("Failed to get records: %v", err)
	}

	for _, record := range records.Records {
		fmt.Printf("Record %d: %d data points from %s (period: %ds)\n",
			record.ID, record.Count,
			time.Unix(record.TS, 0).Format("2006-01-02 15:04:05"),
			record.Period)
	}
	fmt.Println()

	// Get actual data
	fmt.Println("=== Retrieving Historical Data ===")
	data, err := em1data.GetData(ctx, &startTS, &endTS)
	if err != nil {
		log.Fatalf("Failed to get data: %v", err)
	}

	// Analyze data
	var (
		totalSamples  int
		totalEnergyWh float64
		peakPowerW    float64
		peakPowerTime time.Time
		avgVoltage    float64
		avgCurrent    float64
		avgPF         float64
		pfCount       int
	)

	for _, block := range data.Data {
		blockTime := time.Unix(block.TS, 0)
		periodSec := float64(block.Period)

		for _, values := range block.Values {
			totalSamples++

			// Calculate energy consumption (Power * Time / 3600 = Wh)
			energyWh := values.ActivePower * periodSec / 3600.0
			totalEnergyWh += energyWh

			// Track peak power
			if values.ActivePower > peakPowerW {
				peakPowerW = values.ActivePower
				peakPowerTime = blockTime
			}

			// Averages
			avgVoltage += values.Voltage
			avgCurrent += values.Current
			if values.PowerFactor != nil {
				avgPF += *values.PowerFactor
				pfCount++
			}
		}
	}

	// Calculate averages
	if totalSamples > 0 {
		avgVoltage /= float64(totalSamples)
		avgCurrent /= float64(totalSamples)
	}
	if pfCount > 0 {
		avgPF /= float64(pfCount)
	}

	// Display results
	fmt.Printf("=== Energy Analysis (Last %d hours) ===\n", (endTS-startTS)/3600)
	fmt.Printf("Total Samples: %d\n", totalSamples)
	fmt.Printf("Total Energy Consumption: %.2f Wh (%.3f kWh)\n", totalEnergyWh, totalEnergyWh/1000.0)
	fmt.Printf("Peak Power: %.2f W at %s\n", peakPowerW, peakPowerTime.Format("2006-01-02 15:04:05"))
	fmt.Printf("Average Power: %.2f W\n", totalEnergyWh/float64((endTS-startTS)/3600))
	fmt.Printf("Average Voltage: %.1f V\n", avgVoltage)
	fmt.Printf("Average Current: %.2f A\n", avgCurrent)
	if pfCount > 0 {
		fmt.Printf("Average Power Factor: %.3f\n", avgPF)
	}
	fmt.Println()

	// Show cost estimate (example rate: $0.15/kWh)
	costPerKWh := 0.15
	estimatedCost := (totalEnergyWh / 1000.0) * costPerKWh
	fmt.Printf("Estimated Cost (at $%.2f/kWh): $%.2f\n", costPerKWh, estimatedCost)
}
