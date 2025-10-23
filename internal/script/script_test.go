package script

import (
	"testing"

	"xnetperf/config"
)

func TestIsValidMode(t *testing.T) {
	tests := []struct {
		name     string
		mode     TestMode
		expected bool
	}{
		{"Valid BW Fullmesh", ModeBwFullmesh, true},
		{"Valid BW Incast", ModeBwIncast, true},
		{"Valid BW P2P", ModeBwP2P, true},
		{"Valid BW Localtest", ModeBwLocaltest, true},
		{"Valid LAT Fullmesh", ModeLatFullmesh, true},
		{"Valid LAT Incast", ModeLatIncast, true},
		{"Valid LAT P2P", ModeLatP2P, true},
		{"Valid LAT Localtest", ModeLatLocaltest, true},
		{"Invalid Mode", TestMode("invalid"), false},
		{"Empty Mode", TestMode(""), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidMode(tt.mode)
			if result != tt.expected {
				t.Errorf("IsValidMode(%v) = %v, want %v", tt.mode, result, tt.expected)
			}
		})
	}
}

func TestGetSupportedModes(t *testing.T) {
	modes := GetSupportedModes()
	expected := 8
	if len(modes) != expected {
		t.Errorf("GetSupportedModes() returned %d modes, want %d", len(modes), expected)
	}

	// Check all modes are present
	expectedModes := []TestMode{
		ModeBwFullmesh, ModeBwIncast, ModeBwP2P, ModeBwLocaltest,
		ModeLatFullmesh, ModeLatIncast, ModeLatP2P, ModeLatLocaltest,
	}
	modeMap := make(map[TestMode]bool)
	for _, m := range modes {
		modeMap[m] = true
	}

	for _, mode := range expectedModes {
		if !modeMap[mode] {
			t.Errorf("GetSupportedModes() missing mode: %v", mode)
		}
	}
}

func TestNewExecutor(t *testing.T) {
	cfg := &config.Config{
		StartPort:        20000,
		MessageSizeBytes: 4096,
		QpNum:            8,
		RdmaCm:           true,
		Server: config.ServerConfig{
			Hostname: []string{"server1"},
			Hca:      []string{"mlx5_0"},
		},
		Client: config.ClientConfig{
			Hostname: []string{"client1"},
			Hca:      []string{"mlx5_0"},
		},
	}

	tests := []struct {
		name      string
		cfg       *config.Config
		mode      TestMode
		expectErr bool
	}{
		{
			name:      "Valid BW Fullmesh",
			cfg:       cfg,
			mode:      ModeBwFullmesh,
			expectErr: false,
		},
		{
			name:      "Invalid Mode",
			cfg:       cfg,
			mode:      TestMode("invalid"),
			expectErr: true,
		},
		{
			name:      "Nil Config",
			cfg:       nil,
			mode:      ModeBwFullmesh,
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exec := NewExecutor(tt.cfg, tt.mode)
			if tt.expectErr {
				if exec != nil {
					t.Error("Expected nil executor for invalid input")
				}
			} else {
				if exec == nil {
					t.Error("Expected non-nil executor")
				}
				if exec != nil && exec.mode != tt.mode {
					t.Errorf("Executor mode = %v, want %v", exec.mode, tt.mode)
				}
			}
		})
	}
}

