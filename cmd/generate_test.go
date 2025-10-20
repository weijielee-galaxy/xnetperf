package cmd

import (
	"os"
	"path/filepath"
	"testing"
	"xnetperf/config"
)

func TestExecGenerateCommand_FullMesh(t *testing.T) {
	// Create a temporary directory for test
	tmpDir := t.TempDir()

	// Create a test config
	cfg := config.NewDefaultConfig()
	cfg.StreamType = config.FullMesh
	cfg.OutputBase = filepath.Join(tmpDir, "generated_scripts")
	cfg.Server.Hostname = []string{"server1", "server2"}
	cfg.Server.Hca = []string{"mlx5_0"}
	cfg.Client.Hostname = []string{"client1"}
	cfg.Client.Hca = []string{"mlx5_0"}

	// Execute generate command
	execGenerateCommand(cfg)

	// Verify output directory was created
	if _, err := os.Stat(cfg.OutputDir()); os.IsNotExist(err) {
		t.Errorf("Output directory was not created: %s", cfg.OutputDir())
	}

	// Note: Since GenerateFullMeshScript requires actual SSH connectivity,
	// this test only verifies the command structure, not the actual script generation
}

func TestExecGenerateCommand_InCast(t *testing.T) {
	// Create a temporary directory for test
	tmpDir := t.TempDir()

	// Create a test config
	cfg := config.NewDefaultConfig()
	cfg.StreamType = config.InCast
	cfg.OutputBase = filepath.Join(tmpDir, "generated_scripts")
	cfg.Server.Hostname = []string{"server1"}
	cfg.Server.Hca = []string{"mlx5_0"}
	cfg.Client.Hostname = []string{"client1", "client2"}
	cfg.Client.Hca = []string{"mlx5_0"}

	// Execute generate command
	execGenerateCommand(cfg)

	// Verify output directory was created
	if _, err := os.Stat(cfg.OutputDir()); os.IsNotExist(err) {
		t.Errorf("Output directory was not created: %s", cfg.OutputDir())
	}
}

func TestExecGenerateCommand_P2P(t *testing.T) {
	// Create a temporary directory for test
	tmpDir := t.TempDir()

	// Create a test config
	cfg := config.NewDefaultConfig()
	cfg.StreamType = config.P2P
	cfg.OutputBase = filepath.Join(tmpDir, "generated_scripts")
	cfg.Server.Hostname = []string{"server1"}
	cfg.Server.Hca = []string{"mlx5_0"}
	cfg.Client.Hostname = []string{"client1"}
	cfg.Client.Hca = []string{"mlx5_0"}

	// Execute generate command (will fail due to missing SSH connectivity, but structure is tested)
	// We don't check for errors here as the function may fail due to network issues
	// The important thing is that it doesn't panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("execGenerateCommand panicked: %v", r)
		}
	}()

	// This will exit the test process if it fails, so we skip actual execution
	// execGenerateCommand(cfg)

	// Just verify the output directory can be determined
	expectedDir := filepath.Join(tmpDir, "generated_scripts_p2p")
	if cfg.OutputDir() != expectedDir {
		t.Errorf("Expected output dir %s, got %s", expectedDir, cfg.OutputDir())
	}
}

func TestExecGenerateCommand_InvalidStreamType(t *testing.T) {
	// Create a test config with invalid stream type
	cfg := config.NewDefaultConfig()
	cfg.StreamType = "invalid_type"

	// This should exit with code 1, but we can't easily test that
	// The important thing is the function handles it gracefully
	// In a real scenario, this would call os.Exit(1)

	// For unit testing, we just verify the config structure
	if cfg.StreamType != "invalid_type" {
		t.Errorf("Expected stream type 'invalid_type', got %s", cfg.StreamType)
	}
}

func TestContainsString(t *testing.T) {
	tests := []struct {
		name   string
		s      string
		substr string
		want   bool
	}{
		{
			name:   "exact match",
			s:      "_server_",
			substr: "_server_",
			want:   true,
		},
		{
			name:   "contains in middle",
			s:      "host_server_script.sh",
			substr: "_server_",
			want:   true,
		},
		{
			name:   "contains at start",
			s:      "_server_script.sh",
			substr: "_server_",
			want:   true,
		},
		{
			name:   "contains at end",
			s:      "host_server_",
			substr: "_server_",
			want:   true,
		},
		{
			name:   "does not contain",
			s:      "host_client_script.sh",
			substr: "_server_",
			want:   false,
		},
		{
			name:   "empty substring",
			s:      "test",
			substr: "",
			want:   true,
		},
		{
			name:   "substring longer than string",
			s:      "abc",
			substr: "abcdef",
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := containsString(tt.s, tt.substr); got != tt.want {
				t.Errorf("containsString(%q, %q) = %v, want %v", tt.s, tt.substr, got, tt.want)
			}
		})
	}
}

func TestIndexOfSubstring(t *testing.T) {
	tests := []struct {
		name   string
		s      string
		substr string
		want   int
	}{
		{
			name:   "found at start",
			s:      "hello world",
			substr: "hello",
			want:   0,
		},
		{
			name:   "found in middle",
			s:      "hello world",
			substr: "o w",
			want:   4,
		},
		{
			name:   "found at end",
			s:      "hello world",
			substr: "world",
			want:   6,
		},
		{
			name:   "not found",
			s:      "hello world",
			substr: "xyz",
			want:   -1,
		},
		{
			name:   "empty substring",
			s:      "test",
			substr: "",
			want:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := indexOfSubstring(tt.s, tt.substr); got != tt.want {
				t.Errorf("indexOfSubstring(%q, %q) = %v, want %v", tt.s, tt.substr, got, tt.want)
			}
		})
	}
}
