package stream

import (
	"fmt"
	"testing"
	"xnetperf/config"
)

func TestGenerateIncastScriptsV1(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Hostname: []string{"server1", "server2"},
			Hca:      []string{"hca1", "hca2"},
		},
		Client: config.ClientConfig{
			Hostname: []string{"client1", "client2"},
			Hca:      []string{"hcaA", "hcaB"},
		},
		StartPort: 30000,
		Run: config.Run{
			Infinitely:      false,
			DurationSeconds: 20,
		},
		Report: config.Report{
			Enable: true,
			Dir:    "/reports",
		},
		SSH: config.SSH{
			PrivateKey: "",
		},
		OutputBase: "./test_scripts",
	}
	ret := GenerateIncastScriptsV1(cfg)

	for _, sScript := range ret.ServerScripts {
		fmt.Printf("Server Host: %s\nScript:\n%s\n", sScript.Host, sScript.Command)
	}
	for _, cScript := range ret.ClientScripts {
		fmt.Printf("Client Host: %s\nScript:\n%s\n", cScript.Host, cScript.Command)
	}
}
