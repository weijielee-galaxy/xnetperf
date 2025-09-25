package stream

import (
	"fmt"
	"os"
)

const (
	FullMesh string = "fullmesh"
	InCast   string = "incast"
)

// Config holds the entire configuration from the YAML file.
type Config struct {
	StartPort          int          `yaml:"start_port"`
	DurationSeconds    int          `yaml:"duration_seconds"`
	StreamType         string       `yaml:"stream_type"`
	QpNum              int          `yaml:"qp_num"`
	MessageSizeBytes   int          `yaml:"message_size_bytes"`
	OutputBase         string       `yaml:"output_base"`
	WaitingTimeSeconds int          `yaml:"waiting_time_seconds"`
	Server             ServerConfig `yaml:"server"`
	Client             ClientConfig `yaml:"client"`
}

func (c *Config) IsFullMesh() bool {
	return c.StreamType == FullMesh
}

func (c *Config) IsInCast() bool {
	return c.StreamType == InCast
}

func (c *Config) OutputDir() string {
	return fmt.Sprintf("%s_%s", c.OutputBase, c.StreamType)
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

func ClearStreamScriptDir(cfg *Config) {
	dir := cfg.OutputDir()
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
