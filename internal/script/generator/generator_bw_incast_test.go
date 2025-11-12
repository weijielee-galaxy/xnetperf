package generator_test

import (
	"fmt"
	"strings"
	"testing"

	"xnetperf/config"
	"xnetperf/internal/script/generator"
)

func TestBwIncastScriptGenerator_GenerateScripts(t *testing.T) {
	tests := []struct {
		name        string
		cfg         *config.Config
		wantErr     bool
		errContains string
		validate    func(t *testing.T, result *generator.ScriptResult, cfg *config.Config)
	}{
		{
			name: "Basic incast configuration",
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
					Hostname: []string{"server1"},
					Hca:      []string{"mlx5_0"},
				},
				Client: config.ClientConfig{
					Hostname: []string{"client1"},
					Hca:      []string{"mlx5_0"},
				},
			},
			wantErr: false,
			validate: func(t *testing.T, result *generator.ScriptResult, cfg *config.Config) {
				// Should have 1 server script and 1 client script
				if len(result.ServerScripts) != 1 {
					t.Errorf("Expected 1 server script, got %d", len(result.ServerScripts))
				}
				if len(result.ClientScripts) != 1 {
					t.Errorf("Expected 1 client script, got %d", len(result.ClientScripts))
				}

				// Check server script content
				if result.ServerScripts[0].Host != "server1" {
					t.Errorf("Expected server host 'server1', got '%s'", result.ServerScripts[0].Host)
				}
				serverCmd := result.ServerScripts[0].Command
				if !strings.Contains(serverCmd, "ib_write_bw") {
					t.Error("Server command should contain 'ib_write_bw'")
				}
				if !strings.Contains(serverCmd, "-d mlx5_0") {
					t.Error("Server command should contain device '-d mlx5_0'")
				}
				if !strings.Contains(serverCmd, "-p 20000") {
					t.Error("Server command should start with port 20000")
				}
				if !strings.Contains(serverCmd, "-q 8") {
					t.Error("Server command should contain '-q 8'")
				}
				if !strings.Contains(serverCmd, "-m 65536") {
					t.Error("Server command should contain '-m 65536'")
				}

				// Check client script content
				if result.ClientScripts[0].Host != "client1" {
					t.Errorf("Expected client host 'client1', got '%s'", result.ClientScripts[0].Host)
				}
				clientCmd := result.ClientScripts[0].Command
				if !strings.Contains(clientCmd, "ib_write_bw") {
					t.Error("Client command should contain 'ib_write_bw'")
				}
				if !strings.Contains(clientCmd, "-d mlx5_0") {
					t.Error("Client command should contain device '-d mlx5_0'")
				}
			},
		},
		{
			name: "Multiple servers and clients",
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
					Hostname: []string{"server1", "server2"},
					Hca:      []string{"mlx5_0", "mlx5_1"},
				},
				Client: config.ClientConfig{
					Hostname: []string{"client1", "client2"},
					Hca:      []string{"mlx5_0", "mlx5_1"},
				},
			},
			wantErr: false,
			validate: func(t *testing.T, result *generator.ScriptResult, cfg *config.Config) {
				// Should have 2 server scripts and 2 client scripts
				if len(result.ServerScripts) != 2 {
					t.Errorf("Expected 2 server scripts, got %d", len(result.ServerScripts))
				}
				if len(result.ClientScripts) != 2 {
					t.Errorf("Expected 2 client scripts, got %d", len(result.ClientScripts))
				}

				// Total connections: 2 servers * 2 HCAs * 2 clients * 2 HCAs = 16
				totalServerCommands := 0
				for _, script := range result.ServerScripts {
					// Count the number of ib_write_bw commands (wrapped in parentheses)
					totalServerCommands += strings.Count(script.Command, "( ib_write_bw")
				}
				expectedCommands := 2 * 2 * 2 * 2 // servers * server_hcas * clients * client_hcas
				if totalServerCommands != expectedCommands {
					t.Errorf("Expected %d server commands, got %d", expectedCommands, totalServerCommands)
				}

				// Check duration flag
				for _, script := range result.ClientScripts {
					if !strings.Contains(script.Command, "-D 10") {
						t.Error("Client command should contain '-D 10' for duration")
					}
				}
			},
		},
		{
			name: "With report enabled",
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
					Hostname: []string{"server1"},
					Hca:      []string{"mlx5_0"},
				},
				Client: config.ClientConfig{
					Hostname: []string{"client1"},
					Hca:      []string{"mlx5_0"},
				},
			},
			wantErr: false,
			validate: func(t *testing.T, result *generator.ScriptResult, cfg *config.Config) {
				serverCmd := result.ServerScripts[0].Command
				t.Logf("Server command: %s", serverCmd)
				if !strings.Contains(serverCmd, "--report_gbits") {
					t.Error("Server command should contain '--report_gbits'")
				}
				if !strings.Contains(serverCmd, "--out_json") {
					t.Error("Server command should contain '--out_json'")
				}
				if !strings.Contains(serverCmd, "--out_json_file /tmp/reports/report_s_") {
					t.Error("Server command should contain report file path")
				}

				clientCmd := result.ClientScripts[0].Command
				if !strings.Contains(clientCmd, "--report_gbits") {
					t.Error("Client command should contain '--report_gbits'")
				}
				if !strings.Contains(clientCmd, "--out_json_file /tmp/reports/report_c_") {
					t.Error("Client command should contain report file path")
				}
			},
		},
		{
			name: "Port exhaustion error",
			cfg: &config.Config{
				StartPort:        65530,
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
					Hostname: []string{"server1", "server2"},
					Hca:      []string{"mlx5_0", "mlx5_1"},
				},
				Client: config.ClientConfig{
					Hostname: []string{"client1", "client2"},
					Hca:      []string{"mlx5_0", "mlx5_1"},
				},
			},
			wantErr:     true,
			errContains: "not enough available ports",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock IPs for all server hosts
			mockIPs := make(map[string]string)
			for i, host := range tt.cfg.Server.Hostname {
				mockIPs[host] = fmt.Sprintf("192.168.1.%d", 10+i)
			}

			gen := generator.NewBwIncastScriptGenerator(tt.cfg, mockIPs)
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

