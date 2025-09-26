package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

const (
	FullMesh string = "fullmesh"
	InCast   string = "incast"
	P2P      string = "p2p"
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
	Speed              float64      `yaml:"speed"` // in Gbps
	Report             Report       `yaml:"report"`
	Run                Run          `yaml:"run"`
	Server             ServerConfig `yaml:"server"`
	Client             ClientConfig `yaml:"client"`
}

type Report struct {
	Enable bool   `yaml:"enable"`
	Dir    string `yaml:"dir"`
}

type Run struct {
	Infinitely      bool `yaml:"infinitely"`
	DurationSeconds int  `yaml:"duration_seconds"`
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

func (c *Config) IsFullMesh() bool {
	return c.StreamType == FullMesh
}

func (c *Config) IsInCast() bool {
	return c.StreamType == InCast
}

func (c *Config) IsP2P() bool {
	return c.StreamType == P2P
}

func (c *Config) OutputDir() string {
	return fmt.Sprintf("%s_%s", c.OutputBase, c.StreamType)
}

func LoadConfig(filePath string) (*Config, error) {
	var cfg Config
	yamlFile, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file '%s': %w", filePath, err)
	}
	err = yaml.Unmarshal(yamlFile, &cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to parse YAML content from '%s': %w", filePath, err)
	}
	return &cfg, nil
}
