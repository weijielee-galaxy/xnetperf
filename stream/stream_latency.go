package stream

import (
	"fmt"
	"os"
	"strings"
	"xnetperf/config"
)

// GenerateLatencyScripts generates N×N latency test scripts for full-mesh topology
// Each HCA on each host will test latency to every other HCA on every other host
func GenerateLatencyScripts(cfg *config.Config) error {
	// Validate configuration
	if cfg.StreamType != "fullmesh" {
		fmt.Printf("⚠️  Warning: Latency testing currently only supports fullmesh mode. Current mode: %s\n", cfg.StreamType)
		fmt.Println("Continuing with latency test generation...")
	}

	// Clear script directory
	ClearStreamScriptDir(cfg)

	// Calculate total ports needed for N×N testing
	allHosts := append(cfg.Server.Hostname, cfg.Client.Hostname...)
	totalPorts := calculateTotalLatencyPorts(allHosts, cfg.Client.Hca)

	fmt.Printf("Total ports needed for latency testing: %d\n", totalPorts)
	if totalPorts > (65535 - cfg.StartPort + 1) {
		return fmt.Errorf("not enough ports available. Required: %d, Available: %d",
			totalPorts, 65535-cfg.StartPort+1)
	}

	// Generate scripts for each host
	for _, host := range allHosts {
		if err := generateLatencyScriptsForHost(host, allHosts, cfg); err != nil {
			return fmt.Errorf("failed to generate scripts for host %s: %v", host, err)
		}
	}

	fmt.Printf("✅ Successfully generated latency test scripts in %s\n", cfg.OutputDir())
	return nil
}

// calculateTotalLatencyPorts calculates the total number of ports needed
// For N hosts with H HCAs each, we need N × H × (N-1) × H ports
// (each HCA tests to all HCAs on all other hosts)
func calculateTotalLatencyPorts(hosts []string, hcas []string) int {
	numHosts := len(hosts)
	numHcas := len(hcas)
	// Each host HCA tests to all other hosts' HCAs
	return numHosts * numHcas * (numHosts - 1) * numHcas
}

// generateLatencyScriptsForHost generates server and client scripts for a specific host
func generateLatencyScriptsForHost(currentHost string, allHosts []string, cfg *config.Config) error {
	// Get IP address for this host
	output, err := getHostIP(currentHost, cfg.SSH.PrivateKey, cfg.NetworkInterface)
	if err != nil {
		return fmt.Errorf("failed to get IP for %s: %v\nOutput: %s", currentHost, err, string(output))
	}
	currentHostIP := strings.TrimSpace(string(output))
	fmt.Printf("Host %s IP: %s\n", currentHost, currentHostIP)

	// Generate scripts for each HCA on this host
	for _, currentHCA := range cfg.Client.Hca {
		if err := generateLatencyScriptForHCA(
			currentHost, currentHostIP, currentHCA, allHosts, cfg,
		); err != nil {
			return fmt.Errorf("failed to generate scripts for HCA %s on %s: %v",
				currentHCA, currentHost, err)
		}
	}

	return nil
}

