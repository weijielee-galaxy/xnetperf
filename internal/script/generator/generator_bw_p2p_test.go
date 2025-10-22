package generator_test

import (
	"fmt"
	"strings"
	"testing"
	"xnetperf/config"
	"xnetperf/internal/script/generator"
)

func TestBwP2PScriptGenerator_GenerateScripts(t *testing.T) {
	tests := []struct {
		name     string
		cfg      *config.Config
		validate func(t *testing.T, result *generator.ScriptResult, cfg *config.Config)
	}{
		{
			name: "Basic P2P with 2 host pairs, 2 HCAs",
			cfg: &config.Config{
				Server: config.ServerConfig{
					Hostname: []string{"host1", "host2"},
					Hca:      []string{"mlx5_0", "mlx5_1"},
				},
				Client: config.ClientConfig{
					Hostname: []string{"host3", "host4"},
					Hca:      []string{"mlx5_0", "mlx5_1"},
				},
				StartPort:        20000,
				QpNum:            8,
				MessageSizeBytes: 65536,
				Run: config.Run{
					Infinitely:      false,
					DurationSeconds: 10,
				},
				RdmaCm:   true,
				GidIndex: 3,
				Report: config.Report{
					Enable: false,
				},
			},
			validate: func(t *testing.T, result *generator.ScriptResult, cfg *config.Config) {
				t.Log("=== Server Scripts ===")
				for _, script := range result.ServerScripts {
					t.Logf("Host: %s", script.Host)
					t.Logf("Commands:\n%s\n", script.Command)
				}

				t.Log("=== Client Scripts ===")
				for _, script := range result.ClientScripts {
					t.Logf("Host: %s", script.Host)
					t.Logf("Commands:\n%s\n", script.Command)
				}

				// Verify counts
				if len(result.ServerScripts) != 2 {
					t.Errorf("Expected 2 server hosts, got %d", len(result.ServerScripts))
				}
				if len(result.ClientScripts) != 2 {
					t.Errorf("Expected 2 client hosts, got %d", len(result.ClientScripts))
				}

				// Verify host1 has 2 HCA commands
				for _, script := range result.ServerScripts {
					if script.Host == "host1" {
						cmdCount := strings.Count(script.Command, "ib_write_bw")
						if cmdCount != 2 {
							t.Errorf("host1 should have 2 server commands (2 HCAs), got %d", cmdCount)
						}
						// Verify mlx5_0 and mlx5_1
						if !strings.Contains(script.Command, "-d mlx5_0") {
							t.Error("host1 should have mlx5_0 command")
						}
						if !strings.Contains(script.Command, "-d mlx5_1") {
							t.Error("host1 should have mlx5_1 command")
						}
					}
				}

				// Verify staggered HCA pairing
				// host1.mlx5_0 -> port 20000, should connect to host3.mlx5_1
				// host1.mlx5_1 -> port 20001, should connect to host3.mlx5_0
				for _, script := range result.ClientScripts {
					if script.Host == "host3" {
						// Should have mlx5_1 with port 20000 and mlx5_0 with port 20001
						if !strings.Contains(script.Command, "-d mlx5_1") {
							t.Error("host3 should have mlx5_1 command")
						}
						if !strings.Contains(script.Command, "-d mlx5_0") {
							t.Error("host3 should have mlx5_0 command")
						}
						if !strings.Contains(script.Command, "-p 20000") {
							t.Error("host3 should have port 20000 command")
						}
						if !strings.Contains(script.Command, "-p 20001") {
							t.Error("host3 should have port 20001 command")
						}
					}
				}

				// Verify port range
				allCommands := ""
				for _, script := range result.ServerScripts {
					allCommands += script.Command
				}
				// Should use ports 20000-20003 (2 hosts * 2 HCAs = 4 connections)
				if !strings.Contains(allCommands, "-p 20000") {
					t.Error("Should have port 20000")
				}
				if !strings.Contains(allCommands, "-p 20001") {
					t.Error("Should have port 20001")
				}
				if !strings.Contains(allCommands, "-p 20002") {
					t.Error("Should have port 20002")
				}
				if !strings.Contains(allCommands, "-p 20003") {
					t.Error("Should have port 20003")
				}
			},
		},
		{
			name: "P2P with 1 host pair, 3 HCAs - verify staggered pairing",
			cfg: &config.Config{
				Server: config.ServerConfig{
					Hostname: []string{"server1"},
					Hca:      []string{"mlx5_0", "mlx5_1", "mlx5_2"},
				},
				Client: config.ClientConfig{
					Hostname: []string{"client1"},
					Hca:      []string{"mlx5_0", "mlx5_1", "mlx5_2"},
				},
				StartPort:        25000,
				QpNum:            8,
				MessageSizeBytes: 65536,
				Run: config.Run{
					Infinitely:      false,
					DurationSeconds: 10,
				},
				RdmaCm:   true,
				GidIndex: 3,
				Report: config.Report{
					Enable: false,
				},
			},
			validate: func(t *testing.T, result *generator.ScriptResult, cfg *config.Config) {
				t.Log("=== Testing Staggered HCA Pairing ===")

				serverCmd := result.ServerScripts[0].Command
				clientCmd := result.ClientScripts[0].Command

				t.Logf("Server commands:\n%s\n", serverCmd)
				t.Logf("Client commands:\n%s\n", clientCmd)

				// Server side should have mlx5_0, mlx5_1, mlx5_2 in order
				serverLines := strings.Split(serverCmd, " && \\ \n")
				if len(serverLines) != 3 {
					t.Errorf("Expected 3 server commands, got %d", len(serverLines))
				}

				// Client side should have staggered: mlx5_1 (for server's mlx5_0), mlx5_2 (for server's mlx5_1), mlx5_0 (for server's mlx5_2)
				clientLines := strings.Split(clientCmd, " && \\ \n")
				if len(clientLines) != 3 {
					t.Errorf("Expected 3 client commands, got %d", len(clientLines))
				}

				// Verify pairing:
				// server mlx5_0 (port 25000) <-> client mlx5_1 (port 25000)
				// server mlx5_1 (port 25001) <-> client mlx5_2 (port 25001)
				// server mlx5_2 (port 25002) <-> client mlx5_0 (port 25002)

				// Check server mlx5_0 with port 25000
				if !strings.Contains(serverLines[0], "mlx5_0") || !strings.Contains(serverLines[0], "25000") {
					t.Error("First server command should use mlx5_0 and port 25000")
				}
				// Check client mlx5_1 with port 25000
				if !strings.Contains(clientLines[0], "mlx5_1") || !strings.Contains(clientLines[0], "25000") {
					t.Error("First client command should use mlx5_1 and port 25000")
				}

				// Check server mlx5_1 with port 25001
				if !strings.Contains(serverLines[1], "mlx5_1") || !strings.Contains(serverLines[1], "25001") {
					t.Error("Second server command should use mlx5_1 and port 25001")
				}
				// Check client mlx5_2 with port 25001
				if !strings.Contains(clientLines[1], "mlx5_2") || !strings.Contains(clientLines[1], "25001") {
					t.Error("Second client command should use mlx5_2 and port 25001")
				}

				// Check server mlx5_2 with port 25002
				if !strings.Contains(serverLines[2], "mlx5_2") || !strings.Contains(serverLines[2], "25002") {
					t.Error("Third server command should use mlx5_2 and port 25002")
				}
				// Check client mlx5_0 with port 25002
				if !strings.Contains(clientLines[2], "mlx5_0") || !strings.Contains(clientLines[2], "25002") {
					t.Error("Third client command should use mlx5_0 and port 25002")
				}
			},
		},
		{
			name: "P2P with report enabled",
			cfg: &config.Config{
				Server: config.ServerConfig{
					Hostname: []string{"host1"},
					Hca:      []string{"mlx5_0"},
				},
				Client: config.ClientConfig{
					Hostname: []string{"host2"},
					Hca:      []string{"mlx5_0"},
				},
				StartPort:        20000,
				QpNum:            8,
				MessageSizeBytes: 65536,
				Run: config.Run{
					Infinitely:      false,
					DurationSeconds: 10,
				},
				RdmaCm:   true,
				GidIndex: 3,
				Report: config.Report{
					Enable: true,
					Dir:    "/tmp/reports",
				},
			},
			validate: func(t *testing.T, result *generator.ScriptResult, cfg *config.Config) {
				serverCmd := result.ServerScripts[0].Command
				clientCmd := result.ClientScripts[0].Command

				if !strings.Contains(serverCmd, "--report_gbits") {
					t.Error("Server command should contain --report_gbits")
				}
				if !strings.Contains(serverCmd, "/tmp/reports/report_host1_mlx5_0_20000.json") {
					t.Error("Server command should contain correct report file path")
				}

				if !strings.Contains(clientCmd, "--report_gbits") {
					t.Error("Client command should contain --report_gbits")
				}
				if !strings.Contains(clientCmd, "/tmp/reports/report_host2_mlx5_0_20000.json") {
					t.Error("Client command should contain correct report file path")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock IPs
			mockIPs := make(map[string]string)
			for i, host := range tt.cfg.Server.Hostname {
				mockIPs[host] = fmt.Sprintf("192.168.1.%d", 10+i)
			}
			for i, host := range tt.cfg.Client.Hostname {
				mockIPs[host] = fmt.Sprintf("192.168.2.%d", 10+i)
			}

			gen := generator.NewBwP2PScriptGenerator(tt.cfg, mockIPs)
			result, err := gen.GenerateScripts()

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
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

func TestBwP2PScriptGenerator_CheckPortsAvailability(t *testing.T) {
	tests := []struct {
		name        string
		cfg         *config.Config
		wantErr     bool
		errContains string
	}{
		{
			name: "Enough ports available",
			cfg: &config.Config{
				Server: config.ServerConfig{
					Hostname: []string{"host1", "host2"},
					Hca:      []string{"mlx5_0", "mlx5_1"},
				},
				Client: config.ClientConfig{
					Hostname: []string{"host3", "host4"},
					Hca:      []string{"mlx5_0", "mlx5_1"},
				},
				StartPort: 20000,
			},
			wantErr: false,
			// 2 host pairs * 2 HCA pairs = 4 ports
		},
		{
			name: "Not enough ports",
			cfg: &config.Config{
				Server: config.ServerConfig{
					Hostname: func() []string {
						hosts := make([]string, 400)
						for i := range hosts {
							hosts[i] = fmt.Sprintf("host%d", i)
						}
						return hosts
					}(),
					Hca: []string{"mlx5_0", "mlx5_1", "mlx5_2", "mlx5_3", "mlx5_4"},
				},
				Client: config.ClientConfig{
					Hostname: func() []string {
						hosts := make([]string, 400)
						for i := range hosts {
							hosts[i] = fmt.Sprintf("client%d", i)
						}
						return hosts
					}(),
					Hca: []string{"mlx5_0", "mlx5_1", "mlx5_2", "mlx5_3", "mlx5_4"},
				},
				StartPort: 64000,
			},
			wantErr:     true,
			errContains: "not enough available ports",
			// 400 host pairs * 5 HCA pairs = 2000 ports
			// 从64000开始到65535只有1536个端口，肯定不够
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockIPs := make(map[string]string)
			for i, host := range tt.cfg.Server.Hostname {
				mockIPs[host] = fmt.Sprintf("192.168.1.%d", 10+i)
			}

			gen := generator.NewBwP2PScriptGenerator(tt.cfg, mockIPs)
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

func TestBwP2PScriptGenerator_CommandFormat(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Hostname: []string{"host1"},
			Hca:      []string{"mlx5_0", "mlx5_1"},
		},
		Client: config.ClientConfig{
			Hostname: []string{"host2"},
			Hca:      []string{"mlx5_0", "mlx5_1"},
		},
		StartPort:        20000,
		QpNum:            8,
		MessageSizeBytes: 65536,
		Run: config.Run{
			Infinitely:      false,
			DurationSeconds: 10,
		},
		RdmaCm:   true,
		GidIndex: 3,
		Report: config.Report{
			Enable: false,
		},
	}

	mockIPs := map[string]string{
		"host1": "192.168.1.10",
		"host2": "192.168.2.10",
	}

	gen := generator.NewBwP2PScriptGenerator(cfg, mockIPs)
	result, err := gen.GenerateScripts()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Verify command format
	serverCmd := result.ServerScripts[0].Command
	if !strings.Contains(serverCmd, "( ib_write_bw") {
		t.Error("Server commands should wrap ib_write_bw in parentheses")
	}
	if !strings.Contains(serverCmd, " )") {
		t.Error("Server commands should close parentheses")
	}
	if !strings.Contains(serverCmd, " && \\ \n") {
		t.Error("Multiple commands should be joined with delimiter")
	}

	clientCmd := result.ClientScripts[0].Command
	if !strings.Contains(clientCmd, "( ib_write_bw") {
		t.Error("Client commands should wrap ib_write_bw in parentheses")
	}
	if !strings.Contains(clientCmd, " )") {
		t.Error("Client commands should close parentheses")
	}
}
