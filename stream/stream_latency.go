package stream

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
	"xnetperf/config"
)

// getLatencyOutputDir returns the output directory for latency tests
func getLatencyOutputDir(cfg *config.Config) string {
	return fmt.Sprintf("%s_latency", cfg.OutputBase)
}

// clearLatencyScriptDir clears the latency script directory
func clearLatencyScriptDir(cfg *config.Config) {
	dir := getLatencyOutputDir(cfg)
	fmt.Printf("Clearing latency script directory: %s\n", dir)

	// Remove directory if it exists
	if _, err := os.Stat(dir); err == nil {
		if err := os.RemoveAll(dir); err != nil {
			fmt.Printf("Warning: Failed to remove directory %s: %v\n", dir, err)
		}
	}

	// Create fresh directory
	if err := os.MkdirAll(dir, 0755); err != nil {
		fmt.Printf("Warning: Failed to create directory %s: %v\n", dir, err)
	}

	fmt.Printf("Cleared latency script directory: %s\n", dir)
}

// GenerateLatencyScripts generates latency test scripts based on stream_type
// Supports fullmesh (N×N) and incast (client→server only) modes
func GenerateLatencyScripts(cfg *config.Config) error {
	// Clear latency script directory
	clearLatencyScriptDir(cfg)

	// Generate scripts based on stream type
	if cfg.StreamType == config.InCast {
		return generateLatencyScriptsIncast(cfg)
	}

	// Default to fullmesh mode
	return generateLatencyScriptsFullmesh(cfg)
}

// generateLatencyScriptsFullmesh generates N×N latency test scripts for full-mesh topology
// Each HCA on each host will test latency to every other HCA on every other host
func generateLatencyScriptsFullmesh(cfg *config.Config) error {
	fmt.Printf("📊 Generating latency scripts in FULLMESH mode\n")

	// Calculate total ports needed for N×N testing
	allHosts := append(cfg.Server.Hostname, cfg.Client.Hostname...)
	totalPorts := calculateTotalLatencyPortsFullmesh(allHosts, cfg.Client.Hca)

	fmt.Printf("Total latency ports needed: %d (from %d to %d)\n",
		totalPorts, cfg.StartPort, cfg.StartPort+totalPorts-1)

	// Generate scripts for each host with global port counter
	port := cfg.StartPort
	for _, currentHost := range allHosts {
		var err error
		port, err = generateLatencyScriptsForHostFullmesh(currentHost, allHosts, cfg, port)
		if err != nil {
			return fmt.Errorf("failed to generate scripts for %s: %v", currentHost, err)
		}
	}

	outputDir := getLatencyOutputDir(cfg)
	fmt.Printf("✅ Successfully generated fullmesh latency test scripts in %s\n", outputDir)
	return nil
}

// generateLatencyScriptsIncast generates client→server latency test scripts for incast topology
// Only clients test latency to servers, not vice versa
func generateLatencyScriptsIncast(cfg *config.Config) error {
	fmt.Printf("📊 Generating latency scripts in INCAST mode (client → server only)\n")

	// Calculate total ports needed for incast testing
	totalPorts := calculateTotalLatencyPortsIncast(
		cfg.Server.Hostname, cfg.Server.Hca,
		cfg.Client.Hostname, cfg.Client.Hca,
	)

	fmt.Printf("Total latency ports needed: %d (from %d to %d)\n",
		totalPorts, cfg.StartPort, cfg.StartPort+totalPorts-1)

	// Generate scripts for each client (each client tests to all servers)
	port := cfg.StartPort
	for _, clientHost := range cfg.Client.Hostname {
		var err error
		port, err = generateLatencyScriptsForClientIncast(clientHost, cfg, port)
		if err != nil {
			return fmt.Errorf("failed to generate incast scripts for client %s: %v", clientHost, err)
		}
	}

	outputDir := getLatencyOutputDir(cfg)
	fmt.Printf("✅ Successfully generated incast latency test scripts in %s\n", outputDir)
	return nil
}

// calculateTotalLatencyPortsFullmesh calculates the total number of ports needed for fullmesh mode
// For N hosts with H HCAs each, we need N × H × (N × H - 1) ports
// (each HCA tests to all other HCAs except itself)
func calculateTotalLatencyPortsFullmesh(hosts []string, hcas []string) int {
	numHosts := len(hosts)
	numHcas := len(hcas)
	totalHCAs := numHosts * numHcas
	// Each HCA tests to all other HCAs (including same host, different HCA)
	// but not to itself
	return totalHCAs * (totalHCAs - 1)
}