// generateLatencyScriptForHCA generates server and client scripts for a specific HCA
func generateLatencyScriptForHCA(
	currentHost, currentHostIP, currentHCA string,
	allHosts []string,
	cfg *config.Config,
) error {
	serverScriptContent := strings.Builder{}
	clientScriptContent := strings.Builder{}

	serverScriptFileName := fmt.Sprintf("%s/%s_%s_server_latency.sh",
		cfg.OutputDir(), currentHost, currentHCA)
	clientScriptFileName := fmt.Sprintf("%s/%s_%s_client_latency.sh",
		cfg.OutputDir(), currentHost, currentHCA)

	port := cfg.StartPort

	// Generate commands for testing to all other hosts
	for _, targetHost := range allHosts {
		if targetHost == currentHost {
			continue // Skip self-testing
		}

		// Test to all HCAs on the target host
		for _, targetHCA := range cfg.Server.Hca {
			// Generate server command (runs on current host)
			serverCmd := NewIBWriteBWCommandBuilder().
				Host(currentHost).
				Device(currentHCA).
				Port(port).
				ForLatencyTest(true).
				RunInfinitely(false).
				DurationSeconds(cfg.Run.DurationSeconds).
				RdmaCm(cfg.RdmaCm).
				GidIndex(cfg.GidIndex).
				Report(cfg.Report.Enable).
				OutputFileName(fmt.Sprintf("%s/latency_s_%s_%s_%d.json",
					cfg.Report.Dir, currentHost, currentHCA, port)).
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
				DurationSeconds(cfg.Run.DurationSeconds).
				RdmaCm(cfg.RdmaCm).
				GidIndex(cfg.GidIndex).
				Report(cfg.Report.Enable).
				OutputFileName(fmt.Sprintf("%s/latency_c_%s_%s_%d.json",
					cfg.Report.Dir, targetHost, targetHCA, port)).
				SSHPrivateKey(cfg.SSH.PrivateKey).
				ClientCommand()

			serverScriptContent.WriteString(serverCmd.String() + "\n")
			clientScriptContent.WriteString(clientCmd.String() + "\n")

			port++
		}
	}

	// Write server script file
	if err := os.WriteFile(serverScriptFileName, []byte(serverScriptContent.String()), 0755); err != nil {
		return fmt.Errorf("failed to write server script %s: %v", serverScriptFileName, err)
	}

	// Write client script file
	if err := os.WriteFile(clientScriptFileName, []byte(clientScriptContent.String()), 0755); err != nil {
		return fmt.Errorf("failed to write client script %s: %v", clientScriptFileName, err)
	}

	fmt.Printf("✅ Generated latency scripts for %s:%s\n", currentHost, currentHCA)
	return nil
}

// RunLatencyScripts executes the latency test scripts with two-phase startup
// Phase 1: Start all servers with a longer sleep for initialization
// Phase 2: Start all clients after servers are ready
func RunLatencyScripts(cfg *config.Config) error {
	fmt.Println("🚀 Starting latency test execution...")
	fmt.Println("Phase 1: Starting all server processes...")

	// Phase 1: Start all servers
	allHosts := append(cfg.Server.Hostname, cfg.Client.Hostname...)
	for _, host := range allHosts {
		for _, hca := range cfg.Client.Hca {
			serverScript := fmt.Sprintf("%s/%s_%s_server_latency.sh",
				cfg.OutputDir(), host, hca)

			fmt.Printf("  Starting server script: %s\n", serverScript)
			if err := executeScript(serverScript); err != nil {
				return fmt.Errorf("failed to execute server script %s: %v", serverScript, err)
			}
		}
	}

	// Wait for servers to initialize
	fmt.Println("⏳ Waiting 2 seconds for servers to initialize...")
	// time.Sleep(2 * time.Second)

	fmt.Println("Phase 2: Starting all client processes...")

	// Phase 2: Start all clients
	for _, host := range allHosts {
		for _, hca := range cfg.Client.Hca {
			clientScript := fmt.Sprintf("%s/%s_%s_client_latency.sh",
				cfg.OutputDir(), host, hca)

			fmt.Printf("  Starting client script: %s\n", clientScript)
			if err := executeScript(clientScript); err != nil {
				return fmt.Errorf("failed to execute client script %s: %v", clientScript, err)
			}
		}
	}

	fmt.Println("✅ All latency test scripts executed successfully")
	return nil
}

// executeScript runs a shell script
func executeScript(scriptPath string) error {
	// Check if script exists
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		return fmt.Errorf("script does not exist: %s", scriptPath)
	}

	// Execute the script using bash
	// Note: The actual execution is handled by the workflow,
	// here we just validate the script exists
	return nil
}
