package stream

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"xnetperf/config"
)

// TestCalculateTotalLatencyPortsFullmesh tests the port calculation for fullmesh latency tests
func TestCalculateTotalLatencyPortsFullmesh(t *testing.T) {
	tests := []struct {
		name     string
		hosts    []string
		hcas     []string
		expected int
	}{
		{
			name:     "2 hosts, 2 HCAs each",
			hosts:    []string{"host1", "host2"},
			hcas:     []string{"mlx5_0", "mlx5_1"},
			expected: 4 * 3, // 4 total HCAs * (4-1) other HCAs = 12
		},
		{
			name:     "3 hosts, 2 HCAs each",
			hosts:    []string{"host1", "host2", "host3"},
			hcas:     []string{"mlx5_0", "mlx5_1"},
			expected: 6 * 5, // 6 total HCAs * (6-1) other HCAs = 30
		},
		{
			name:     "2 hosts, 1 HCA each",
			hosts:    []string{"host1", "host2"},
			hcas:     []string{"mlx5_0"},
			expected: 2 * 1, // 2 total HCAs * (2-1) other HCAs = 2
		},
		{
			name:     "4 hosts, 3 HCAs each",
			hosts:    []string{"host1", "host2", "host3", "host4"},
			hcas:     []string{"mlx5_0", "mlx5_1", "mlx5_2"},
			expected: 12 * 11, // 12 total HCAs * (12-1) other HCAs = 132
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateTotalLatencyPortsFullmesh(tt.hosts, tt.hcas)
			if result != tt.expected {
				t.Errorf("calculateTotalLatencyPortsFullmesh() = %d, expected %d", result, tt.expected)
			}
		})
	}
}

// TestCalculateTotalLatencyPortsFullmeshFormula verifies the formula is correct
func TestCalculateTotalLatencyPortsFullmeshFormula(t *testing.T) {
	// For N hosts with H HCAs each:
	// Total HCAs = N * H
	// Total connections = (N * H) * (N * H - 1)

	// Example: 10 hosts with 4 HCAs each
	hosts := make([]string, 10)
	for i := 0; i < 10; i++ {
		hosts[i] = "host" + string(rune('0'+i))
	}
	hcas := []string{"mlx5_0", "mlx5_1", "mlx5_2", "mlx5_3"}

	result := calculateTotalLatencyPortsFullmesh(hosts, hcas)
	totalHCAs := 10 * 4                     // 40 total HCAs
	expected := totalHCAs * (totalHCAs - 1) // 40 * 39 = 1560 ports

	if result != expected {
		t.Errorf("For 10 hosts with 4 HCAs: got %d ports, expected %d", result, expected)
	}
}