// calculateTotalLatencyPortsIncast calculates the total number of ports needed for incast mode
// Only clients test to servers: num_clients × num_client_hcas × num_servers × num_server_hcas
func calculateTotalLatencyPortsIncast(serverHosts []string, serverHcas []string, clientHosts []string, clientHcas []string) int {
	numServers := len(serverHosts)
	numServerHcas := len(serverHcas)
	numClients := len(clientHosts)
	numClientHcas := len(clientHcas)

	// Each client HCA tests to all server HCAs
	return numServers * numServerHcas * numClients * numClientHcas
}

// generateLatencyScriptsForHostFullmesh generates server and client scripts for a specific host in fullmesh mode
// Returns the next available port number
func generateLatencyScriptsForHostFullmesh(currentHost string, allHosts []string, cfg *config.Config, startPort int) (int, error) {
	// Get IP address for this host
	output, err := getHostIP(currentHost, cfg.SSH.PrivateKey, cfg.NetworkInterface)
	if err != nil {
		return startPort, fmt.Errorf("failed to get IP for %s: %v\nOutput: %s", currentHost, err, string(output))
	}
	currentHostIP := strings.TrimSpace(string(output))
	fmt.Printf("Host %s IP: %s\n", currentHost, currentHostIP)

	// Generate scripts for each HCA on this host
	port := startPort
	for _, currentHCA := range cfg.Client.Hca {
		var err error
		port, err = generateLatencyScriptForHCA(
			currentHost, currentHostIP, currentHCA, allHosts, cfg, port,
		)
		if err != nil {
			return port, fmt.Errorf("failed to generate scripts for HCA %s on %s: %v",
				currentHCA, currentHost, err)
		}
	}

	return port, nil
}

// generateLatencyScriptForHCA generates server and client scripts for a specific HCA
// Returns the next available port number
func generateLatencyScriptForHCA(
	currentHost, currentHostIP, currentHCA string,
	allHosts []string,
	cfg *config.Config,
	startPort int,
) (int, error) {
	serverScriptContent := strings.Builder{}
	clientScriptContent := strings.Builder{}

	outputDir := getLatencyOutputDir(cfg)
	serverScriptFileName := fmt.Sprintf("%s/%s_%s_server_latency.sh",
		outputDir, currentHost, currentHCA)
	clientScriptFileName := fmt.Sprintf("%s/%s_%s_client_latency.sh",
		outputDir, currentHost, currentHCA)

	port := startPort

	// Generate commands for testing to all other hosts
	for _, targetHost := range allHosts {
		// Test to all HCAs on the target host
		for _, targetHCA := range cfg.Server.Hca {
			// Skip only if same host AND same HCA
			if targetHost == currentHost && targetHCA == currentHCA {
				continue // Skip testing same HCA to itself
			}
			// Generate server command (runs on current host)
			serverCmd := NewIBWriteBWCommandBuilder().
				Host(currentHost).
				Device(currentHCA).
				Port(port).
				ForLatencyTest(true).
				RunInfinitely(false).
				DurationSeconds(5).
				RdmaCm(cfg.RdmaCm).
				GidIndex(cfg.GidIndex).
				Report(cfg.Report.Enable).
				OutputFileName(fmt.Sprintf("%s/latency_fullmesh_s_%s_%s_from_%s_%s_p%d.json",
					cfg.Report.Dir, currentHost, currentHCA, targetHost, targetHCA, port)).
				SSHPrivateKey(cfg.SSH.PrivateKey).
				ServerCommand()

			// Generate client command (connects from target to current host)
			clientCmd := NewIBWriteBWCommandBuilder().
				Host(targetHost).
				Device(targetHCA).
				Port(port).
				ForLatencyTest(true).
				TargetIP(currentHostIP).
				RunInfinitely(false).
				DurationSeconds(5).
				RdmaCm(cfg.RdmaCm).
				GidIndex(cfg.GidIndex).
				Report(cfg.Report.Enable).
				OutputFileName(fmt.Sprintf("%s/latency_fullmesh_c_%s_%s_to_%s_%s_p%d.json",
					cfg.Report.Dir, targetHost, targetHCA, currentHost, currentHCA, port)).
				SSHPrivateKey(cfg.SSH.PrivateKey).
				ClientCommand()

			serverScriptContent.WriteString(serverCmd.String() + "\n")
			clientScriptContent.WriteString(clientCmd.String() + "\n")

			port++
		}
	}

	// Write server script file
	if err := os.WriteFile(serverScriptFileName, []byte(serverScriptContent.String()), 0755); err != nil {
		return port, fmt.Errorf("failed to write server script %s: %v", serverScriptFileName, err)
	}

	// Write client script file
	if err := os.WriteFile(clientScriptFileName, []byte(clientScriptContent.String()), 0755); err != nil {
		return port, fmt.Errorf("failed to write client script %s: %v", clientScriptFileName, err)
	}

	fmt.Printf("✅ Generated latency scripts for %s:%s (ports %d-%d)\n",
		currentHost, currentHCA, startPort, port-1)

	// Print first few lines of scripts for debugging
	serverLines := strings.Split(serverScriptContent.String(), "\n")
	if len(serverLines) > 0 {
		fmt.Printf("   Server script preview (first command): %s\n", serverLines[0])
	}
	clientLines := strings.Split(clientScriptContent.String(), "\n")
	if len(clientLines) > 0 {
		fmt.Printf("   Client script preview (first command): %s\n", clientLines[0])
	}

	return port, nil
}

