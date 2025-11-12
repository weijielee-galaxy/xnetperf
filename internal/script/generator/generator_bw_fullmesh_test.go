package generator_test

import (
	"fmt"
	"strings"
	"testing"

	"xnetperf/config"
	"xnetperf/internal/script/generator"
)

func TestBwFullmeshScriptGenerator_GenerateScripts(t *testing.T) {
	tests := []struct {
		name        string
		cfg         *config.Config
		wantErr     bool
		errContains string
		validate    func(t *testing.T, result *generator.ScriptResult, cfg *config.Config)
	}{
		{
			name: "Basic fullmesh configuration with 2 hosts",
			cfg: &config.Config{
				StartPort:        20000,
				QpNum:            8,
				MessageSizeBytes: 65536,
				RdmaCm:           true,
				GidIndex:         3,
				Report: config.Report{
					Enable: false,
				},
				Run: config.Run{
					Infinitely:      true,
					DurationSeconds: 0,
				},
				Server: config.ServerConfig{
					Hostname: []string{"host1"},
					Hca:      []string{"mlx5_0"},
				},
				Client: config.ClientConfig{
					Hostname: []string{"host2"},
					Hca:      []string{"mlx5_0"},
				},
			},
			wantErr: false,
			validate: func(t *testing.T, result *generator.ScriptResult, cfg *config.Config) {
				// For fullmesh with 2 hosts, each host should have scripts
				// host1 and host2 both act as server and client
				if len(result.ServerScripts) != 2 {
					t.Errorf("Expected 2 server scripts (one per host), got %d", len(result.ServerScripts))
				}
				if len(result.ClientScripts) != 2 {
					t.Errorf("Expected 2 client scripts (one per host), got %d", len(result.ClientScripts))
				}

				// Check that both hosts appear in server scripts
				serverHosts := make(map[string]bool)
				for _, script := range result.ServerScripts {
					serverHosts[script.Host] = true
					if !strings.Contains(script.Command, "ib_write_bw") {
						t.Errorf("Server command for %s should contain 'ib_write_bw'", script.Host)
					}
				}
				if !serverHosts["host1"] || !serverHosts["host2"] {
					t.Error("Both host1 and host2 should have server scripts")
				}

				// Check that both hosts appear in client scripts
				clientHosts := make(map[string]bool)
				for _, script := range result.ClientScripts {
					clientHosts[script.Host] = true
					if !strings.Contains(script.Command, "ib_write_bw") {
						t.Errorf("Client command for %s should contain 'ib_write_bw'", script.Host)
					}
				}
				if !clientHosts["host1"] || !clientHosts["host2"] {
					t.Error("Both host1 and host2 should have client scripts")
				}
			},
		},
		{
			name: "Fullmesh with 3 hosts and multiple HCAs",
			cfg: &config.Config{
				StartPort:        30000,
				QpNum:            4,
				MessageSizeBytes: 4096,
				RdmaCm:           false,
				GidIndex:         0,
				Report: config.Report{
					Enable: false,
				},
				Run: config.Run{
					Infinitely:      false,
					DurationSeconds: 10,
				},
				Server: config.ServerConfig{
					Hostname: []string{"host1", "host2"},
					Hca:      []string{"mlx5_0", "mlx5_1"},
				},
				Client: config.ClientConfig{
					Hostname: []string{"host3"},
					Hca:      []string{"mlx5_0", "mlx5_1"},
				},
			},
			wantErr: false,
			validate: func(t *testing.T, result *generator.ScriptResult, cfg *config.Config) {
				// Total hosts: 3 (host1, host2, host3)
				// Each host should have server and client scripts
				if len(result.ServerScripts) != 3 {
					t.Errorf("Expected 3 server scripts, got %d", len(result.ServerScripts))
				}
				if len(result.ClientScripts) != 3 {
					t.Errorf("Expected 3 client scripts, got %d", len(result.ClientScripts))
				}

				// Count total connections
				// For fullmesh: each host (with each HCA) connects to every other host (with each HCA)
				// 3 hosts * 2 HCAs * 2 other hosts * 2 HCAs = 24 connections per host type
				totalServerCommands := 0
				for _, script := range result.ServerScripts {
					totalServerCommands += strings.Count(script.Command, "( ib_write_bw")
				}
				// Total connections: 3 hosts * (3-1) other hosts * 2 HCAs * 2 HCAs = 24
				expectedConnections := 3 * 2 * 2 * 2
				if totalServerCommands != expectedConnections {
					t.Errorf("Expected %d server commands, got %d", expectedConnections, totalServerCommands)
				}

				// Check duration flag
				for _, script := range result.ClientScripts {
					if !strings.Contains(script.Command, "-D 10") {
						t.Errorf("Client command for %s should contain '-D 10' for duration", script.Host)
					}
				}
			},
		},
		{
			name: "Fullmesh with report enabled",
			cfg: &config.Config{
				StartPort:        25000,
				QpNum:            8,
				MessageSizeBytes: 65536,
				RdmaCm:           true,
				GidIndex:         3,
				Report: config.Report{
					Enable: true,
					Dir:    "/tmp/reports",
				},
				Run: config.Run{
					Infinitely:      false,
					DurationSeconds: 10,
				},
				Server: config.ServerConfig{
					Hostname: []string{"host1"},
					Hca:      []string{"mlx5_0"},
				},
				Client: config.ClientConfig{
					Hostname: []string{"host2"},
					Hca:      []string{"mlx5_0"},
				},
			},
			wantErr: false,
			validate: func(t *testing.T, result *generator.ScriptResult, cfg *config.Config) {
				for _, script := range result.ServerScripts {
					if !strings.Contains(script.Command, "--report_gbits") {
						t.Errorf("Server command for %s should contain '--report_gbits'", script.Host)
					}
					if !strings.Contains(script.Command, "--out_json") {
						t.Errorf("Server command for %s should contain '--out_json'", script.Host)
					}
					if !strings.Contains(script.Command, "--out_json_file /tmp/reports/report_s_") {
						t.Errorf("Server command for %s should contain report file path", script.Host)
					}
				}

				for _, script := range result.ClientScripts {
					if !strings.Contains(script.Command, "--report_gbits") {
						t.Errorf("Client command for %s should contain '--report_gbits'", script.Host)
					}
					if !strings.Contains(script.Command, "--out_json_file /tmp/reports/report_c_") {
						t.Errorf("Client command for %s should contain report file path", script.Host)
					}
				}
			},
		},
		{
			name: "Fullmesh - verify no self-connections",
			cfg: &config.Config{
				StartPort:        20000,
				QpNum:            8,
				MessageSizeBytes: 65536,
				RdmaCm:           false,
				GidIndex:         0,
				Report: config.Report{
					Enable: false,
				},
				Run: config.Run{
					Infinitely:      true,
					DurationSeconds: 0,
				},
				Server: config.ServerConfig{
					Hostname: []string{"host1"},
					Hca:      []string{"mlx5_0"},
				},
				Client: config.ClientConfig{
					Hostname: []string{"host2"},
					Hca:      []string{"mlx5_0"},
				},
			},
			wantErr: false,
			validate: func(t *testing.T, result *generator.ScriptResult, cfg *config.Config) {
				// For 2 hosts with 1 HCA each:
				// host1 should connect to host2 (1 connection)
				// host2 should connect to host1 (1 connection)
				// Total: 2 connections, no self-connections

				for _, script := range result.ServerScripts {
					cmdCount := strings.Count(script.Command, "( ib_write_bw")
					// Each host should have 1 server command (receiving from 1 other host)
					if cmdCount != 1 {
						t.Errorf("Host %s should have 1 server command, got %d", script.Host, cmdCount)
					}
				}

				for _, script := range result.ClientScripts {
					cmdCount := strings.Count(script.Command, "( ib_write_bw")
					// Each host should have 1 client command (sending to 1 other host)
					if cmdCount != 1 {
						t.Errorf("Host %s should have 1 client command, got %d", script.Host, cmdCount)
					}
				}
			},
		},
		{
			name: "Port exhaustion error",
			cfg: &config.Config{
				StartPort:        65500, // Very high start port
				QpNum:            8,
				MessageSizeBytes: 65536,
				RdmaCm:           true,
				GidIndex:         3,
				Report: config.Report{
					Enable: false,
				},
				Run: config.Run{
					Infinitely:      true,
					DurationSeconds: 0,
				},
				Server: config.ServerConfig{
					Hostname: []string{"host1", "host2"},
					Hca:      []string{"mlx5_0", "mlx5_1"},
				},
				Client: config.ClientConfig{
					Hostname: []string{"host3", "host4"},
					Hca:      []string{"mlx5_0", "mlx5_1"},
				},
			},
			wantErr:     true,
			errContains: "not enough available ports",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock IPs for all hosts (server + client)
			mockIPs := make(map[string]string)
			allHosts := append(tt.cfg.Server.Hostname, tt.cfg.Client.Hostname...)
			for i, host := range allHosts {
				mockIPs[host] = fmt.Sprintf("192.168.1.%d", 10+i)
			}

			gen := generator.NewBwFullmeshScriptGenerator(tt.cfg, mockIPs)
			result, err := gen.GenerateScripts()

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				} else if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("Expected error to contain '%s', got '%s'", tt.errContains, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if result == nil {
				t.Fatal("Expected result, got nil")
			}

			if tt.validate != nil {
				tt.validate(t, result, tt.cfg)
			}
		})
	}
}