func TestExecutorGenerateScripts_BwFullmesh(t *testing.T) {
	cfg := &config.Config{
		StartPort:        20000,
		MessageSizeBytes: 4096,
		QpNum:            8,
		RdmaCm:           true,
		Run: config.Run{
			Infinitely:      false,
			DurationSeconds: 10,
		},
		Server: config.ServerConfig{
			Hostname: []string{"192.168.1.1"},
			Hca:      []string{"mlx5_0"},
		},
		Client: config.ClientConfig{
			Hostname: []string{"192.168.1.2"},
			Hca:      []string{"mlx5_0"},
		},
	}

	exec := NewExecutor(cfg, ModeBwFullmesh)
	if exec == nil {
		t.Fatal("NewExecutor returned nil")
	}

	result, err := exec.GenerateScripts()
	if err != nil {
		t.Fatalf("GenerateScripts failed: %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	if len(result.ServerScripts) == 0 {
		t.Error("Expected non-empty server scripts")
	}
	if len(result.ClientScripts) == 0 {
		t.Error("Expected non-empty client scripts")
	}
}

func TestExecutorGenerateScripts_BwIncast(t *testing.T) {
	cfg := &config.Config{
		StartPort:        20000,
		MessageSizeBytes: 4096,
		QpNum:            8,
		RdmaCm:           true,
		Run: config.Run{
			Infinitely:      false,
			DurationSeconds: 10,
		},
		Server: config.ServerConfig{
			Hostname: []string{"192.168.1.1"},
			Hca:      []string{"mlx5_0"},
		},
		Client: config.ClientConfig{
			Hostname: []string{"192.168.1.2", "192.168.1.3"},
			Hca:      []string{"mlx5_0"},
		},
	}

	exec := NewExecutor(cfg, ModeBwIncast)
	if exec == nil {
		t.Fatal("NewExecutor returned nil")
	}

	result, err := exec.GenerateScripts()
	if err != nil {
		t.Fatalf("GenerateScripts failed: %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	if len(result.ServerScripts) == 0 {
		t.Error("Expected non-empty server scripts")
	}
	if len(result.ClientScripts) == 0 {
		t.Error("Expected non-empty client scripts")
	}
}

func TestExecutorGenerateScripts_BwP2P(t *testing.T) {
	cfg := &config.Config{
		StartPort:        20000,
		MessageSizeBytes: 4096,
		QpNum:            8,
		RdmaCm:           true,
		Run: config.Run{
			Infinitely:      false,
			DurationSeconds: 10,
		},
		Server: config.ServerConfig{
			Hostname: []string{"192.168.1.1"},
			Hca:      []string{"mlx5_0"},
		},
		Client: config.ClientConfig{
			Hostname: []string{"192.168.1.2"},
			Hca:      []string{"mlx5_0"},
		},
	}

	exec := NewExecutor(cfg, ModeBwP2P)
	if exec == nil {
		t.Fatal("NewExecutor returned nil")
	}

	result, err := exec.GenerateScripts()
	if err != nil {
		t.Fatalf("GenerateScripts failed: %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	if len(result.ServerScripts) == 0 {
		t.Error("Expected non-empty server scripts")
	}
	if len(result.ClientScripts) == 0 {
		t.Error("Expected non-empty client scripts")
	}
}

func TestExecutorGenerateScripts_BwLocaltest(t *testing.T) {
	cfg := &config.Config{
		StartPort:        20000,
		MessageSizeBytes: 4096,
		QpNum:            8,
		RdmaCm:           true,
		Run: config.Run{
			Infinitely:      false,
			DurationSeconds: 10,
		},
		Server: config.ServerConfig{
			Hostname: []string{"192.168.1.1", "192.168.1.2"},
			Hca:      []string{"mlx5_0"},
		},
	}

	exec := NewExecutor(cfg, ModeBwLocaltest)
	if exec == nil {
		t.Fatal("NewExecutor returned nil")
	}

	result, err := exec.GenerateScripts()
	if err != nil {
		t.Fatalf("GenerateScripts failed: %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	if len(result.ServerScripts) == 0 {
		t.Error("Expected non-empty server scripts")
	}
	if len(result.ClientScripts) == 0 {
		t.Error("Expected non-empty client scripts")
	}
}

func TestExecutorGenerateScripts_LatFullmesh(t *testing.T) {
	cfg := &config.Config{
		StartPort:        20000,
		MessageSizeBytes: 4096,
		QpNum:            8,
		RdmaCm:           true,
		Run: config.Run{
			Infinitely:      false,
			DurationSeconds: 10,
		},
		Report: config.Report{
			Enable: true,
			Dir:    "/tmp/report",
		},
		Server: config.ServerConfig{
			Hostname: []string{"192.168.1.1"},
			Hca:      []string{"mlx5_0"},
		},
		Client: config.ClientConfig{
			Hostname: []string{"192.168.1.2"},
			Hca:      []string{"mlx5_0"},
		},
	}

	exec := NewExecutor(cfg, ModeLatFullmesh)
	if exec == nil {
		t.Fatal("NewExecutor returned nil")
	}

	result, err := exec.GenerateScripts()
	if err != nil {
		t.Fatalf("GenerateScripts failed: %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	if len(result.ServerScripts) == 0 {
		t.Error("Expected non-empty server scripts")
	}
	if len(result.ClientScripts) == 0 {
		t.Error("Expected non-empty client scripts")
	}
}

func TestExecutorGenerateScripts_LatIncast(t *testing.T) {
	cfg := &config.Config{
		StartPort:        20000,
		MessageSizeBytes: 4096,
		QpNum:            8,
		RdmaCm:           true,
		Run: config.Run{
			Infinitely:      false,
			DurationSeconds: 10,
		},
		Report: config.Report{
			Enable: true,
			Dir:    "/tmp/report",
		},
		Server: config.ServerConfig{
			Hostname: []string{"192.168.1.1"},
			Hca:      []string{"mlx5_0"},
		},
		Client: config.ClientConfig{
			Hostname: []string{"192.168.1.2", "192.168.1.3"},
			Hca:      []string{"mlx5_0"},
		},
	}

	exec := NewExecutor(cfg, ModeLatIncast)
	if exec == nil {
		t.Fatal("NewExecutor returned nil")
	}

	result, err := exec.GenerateScripts()
	if err != nil {
		t.Fatalf("GenerateScripts failed: %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	if len(result.ServerScripts) == 0 {
		t.Error("Expected non-empty server scripts")
	}
	if len(result.ClientScripts) == 0 {
		t.Error("Expected non-empty client scripts")
	}
}

func TestExecutorGenerateScripts_LatP2PNotImplemented(t *testing.T) {
	cfg := &config.Config{
		StartPort:        20000,
		MessageSizeBytes: 4096,
		QpNum:            8,
		RdmaCm:           true,
		Run: config.Run{
			Infinitely:      false,
			DurationSeconds: 10,
		},
		Server: config.ServerConfig{
			Hostname: []string{"192.168.1.1"},
			Hca:      []string{"mlx5_0"},
		},
		Client: config.ClientConfig{
			Hostname: []string{"192.168.1.2"},
			Hca:      []string{"mlx5_0"},
		},
	}

	exec := NewExecutor(cfg, ModeLatP2P)
	if exec == nil {
		t.Fatal("NewExecutor returned nil")
	}

	_, err := exec.GenerateScripts()
	if err == nil {
		t.Error("Expected error for unimplemented lat_p2p mode, got nil")
	}
}

func TestExecutorGenerateScripts_LatLocaltestNotImplemented(t *testing.T) {
	cfg := &config.Config{
		StartPort:        20000,
		MessageSizeBytes: 4096,
		QpNum:            8,
		RdmaCm:           true,
		Run: config.Run{
			Infinitely:      false,
			DurationSeconds: 10,
		},
		Server: config.ServerConfig{
			Hostname: []string{"192.168.1.1", "192.168.1.2"},
			Hca:      []string{"mlx5_0"},
		},
	}

	exec := NewExecutor(cfg, ModeLatLocaltest)
	if exec == nil {
		t.Fatal("NewExecutor returned nil")
	}

	_, err := exec.GenerateScripts()
	if err == nil {
		t.Error("Expected error for unimplemented lat_localtest mode, got nil")
	}
}
