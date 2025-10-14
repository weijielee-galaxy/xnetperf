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
	StartPort          int          `yaml:"start_port" json:"start_port"`
	StreamType         string       `yaml:"stream_type" json:"stream_type"`
	QpNum              int          `yaml:"qp_num" json:"qp_num"`
	MessageSizeBytes   int          `yaml:"message_size_bytes" json:"message_size_bytes"`
	OutputBase         string       `yaml:"output_base" json:"output_base"`
	WaitingTimeSeconds int          `yaml:"waiting_time_seconds" json:"waiting_time_seconds"`
	Speed              float64      `yaml:"speed" json:"speed"` // in Gbps
	RdmaCm             bool         `yaml:"rdma_cm" json:"rdma_cm"`
	Report             Report       `yaml:"report" json:"report"`
	Run                Run          `yaml:"run" json:"run"`
	Server             ServerConfig `yaml:"server" json:"server"`
	Client             ClientConfig `yaml:"client" json:"client"`
}

type Report struct {
	Enable bool   `yaml:"enable" json:"enable"`
	Dir    string `yaml:"dir" json:"dir"`
}

type Run struct {
	Infinitely      bool `yaml:"infinitely" json:"infinitely"`
	DurationSeconds int  `yaml:"duration_seconds" json:"duration_seconds"`
}

// ServerConfig holds the server-specific settings.
type ServerConfig struct {
	Hostname []string `yaml:"hostname" json:"hostname"`
	Hca      []string `yaml:"hca" json:"hca"`
}

// ClientConfig holds the client-specific settings.
type ClientConfig struct {
	Hostname []string `yaml:"hostname" json:"hostname"`
	Hca      []string `yaml:"hca" json:"hca"`
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

// NewDefaultConfig creates a new config with default values
func NewDefaultConfig() *Config {
	return &Config{
		StartPort:          20000,
		StreamType:         InCast,
		QpNum:              10,
		MessageSizeBytes:   4096,
		OutputBase:         "./generated_scripts",
		WaitingTimeSeconds: 15,
		Speed:              400,
		RdmaCm:             false,
		Report: Report{
			Enable: true,
			Dir:    "/root",
		},
		Run: Run{
			Infinitely:      false,
			DurationSeconds: 10,
		},
		Server: ServerConfig{
			Hostname: []string{},
			Hca:      []string{},
		},
		Client: ClientConfig{
			Hostname: []string{},
			Hca:      []string{},
		},
	}
}

// ApplyDefaults applies default values to missing fields in the config
func (c *Config) ApplyDefaults() {
	if c.StartPort == 0 {
		c.StartPort = 20000
	}
	if c.StreamType == "" {
		c.StreamType = InCast
	}
	if c.QpNum == 0 {
		c.QpNum = 10
	}
	if c.MessageSizeBytes == 0 {
		c.MessageSizeBytes = 4096
	}
	if c.OutputBase == "" {
		c.OutputBase = "./generated_scripts"
	}
	if c.WaitingTimeSeconds == 0 {
		c.WaitingTimeSeconds = 15
	}
	if c.Speed == 0 {
		c.Speed = 400
	}
	// Report defaults
	if c.Report.Dir == "" {
		c.Report.Dir = "/root"
	}
	// Run defaults
	if c.Run.DurationSeconds == 0 {
		c.Run.DurationSeconds = 10
	}
}
