package cmd

import (
	"strings"
	"testing"
)

// TestHCANameParsing 测试 HCA 设备名称解析的灵活性
func TestHCANameParsing(t *testing.T) {
	tests := []struct {
		name         string
		filename     string
		reportType   string // "fullmesh", "incast", or "p2p"
		expectedHost string
		expectedHCA  string
	}{
		// FullMesh/Incast Client Reports
		{
			name:         "FullMesh client - standard mlx5_0",
			filename:     "report_c_cetus-g88-061_mlx5_0_20000.json",
			reportType:   "fullmesh",
			expectedHost: "cetus-g88-061",
			expectedHCA:  "mlx5_0",
		},
		{
			name:         "FullMesh client - mlx5_bond_0",
			filename:     "report_c_cetus-g88-061_mlx5_bond_0_20000.json",
			reportType:   "fullmesh",
			expectedHost: "cetus-g88-061",
			expectedHCA:  "mlx5_bond_0",
		},
		{
			name:         "FullMesh client - mlx5_1_bond",
			filename:     "report_c_host1_mlx5_1_bond_20001.json",
			reportType:   "fullmesh",
			expectedHost: "host1",
			expectedHCA:  "mlx5_1_bond",
		},
		{
			name:         "FullMesh client - ib0 simple name",
			filename:     "report_c_server_ib0_20000.json",
			reportType:   "fullmesh",
			expectedHost: "server",
			expectedHCA:  "ib0",
		},
		{
			name:         "FullMesh client - complex HCA name",
			filename:     "report_c_myhost_custom_hca_name_123_20000.json",
			reportType:   "fullmesh",
			expectedHost: "myhost",
			expectedHCA:  "custom_hca_name_123",
		},

		// FullMesh/Incast Server Reports
		{
			name:         "FullMesh server - mlx5_0",
			filename:     "report_s_cetus-g88-094_mlx5_0_20000.json",
			reportType:   "fullmesh",
			expectedHost: "cetus-g88-094",
			expectedHCA:  "mlx5_0",
		},
		{
			name:         "FullMesh server - mlx5_bond_0",
			filename:     "report_s_cetus-g88-094_mlx5_bond_0_20000.json",
			reportType:   "fullmesh",
			expectedHost: "cetus-g88-094",
			expectedHCA:  "mlx5_bond_0",
		},

		// P2P Reports
		{
			name:         "P2P - standard mlx5_0",
			filename:     "report_cetus-g88-061_mlx5_0_20000.json",
			reportType:   "p2p",
			expectedHost: "cetus-g88-061",
			expectedHCA:  "mlx5_0",
		},
		{
			name:         "P2P - mlx5_bond_0",
			filename:     "report_cetus-g88-061_mlx5_bond_0_20000.json",
			reportType:   "p2p",
			expectedHost: "cetus-g88-061",
			expectedHCA:  "mlx5_bond_0",
		},
		{
			name:         "P2P - mlx5_1_bond",
			filename:     "report_host1_mlx5_1_bond_20001.json",
			reportType:   "p2p",
			expectedHost: "host1",
			expectedHCA:  "mlx5_1_bond",
		},
		{
			name:         "P2P - complex HCA name",
			filename:     "report_myserver_hca_dev_name_v2_20000.json",
			reportType:   "p2p",
			expectedHost: "myserver",
			expectedHCA:  "hca_dev_name_v2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parts := strings.Split(tt.filename, "_")

			var hostname, device string

			if tt.reportType == "p2p" {
				// P2P format: report_hostname_hca_port.json
				if len(parts) < 4 {
					t.Fatalf("Not enough parts for P2P format: %v", parts)
				}
				hostname = parts[1]
				device = strings.Join(parts[2:len(parts)-1], "_")
			} else {
				// FullMesh/Incast format: report_c/s_hostname_hca_port.json
				if len(parts) < 5 {
					t.Fatalf("Not enough parts for FullMesh/Incast format: %v", parts)
				}
				hostname = parts[2]
				device = strings.Join(parts[3:len(parts)-1], "_")
			}

			if hostname != tt.expectedHost {
				t.Errorf("Hostname mismatch: expected %q, got %q", tt.expectedHost, hostname)
			}

			if device != tt.expectedHCA {
				t.Errorf("HCA device mismatch: expected %q, got %q", tt.expectedHCA, device)
			}
		})
	}
}

// TestHCANameParsingEdgeCases 测试边界情况
func TestHCANameParsingEdgeCases(t *testing.T) {
	tests := []struct {
		name       string
		filename   string
		reportType string
		shouldFail bool
	}{
		{
			name:       "FullMesh - too few parts",
			filename:   "report_c_host.json",
			reportType: "fullmesh",
			shouldFail: true,
		},
		{
			name:       "P2P - too few parts",
			filename:   "report_host.json",
			reportType: "p2p",
			shouldFail: true,
		},
		{
			name:       "FullMesh - minimum valid parts",
			filename:   "report_c_host_hca_20000.json",
			reportType: "fullmesh",
			shouldFail: false,
		},
		{
			name:       "P2P - minimum valid parts",
			filename:   "report_host_hca_20000.json",
			reportType: "p2p",
			shouldFail: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parts := strings.Split(tt.filename, "_")

			if tt.reportType == "p2p" {
				if len(parts) < 4 {
					if !tt.shouldFail {
						t.Errorf("Expected parsing to succeed but got too few parts")
					}
					return
				}
			} else {
				if len(parts) < 5 {
					if !tt.shouldFail {
						t.Errorf("Expected parsing to succeed but got too few parts")
					}
					return
				}
			}

			if tt.shouldFail {
				t.Errorf("Expected parsing to fail but got enough parts")
			}
		})
	}
}
