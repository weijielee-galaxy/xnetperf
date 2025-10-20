package config

import (
	"os"
	"testing"
)

func TestSSHConfigDefaults(t *testing.T) {
	cfg := NewDefaultConfig()

	if cfg.SSH.PrivateKey != "~/.ssh/id_rsa" {
		t.Errorf("Expected default SSH private key to be '~/.ssh/id_rsa', got '%s'", cfg.SSH.PrivateKey)
	}
}

func TestLoadSSHConfig(t *testing.T) {
	// 创建临时配置文件
	content := `
start_port: 20000
stream_type: "p2p"
ssh:
  private_key: "/custom/path/id_rsa"
server:
  hostname: []
  hca: []
client:
  hostname: []
  hca: []
`
	tmpFile, err := os.CreateTemp("", "config_ssh_test_*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()

	cfg, err := LoadConfig(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if cfg.SSH.PrivateKey != "/custom/path/id_rsa" {
		t.Errorf("Expected SSH private key to be '/custom/path/id_rsa', got '%s'", cfg.SSH.PrivateKey)
	}
}

func TestApplyDefaultsWithSSH(t *testing.T) {
	cfg := &Config{}
	cfg.ApplyDefaults()

	if cfg.SSH.PrivateKey != "~/.ssh/id_rsa" {
		t.Errorf("Expected default SSH private key after ApplyDefaults to be '~/.ssh/id_rsa', got '%s'", cfg.SSH.PrivateKey)
	}
}