// RunLatencyScripts executes the latency test scripts with two-phase startup
// Phase 1: Start all servers with a longer sleep for initialization
// Phase 2: Start all clients after servers are ready
func RunLatencyScripts(cfg *config.Config) error {
	outputDir := getLatencyOutputDir(cfg)
	fmt.Println("🚀 Starting latency test execution...")
	fmt.Println("Phase 1: Starting all server processes...")

	// Phase 1: Start all servers
	allHosts := append(cfg.Server.Hostname, cfg.Client.Hostname...)
	for _, host := range allHosts {
		for _, hca := range cfg.Client.Hca {
			serverScript := fmt.Sprintf("%s/%s_%s_server_latency.sh",
				outputDir, host, hca)

			fmt.Printf("  Executing: bash %s\n", serverScript)
			if err := executeScript(serverScript); err != nil {
				return fmt.Errorf("failed to execute server script %s: %v", serverScript, err)
			}
		}
	}

	// Wait for servers to initialize
	fmt.Printf("\nWaiting %d seconds for servers to initialize...\n", cfg.WaitingTimeSeconds)
	time.Sleep(time.Second * time.Duration(cfg.WaitingTimeSeconds))

	fmt.Println("Phase 2: Starting all client processes...")

	// Phase 2: Start all clients
	for _, host := range allHosts {
		for _, hca := range cfg.Client.Hca {
			clientScript := fmt.Sprintf("%s/%s_%s_client_latency.sh",
				outputDir, host, hca)

			fmt.Printf("  Executing: bash %s\n", clientScript)
			if err := executeScript(clientScript); err != nil {
				return fmt.Errorf("failed to execute client script %s: %v", clientScript, err)
			}
		}
	}

	fmt.Println("✅ All latency test scripts executed successfully")
	return nil
}

// executeScript runs a shell script using bash
func executeScript(scriptPath string) error {
	// Check if script exists
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		return fmt.Errorf("script does not exist: %s", scriptPath)
	}

	// Print script path for debugging
	fmt.Printf("    → Running: bash %s\n", scriptPath)

	// Read script content
	content, err := os.ReadFile(scriptPath)
	if err != nil {
		return fmt.Errorf("failed to read script %s: %v", scriptPath, err)
	}

	// Execute via bash
	cmd := exec.Command("bash", "-c", string(content))
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start script %s: %v", scriptPath, err)
	}

	return nil
}

