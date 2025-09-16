package stream

import (
	"fmt"
	"os"
)

// Config holds the entire configuration from the YAML file.
type Config struct {
	StartPort        int          `yaml:"start_port"`
	DurationSeconds  int          `yaml:"duration_seconds"`
	StreamType       string       `yaml:"stream_type"`
	QpNum            int          `yaml:"qp_num"`
	MessageSizeBytes int          `yaml:"message_size_bytes"`
	Server           ServerConfig `yaml:"server"`
	Client           ClientConfig `yaml:"client"`
}

// ServerConfig holds the server-specific settings.
type ServerConfig struct {
	Hostname []string `yaml:"hostname"`
	Hca      []string `yaml:"hca"`
}

// ClientConfig holds the client-specific settings.
type ClientConfig struct {
	Hostname []string `yaml:"hostname"`
	Hca      []string `yaml:"hca"`
}

func ClearStreamScriptDir() {
	dir := "streamScript"
	err := os.RemoveAll(dir)
	if err != nil {
		fmt.Printf("Error clearing stream script directory: %v\n", err)
		return
	}
	err = os.Mkdir(dir, 0755)
	if err != nil {
		fmt.Printf("Error creating stream script directory: %v\n", err)
		return
	}
	fmt.Printf("Cleared stream script directory: %s\n", dir)
}
