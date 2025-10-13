package cmd

import (
	"strings"
	"testing"
)

func TestDisplayProbeResults(t *testing.T) {
	// 测试各种不同的场景
	testCases := []struct {
		name    string
		results []ProbeResult
		desc    string
	}{
		{
			name: "Mixed Status Results",
			results: []ProbeResult{
				{
					Hostname:     "cetus-g88-062",
					ProcessCount: 8,
					Status:       "RUNNING",
				},
				{
					Hostname:     "cetus-g88-065",
					ProcessCount: 8,
					Status:       "RUNNING",
				},
				{
					Hostname:     "cetus-g88-061",
					ProcessCount: 8,
					Status:       "RUNNING",
				},
				{
					Hostname:     "cetus-g88-094",
					ProcessCount: 8,
					Status:       "RUNNING",
				},
			},
			desc: "All hosts running with 8 processes each",
		},
		{
			name: "Long Hostname Test",
			results: []ProbeResult{
				{
					Hostname:     "very-long-hostname-that-exceeds-normal-width",
					ProcessCount: 16,
					Status:       "RUNNING",
				},
				{
					Hostname:     "short",
					ProcessCount: 2,
					Status:       "RUNNING",
				},
				{
					Hostname:     "completed-host",
					ProcessCount: 0,
					Status:       "COMPLETED",
				},
			},
			desc: "Test with varying hostname lengths",
		},
		{
			name: "Error Status Test",
			results: []ProbeResult{
				{
					Hostname:     "working-host",
					ProcessCount: 5,
					Status:       "RUNNING",
				},
				{
					Hostname:     "error-host",
					ProcessCount: 0,
					Status:       "ERROR",
					Error:        "SSH connection timeout",
				},
				{
					Hostname:     "completed-host",
					ProcessCount: 0,
					Status:       "COMPLETED",
				},
			},
			desc: "Test with error status and error messages",
		},
		{
			name: "All Completed",
			results: []ProbeResult{
				{
					Hostname:     "host-01",
					ProcessCount: 0,
					Status:       "COMPLETED",
				},
				{
					Hostname:     "host-02",
					ProcessCount: 0,
					Status:       "COMPLETED",
				},
				{
					Hostname:     "host-03",
					ProcessCount: 0,
					Status:       "COMPLETED",
				},
			},
			desc: "All hosts completed",
		},
		{
			name: "Single Host High Process Count",
			results: []ProbeResult{
				{
					Hostname:     "high-load-server",
					ProcessCount: 128,
					Status:       "RUNNING",
				},
			},
			desc: "Single host with high process count",
		},
		{
			name:    "Empty Results",
			results: []ProbeResult{},
			desc:    "Empty results list",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Logf("\n=== Test Case: %s ===", tc.name)
			t.Logf("Description: %s", tc.desc)
			t.Log("Output:")

			// 调用被测试的函数
			displayProbeResults(tc.results)

			t.Log("\n" + strings.Repeat("=", 60))
		})
	}
}

func TestDisplayProbeResultsRealScenario(t *testing.T) {
	// 模拟真实场景的测试数据
	t.Log("\n=== Real Scenario Test ===")
	t.Log("Simulating the exact scenario from the screenshot")

	results := []ProbeResult{
		{
			Hostname:     "cetus-g88-062",
			ProcessCount: 8,
			Status:       "RUNNING",
		},
		{
			Hostname:     "cetus-g88-065",
			ProcessCount: 8,
			Status:       "RUNNING",
		},
		{
			Hostname:     "cetus-g88-061",
			ProcessCount: 8,
			Status:       "RUNNING",
		},
		{
			Hostname:     "cetus-g88-094",
			ProcessCount: 8,
			Status:       "RUNNING",
		},
	}

	displayProbeResults(results)
	t.Log("\n" + strings.Repeat("=", 60))
}

func TestDisplayProbeResultsEdgeCases(t *testing.T) {
	// 边界情况测试
	edgeCases := []struct {
		name    string
		results []ProbeResult
	}{
		{
			name: "Very Long Error Message",
			results: []ProbeResult{
				{
					Hostname:     "problematic-host",
					ProcessCount: 0,
					Status:       "ERROR",
					Error:        "This is a very long error message that should test the table formatting when error messages exceed normal column widths",
				},
			},
		},
		{
			name: "Unicode Characters in Hostname",
			results: []ProbeResult{
				{
					Hostname:     "服务器-001",
					ProcessCount: 4,
					Status:       "RUNNING",
				},
				{
					Hostname:     "хост-002",
					ProcessCount: 2,
					Status:       "RUNNING",
				},
			},
		},
		{
			name: "Mixed Status with Long Names",
			results: []ProbeResult{
				{
					Hostname:     "extremely-long-hostname-for-testing-table-alignment-issues",
					ProcessCount: 999,
					Status:       "RUNNING",
				},
				{
					Hostname:     "a",
					ProcessCount: 1,
					Status:       "RUNNING",
				},
				{
					Hostname:     "medium-length-hostname",
					ProcessCount: 0,
					Status:       "COMPLETED",
				},
			},
		},
	}

	for _, tc := range edgeCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Logf("\n=== Edge Case: %s ===", tc.name)
			displayProbeResults(tc.results)
			t.Log("\n" + strings.Repeat("=", 50))
		})
	}
}