// generateLatencyScriptsForClientIncast generates scripts for a client in incast mode
// Client tests latency to all servers
// Returns the next available port number
func generateLatencyScriptsForClientIncast(clientHost string, cfg *config.Config, startPort int) (int, error) {
	// Get IP addresses for all servers
	serverIPs := make(map[string]string)
	for _, serverHost := range cfg.Server.Hostname {
		output, err := getHostIP(serverHost, cfg.SSH.PrivateKey, cfg.NetworkInterface)
		if err != nil {
			return startPort, fmt.Errorf("failed to get IP for server %s: %v\nOutput: %s", serverHost, err, string(output))
		}
		serverIPs[serverHost] = strings.TrimSpace(string(output))
		fmt.Printf("Server %s IP: %s\n", serverHost, serverIPs[serverHost])
	}

	// Generate scripts for each HCA on this client
	port := startPort
	for _, clientHCA := range cfg.Client.Hca {
		var err error
		port, err = generateLatencyScriptForClientHCAIncast(
			clientHost, clientHCA, serverIPs, cfg, port,
		)
		if err != nil {
			return port, fmt.Errorf("failed to generate incast scripts for HCA %s on client %s: %v",
				clientHCA, clientHost, err)
		}
	}

	return port, nil
}

// generateLatencyScriptForClientHCAIncast generates server and client scripts for a specific client HCA in incast mode
// Returns the next available port number
func generateLatencyScriptForClientHCAIncast(
	clientHost, clientHCA string,
	serverIPs map[string]string,
	cfg *config.Config,
	startPort int,
) (int, error) {
	serverScriptContent := strings.Builder{}
	clientScriptContent := strings.Builder{}

	outputDir := getLatencyOutputDir(cfg)
	serverScriptFileName := fmt.Sprintf("%s/%s_%s_server_latency.sh",
		outputDir, clientHost, clientHCA)
	clientScriptFileName := fmt.Sprintf("%s/%s_%s_client_latency.sh",
		outputDir, clientHost, clientHCA)

	port := startPort

	// In incast mode: this client HCA tests to all server HCAs
	for _, serverHost := range cfg.Server.Hostname {
		serverIP := serverIPs[serverHost]
		for _, serverHCA := range cfg.Server.Hca {
			// Generate server command (runs on server host)
			serverCmd := NewIBWriteBWCommandBuilder().
				Host(serverHost).
				Device(serverHCA).
				Port(port).
				ForLatencyTest(true).
				RunInfinitely(false).
				DurationSeconds(5).
				RdmaCm(cfg.RdmaCm).
				GidIndex(cfg.GidIndex).
				Report(cfg.Report.Enable).
				OutputFileName(fmt.Sprintf("%s/latency_incast_s_%s_%s_from_%s_%s_p%d.json",
					cfg.Report.Dir, serverHost, serverHCA, clientHost, clientHCA, port)).
				SSHPrivateKey(cfg.SSH.PrivateKey).
				ServerCommand()

			// Generate client command (connects from client to server)
			clientCmd := NewIBWriteBWCommandBuilder().
				Host(clientHost).
				Device(clientHCA).
				Port(port).
				ForLatencyTest(true).
				TargetIP(serverIP).
				RunInfinitely(false).
				DurationSeconds(5).
				RdmaCm(cfg.RdmaCm).
				GidIndex(cfg.GidIndex).
				Report(cfg.Report.Enable).
				OutputFileName(fmt.Sprintf("%s/latency_incast_c_%s_%s_to_%s_%s_p%d.json",
					cfg.Report.Dir, clientHost, clientHCA, serverHost, serverHCA, port)).
				SSHPrivateKey(cfg.SSH.PrivateKey).
				ClientCommand()

			serverScriptContent.WriteString(serverCmd.String() + "\n")
			clientScriptContent.WriteString(clientCmd.String() + "\n")

			port++
		}
	}

	// Write server script file
	if err := os.WriteFile(serverScriptFileName, []byte(serverScriptContent.String()), 0755); err != nil {
		return port, fmt.Errorf("failed to write server script %s: %v", serverScriptFileName, err)
	}

	// Write client script file
	if err := os.WriteFile(clientScriptFileName, []byte(clientScriptContent.String()), 0755); err != nil {
		return port, fmt.Errorf("failed to write client script %s: %v", clientScriptFileName, err)
	}

	fmt.Printf("✅ Generated incast latency scripts for %s:%s (ports %d-%d)\n",
		clientHost, clientHCA, startPort, port-1)

	// Print first few lines of scripts for debugging
	serverLines := strings.Split(serverScriptContent.String(), "\n")
	if len(serverLines) > 0 {
		fmt.Printf("   Server script preview (first command): %s\n", serverLines[0])
	}
	clientLines := strings.Split(clientScriptContent.String(), "\n")
	if len(clientLines) > 0 {
		fmt.Printf("   Client script preview (first command): %s\n", clientLines[0])
	}

	return port, nil
}