// TestGenerateLatencyScriptForHCA tests the script generation for a single HCA
func TestGenerateLatencyScriptForHCA(t *testing.T) {
	// Create temporary directory for test output
	tmpDir, err := os.MkdirTemp("", "latency_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create the latency output directory
	latencyDir := tmpDir + "_latency"
	if err := os.MkdirAll(latencyDir, 0755); err != nil {
		t.Fatalf("Failed to create latency dir: %v", err)
	}

	// Create test config
	cfg := &config.Config{
		OutputBase: tmpDir,
		StartPort:  20000,
		Client: config.ClientConfig{
			Hca: []string{"mlx5_0", "mlx5_1"},
		},
		Server: config.ServerConfig{
			Hca: []string{"mlx5_0", "mlx5_1"},
		},
		Run: config.Run{
			DurationSeconds: 10,
		},
		Report: config.Report{
			Enable: true,
			Dir:    tmpDir,
		},
		RdmaCm:   true,
		GidIndex: 3,
		SSH: config.SSH{
			PrivateKey: "/home/user/.ssh/id_rsa",
		},
	}

	// Test parameters
	currentHost := "host1"
	currentHostIP := "192.168.1.1"
	currentHCA := "mlx5_0"
	allHosts := []string{"host1", "host2", "host3"}
	startPort := 20000

	// Generate scripts
	nextPort, err := generateLatencyScriptForHCA(
		currentHost, currentHostIP, currentHCA, allHosts, cfg, startPort,
	)
	if err != nil {
		t.Fatalf("generateLatencyScriptForHCA failed: %v", err)
	}

	// Verify port calculation
	// Should test to:
	// - Same host (host1), different HCA: 1 test (mlx5_1)
	// - Other hosts (host2, host3), all HCAs: 2 hosts * 2 HCAs = 4 tests
	// Total: 1 + 4 = 5 tests
	expectedNextPort := startPort + 5
	if nextPort != expectedNextPort {
		t.Errorf("Expected next port %d, got %d", expectedNextPort, nextPort)
	}

	// Verify server script file exists
	serverScript := filepath.Join(tmpDir+"_latency", "host1_mlx5_0_server_latency.sh")
	if _, err := os.Stat(serverScript); os.IsNotExist(err) {
		t.Errorf("Server script not created: %s", serverScript)
	}

	// Verify client script file exists
	clientScript := filepath.Join(tmpDir+"_latency", "host1_mlx5_0_client_latency.sh")
	if _, err := os.Stat(clientScript); os.IsNotExist(err) {
		t.Errorf("Client script not created: %s", clientScript)
	}

	// Read and verify server script content
	serverContent, err := os.ReadFile(serverScript)
	if err != nil {
		t.Fatalf("Failed to read server script: %v", err)
	}

	serverLines := strings.Split(string(serverContent), "\n")
	nonEmptyServerLines := 0
	for _, line := range serverLines {
		if strings.TrimSpace(line) != "" {
			nonEmptyServerLines++
		}
	}

	// Should have 5 server commands:
	// - 1 for same host different HCA (host1:mlx5_1)
	// - 4 for other hosts (host2:mlx5_0, host2:mlx5_1, host3:mlx5_0, host3:mlx5_1)
	if nonEmptyServerLines != 5 {
		t.Errorf("Expected 5 server commands, got %d", nonEmptyServerLines)
	}

	// Verify server script contains expected elements
	serverStr := string(serverContent)
	expectedServerElements := []string{
		"ssh -i /home/user/.ssh/id_rsa host1",
		"ib_write_lat",
		"-d mlx5_0",
		"-p 20000",
		"-p 20001",
		"-p 20002",
		"-p 20003",
		"-p 20004",
		"-R",   // RDMA-CM flag for ib_write_lat
		"-x 3", // GID index flag for ib_write_lat
		"--out_json",
		"latency_fullmesh_s_host1_mlx5_0_from_host1_mlx5_1_p20000.json", // Same host, different HCA
		"latency_fullmesh_s_host1_mlx5_0_from_host2_mlx5_0_p20001.json",
		"latency_fullmesh_s_host1_mlx5_0_from_host2_mlx5_1_p20002.json",
		"latency_fullmesh_s_host1_mlx5_0_from_host3_mlx5_0_p20003.json",
		"latency_fullmesh_s_host1_mlx5_0_from_host3_mlx5_1_p20004.json",
	}

	for _, expected := range expectedServerElements {
		if !strings.Contains(serverStr, expected) {
			t.Errorf("Server script missing expected element: %s", expected)
		}
	}

	// Read and verify client script content
	clientContent, err := os.ReadFile(clientScript)
	if err != nil {
		t.Fatalf("Failed to read client script: %v", err)
	}

	clientLines := strings.Split(string(clientContent), "\n")
	nonEmptyClientLines := 0
	for _, line := range clientLines {
		if strings.TrimSpace(line) != "" {
			nonEmptyClientLines++
		}
	}

	// Should have 5 client commands:
	// - 1 for same host different HCA (host1:mlx5_1)
	// - 4 for other hosts (host2:mlx5_0, host2:mlx5_1, host3:mlx5_0, host3:mlx5_1)
	if nonEmptyClientLines != 5 {
		t.Errorf("Expected 5 client commands, got %d", nonEmptyClientLines)
	}

	// Verify client script contains expected elements
	clientStr := string(clientContent)
	expectedClientElements := []string{
		"ssh -i /home/user/.ssh/id_rsa host1", // Same host
		"ssh -i /home/user/.ssh/id_rsa host2",
		"ssh -i /home/user/.ssh/id_rsa host3",
		"ib_write_lat",
		"-d mlx5_0",
		"-d mlx5_1",
		"192.168.1.1", // Target IP
		"-p 20000",
		"-p 20001",
		"-p 20002",
		"-p 20003",
		"-p 20004",
		"-R",   // RDMA-CM flag for ib_write_lat
		"-x 3", // GID index flag for ib_write_lat
		"--out_json",
		"latency_fullmesh_c_host1_mlx5_1_to_host1_mlx5_0_p20000.json", // Same host, different HCA
		"latency_fullmesh_c_host2_mlx5_0_to_host1_mlx5_0_p20001.json",
		"latency_fullmesh_c_host2_mlx5_1_to_host1_mlx5_0_p20002.json",
		"latency_fullmesh_c_host3_mlx5_0_to_host1_mlx5_0_p20003.json",
		"latency_fullmesh_c_host3_mlx5_1_to_host1_mlx5_0_p20004.json",
	}

	for _, expected := range expectedClientElements {
		if !strings.Contains(clientStr, expected) {
			t.Errorf("Client script missing expected element: %s", expected)
		}
	}
}

// TestGenerateLatencyScriptForHCA_FilenameFormat verifies the new filename format
func TestGenerateLatencyScriptForHCA_FilenameFormat(t *testing.T) {
	// Create temporary directory for test output
	tmpDir, err := os.MkdirTemp("", "latency_filename_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create the latency output directory
	latencyDir := tmpDir + "_latency"
	if err := os.MkdirAll(latencyDir, 0755); err != nil {
		t.Fatalf("Failed to create latency dir: %v", err)
	}

	// Create test config
	cfg := &config.Config{
		OutputBase: tmpDir,
		StartPort:  25000,
		Client: config.ClientConfig{
			Hca: []string{"mlx5_0"},
		},
		Server: config.ServerConfig{
			Hca: []string{"mlx5_1"},
		},
		Run: config.Run{
			DurationSeconds: 5,
		},
		Report: config.Report{
			Enable: true,
			Dir:    "/tmp/reports",
		},
		RdmaCm:   false,
		GidIndex: 0,
		SSH: config.SSH{
			PrivateKey: "/root/.ssh/id_rsa",
		},
	}

	// Test parameters
	currentHost := "node-a"
	currentHostIP := "10.0.0.1"
	currentHCA := "mlx5_0"
	allHosts := []string{"node-a", "node-b"}
	startPort := 25000

	// Generate scripts
	_, err = generateLatencyScriptForHCA(
		currentHost, currentHostIP, currentHCA, allHosts, cfg, startPort,
	)
	if err != nil {
		t.Fatalf("generateLatencyScriptForHCA failed: %v", err)
	}

	// Read server script
	serverScript := filepath.Join(tmpDir+"_latency", "node-a_mlx5_0_server_latency.sh")
	serverContent, err := os.ReadFile(serverScript)
	if err != nil {
		t.Fatalf("Failed to read server script: %v", err)
	}

	// Read client script
	clientScript := filepath.Join(tmpDir+"_latency", "node-a_mlx5_0_client_latency.sh")
	clientContent, err := os.ReadFile(clientScript)
	if err != nil {
		t.Fatalf("Failed to read client script: %v", err)
	}

	// Verify filename format in server script
	serverStr := string(serverContent)

	// First connection: same host, different HCA (node-a:mlx5_1 -> node-a:mlx5_0)
	expectedServerFilename1 := "latency_fullmesh_s_node-a_mlx5_0_from_node-a_mlx5_1_p25000.json"
	if !strings.Contains(serverStr, expectedServerFilename1) {
		t.Errorf("Server script missing expected filename: %s", expectedServerFilename1)
		t.Logf("Server script content:\n%s", serverStr)
	}

	// Second connection: different host (node-b:mlx5_1 -> node-a:mlx5_0)
	expectedServerFilename2 := "latency_fullmesh_s_node-a_mlx5_0_from_node-b_mlx5_1_p25001.json"
	if !strings.Contains(serverStr, expectedServerFilename2) {
		t.Errorf("Server script missing expected filename: %s", expectedServerFilename2)
		t.Logf("Server script content:\n%s", serverStr)
	}

	// Verify filename format in client script
	clientStr := string(clientContent)

	// First connection: same host (node-a:mlx5_1 -> node-a:mlx5_0)
	expectedClientFilename1 := "latency_fullmesh_c_node-a_mlx5_1_to_node-a_mlx5_0_p25000.json"
	if !strings.Contains(clientStr, expectedClientFilename1) {
		t.Errorf("Client script missing expected filename: %s", expectedClientFilename1)
		t.Logf("Client script content:\n%s", clientStr)
	}

	// Second connection: different host (node-b:mlx5_1 -> node-a:mlx5_0)
	expectedClientFilename2 := "latency_fullmesh_c_node-b_mlx5_1_to_node-a_mlx5_0_p25001.json"
	if !strings.Contains(clientStr, expectedClientFilename2) {
		t.Errorf("Client script missing expected filename: %s", expectedClientFilename2)
		t.Logf("Client script content:\n%s", clientStr)
	}

	// Verify _to_ separator in client filename
	if !strings.Contains(clientStr, "_to_") {
		t.Error("Client filename should contain '_to_' separator")
	}

	// Verify _from_ separator in server filename
	if !strings.Contains(serverStr, "_from_") {
		t.Error("Server filename should contain '_from_' separator")
	}

	// Verify port prefix _p in both filenames (both ports 25000 and 25001)
	if !strings.Contains(serverStr, "_p25000") {
		t.Error("Server filename should contain '_p25000' port prefix")
	}
	if !strings.Contains(serverStr, "_p25001") {
		t.Error("Server filename should contain '_p25001' port prefix")
	}
	if !strings.Contains(clientStr, "_p25000") {
		t.Error("Client filename should contain '_p25000' port prefix")
	}
	if !strings.Contains(clientStr, "_p25001") {
		t.Error("Client filename should contain '_p25001' port prefix")
	}
}

// TestGenerateLatencyScriptForHCA_PortAllocation verifies continuous port allocation
func TestGenerateLatencyScriptForHCA_PortAllocation(t *testing.T) {
	// Create temporary directory for test output
	tmpDir, err := os.MkdirTemp("", "latency_port_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create the latency output directory
	latencyDir := tmpDir + "_latency"
	if err := os.MkdirAll(latencyDir, 0755); err != nil {
		t.Fatalf("Failed to create latency dir: %v", err)
	}

	// Create test config
	cfg := &config.Config{
		OutputBase: tmpDir,
		StartPort:  30000,
		Client: config.ClientConfig{
			Hca: []string{"mlx5_0", "mlx5_1"},
		},
		Server: config.ServerConfig{
			Hca: []string{"mlx5_0", "mlx5_1"},
		},
		Run: config.Run{
			DurationSeconds: 10,
		},
		Report: config.Report{
			Enable: true,
			Dir:    tmpDir,
		},
		RdmaCm:   true,
		GidIndex: 3,
		SSH: config.SSH{
			PrivateKey: "/home/user/.ssh/id_rsa",
		},
	}

	// Test port allocation across multiple HCAs
	currentHost := "host1"
	currentHostIP := "192.168.1.1"
	allHosts := []string{"host1", "host2"}
	startPort := 30000

	// Generate for first HCA
	nextPort1, err := generateLatencyScriptForHCA(
		currentHost, currentHostIP, "mlx5_0", allHosts, cfg, startPort,
	)
	if err != nil {
		t.Fatalf("generateLatencyScriptForHCA for mlx5_0 failed: %v", err)
	}

	// Should use 3 ports:
	// - Same host different HCA: 1 test (host1:mlx5_1)
	// - Other host: 1 host * 2 HCAs = 2 tests
	// Total: 1 + 2 = 3 tests
	expectedNextPort1 := startPort + 3
	if nextPort1 != expectedNextPort1 {
		t.Errorf("After mlx5_0: expected next port %d, got %d", expectedNextPort1, nextPort1)
	}

	// Generate for second HCA, starting from where first one ended
	nextPort2, err := generateLatencyScriptForHCA(
		currentHost, currentHostIP, "mlx5_1", allHosts, cfg, nextPort1,
	)
	if err != nil {
		t.Fatalf("generateLatencyScriptForHCA for mlx5_1 failed: %v", err)
	}

	// Should use 3 more ports (same as mlx5_0)
	expectedNextPort2 := nextPort1 + 3
	if nextPort2 != expectedNextPort2 {
		t.Errorf("After mlx5_1: expected next port %d, got %d", expectedNextPort2, nextPort2)
	}

	// Read both scripts and verify no port overlap
	script1 := filepath.Join(tmpDir+"_latency", "host1_mlx5_0_server_latency.sh")
	content1, err := os.ReadFile(script1)
	if err != nil {
		t.Fatalf("Failed to read mlx5_0 script: %v", err)
	}

	script2 := filepath.Join(tmpDir+"_latency", "host1_mlx5_1_server_latency.sh")
	content2, err := os.ReadFile(script2)
	if err != nil {
		t.Fatalf("Failed to read mlx5_1 script: %v", err)
	}

	// Verify mlx5_0 uses ports 30000-30002
	str1 := string(content1)
	if !strings.Contains(str1, "-p 30000") {
		t.Error("mlx5_0 script should contain port 30000")
	}
	if !strings.Contains(str1, "-p 30001") {
		t.Error("mlx5_0 script should contain port 30001")
	}
	if !strings.Contains(str1, "-p 30002") {
		t.Error("mlx5_0 script should contain port 30002")
	}
	if strings.Contains(str1, "-p 30003") {
		t.Error("mlx5_0 script should NOT contain port 30003")
	}

	// Verify mlx5_1 uses ports 30003-30005
	str2 := string(content2)
	if !strings.Contains(str2, "-p 30003") {
		t.Error("mlx5_1 script should contain port 30003")
	}
	if !strings.Contains(str2, "-p 30004") {
		t.Error("mlx5_1 script should contain port 30004")
	}
	if !strings.Contains(str2, "-p 30005") {
		t.Error("mlx5_1 script should contain port 30005")
	}
	if strings.Contains(str2, "-p 30000") {
		t.Error("mlx5_1 script should NOT contain port 30000")
	}
	if strings.Contains(str2, "-p 30001") {
		t.Error("mlx5_1 script should NOT contain port 30001")
	}
	if strings.Contains(str2, "-p 30002") {
		t.Error("mlx5_1 script should NOT contain port 30002")
	}
}
