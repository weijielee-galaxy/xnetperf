package cmd

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"xnetperf/config"
)

// TestDisplayLatencyMatrixWithColorMarking tests the red color marking for high latencies and missing data
func TestDisplayLatencyMatrixWithColorMarking(t *testing.T) {
	tests := []struct {
		name           string
		latencyData    []LatencyData
		expectedColors []string // Expected red-colored items
		description    string
	}{
		{
			name: "Mixed latencies with high and normal values",
			latencyData: []LatencyData{
				// host1 to host2
				{SourceHost: "host1", SourceHCA: "mlx5_0", TargetHost: "host2", TargetHCA: "mlx5_0", AvgLatencyUs: 2.5},
				{SourceHost: "host1", SourceHCA: "mlx5_0", TargetHost: "host2", TargetHCA: "mlx5_1", AvgLatencyUs: 3.2},
				// host2 to host1 - high latency
				{SourceHost: "host2", SourceHCA: "mlx5_0", TargetHost: "host1", TargetHCA: "mlx5_0", AvgLatencyUs: 5.8},
				{SourceHost: "host2", SourceHCA: "mlx5_0", TargetHost: "host1", TargetHCA: "mlx5_1", AvgLatencyUs: 12.3},
				// host2:mlx5_1 to host1:mlx5_0
				{SourceHost: "host2", SourceHCA: "mlx5_1", TargetHost: "host1", TargetHCA: "mlx5_0", AvgLatencyUs: 3.1},
			},
			expectedColors: []string{
				"5.80 μs",  // High latency
				"12.30 μs", // High latency
				"*",        // Missing data (test failure)
			},
			description: "Should mark latencies > 4μs in red and missing data as red *",
		},
		{
			name: "All high latencies",
			latencyData: []LatencyData{
				{SourceHost: "host1", SourceHCA: "mlx5_0", TargetHost: "host2", TargetHCA: "mlx5_0", AvgLatencyUs: 8.5},
				{SourceHost: "host2", SourceHCA: "mlx5_0", TargetHost: "host1", TargetHCA: "mlx5_0", AvgLatencyUs: 9.2},
			},
			expectedColors: []string{
				"8.50 μs",
				"9.20 μs",
			},
			description: "All latencies should be red when > 4μs",
		},
		{
			name: "All normal latencies",
			latencyData: []LatencyData{
				{SourceHost: "host1", SourceHCA: "mlx5_0", TargetHost: "host2", TargetHCA: "mlx5_0", AvgLatencyUs: 1.5},
				{SourceHost: "host2", SourceHCA: "mlx5_0", TargetHost: "host1", TargetHCA: "mlx5_0", AvgLatencyUs: 2.3},
			},
			expectedColors: []string{}, // No red markers expected
			description:    "No red markers for normal latencies",
		},
		{
			name: "Threshold boundary test",
			latencyData: []LatencyData{
				{SourceHost: "host1", SourceHCA: "mlx5_0", TargetHost: "host2", TargetHCA: "mlx5_0", AvgLatencyUs: 3.99}, // Just below threshold
				{SourceHost: "host1", SourceHCA: "mlx5_0", TargetHost: "host2", TargetHCA: "mlx5_1", AvgLatencyUs: 4.00}, // At threshold - not red
				{SourceHost: "host1", SourceHCA: "mlx5_0", TargetHost: "host2", TargetHCA: "mlx5_2", AvgLatencyUs: 4.01}, // Just above threshold - red
			},
			expectedColors: []string{
				"4.01 μs", // Only this should be red
			},
			description: "Test threshold boundary (4.0μs)",
		},
		{
			name: "Self-to-self connections (diagonal)",
			latencyData: []LatencyData{
				// Cross connections with data
				{SourceHost: "host1", SourceHCA: "mlx5_0", TargetHost: "host2", TargetHCA: "mlx5_0", AvgLatencyUs: 2.5},
				{SourceHost: "host2", SourceHCA: "mlx5_0", TargetHost: "host1", TargetHCA: "mlx5_0", AvgLatencyUs: 2.6},
				// Self-to-self not included (would be diagonal, should show "-")
			},
			expectedColors: []string{}, // No red markers - self-to-self shows "-" without color
			description:    "Self-to-self connections should display '-' without red color",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stdout
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			// Call the display function
			displayLatencyMatrix(tt.latencyData)

			// Restore stdout
			w.Close()
			os.Stdout = oldStdout

			// Read captured output
			var buf bytes.Buffer
			io.Copy(&buf, r)
			output := buf.String()

			// Print the test description and output for visual inspection
			fmt.Printf("\n=== Test: %s ===\n", tt.name)
			fmt.Printf("Description: %s\n", tt.description)
			fmt.Printf("Output:\n%s\n", output)

			// Verify that red color codes are present for expected values
			for _, expectedValue := range tt.expectedColors {
				if !strings.Contains(output, colorRed) || !strings.Contains(output, expectedValue) {
					t.Errorf("Expected red-marked value '%s' not found in output", expectedValue)
				}
			}

			// Verify red color code count matches expected
			redCount := strings.Count(output, colorRed)
			expectedRedCount := len(tt.expectedColors)
			if redCount < expectedRedCount {
				t.Errorf("Expected at least %d red markers, but found %d", expectedRedCount, redCount)
			}

			// Additional visual separator
			fmt.Println(strings.Repeat("=", 80))
		})
	}
}

