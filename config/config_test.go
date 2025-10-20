package config

import (
	"testing"
)

func TestApplyDefaults(t *testing.T) {
	tests := []struct {
		name     string
		input    Config
		expected Config
	}{
		{
			name:  "empty config should get all defaults",
			input: Config{},
			expected: Config{
				StartPort:          20000,
				StreamType:         InCast,
				QpNum:              10,
				MessageSizeBytes:   4096,
				OutputBase:         "./generated_scripts",
				WaitingTimeSeconds: 15,
				Speed:              400,
				RdmaCm:             false,
				Report: Report{
					Enable: false, // Not set by ApplyDefaults since it's a bool
					Dir:    "/root",
				},
				Run: Run{
					Infinitely:      false, // Not set by ApplyDefaults since it's a bool
					DurationSeconds: 10,
				},
			},
		},
		{
			name: "partial config should only fill missing fields",
			input: Config{
				StartPort:        25000,
				StreamType:       P2P,
				MessageSizeBytes: 8192,
			},
			expected: Config{
				StartPort:          25000, // User value preserved
				StreamType:         P2P,   // User value preserved
				QpNum:              10,    // Default applied
				MessageSizeBytes:   8192,  // User value preserved
				OutputBase:         "./generated_scripts",
				WaitingTimeSeconds: 15,
				Speed:              400,
				RdmaCm:             false,
				Report: Report{
					Dir: "/root",
				},
				Run: Run{
					DurationSeconds: 10,
				},
			},
		},
		{
			name: "full config should not change",
			input: Config{
				StartPort:          30000,
				StreamType:         FullMesh,
				QpNum:              20,
				MessageSizeBytes:   16384,
				OutputBase:         "./custom",
				WaitingTimeSeconds: 30,
				Speed:              200,
				RdmaCm:             true,
				Report: Report{
					Enable: true,
					Dir:    "/tmp",
				},
				Run: Run{
					Infinitely:      true,
					DurationSeconds: 60,
				},
			},
			expected: Config{
				StartPort:          30000,
				StreamType:         FullMesh,
				QpNum:              20,
				MessageSizeBytes:   16384,
				OutputBase:         "./custom",
				WaitingTimeSeconds: 30,
				Speed:              200,
				RdmaCm:             true,
				Report: Report{
					Enable: true,
					Dir:    "/tmp",
				},
				Run: Run{
					Infinitely:      true,
					DurationSeconds: 60,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := tt.input
			cfg.ApplyDefaults()

			// Check all fields
			if cfg.StartPort != tt.expected.StartPort {
				t.Errorf("StartPort = %d, want %d", cfg.StartPort, tt.expected.StartPort)
			}
			if cfg.StreamType != tt.expected.StreamType {
				t.Errorf("StreamType = %s, want %s", cfg.StreamType, tt.expected.StreamType)
			}
			if cfg.QpNum != tt.expected.QpNum {
				t.Errorf("QpNum = %d, want %d", cfg.QpNum, tt.expected.QpNum)
			}
			if cfg.MessageSizeBytes != tt.expected.MessageSizeBytes {
				t.Errorf("MessageSizeBytes = %d, want %d", cfg.MessageSizeBytes, tt.expected.MessageSizeBytes)
			}
			if cfg.OutputBase != tt.expected.OutputBase {
				t.Errorf("OutputBase = %s, want %s", cfg.OutputBase, tt.expected.OutputBase)
			}
			if cfg.WaitingTimeSeconds != tt.expected.WaitingTimeSeconds {
				t.Errorf("WaitingTimeSeconds = %d, want %d", cfg.WaitingTimeSeconds, tt.expected.WaitingTimeSeconds)
			}
			if cfg.Speed != tt.expected.Speed {
				t.Errorf("Speed = %f, want %f", cfg.Speed, tt.expected.Speed)
			}
			if cfg.Report.Dir != tt.expected.Report.Dir {
				t.Errorf("Report.Dir = %s, want %s", cfg.Report.Dir, tt.expected.Report.Dir)
			}
			if cfg.Run.DurationSeconds != tt.expected.Run.DurationSeconds {
				t.Errorf("Run.DurationSeconds = %d, want %d", cfg.Run.DurationSeconds, tt.expected.Run.DurationSeconds)
			}
		})
	}
}

func TestNewDefaultConfig(t *testing.T) {
	cfg := NewDefaultConfig()

	if cfg.StartPort != 20000 {
		t.Errorf("StartPort = %d, want 20000", cfg.StartPort)
	}
	if cfg.StreamType != InCast {
		t.Errorf("StreamType = %s, want %s", cfg.StreamType, InCast)
	}
	if cfg.QpNum != 10 {
		t.Errorf("QpNum = %d, want 10", cfg.QpNum)
	}
	if cfg.MessageSizeBytes != 4096 {
		t.Errorf("MessageSizeBytes = %d, want 4096", cfg.MessageSizeBytes)
	}
	if cfg.OutputBase != "./generated_scripts" {
		t.Errorf("OutputBase = %s, want ./generated_scripts", cfg.OutputBase)
	}
	if cfg.WaitingTimeSeconds != 15 {
		t.Errorf("WaitingTimeSeconds = %d, want 15", cfg.WaitingTimeSeconds)
	}
	if cfg.Speed != 400 {
		t.Errorf("Speed = %f, want 400", cfg.Speed)
	}
	if cfg.RdmaCm != false {
		t.Errorf("RdmaCm = %v, want false", cfg.RdmaCm)
	}
	if !cfg.Report.Enable {
		t.Errorf("Report.Enable = %v, want true", cfg.Report.Enable)
	}
	if cfg.Report.Dir != "/root" {
		t.Errorf("Report.Dir = %s, want /root", cfg.Report.Dir)
	}
	if cfg.Run.Infinitely != false {
		t.Errorf("Run.Infinitely = %v, want false", cfg.Run.Infinitely)
	}
	if cfg.Run.DurationSeconds != 10 {
		t.Errorf("Run.DurationSeconds = %d, want 10", cfg.Run.DurationSeconds)
	}
	if len(cfg.Server.Hostname) != 0 {
		t.Errorf("Server.Hostname should be empty")
	}
	if len(cfg.Server.Hca) != 0 {
		t.Errorf("Server.Hca should be empty")
	}
	if len(cfg.Client.Hostname) != 0 {
		t.Errorf("Client.Hostname should be empty")
	}
	if len(cfg.Client.Hca) != 0 {
		t.Errorf("Client.Hca should be empty")
	}
}
