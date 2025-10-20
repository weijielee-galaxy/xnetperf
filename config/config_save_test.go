package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveConfig(t *testing.T) {
	// Create a temporary directory for test
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test_config.yaml")

	// Create a test config
	cfg := NewDefaultConfig()
	cfg.StartPort = 30000
	cfg.StreamType = FullMesh

	// Save the config
	err := SaveConfig(testFile, cfg)
	if err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Load the config back
	loadedCfg, err := LoadConfig(testFile)
	if err != nil {
		t.Fatalf("Failed to load saved config: %v", err)
	}

	// Verify the values
	if loadedCfg.StartPort != 30000 {
		t.Errorf("Expected StartPort 30000, got %d", loadedCfg.StartPort)
	}
	if loadedCfg.StreamType != FullMesh {
		t.Errorf("Expected StreamType %s, got %s", FullMesh, loadedCfg.StreamType)
	}
}

func TestEnsureConfigFile_CreatesNewFile(t *testing.T) {
	// Create a temporary directory for test
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "config.yaml")

	// Ensure the file doesn't exist yet
	if _, err := os.Stat(testFile); !os.IsNotExist(err) {
		t.Fatal("Test file should not exist yet")
	}

	// Call EnsureConfigFile
	err := EnsureConfigFile(testFile)
	if err != nil {
		t.Fatalf("EnsureConfigFile failed: %v", err)
	}

	// Verify the file was created
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Fatal("Config file should have been created")
	}

	// Load the config and verify it has default values
	cfg, err := LoadConfig(testFile)
	if err != nil {
		t.Fatalf("Failed to load created config: %v", err)
	}

	if cfg.StartPort != 20000 {
		t.Errorf("Expected default StartPort 20000, got %d", cfg.StartPort)
	}
	if cfg.StreamType != InCast {
		t.Errorf("Expected default StreamType %s, got %s", InCast, cfg.StreamType)
	}
	if cfg.SSH.PrivateKey != "~/.ssh/id_rsa" {
		t.Errorf("Expected default SSH key ~/.ssh/id_rsa, got %s", cfg.SSH.PrivateKey)
	}
}

func TestEnsureConfigFile_ExistingFile(t *testing.T) {
	// Create a temporary directory for test
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "config.yaml")

	// Create a config file with custom values
	customCfg := NewDefaultConfig()
	customCfg.StartPort = 40000
	customCfg.StreamType = P2P
	err := SaveConfig(testFile, customCfg)
	if err != nil {
		t.Fatalf("Failed to create custom config: %v", err)
	}

	// Call EnsureConfigFile on existing file
	err = EnsureConfigFile(testFile)
	if err != nil {
		t.Fatalf("EnsureConfigFile failed: %v", err)
	}

	// Load the config and verify it still has custom values (not overwritten)
	cfg, err := LoadConfig(testFile)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if cfg.StartPort != 40000 {
		t.Errorf("Expected StartPort 40000 (should not be overwritten), got %d", cfg.StartPort)
	}
	if cfg.StreamType != P2P {
		t.Errorf("Expected StreamType %s (should not be overwritten), got %s", P2P, cfg.StreamType)
	}
}
