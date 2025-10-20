package stream

import (
	"testing"
)

// TestCalculateTotalLatencyPorts tests the port calculation for latency tests
func TestCalculateTotalLatencyPorts(t *testing.T) {
	tests := []struct {
		name     string
		hosts    []string
		hcas     []string
		expected int
	}{
		{
			name:     "2 hosts, 2 HCAs each",
			hosts:    []string{"host1", "host2"},
			hcas:     []string{"mlx5_0", "mlx5_1"},
			expected: 2 * 2 * 1 * 2, // 2 hosts * 2 HCAs * (2-1) other hosts * 2 HCAs = 8
		},
		{
			name:     "3 hosts, 2 HCAs each",
			hosts:    []string{"host1", "host2", "host3"},
			hcas:     []string{"mlx5_0", "mlx5_1"},
			expected: 3 * 2 * 2 * 2, // 3 hosts * 2 HCAs * (3-1) other hosts * 2 HCAs = 24
		},
		{
			name:     "2 hosts, 1 HCA each",
			hosts:    []string{"host1", "host2"},
			hcas:     []string{"mlx5_0"},
			expected: 2 * 1 * 1 * 1, // 2 hosts * 1 HCA * (2-1) other hosts * 1 HCA = 2
		},
		{
			name:     "4 hosts, 3 HCAs each",
			hosts:    []string{"host1", "host2", "host3", "host4"},
			hcas:     []string{"mlx5_0", "mlx5_1", "mlx5_2"},
			expected: 4 * 3 * 3 * 3, // 4 hosts * 3 HCAs * (4-1) other hosts * 3 HCAs = 108
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateTotalLatencyPorts(tt.hosts, tt.hcas)
			if result != tt.expected {
				t.Errorf("calculateTotalLatencyPorts() = %d, expected %d", result, tt.expected)
			}
		})
	}
}

// TestCalculateTotalLatencyPortsFormula verifies the formula is correct
func TestCalculateTotalLatencyPortsFormula(t *testing.T) {
	// For N hosts with H HCAs each:
	// Total connections = N * H * (N-1) * H = N * H^2 * (N-1)

	// Example: 10 hosts with 4 HCAs each
	hosts := make([]string, 10)
	for i := 0; i < 10; i++ {
		hosts[i] = "host" + string(rune('0'+i))
	}
	hcas := []string{"mlx5_0", "mlx5_1", "mlx5_2", "mlx5_3"}

	result := calculateTotalLatencyPorts(hosts, hcas)
	expected := 10 * 4 * 9 * 4 // 1440 ports

	if result != expected {
		t.Errorf("For 10 hosts with 4 HCAs: got %d ports, expected %d", result, expected)
	}
}
