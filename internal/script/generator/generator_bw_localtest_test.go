package generator_test

import (
	"fmt"
	"strings"
	"testing"
	"xnetperf/config"
	"xnetperf/internal/script/generator"
)

func TestBwLocaltestScriptGenerator_GenerateScripts(t *testing.T) {
	tests := []struct {
		name        string
		cfg         *config.Config
		wantErr     bool
		errContains string
		validate    func(t *testing.T, result *generator.ScriptResult, cfg *config.Config)
	}{
		{
			name: "Basic localtest configuration with 2 hosts, 1 HCA",
			cfg: &config.Config{
				Server: config.ServerConfig{
					Hostname: []string{"host1", "host2"},
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
					Enable: false,
				},
			},
			validate: func(t *testing.T, result *generator.ScriptResult, cfg *config.Config) {
				if result == nil {
					t.Fatal("Expected result, got nil")
				}

				// Localtest: 2 hosts, key按host分组
				expectedHosts := 2
				if len(result.ServerScripts) != expectedHosts {
					t.Errorf("Expected %d server scripts, got %d", expectedHosts, len(result.ServerScripts))
				}
				if len(result.ClientScripts) != expectedHosts {
					t.Errorf("Expected %d client scripts, got %d", expectedHosts, len(result.ClientScripts))
				}

				// Verify each host has correct number of commands
				// 对于host1: 作为server, 要接收来自所有host_hca的连接
				// 2 hosts * 1 HCA = 2个server命令 (host1_mlx5_0作为server, host2_mlx5_0作为server)
				// 作为client: 要连接到所有host_hca
				// 2 hosts * 1 HCA = 2个client命令
				for _, script := range result.ServerScripts {
					cmdCount := strings.Count(script.Command, "ib_write_bw")
					expectedCmds := 2
					if cmdCount != expectedCmds {
						t.Errorf("Host %s: expected %d server commands, got %d", script.Host, expectedCmds, cmdCount)
					}
				}

				for _, script := range result.ClientScripts {
					cmdCount := strings.Count(script.Command, "ib_write_bw")
					expectedCmds := 2
					if cmdCount != expectedCmds {
						t.Errorf("Host %s: expected %d client commands, got %d", script.Host, expectedCmds, cmdCount)
					}
				}
			},
		},
		{
			name: "Localtest with multiple HCAs",
			cfg: &config.Config{
				Server: config.ServerConfig{
					Hostname: []string{"host1", "host2"},
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
				// 2 hosts, key按host分组
				expectedHosts := 2
				if len(result.ServerScripts) != expectedHosts {
					t.Errorf("Expected %d server scripts, got %d", expectedHosts, len(result.ServerScripts))
				}

				// 每个host的每个HCA都要作为server接收来自所有host_hca的连接
				// host1有2个HCA，每个HCA作为server接收来自 2hosts*2HCAs=4 个连接
				// 所以host1总共: 2 HCAs * 4 connections = 8 server commands
				for _, script := range result.ServerScripts {
					cmdCount := strings.Count(script.Command, "ib_write_bw")
					expectedCmds := 8
					if cmdCount != expectedCmds {
						t.Errorf("Host %s: expected %d server commands, got %d", script.Host, expectedCmds, cmdCount)
					}
				}

				// 每个host的每个HCA都要作为client连接到所有host_hca
				// host1有2个HCA，每个HCA作为client连接到 2hosts*2HCAs=4 个server
				// 所以host1总共: 2 HCAs * 4 connections = 8 client commands
				for _, script := range result.ClientScripts {
					cmdCount := strings.Count(script.Command, "ib_write_bw")
					expectedCmds := 8
					if cmdCount != expectedCmds {
						t.Errorf("Host %s: expected %d client commands, got %d", script.Host, expectedCmds, cmdCount)
					}
				}
			},
		},
		{
			name: "Localtest with report enabled",
			cfg: &config.Config{
				Server: config.ServerConfig{
					Hostname: []string{"host1"},
					Hca:      []string{"mlx5_0"},
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
					Enable: true,
					Dir:    "/tmp/reports",
				},
			},
			validate: func(t *testing.T, result *generator.ScriptResult, cfg *config.Config) {
				// Verify report flags are present
				serverCmd := result.ServerScripts[0].Command
				if !strings.Contains(serverCmd, "--report_gbits") {
					t.Error("Server command should contain --report_gbits flag")
				}
				if !strings.Contains(serverCmd, "/tmp/reports/report_s_") {
					t.Error("Server command should contain report file path")
				}

				clientCmd := result.ClientScripts[0].Command
				if !strings.Contains(clientCmd, "--report_gbits") {
					t.Error("Client command should contain --report_gbits flag")
				}
				if !strings.Contains(clientCmd, "/tmp/reports/report_c_") {
					t.Error("Client command should contain report file path")
				}
			},
		},
		{
			name: "Verify self-connections are included",
			cfg: &config.Config{
				Server: config.ServerConfig{
					Hostname: []string{"host1"},
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
				// 1 host按host分组
				if len(result.ServerScripts) != 1 {
					t.Errorf("Expected 1 server script, got %d", len(result.ServerScripts))
				}

				// host1有2个HCA，每个HCA要接收来自2个HCA的连接（包括自己）
				// 2 HCAs * 2 connections = 4 server commands
				cmdCount := strings.Count(result.ServerScripts[0].Command, "ib_write_bw")
				expectedCmds := 4
				if cmdCount != expectedCmds {
					t.Errorf("Expected %d server commands (including self-connections), got %d", expectedCmds, cmdCount)
				}

				// Verify self-connection exists (mlx5_0 -> mlx5_0)
				// 通过检查port 20000的命令（mlx5_0作为server，mlx5_0作为client）
				if !strings.Contains(result.ServerScripts[0].Command, "-p 20000") {
					t.Error("Should have command with port 20000 (self-connection)")
				}
			},
		},
		{
			name: "Port exhaustion error",
			cfg: &config.Config{
				Server: config.ServerConfig{
					Hostname: []string{"host1", "host2", "host3", "host4", "host5"},
					Hca:      []string{"mlx5_0", "mlx5_1", "mlx5_2", "mlx5_3", "mlx5_4"},
				},
				StartPort:        65000, // Very high start port
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
			wantErr:     true,
			errContains: "not enough available ports",
			// 5 hosts * 5 HCAs = 25 combinations, 25*25 = 625 ports needed
			// 从65000开始只有536个端口，不够
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock IPs
			mockIPs := make(map[string]string)
			for i, host := range tt.cfg.Server.Hostname {
				mockIPs[host] = fmt.Sprintf("192.168.1.%d", 10+i)
			}

			gen := generator.NewBwLocaltestScriptGenerator(tt.cfg, mockIPs)
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

func TestBwLocaltestScriptGenerator_CheckPortsAvailability(t *testing.T) {
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
					Hca:      []string{"mlx5_0"},
				},
				StartPort: 20000,
			},
			wantErr: false,
			// 2 hosts * 1 HCA = 2 combinations, 2*2 = 4 ports
		},
		{
			name: "Not enough ports - starting from high port",
			cfg: &config.Config{
				Server: config.ServerConfig{
					Hostname: []string{"host1", "host2", "host3", "host4", "host5"},
					Hca:      []string{"mlx5_0", "mlx5_1", "mlx5_2", "mlx5_3", "mlx5_4"},
				},
				StartPort: 65000,
			},
			wantErr:     true,
			errContains: "not enough available ports",
			// 5 hosts * 5 HCAs = 25 combinations, 25*25 = 625 ports needed
			// 从65000开始只有536个端口，不够
		},
		{
			name: "Large scale localtest configuration",
			cfg: &config.Config{
				Server: config.ServerConfig{
					Hostname: []string{"host1", "host2", "host3", "host4"},
					Hca:      []string{"mlx5_0", "mlx5_1", "mlx5_2"},
				},
				StartPort: 20000,
			},
			wantErr: false,
			// 4 hosts * 3 HCAs = 12 combinations, 12*12 = 144 ports needed
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockIPs := make(map[string]string)
			for i, host := range tt.cfg.Server.Hostname {
				mockIPs[host] = fmt.Sprintf("192.168.1.%d", 10+i)
			}

			gen := generator.NewBwLocaltestScriptGenerator(tt.cfg, mockIPs)
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

func TestBwLocaltestScriptGenerator_CommandFormat(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Hostname: []string{"host1", "host2"},
			Hca:      []string{"mlx5_0"},
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
	}

	mockIPs := map[string]string{"host1": "192.168.1.10", "host2": "192.168.1.11"}
	gen := generator.NewBwLocaltestScriptGenerator(cfg, mockIPs)
	result, err := gen.GenerateScripts()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Verify command delimiter and parentheses
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
	if !strings.Contains(clientCmd, " )") {
		t.Error("Client command should close parentheses")
	}

	// Verify delimiter (有多个命令时才有delimiter)
	// 2 hosts * 1 HCA = 2 commands per host，所以会有delimiter
	if !strings.Contains(serverCmd, " && \\ \n") {
		t.Error("Server commands should be joined with proper delimiter")
	}
}

func TestBwLocaltestScriptGenerator_HostKeyFormat(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Hostname: []string{"host1", "host2"},
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
		"host2": "192.168.1.11",
	}
	gen := generator.NewBwLocaltestScriptGenerator(cfg, mockIPs)
	result, err := gen.GenerateScripts()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Verify host keys are in "hostname" format (not "hostname_hca")
	expectedKeys := map[string]bool{
		"host1": false,
		"host2": false,
	}

	for _, script := range result.ServerScripts {
		if _, exists := expectedKeys[script.Host]; !exists {
			t.Errorf("Unexpected server script key: %s", script.Host)
		}
		expectedKeys[script.Host] = true
	}

	for key, found := range expectedKeys {
		if !found {
			t.Errorf("Expected server script for key %s not found", key)
		}
	}

	// Verify no "host_hca" format keys
	for _, script := range result.ServerScripts {
		if strings.Contains(script.Host, "_mlx5") {
			t.Errorf("Script key should not contain HCA name: %s", script.Host)
		}
	}
}