func TestBwFullmeshScriptGenerator_CheckPortsAvailability(t *testing.T) {
	tests := []struct {
		name        string
		cfg         *config.Config
		wantErr     bool
		errContains string
	}{
		{
			name: "Enough ports available",
			cfg: &config.Config{
				StartPort: 20000,
				Server: config.ServerConfig{
					Hostname: []string{"host1"},
					Hca:      []string{"mlx5_0"},
				},
				Client: config.ClientConfig{
					Hostname: []string{"host2"},
					Hca:      []string{"mlx5_0"},
				},
			},
			wantErr: false,
		},
		{
			name: "Not enough ports - starting from high port",
			cfg: &config.Config{
				StartPort: 65535,
				Server: config.ServerConfig{
					Hostname: []string{"host1"},
					Hca:      []string{"mlx5_0", "mlx5_1"},
				},
				Client: config.ClientConfig{
					Hostname: []string{"host2"},
					Hca:      []string{"mlx5_0", "mlx5_1"},
				},
			},
			wantErr:     true,
			errContains: "not enough available ports",
		},
		{
			name: "Large scale fullmesh configuration",
			cfg: &config.Config{
				StartPort: 20000,
				Server: config.ServerConfig{
					Hostname: []string{"host1", "host2"},
					Hca:      []string{"mlx5_0", "mlx5_1"},
				},
				Client: config.ClientConfig{
					Hostname: []string{"host3", "host4"},
					Hca:      []string{"mlx5_0", "mlx5_1"},
				},
			},
			wantErr: false, // 4 hosts * 3 other hosts * 2 HCAs * 2 HCAs = 48 ports needed
		},
		{
			name: "Single host should use 0 ports (no self-connection)",
			cfg: &config.Config{
				StartPort: 20000,
				Server: config.ServerConfig{
					Hostname: []string{"host1"},
					Hca:      []string{"mlx5_0"},
				},
				Client: config.ClientConfig{
					Hostname: []string{},
					Hca:      []string{"mlx5_0"},
				},
			},
			wantErr: false, // 1 host * (1-1) other hosts = 0 connections
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gen := generator.NewBwFullmeshScriptGenerator(tt.cfg, nil)
			err := gen.CheckPortsAvailability()

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				} else if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("Expected error to contain '%s', got '%s'", tt.errContains, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestBwFullmeshScriptGenerator_CommandFormat(t *testing.T) {
	cfg := &config.Config{
		StartPort:        20000,
		QpNum:            8,
		MessageSizeBytes: 65536,
		RdmaCm:           true,
		GidIndex:         3,
		Report: config.Report{
			Enable: false,
		},
		Run: config.Run{
			Infinitely:      true,
			DurationSeconds: 0,
		},
		Server: config.ServerConfig{
			Hostname: []string{"host1"},
			Hca:      []string{"mlx5_0"},
		},
		Client: config.ClientConfig{
			Hostname: []string{"host2"},
			Hca:      []string{"mlx5_0"},
		},
	}

	mockIPs := map[string]string{"host1": "192.168.1.10", "host2": "192.168.1.11"}
	gen := generator.NewBwFullmeshScriptGenerator(cfg, mockIPs)
	result, err := gen.GenerateScripts()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Validate command delimiter and parentheses
	for _, script := range result.ServerScripts {
		if !strings.Contains(script.Command, "( ib_write_bw") {
			t.Errorf("Server command for %s should wrap ib_write_bw in parentheses", script.Host)
		}
		if !strings.Contains(script.Command, " )") {
			t.Errorf("Server command for %s should close parentheses", script.Host)
		}
	}

	for _, script := range result.ClientScripts {
		if !strings.Contains(script.Command, "( ib_write_bw") {
			t.Errorf("Client command for %s should wrap ib_write_bw in parentheses", script.Host)
		}
		if !strings.Contains(script.Command, " )") {
			t.Errorf("Client command for %s should close parentheses", script.Host)
		}
	}
}

func TestBwFullmeshScriptGenerator_PortAllocation(t *testing.T) {
	cfg := &config.Config{
		StartPort:        20000,
		QpNum:            8,
		MessageSizeBytes: 65536,
		RdmaCm:           false,
		GidIndex:         0,
		Report: config.Report{
			Enable: false,
		},
		Run: config.Run{
			Infinitely:      true,
			DurationSeconds: 0,
		},
		Server: config.ServerConfig{
			Hostname: []string{"host1"},
			Hca:      []string{"mlx5_0", "mlx5_1"},
		},
		Client: config.ClientConfig{
			Hostname: []string{"host2"},
			Hca:      []string{"mlx5_0", "mlx5_1"},
		},
	}

	mockIPs := map[string]string{"host1": "192.168.1.10", "host2": "192.168.1.11"}
	gen := generator.NewBwFullmeshScriptGenerator(cfg, mockIPs)
	result, err := gen.GenerateScripts()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// For 2 hosts with 2 HCAs each:
	// host1 -> host2: 2 * 2 = 4 connections (ports 20000, 20001, 20002, 20003)
	// host2 -> host1: 2 * 2 = 4 connections (ports 20004, 20005, 20006, 20007)
	// Total: 8 ports used

	// Verify port ranges
	allCommands := ""
	for _, script := range result.ServerScripts {
		allCommands += script.Command
	}
	for _, script := range result.ClientScripts {
		allCommands += script.Command
	}

	// Verify first port
	if !strings.Contains(allCommands, "-p 20000") {
		t.Error("Commands should start with port 20000")
	}

	// Verify that multiple ports are used
	if !strings.Contains(allCommands, "-p 20001") {
		t.Error("Commands should use port 20001")
	}
}