func TestBwIncastScriptGenerator_CheckPortsAvailability(t *testing.T) {
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
					Hostname: []string{"server1"},
					Hca:      []string{"mlx5_0"},
				},
				Client: config.ClientConfig{
					Hostname: []string{"client1"},
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
					Hostname: []string{"server1"},
					Hca:      []string{"mlx5_0", "mlx5_1"},
				},
				Client: config.ClientConfig{
					Hostname: []string{"client1"},
					Hca:      []string{"mlx5_0", "mlx5_1"},
				},
			},
			wantErr:     true,
			errContains: "not enough available ports",
		},
		{
			name: "Large scale configuration",
			cfg: &config.Config{
				StartPort: 20000,
				Server: config.ServerConfig{
					Hostname: []string{"server1", "server2"},
					Hca:      []string{"mlx5_0", "mlx5_1"},
				},
				Client: config.ClientConfig{
					Hostname: []string{"client1", "client2", "client3"},
					Hca:      []string{"mlx5_0", "mlx5_1"},
				},
			},
			wantErr: false, // 2*2*3*2 = 24 ports needed, plenty available
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gen := generator.NewBwIncastScriptGenerator(tt.cfg, nil)
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

func TestBwIncastScriptGenerator_CommandFormat(t *testing.T) {
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
			Hostname: []string{"server1"},
			Hca:      []string{"mlx5_0"},
		},
		Client: config.ClientConfig{
			Hostname: []string{"client1"},
			Hca:      []string{"mlx5_0"},
		},
	}

	mockIPs := map[string]string{"server1": "192.168.1.10"}
	gen := generator.NewBwIncastScriptGenerator(cfg, mockIPs)
	result, err := gen.GenerateScripts()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Validate command delimiter
	serverCmd := result.ServerScripts[0].Command
	if !strings.Contains(serverCmd, "( ib_write_bw") {
		t.Error("Server command should wrap ib_write_bw in parentheses")
	}
	if !strings.Contains(serverCmd, " )") {
		t.Error("Server command should close parentheses")
	}

	clientCmd := result.ClientScripts[0].Command
	if !strings.Contains(clientCmd, "( ib_write_bw") {
		t.Error("Client command should wrap ib_write_bw in parentheses")
	}
}