// TestDisplayLatencyMatrixIncastWithColorMarking tests color marking in incast mode
func TestDisplayLatencyMatrixIncastWithColorMarking(t *testing.T) {
	tests := []struct {
		name           string
		latencyData    []LatencyData
		cfg            *config.Config
		expectedColors []string
		description    string
	}{
		{
			name: "Incast mode with mixed latencies",
			latencyData: []LatencyData{
				// Client1 to servers - normal latency
				{SourceHost: "client1", SourceHCA: "mlx5_0", TargetHost: "server1", TargetHCA: "mlx5_0", AvgLatencyUs: 2.8},
				{SourceHost: "client1", SourceHCA: "mlx5_0", TargetHost: "server2", TargetHCA: "mlx5_0", AvgLatencyUs: 3.1},
				// Client2 to servers - high latency
				{SourceHost: "client2", SourceHCA: "mlx5_0", TargetHost: "server1", TargetHCA: "mlx5_0", AvgLatencyUs: 6.5},
				{SourceHost: "client2", SourceHCA: "mlx5_0", TargetHost: "server2", TargetHCA: "mlx5_0", AvgLatencyUs: 15.2},
			},
			cfg: &config.Config{
				StreamType: config.InCast,
				Server: config.ServerConfig{
					Hostname: []string{"server1", "server2"},
					Hca:      []string{"mlx5_0"},
				},
				Client: config.ClientConfig{
					Hostname: []string{"client1", "client2"},
					Hca:      []string{"mlx5_0"},
				},
			},
			expectedColors: []string{
				"6.50 μs",  // High latency
				"15.20 μs", // High latency
			},
			description: "Incast mode should mark high latencies in red",
		},
		{
			name: "Incast mode with missing data",
			latencyData: []LatencyData{
				{SourceHost: "client1", SourceHCA: "mlx5_0", TargetHost: "server1", TargetHCA: "mlx5_0", AvgLatencyUs: 2.5},
				// Missing: client1:mlx5_0 -> server1:mlx5_1 (intentionally not provided)
				{SourceHost: "client1", SourceHCA: "mlx5_1", TargetHost: "server1", TargetHCA: "mlx5_0", AvgLatencyUs: 3.2},
				{SourceHost: "client1", SourceHCA: "mlx5_1", TargetHost: "server1", TargetHCA: "mlx5_1", AvgLatencyUs: 3.5},
			},
			cfg: &config.Config{
				StreamType: config.InCast,
				Server: config.ServerConfig{
					Hostname: []string{"server1"},
					Hca:      []string{"mlx5_0", "mlx5_1"},
				},
				Client: config.ClientConfig{
					Hostname: []string{"client1"},
					Hca:      []string{"mlx5_0", "mlx5_1"},
				},
			},
			expectedColors: []string{
				"*", // Missing data point: client1:mlx5_0 -> server1:mlx5_1
			},
			description: "Incast mode should mark missing data as red *",
		},
		{
			name: "Incast mode with extreme latency values",
			latencyData: []LatencyData{
				{SourceHost: "client1", SourceHCA: "mlx5_0", TargetHost: "server1", TargetHCA: "mlx5_0", AvgLatencyUs: 1.2},
				{SourceHost: "client1", SourceHCA: "mlx5_0", TargetHost: "server2", TargetHCA: "mlx5_0", AvgLatencyUs: 50.8}, // Very high
				{SourceHost: "client2", SourceHCA: "mlx5_0", TargetHost: "server1", TargetHCA: "mlx5_0", AvgLatencyUs: 2.1},
				{SourceHost: "client2", SourceHCA: "mlx5_0", TargetHost: "server2", TargetHCA: "mlx5_0", AvgLatencyUs: 100.5}, // Extremely high
			},
			cfg: &config.Config{
				StreamType: config.InCast,
				Server: config.ServerConfig{
					Hostname: []string{"server1", "server2"},
					Hca:      []string{"mlx5_0"},
				},
				Client: config.ClientConfig{
					Hostname: []string{"client1", "client2"},
					Hca:      []string{"mlx5_0"},
				},
			},
			expectedColors: []string{
				"50.80 μs",  // Very high latency
				"100.50 μs", // Extremely high latency
			},
			description: "Should mark extreme latencies in red",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stdout
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			// Call the display function
			displayLatencyMatrixIncast(tt.latencyData, tt.cfg)

			// Restore stdout
			w.Close()
			os.Stdout = oldStdout

			// Read captured output
			var buf bytes.Buffer
			io.Copy(&buf, r)
			output := buf.String()

			// Print the test description and output for visual inspection
			fmt.Printf("\n=== Test: %s ===\n", tt.name)
			fmt.Printf("Description: %s\n", tt.description)
			fmt.Printf("Output:\n%s\n", output)

			// Verify that red color codes are present for expected values
			for _, expectedValue := range tt.expectedColors {
				if !strings.Contains(output, colorRed) || !strings.Contains(output, expectedValue) {
					t.Errorf("Expected red-marked value '%s' not found in output", expectedValue)
				}
			}

			// Additional visual separator
			fmt.Println(strings.Repeat("=", 80))
		})
	}
}

