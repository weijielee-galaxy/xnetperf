package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

// TestMinFloat tests the minFloat helper function
func TestMinFloat(t *testing.T) {
	tests := []struct {
		name     string
		values   []float64
		expected float64
	}{
		{
			name:     "Normal values",
			values:   []float64{3.5, 1.2, 4.8, 2.1},
			expected: 1.2,
		},
		{
			name:     "Single value",
			values:   []float64{5.5},
			expected: 5.5,
		},
		{
			name:     "Empty slice",
			values:   []float64{},
			expected: 0,
		},
		{
			name:     "Negative values",
			values:   []float64{-1.5, -3.2, 0.5, -2.1},
			expected: -3.2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := minFloat(tt.values)
			if result != tt.expected {
				t.Errorf("minFloat() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

// TestMaxFloat tests the maxFloat helper function
func TestMaxFloat(t *testing.T) {
	tests := []struct {
		name     string
		values   []float64
		expected float64
	}{
		{
			name:     "Normal values",
			values:   []float64{3.5, 1.2, 4.8, 2.1},
			expected: 4.8,
		},
		{
			name:     "Single value",
			values:   []float64{5.5},
			expected: 5.5,
		},
		{
			name:     "Empty slice",
			values:   []float64{},
			expected: 0,
		},
		{
			name:     "Negative values",
			values:   []float64{-1.5, -3.2, 0.5, -2.1},
			expected: 0.5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := maxFloat(tt.values)
			if result != tt.expected {
				t.Errorf("maxFloat() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

// TestAvgFloat tests the avgFloat helper function
func TestAvgFloat(t *testing.T) {
	tests := []struct {
		name     string
		values   []float64
		expected float64
	}{
		{
			name:     "Normal values",
			values:   []float64{2.0, 4.0, 6.0, 8.0},
			expected: 5.0,
		},
		{
			name:     "Single value",
			values:   []float64{5.5},
			expected: 5.5,
		},
		{
			name:     "Empty slice",
			values:   []float64{},
			expected: 0,
		},
		{
			name:     "Mixed positive and negative",
			values:   []float64{-2.0, 2.0, -1.0, 1.0},
			expected: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := avgFloat(tt.values)
			if result != tt.expected {
				t.Errorf("avgFloat() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

// TestParseLatencyReport tests parsing of latency JSON reports
func TestParseLatencyReport(t *testing.T) {
	// Create temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "latency_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	tests := []struct {
		name            string
		filename        string
		jsonContent     string
		expectError     bool
		expectNil       bool
		expectedSource  string
		expectedTarget  string
		expectedLatency float64
	}{
		{
			name:     "Valid client report",
			filename: "latency_c_host1_mlx5_0_to_host2_mlx5_1_p20000.json",
			jsonContent: `{
				"results": {
					"t_avg": 1.23
				}
			}`,
			expectError:     false,
			expectNil:       false,
			expectedSource:  "host1:mlx5_0",
			expectedTarget:  "host2:mlx5_1",
			expectedLatency: 1.23,
		},
		{
			name:     "Server report should be skipped",
			filename: "latency_s_host1_mlx5_0_from_host2_mlx5_1_p20000.json",
			jsonContent: `{
				"results": {
					"t_avg": 1.23
				}
			}`,
			expectError: false,
			expectNil:   true,
		},
		{
			name:        "Invalid filename format - old format",
			filename:    "latency_c_host1_mlx5_0_20000.json",
			jsonContent: `{"results": {"t_avg": 1.23}}`,
			expectError: true,
		},
		{
			name:     "Missing results field - should parse as zero",
			filename: "latency_c_host1_mlx5_0_to_host2_mlx5_1_p20000.json",
			jsonContent: `{
				"test_info": {}
			}`,
			expectError:     false,
			expectNil:       false,
			expectedSource:  "host1:mlx5_0",
			expectedTarget:  "host2:mlx5_1",
			expectedLatency: 0.0,
		},
		{
			name:        "Invalid JSON",
			filename:    "latency_c_host1_mlx5_0_to_host2_mlx5_1_p20000.json",
			jsonContent: `invalid json content`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test file
			filePath := filepath.Join(tmpDir, tt.filename)
			err := os.WriteFile(filePath, []byte(tt.jsonContent), 0644)
			if err != nil {
				t.Fatalf("Failed to write test file: %v", err)
			}

			// Parse the report
			result, err := parseLatencyReport(filePath)

			// Check error expectations
			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			// Check nil expectations
			if tt.expectNil {
				if result != nil {
					t.Error("Expected nil result but got data")
				}
				return
			}

			if result == nil {
				t.Error("Expected data but got nil")
				return
			}

			// Verify parsed data
			if tt.expectedSource != "" {
				actualSource := result.SourceHost + ":" + result.SourceHCA
				if actualSource != tt.expectedSource {
					t.Errorf("Source = %v, expected %v", actualSource, tt.expectedSource)
				}
			}

			if tt.expectedTarget != "" {
				actualTarget := result.TargetHost + ":" + result.TargetHCA
				if actualTarget != tt.expectedTarget {
					t.Errorf("Target = %v, expected %v", actualTarget, tt.expectedTarget)
				}
			}

			if result.AvgLatencyUs != tt.expectedLatency {
				t.Errorf("AvgLatencyUs = %v, expected %v", result.AvgLatencyUs, tt.expectedLatency)
			}
		})
	}
}

// TestCollectLatencyReportData tests collecting multiple latency reports
func TestCollectLatencyReportData(t *testing.T) {
	// Create temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "latency_collect_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create sample latency report files
	reports := []struct {
		filename string
		content  string
	}{
		{
			filename: "latency_c_host1_mlx5_0_to_host2_mlx5_1_p20000.json",
			content:  `{"results": {"t_avg": 1.5}}`,
		},
		{
			filename: "latency_c_host2_mlx5_1_to_host1_mlx5_0_p20001.json",
			content:  `{"results": {"t_avg": 2.3}}`,
		},
		{
			filename: "latency_s_host1_mlx5_0_from_host2_mlx5_1_p20000.json", // Should be skipped
			content:  `{"results": {"t_avg": 1.0}}`,
		},
	}

	for _, report := range reports {
		filePath := filepath.Join(tmpDir, report.filename)
		err = os.WriteFile(filePath, []byte(report.content), 0644)
		if err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}
	}

	// Collect the reports
	latencyData, err := collectLatencyReportData(tmpDir)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}

	// Should have 2 client reports (server reports are skipped)
	if len(latencyData) != 2 {
		t.Errorf("Expected 2 latency data entries, got %d", len(latencyData))
	}

	// Verify the data
	expectedLatencies := []float64{1.5, 2.3}
	actualLatencies := []float64{latencyData[0].AvgLatencyUs, latencyData[1].AvgLatencyUs}

	// Sort to ensure consistent comparison
	for i := 0; i < len(actualLatencies); i++ {
		found := false
		for _, expected := range expectedLatencies {
			if actualLatencies[i] == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Unexpected latency value: %v", actualLatencies[i])
		}
	}
}

// TestCollectLatencyReportDataEmptyDir tests handling of empty directory
func TestCollectLatencyReportDataEmptyDir(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "latency_empty_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	_, err = collectLatencyReportData(tmpDir)
	if err == nil {
		t.Error("Expected error for empty directory but got none")
	}
}

// TestDisplayLatencyMatrix tests the matrix display function
func TestDisplayLatencyMatrix(t *testing.T) {
	tests := []struct {
		name         string
		latencyData  []LatencyData
		shouldOutput bool
	}{
		{
			name:         "Empty data",
			latencyData:  []LatencyData{},
			shouldOutput: false,
		},
		{
			name: "Single measurement",
			latencyData: []LatencyData{
				{
					SourceHost:   "host1",
					SourceHCA:    "mlx5_0",
					TargetHost:   "host2",
					TargetHCA:    "mlx5_0",
					AvgLatencyUs: 1.5,
				},
			},
			shouldOutput: true,
		},
		{
			name: "Multiple measurements",
			latencyData: []LatencyData{
				{SourceHost: "host1", SourceHCA: "mlx5_0", TargetHost: "host2", TargetHCA: "mlx5_0", AvgLatencyUs: 1.5},
				{SourceHost: "host1", SourceHCA: "mlx5_0", TargetHost: "host3", TargetHCA: "mlx5_0", AvgLatencyUs: 2.3},
				{SourceHost: "host2", SourceHCA: "mlx5_0", TargetHost: "host1", TargetHCA: "mlx5_0", AvgLatencyUs: 1.6},
			},
			shouldOutput: true,
		},
		{
			name: "Full N×N matrix (3x3)",
			latencyData: []LatencyData{
				// host1 -> others
				{SourceHost: "host1", SourceHCA: "mlx5_0", TargetHost: "host2", TargetHCA: "mlx5_0", AvgLatencyUs: 1.45},
				{SourceHost: "host1", SourceHCA: "mlx5_0", TargetHost: "host3", TargetHCA: "mlx5_0", AvgLatencyUs: 1.52},
				// host2 -> others
				{SourceHost: "host2", SourceHCA: "mlx5_0", TargetHost: "host1", TargetHCA: "mlx5_0", AvgLatencyUs: 1.46},
				{SourceHost: "host2", SourceHCA: "mlx5_0", TargetHost: "host3", TargetHCA: "mlx5_0", AvgLatencyUs: 1.50},
				// host3 -> others
				{SourceHost: "host3", SourceHCA: "mlx5_0", TargetHost: "host1", TargetHCA: "mlx5_0", AvgLatencyUs: 1.53},
				{SourceHost: "host3", SourceHCA: "mlx5_0", TargetHost: "host2", TargetHCA: "mlx5_0", AvgLatencyUs: 1.51},
			},
			shouldOutput: true,
		},
		{
			name: "Long hostname handling",
			latencyData: []LatencyData{
				// Very long hostnames should be truncated
				{SourceHost: "very-long-hostname-that-exceeds-limits", SourceHCA: "mlx5_0", TargetHost: "another-very-long-hostname", TargetHCA: "mlx5_1", AvgLatencyUs: 2.35},
				{SourceHost: "another-very-long-hostname", SourceHCA: "mlx5_1", TargetHost: "very-long-hostname-that-exceeds-limits", TargetHCA: "mlx5_0", AvgLatencyUs: 2.40},
			},
			shouldOutput: true,
		},
		{
			name: "Mixed short and long names",
			latencyData: []LatencyData{
				{SourceHost: "h1", SourceHCA: "mlx5_0", TargetHost: "very-long-hostname-with-many-characters", TargetHCA: "mlx5_0", AvgLatencyUs: 1.25},
				{SourceHost: "very-long-hostname-with-many-characters", SourceHCA: "mlx5_0", TargetHost: "h1", TargetHCA: "mlx5_0", AvgLatencyUs: 1.30},
			},
			shouldOutput: true,
		},
		{
			name: "Multiple HCAs per host (merged cells)",
			latencyData: []LatencyData{
				// host1:mlx5_0 -> others
				{SourceHost: "host1", SourceHCA: "mlx5_0", TargetHost: "host1", TargetHCA: "mlx5_1", AvgLatencyUs: 1.23},
				{SourceHost: "host1", SourceHCA: "mlx5_0", TargetHost: "host2", TargetHCA: "mlx5_0", AvgLatencyUs: 1.45},
				{SourceHost: "host1", SourceHCA: "mlx5_0", TargetHost: "host2", TargetHCA: "mlx5_1", AvgLatencyUs: 1.50},
				// host1:mlx5_1 -> others
				{SourceHost: "host1", SourceHCA: "mlx5_1", TargetHost: "host1", TargetHCA: "mlx5_0", AvgLatencyUs: 1.24},
				{SourceHost: "host1", SourceHCA: "mlx5_1", TargetHost: "host2", TargetHCA: "mlx5_0", AvgLatencyUs: 1.46},
				{SourceHost: "host1", SourceHCA: "mlx5_1", TargetHost: "host2", TargetHCA: "mlx5_1", AvgLatencyUs: 1.51},
				// host2:mlx5_0 -> others
				{SourceHost: "host2", SourceHCA: "mlx5_0", TargetHost: "host1", TargetHCA: "mlx5_0", AvgLatencyUs: 1.47},
				{SourceHost: "host2", SourceHCA: "mlx5_0", TargetHost: "host1", TargetHCA: "mlx5_1", AvgLatencyUs: 1.48},
				{SourceHost: "host2", SourceHCA: "mlx5_0", TargetHost: "host2", TargetHCA: "mlx5_1", AvgLatencyUs: 1.25},
				// host2:mlx5_1 -> others
				{SourceHost: "host2", SourceHCA: "mlx5_1", TargetHost: "host1", TargetHCA: "mlx5_0", AvgLatencyUs: 1.49},
				{SourceHost: "host2", SourceHCA: "mlx5_1", TargetHost: "host1", TargetHCA: "mlx5_1", AvgLatencyUs: 1.50},
				{SourceHost: "host2", SourceHCA: "mlx5_1", TargetHost: "host2", TargetHCA: "mlx5_0", AvgLatencyUs: 1.26},
			},
			shouldOutput: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This test just ensures the function doesn't panic
			// In a real scenario, we'd capture stdout and verify the output
			displayLatencyMatrix(tt.latencyData)
		})
	}
}
