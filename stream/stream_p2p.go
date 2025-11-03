package stream

import (
	"fmt"
	"os"
	"strings"
	"xnetperf/config"
	"xnetperf/pkg/tools"
)

// ValidateP2PConfig validates P2P configuration requirements
func ValidateP2PConfig(cfg *config.Config) error {
	// Check if server and client hostname counts are equal
	if len(cfg.Server.Hostname) != len(cfg.Client.Hostname) {
		return fmt.Errorf("P2P mode requires equal number of server and client hostnames. Server: %d, Client: %d",
			len(cfg.Server.Hostname), len(cfg.Client.Hostname))
	}

	// Check if server and client HCA counts are equal
	if len(cfg.Server.Hca) != len(cfg.Client.Hca) {
		return fmt.Errorf("P2P mode requires equal number of server and client HCAs. Server: %d, Client: %d",
			len(cfg.Server.Hca), len(cfg.Client.Hca))
	}

	// Check if P2P with infinitely=true, which is not supported
	// if cfg.Run.Infinitely {
	// 	return fmt.Errorf("P2P mode does not support infinitely=true. Please set run.infinitely=false in config file")
	// }

	return nil
}

// GenerateP2PScripts generates point-to-point scripts for network performance testing
func GenerateP2PScripts(cfg *config.Config) error {
	// Validate P2P configuration
	if err := ValidateP2PConfig(cfg); err != nil {
		fmt.Printf("❌ P2P Configuration Error: %v\n", err)
		return err
	}

	// Clear streamScript directory
	ClearStreamScriptDir(cfg)

	// Calculate total ports needed
	totalPairs := len(cfg.Server.Hostname) * len(cfg.Server.Hca)
	fmt.Printf("Generating P2P scripts for %d host pairs with %d HCA pairs each...\n",
		len(cfg.Server.Hostname), len(cfg.Server.Hca))
	fmt.Printf("Total P2P pairs: %d\n", totalPairs)

	if totalPairs > (65535 - cfg.StartPort + 1) {
		fmt.Printf("❌ Error: Not enough ports available. Required: %d, Available: %d\n",
			totalPairs, 65535-cfg.StartPort+1)
		return fmt.Errorf("not enough ports available")
	}

	// Generate P2P connection pairs
	port := cfg.StartPort

	for hostIndex, serverHost := range cfg.Server.Hostname {
		clientHost := cfg.Client.Hostname[hostIndex]

		// Get server IP
		serverIP, err := getHostIP(serverHost, cfg.SSH.PrivateKey, cfg.NetworkInterface)
		if err != nil {
			fmt.Printf("❌ Error getting IP for server %s: %v\n", serverHost, err)
			// continue
		}

		// Get client IP
		clientIP, err := getHostIP(clientHost, cfg.SSH.PrivateKey, cfg.NetworkInterface)
		if err != nil {
			fmt.Printf("❌ Error getting IP for client %s: %v\n", clientHost, err)
			// continue
		}

		fmt.Printf("Creating P2P pair: %s ↔ %s\n", serverHost, clientHost)

		for hcaIndex, serverHca := range cfg.Server.Hca {
			// Use staggered HCA pairing to avoid same-index connections
			clientHcaIndex := (hcaIndex + 1) % len(cfg.Client.Hca)
			clientHca := cfg.Client.Hca[clientHcaIndex]

			fmt.Printf("  HCA pair: %s.%s ↔ %s.%s (port %d)\n",
				serverHost, serverHca, clientHost, clientHca, port)

			// Generate scripts for this P2P pair
			err := generateP2PScriptPair(cfg, serverHost, serverHca, serverIP,
				clientHost, clientHca, clientIP, port)
			if err != nil {
				fmt.Printf("❌ Error generating scripts for %s.%s ↔ %s.%s: %v\n",
					serverHost, serverHca, clientHost, clientHca, err)
			}

			port++
		}
	}

	fmt.Println("✅ P2P script generation completed")
	return nil
}

// generateP2PScriptPair generates server and client scripts for a P2P connection pair
func generateP2PScriptPair(cfg *config.Config, serverHost, serverHca, serverIP,
	clientHost, clientHca, clientIP string, port int) error {

	// Create script file names
	serverScriptFileName := fmt.Sprintf("%s/%s_%s_server_p2p.sh", cfg.OutputDir(), serverHost, serverHca)
	clientScriptFileName := fmt.Sprintf("%s/%s_%s_client_p2p.sh", cfg.OutputDir(), clientHost, clientHca)

	// Generate server command (listener)
	serverCmd := NewIBWriteBWCommandBuilder(true).
		Host(serverHost).
		Device(serverHca).
		QueuePairNum(cfg.QpNum).
		MessageSize(cfg.MessageSizeBytes).
		Port(port).
		RunInfinitely(cfg.Run.Infinitely).
		DurationSeconds(cfg.Run.DurationSeconds).
		Bidirectional(true). // P2P mode uses bidirectional testing
		RdmaCm(cfg.RdmaCm).
		GidIndex(cfg.GidIndex).
		Report(cfg.Report.Enable).
		OutputFileName(fmt.Sprintf("%s/report_%s_%s_%d.json", cfg.Report.Dir, serverHost, serverHca, port)).
		SSHPrivateKey(cfg.SSH.PrivateKey).
		ServerCommand()

	// Generate client command (initiator)
	clientCmd := NewIBWriteBWCommandBuilder(true).
		Host(clientHost).
		Device(clientHca).
		QueuePairNum(cfg.QpNum).
		MessageSize(cfg.MessageSizeBytes).
		Port(port).
		TargetIP(serverIP).
		RunInfinitely(cfg.Run.Infinitely).
		DurationSeconds(cfg.Run.DurationSeconds).
		Bidirectional(true). // P2P mode uses bidirectional testing
		RdmaCm(cfg.RdmaCm).
		GidIndex(cfg.GidIndex).
		Report(cfg.Report.Enable).
		OutputFileName(fmt.Sprintf("%s/report_%s_%s_%d.json", cfg.Report.Dir, clientHost, clientHca, port)).
		SSHPrivateKey(cfg.SSH.PrivateKey).
		ClientCommand()

	// Write server script
	err := os.WriteFile(serverScriptFileName, []byte(serverCmd.String()+"\n"), 0755)
	if err != nil {
		return fmt.Errorf("failed to write server script %s: %w", serverScriptFileName, err)
	}

	// Write client script
	err = os.WriteFile(clientScriptFileName, []byte(clientCmd.String()+"\n"), 0755)
	if err != nil {
		return fmt.Errorf("failed to write client script %s: %w", clientScriptFileName, err)
	}

	return nil
}

// getHostIP retrieves the IP address of a host using specified network interface
func getHostIP(hostname string, sshKeyPath string, networkInterface string) (string, error) {
	command := fmt.Sprintf("ip addr show %s | grep 'inet ' | awk '{print $2}' | cut -d'/' -f1", networkInterface)
	cmd := tools.BuildSSHCommand(hostname, command, sshKeyPath)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "127.0.0.1", fmt.Errorf("SSH command failed on %s: %v, output: %s", hostname, err, string(output))
	}

	ip := strings.TrimSpace(string(output))
	if ip == "" {
		return "127.0.0.1", fmt.Errorf("no IP address found for %s on %s interface", hostname, networkInterface)
	}

	return ip, nil
}