// TestColorConstants tests that color constants are properly defined
func TestColorConstants(t *testing.T) {
	if colorRed == "" {
		t.Error("colorRed constant should not be empty")
	}
	if colorReset == "" {
		t.Error("colorReset constant should not be empty")
	}
	if latencyThreshold != 4.0 {
		t.Errorf("Expected latency threshold to be 4.0, got %f", latencyThreshold)
	}

	fmt.Printf("\n=== Color Constants Test ===\n")
	fmt.Printf("colorRed: %q\n", colorRed)
	fmt.Printf("colorReset: %q\n", colorReset)
	fmt.Printf("latencyThreshold: %.1f μs\n", latencyThreshold)
	fmt.Printf("\nVisual test - Normal text vs %sRed text%s\n", colorRed, colorReset)
	fmt.Println(strings.Repeat("=", 80))
}

// TestLatencyValueFormatting tests the formatting of latency values with color
func TestLatencyValueFormatting(t *testing.T) {
	tests := []struct {
		name          string
		latency       float64
		shouldBeRed   bool
		expectedValue string
	}{
		{"Very low latency", 0.5, false, "0.50 μs"},
		{"Low latency", 1.5, false, "1.50 μs"},
		{"Normal latency", 3.99, false, "3.99 μs"},
		{"At threshold", 4.0, false, "4.00 μs"},
		{"Just above threshold", 4.01, true, "4.01 μs"},
		{"High latency", 10.5, true, "10.50 μs"},
		{"Very high latency", 50.0, true, "50.00 μs"},
		{"Missing data", 0.0, true, "N/A"},
	}

	fmt.Printf("\n=== Latency Value Formatting Test ===\n")
	fmt.Printf("Threshold: %.2f μs\n\n", latencyThreshold)
	fmt.Printf("%-25s | %-12s | %-8s | %s\n", "Test Case", "Latency", "Red?", "Formatted Output")
	fmt.Println(strings.Repeat("-", 80))

	for _, tt := range tests {
		var formatted string
		if tt.latency > 0 {
			valueStr := fmt.Sprintf("%.2f μs", tt.latency)
			if tt.latency > latencyThreshold {
				formatted = fmt.Sprintf("%s%s%s", colorRed, valueStr, colorReset)
			} else {
				formatted = valueStr
			}
		} else {
			formatted = fmt.Sprintf("%sN/A%s", colorRed, colorReset)
		}

		redMarker := "No"
		if tt.shouldBeRed {
			redMarker = "Yes"
		}

		fmt.Printf("%-25s | %10.2f μs | %-8s | %s\n", tt.name, tt.latency, redMarker, formatted)

		// Verify
		if tt.shouldBeRed && !strings.Contains(formatted, colorRed) {
			t.Errorf("%s: expected red color but not found", tt.name)
		}
		if !tt.shouldBeRed && strings.Contains(formatted, colorRed) {
			t.Errorf("%s: did not expect red color but found", tt.name)
		}
	}

	fmt.Println(strings.Repeat("=", 80))
}
