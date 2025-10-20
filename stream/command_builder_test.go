package stream

import (
	"fmt"
	"strings"
	"testing"
)

func TestBuilder(t *testing.T) {
	cmdBuilder := NewIBWriteBWCommandBuilder().
		Host("test-host").
		Device("mlx5_0").
		RunInfinitely(false).
		QueuePairNum(0).
		DurationSeconds(30).
		MessageSize(4096).
		Port(20000).
		Report(true).
		OutputFileName("report_c_061.json").
		TargetIP("192.168.1.100").
		ClientCommand()

	if true {
		cmdBuilder = cmdBuilder.DurationSeconds(10)
	}

	fmt.Println(cmdBuilder.String())
}

func TestNewIBWriteBWCommandBuilder(t *testing.T) {
	builder := NewIBWriteBWCommandBuilder()

	if builder == nil {
		t.Fatal("NewIBWriteBWCommandBuilder should not return nil")
	}

	// Test default values
	if !builder.runInfinitely {
		t.Error("Default runInfinitely should be true")
	}

	if builder.redirectOutput != ">/dev/null 2>&1" {
		t.Errorf("Default redirectOutput should be '>/dev/null 2>&1', got '%s'", builder.redirectOutput)
	}

	if !builder.background {
		t.Error("Default background should be true")
	}

	if builder.sleepTime != "0.02" {
		t.Errorf("Default sleepTime should be '0.02', got '%s'", builder.sleepTime)
	}

	if !builder.sshWrapper {
		t.Error("Default sshWrapper should be true")
	}
}

func TestIBWriteBWCommandBuilder_ChainableMethods(t *testing.T) {
	builder := NewIBWriteBWCommandBuilder()

	// Test that all methods return *IBWriteBWCommandBuilder for chaining
	result := builder.Host("testhost").
		Device("mlx5_0").
		RunInfinitely(false).
		DurationSeconds(30).
		QueuePairNum(10).
		MessageSize(4096).
		Port(20000).
		TargetIP("192.168.1.100").
		Report(true).
		OutputFileName("test_report.json")

	if result != builder {
		t.Error("Methods should return the same builder instance for chaining")
	}
}

func TestIBWriteBWCommandBuilder_Host(t *testing.T) {
	builder := NewIBWriteBWCommandBuilder()
	builder.Host("test-host")

	if builder.host != "test-host" {
		t.Errorf("Expected host to be 'test-host', got '%s'", builder.host)
	}
}

func TestIBWriteBWCommandBuilder_Device(t *testing.T) {
	builder := NewIBWriteBWCommandBuilder()
	builder.Device("mlx5_1")

	if builder.device != "mlx5_1" {
		t.Errorf("Expected device to be 'mlx5_1', got '%s'", builder.device)
	}
}

func TestIBWriteBWCommandBuilder_RunInfinitely(t *testing.T) {
	builder := NewIBWriteBWCommandBuilder()
	builder.RunInfinitely(false)

	if builder.runInfinitely {
		t.Error("Expected runInfinitely to be false")
	}
}

func TestIBWriteBWCommandBuilder_DurationSeconds(t *testing.T) {
	builder := NewIBWriteBWCommandBuilder()
	builder.DurationSeconds(30)

	if builder.durationSeconds != 30 {
		t.Errorf("Expected durationSeconds to be 30, got %d", builder.durationSeconds)
	}
}

func TestIBWriteBWCommandBuilder_QueuePairNum(t *testing.T) {
	builder := NewIBWriteBWCommandBuilder()
	builder.QueuePairNum(15)

	if builder.queuePairNum != 15 {
		t.Errorf("Expected queuePairNum to be 15, got %d", builder.queuePairNum)
	}
}

func TestIBWriteBWCommandBuilder_MessageSize(t *testing.T) {
	builder := NewIBWriteBWCommandBuilder()
	builder.MessageSize(8192)

	if builder.messageSize != 8192 {
		t.Errorf("Expected messageSize to be 8192, got %d", builder.messageSize)
	}
}

func TestIBWriteBWCommandBuilder_Port(t *testing.T) {
	builder := NewIBWriteBWCommandBuilder()
	builder.Port(25000)

	if builder.port != 25000 {
		t.Errorf("Expected port to be 25000, got %d", builder.port)
	}
}

func TestIBWriteBWCommandBuilder_TargetIP(t *testing.T) {
	builder := NewIBWriteBWCommandBuilder()
	builder.TargetIP("10.0.0.1")

	if builder.targetIP != "10.0.0.1" {
		t.Errorf("Expected targetIP to be '10.0.0.1', got '%s'", builder.targetIP)
	}
}

func TestIBWriteBWCommandBuilder_RedirectOutput(t *testing.T) {
	builder := NewIBWriteBWCommandBuilder()
	builder.RedirectOutput(">test.log 2>&1")

	if builder.redirectOutput != ">test.log 2>&1" {
		t.Errorf("Expected redirectOutput to be '>test.log 2>&1', got '%s'", builder.redirectOutput)
	}
}

func TestIBWriteBWCommandBuilder_Background(t *testing.T) {
	builder := NewIBWriteBWCommandBuilder()
	builder.Background(false)

	if builder.background {
		t.Error("Expected background to be false")
	}
}

func TestIBWriteBWCommandBuilder_SleepTime(t *testing.T) {
	builder := NewIBWriteBWCommandBuilder()
	builder.SleepTime("0.1")

	if builder.sleepTime != "0.1" {
		t.Errorf("Expected sleepTime to be '0.1', got '%s'", builder.sleepTime)
	}
}

func TestIBWriteBWCommandBuilder_SSHWrapper(t *testing.T) {
	builder := NewIBWriteBWCommandBuilder()
	builder.SSHWrapper(false)

	if builder.sshWrapper {
		t.Error("Expected sshWrapper to be false")
	}
}

func TestIBWriteBWCommandBuilder_Report(t *testing.T) {
	builder := NewIBWriteBWCommandBuilder()
	builder.Report(true)

	if !builder.report {
		t.Error("Expected report to be true")
	}
}

func TestIBWriteBWCommandBuilder_OutputFileName(t *testing.T) {
	builder := NewIBWriteBWCommandBuilder()
	builder.OutputFileName("test_report.json")

	if builder.outputFileName != "test_report.json" {
		t.Errorf("Expected outputFileName to be 'test_report.json', got '%s'", builder.outputFileName)
	}
}

func TestIBWriteBWCommandBuilder_ServerCommand(t *testing.T) {
	builder := NewIBWriteBWCommandBuilder()
	builder.TargetIP("192.168.1.100") // Set an IP first
	result := builder.ServerCommand()

	if result != builder {
		t.Error("ServerCommand should return the same builder instance")
	}

	if builder.targetIP != "" {
		t.Error("ServerCommand should clear the targetIP")
	}
}

func TestIBWriteBWCommandBuilder_ClientCommand(t *testing.T) {
	builder := NewIBWriteBWCommandBuilder()
	result := builder.ClientCommand()

	if result != builder {
		t.Error("ClientCommand should return the same builder instance")
	}

	if builder.sleepTime != "0.06" {
		t.Errorf("ClientCommand should set sleepTime to '0.06', got '%s'", builder.sleepTime)
	}
}

func TestIBWriteBWCommandBuilder_String_BasicServer(t *testing.T) {
	builder := NewIBWriteBWCommandBuilder().
		Host("test-server").
		Device("mlx5_0").
		QueuePairNum(10).
		MessageSize(4096).
		Port(20000).
		ServerCommand()

	result := builder.String()
	expected := "ssh test-server 'ib_write_bw -d mlx5_0 --run_infinitely -q 10 -m 4096 -p 20000 >/dev/null 2>&1 &'; sleep 0.02"

	if result != expected {
		t.Errorf("Expected:\n%s\nGot:\n%s", expected, result)
	}
}

func TestIBWriteBWCommandBuilder_String_BasicClient(t *testing.T) {
	builder := NewIBWriteBWCommandBuilder().
		Host("test-client").
		Device("mlx5_1").
		QueuePairNum(5).
		MessageSize(2048).
		Port(21000).
		TargetIP("192.168.1.100").
		ClientCommand()

	result := builder.String()
	expected := "ssh test-client 'ib_write_bw -d mlx5_1 --run_infinitely -q 5 -m 2048 -p 21000 192.168.1.100 >/dev/null 2>&1 &'; sleep 0.06"

	if result != expected {
		t.Errorf("Expected:\n%s\nGot:\n%s", expected, result)
	}
}

func TestIBWriteBWCommandBuilder_String_NoSSHWrapper(t *testing.T) {
	builder := NewIBWriteBWCommandBuilder().
		Host("test-host").
		Device("mlx5_0").
		Port(20000).
		SSHWrapper(false).
		SleepTime("")

	result := builder.String()
	expected := "ib_write_bw -d mlx5_0 --run_infinitely -p 20000 >/dev/null 2>&1 &"

	if result != expected {
		t.Errorf("Expected:\n%s\nGot:\n%s", expected, result)
	}
}

func TestIBWriteBWCommandBuilder_String_MinimalCommand(t *testing.T) {
	builder := NewIBWriteBWCommandBuilder().
		RunInfinitely(false).
		Background(false).
		RedirectOutput("").
		SleepTime("").
		SSHWrapper(false)

	result := builder.String()
	expected := "ib_write_bw"

	if result != expected {
		t.Errorf("Expected:\n%s\nGot:\n%s", expected, result)
	}
}

func TestIBWriteBWCommandBuilder_String_AllParameters(t *testing.T) {
	builder := NewIBWriteBWCommandBuilder().
		Host("full-test").
		Device("mlx5_7").
		RunInfinitely(true).
		QueuePairNum(20).
		MessageSize(8192).
		Port(30000).
		TargetIP("10.0.0.50").
		RedirectOutput(">custom.log").
		Background(true).
		SleepTime("0.5").
		SSHWrapper(true)

	result := builder.String()
	expected := "ssh full-test 'ib_write_bw -d mlx5_7 --run_infinitely -q 20 -m 8192 -p 30000 10.0.0.50 >custom.log &'; sleep 0.5"

	if result != expected {
		t.Errorf("Expected:\n%s\nGot:\n%s", expected, result)
	}
}

func TestIBWriteBWCommandBuilder_String_ZeroValues(t *testing.T) {
	builder := NewIBWriteBWCommandBuilder().
		QueuePairNum(0).
		MessageSize(0).
		Port(0)

	result := builder.String()

	// Zero values should not appear in the command
	if strings.Contains(result, " -q 0") {
		t.Error("Zero queuePairNum should not appear in command")
	}
	if strings.Contains(result, " -m 0") {
		t.Error("Zero messageSize should not appear in command")
	}
	if strings.Contains(result, " -p 0") {
		t.Error("Zero port should not appear in command")
	}
}

func TestIBWriteBWCommandBuilder_String_EmptyStrings(t *testing.T) {
	builder := NewIBWriteBWCommandBuilder().
		Device("").
		TargetIP("").
		RedirectOutput("").
		SleepTime("")

	result := builder.String()

	// Empty strings should not appear in the command
	if strings.Contains(result, " -d ") {
		t.Error("Empty device should not appear in command")
	}
	if strings.Contains(result, "; sleep") {
		t.Error("Empty sleepTime should not add sleep command")
	}
}

func TestIBWriteBWCommandBuilder_String_WithDuration(t *testing.T) {
	builder := NewIBWriteBWCommandBuilder().
		Host("test-host").
		Device("mlx5_0").
		RunInfinitely(false).
		DurationSeconds(30).
		Port(20000)

	result := builder.String()
	expected := "ssh test-host 'ib_write_bw -d mlx5_0 -D 30 -p 20000 >/dev/null 2>&1 &'; sleep 0.02"

	if result != expected {
		t.Errorf("Expected:\n%s\nGot:\n%s", expected, result)
	}
}

func TestIBWriteBWCommandBuilder_String_DurationIgnoredWhenInfinite(t *testing.T) {
	builder := NewIBWriteBWCommandBuilder().
		Host("test-host").
		Device("mlx5_0").
		RunInfinitely(true).
		DurationSeconds(30).
		Port(20000)

	result := builder.String()
	expected := "ssh test-host 'ib_write_bw -d mlx5_0 --run_infinitely -p 20000 >/dev/null 2>&1 &'; sleep 0.02"

	if result != expected {
		t.Errorf("Expected:\n%s\nGot:\n%s", expected, result)
	}

	// Should contain --run_infinitely but not -D
	if !strings.Contains(result, "--run_infinitely") {
		t.Error("Should contain --run_infinitely when runInfinitely is true")
	}
	if strings.Contains(result, " -D ") {
		t.Error("Should not contain -D when runInfinitely is true")
	}
}

func TestIBWriteBWCommandBuilder_String_NoDurationWhenZero(t *testing.T) {
	builder := NewIBWriteBWCommandBuilder().
		Host("test-host").
		Device("mlx5_0").
		RunInfinitely(false).
		DurationSeconds(0).
		Port(20000)

	result := builder.String()

	// Should not contain -D when duration is 0
	if strings.Contains(result, " -D ") {
		t.Error("Should not contain -D when duration is 0")
	}
	// Should not contain --run_infinitely when runInfinitely is false
	if strings.Contains(result, "--run_infinitely") {
		t.Error("Should not contain --run_infinitely when runInfinitely is false")
	}
}

func TestIBWriteBWCommandBuilder_String_WithReport(t *testing.T) {
	builder := NewIBWriteBWCommandBuilder().
		Host("test-host").
		Device("mlx5_0").
		Port(20000).
		TargetIP("192.168.1.100").
		RunInfinitely(false).
		Report(true).
		OutputFileName("report_c_061.json")

	result := builder.String()
	expected := "ssh test-host 'ib_write_bw -d mlx5_0 -p 20000 192.168.1.100 --report_gbits --out_json --out_json_file report_c_061.json >/dev/null 2>&1 &'; sleep 0.02"

	if result != expected {
		t.Errorf("Expected:\n%s\nGot:\n%s", expected, result)
	}
}

func TestIBWriteBWCommandBuilder_String_WithReportNoTargetIP(t *testing.T) {
	builder := NewIBWriteBWCommandBuilder().
		Host("test-host").
		Device("mlx5_0").
		Port(20000).
		RunInfinitely(false).
		Report(true).
		OutputFileName("server_report.json").
		ServerCommand()

	result := builder.String()
	expected := "ssh test-host 'ib_write_bw -d mlx5_0 -p 20000 --report_gbits --out_json --out_json_file server_report.json >/dev/null 2>&1 &'; sleep 0.02"

	if result != expected {
		t.Errorf("Expected:\n%s\nGot:\n%s", expected, result)
	}
}

func TestIBWriteBWCommandBuilder_String_NoReportParameters(t *testing.T) {
	builder := NewIBWriteBWCommandBuilder().
		Host("test-host").
		Device("mlx5_0").
		Port(20000).
		Report(false)

	result := builder.String()

	// Should not contain report parameters when report is false
	if strings.Contains(result, "--report_gbits") {
		t.Error("Should not contain --report_gbits when report is false")
	}
	if strings.Contains(result, "--out_json") {
		t.Error("Should not contain --out_json when report is false")
	}
	if strings.Contains(result, "--out_json_file") {
		t.Error("Should not contain --out_json_file when report is false")
	}
}

func TestIBWriteBWCommandBuilder_String_ReportWithoutFileName_ShouldPanic(t *testing.T) {
	builder := NewIBWriteBWCommandBuilder().
		Host("test-host").
		Device("mlx5_0").
		Port(20000).
		RunInfinitely(false).
		Report(true)
		// OutputFileName not set

	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic when report is true, runInfinitely is false, but outputFileName is not set")
		}
	}()

	_ = builder.String()
}

func TestIBWriteBWCommandBuilder_String_ReportWithEmptyFileName_ShouldPanic(t *testing.T) {
	builder := NewIBWriteBWCommandBuilder().
		Host("test-host").
		Device("mlx5_0").
		Port(20000).
		RunInfinitely(false).
		Report(true).
		OutputFileName("")

	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic when report is true but outputFileName is empty")
		}
	}()

	_ = builder.String()
}

func TestIBWriteBWCommandBuilder_String_ReportIgnoredWhenInfinite(t *testing.T) {
	builder := NewIBWriteBWCommandBuilder().
		Host("test-host").
		Device("mlx5_0").
		Port(20000).
		RunInfinitely(true).
		Report(true).
		OutputFileName("report.json")

	result := builder.String()
	expected := "ssh test-host 'ib_write_bw -d mlx5_0 --run_infinitely -p 20000 >/dev/null 2>&1 &'; sleep 0.02"

	if result != expected {
		t.Errorf("Expected:\n%s\nGot:\n%s", expected, result)
	}

	// Should not contain report parameters when runInfinitely is true
	if strings.Contains(result, "--report_gbits") {
		t.Error("Should not contain --report_gbits when runInfinitely is true")
	}
	if strings.Contains(result, "--out_json") {
		t.Error("Should not contain --out_json when runInfinitely is true")
	}
	if strings.Contains(result, "--out_json_file") {
		t.Error("Should not contain --out_json_file when runInfinitely is true")
	}
}

func TestIBWriteBWCommandBuilder_String_ReportWithInfiniteNoFilenameRequired(t *testing.T) {
	// When runInfinitely is true, report should be ignored and no filename should be required
	builder := NewIBWriteBWCommandBuilder().
		Host("test-host").
		Device("mlx5_0").
		Port(20000).
		RunInfinitely(true).
		Report(true)
		// No OutputFileName set, but should not panic since runInfinitely=true

	// This should not panic
	result := builder.String()

	// Should contain --run_infinitely but no report parameters
	if !strings.Contains(result, "--run_infinitely") {
		t.Error("Should contain --run_infinitely")
	}
	if strings.Contains(result, "--report_gbits") {
		t.Error("Should not contain --report_gbits when runInfinitely is true")
	}
}

func TestRdmaCm(t *testing.T) {
	builder := NewIBWriteBWCommandBuilder()

	// Test RdmaCm method returns builder for chaining
	result := builder.RdmaCm(true)
	if result != builder {
		t.Error("RdmaCm should return the same builder instance for chaining")
	}

	// Test RdmaCm enabled
	builder.Host("test-host").Device("mlx5_0").Port(20000).RdmaCm(true)
	cmd := builder.String()
	if !strings.Contains(cmd, " -R") {
		t.Error("Command should contain -R flag when RdmaCm is true")
	}

	// Test RdmaCm disabled
	builder2 := NewIBWriteBWCommandBuilder()
	builder2.Host("test-host").Device("mlx5_0").Port(20000).RdmaCm(false)
	cmd2 := builder2.String()
	if strings.Contains(cmd2, " -R") {
		t.Error("Command should not contain -R flag when RdmaCm is false")
	}
}

func TestRdmaCmWithOtherFlags(t *testing.T) {
	builder := NewIBWriteBWCommandBuilder()
	builder.Host("test-host").
		Device("mlx5_0").
		Port(20000).
		Bidirectional(true).
		RdmaCm(true).
		QueuePairNum(10)

	cmd := builder.String()

	// Should contain both -b and -R flags
	if !strings.Contains(cmd, " -b") {
		t.Error("Command should contain -b flag")
	}
	if !strings.Contains(cmd, " -R") {
		t.Error("Command should contain -R flag")
	}

	// Check flag order (should be -b before -R)
	bIndex := strings.Index(cmd, " -b")
	rIndex := strings.Index(cmd, " -R")
	if bIndex == -1 || rIndex == -1 {
		t.Error("Both -b and -R flags should be present")
	}
	if bIndex >= rIndex {
		t.Error("Expected -b flag to appear before -R flag in command")
	}
}

// TestForLatencyTest tests the latency test mode
func TestForLatencyTest(t *testing.T) {
	builder := NewIBWriteBWCommandBuilder()

	// Test chaining
	result := builder.ForLatencyTest(true)
	if result != builder {
		t.Error("ForLatencyTest should return the same builder instance for chaining")
	}

	// Test latencyTest field is set
	if !builder.latencyTest {
		t.Error("latencyTest field should be true")
	}

	// Test disabling latency test
	builder.ForLatencyTest(false)
	if builder.latencyTest {
		t.Error("latencyTest field should be false")
	}
}

// TestLatencyTestCommand tests ib_write_lat command generation
func TestLatencyTestCommand(t *testing.T) {
	builder := NewIBWriteBWCommandBuilder()
	builder.Host("test-host").
		Device("mlx5_0").
		Port(20000).
		ForLatencyTest(true).
		RunInfinitely(false).
		DurationSeconds(5)

	cmd := builder.String()

	// Should use ib_write_lat instead of ib_write_bw
	if !strings.Contains(cmd, "ib_write_lat") {
		t.Error("Command should contain 'ib_write_lat'")
	}
	if strings.Contains(cmd, "ib_write_bw") {
		t.Error("Command should not contain 'ib_write_bw'")
	}

	// Should contain duration
	if !strings.Contains(cmd, " -D 5") {
		t.Error("Command should contain -D 5 for duration")
	}

	// Should NOT contain queue pair or message size parameters
	if strings.Contains(cmd, " -q ") {
		t.Error("Latency command should not contain -q flag")
	}
	if strings.Contains(cmd, " -m ") {
		t.Error("Latency command should not contain -m flag")
	}
}

// TestLatencyTestWithReport tests latency test with JSON report output
func TestLatencyTestWithReport(t *testing.T) {
	builder := NewIBWriteBWCommandBuilder()
	builder.Host("test-host").
		Device("mlx5_0").
		Port(20000).
		ForLatencyTest(true).
		RunInfinitely(false).
		DurationSeconds(5).
		Report(true).
		OutputFileName("latency_report.json")

	cmd := builder.String()

	// Should contain --output-format json
	if !strings.Contains(cmd, "--output-format json") {
		t.Error("Latency command should contain '--output-format json'")
	}

	// Should contain output file
	if !strings.Contains(cmd, "--out_json_file latency_report.json") {
		t.Error("Latency command should contain '--out_json_file latency_report.json'")
	}

	// Should NOT contain --report_gbits (bandwidth-specific)
	if strings.Contains(cmd, "--report_gbits") {
		t.Error("Latency command should not contain '--report_gbits'")
	}
}

// TestLatencyTestServerCommand tests latency server command
func TestLatencyTestServerCommand(t *testing.T) {
	builder := NewIBWriteBWCommandBuilder()
	builder.Host("server-host").
		Device("mlx5_0").
		Port(20000).
		ForLatencyTest(true).
		RunInfinitely(false).
		DurationSeconds(5).
		ServerCommand()

	cmd := builder.String()

	// Should use ib_write_lat
	if !strings.Contains(cmd, "ib_write_lat") {
		t.Error("Command should contain 'ib_write_lat'")
	}

	// Should not contain target IP (server mode)
	if strings.Contains(cmd, "192.168") {
		t.Error("Server command should not contain target IP")
	}
}

// TestLatencyTestClientCommand tests latency client command
func TestLatencyTestClientCommand(t *testing.T) {
	builder := NewIBWriteBWCommandBuilder()
	builder.Host("client-host").
		Device("mlx5_0").
		Port(20000).
		ForLatencyTest(true).
		RunInfinitely(false).
		DurationSeconds(5).
		TargetIP("192.168.1.100").
		ClientCommand()

	cmd := builder.String()

	// Should use ib_write_lat
	if !strings.Contains(cmd, "ib_write_lat") {
		t.Error("Command should contain 'ib_write_lat'")
	}

	// Should contain target IP (client mode)
	if !strings.Contains(cmd, "192.168.1.100") {
		t.Error("Client command should contain target IP")
	}

	// Should have appropriate sleep time for client
	if !strings.Contains(cmd, "sleep 0.06") {
		t.Error("Client command should have sleep 0.06")
	}
}

// TestLatencyTestWithSSHKey tests latency command with SSH private key
func TestLatencyTestWithSSHKey(t *testing.T) {
	builder := NewIBWriteBWCommandBuilder()
	builder.Host("test-host").
		Device("mlx5_0").
		Port(20000).
		ForLatencyTest(true).
		RunInfinitely(false).
		DurationSeconds(5).
		SSHPrivateKey("/path/to/key.pem")

	cmd := builder.String()

	// Should contain SSH with private key
	if !strings.Contains(cmd, "ssh -i /path/to/key.pem test-host") {
		t.Error("Command should contain 'ssh -i /path/to/key.pem test-host'")
	}

	// Should contain ib_write_lat
	if !strings.Contains(cmd, "ib_write_lat") {
		t.Error("Command should contain 'ib_write_lat'")
	}
}

// TestLatencyTestWithGidIndex tests latency command with GID index
func TestLatencyTestWithGidIndex(t *testing.T) {
	builder := NewIBWriteBWCommandBuilder()
	builder.Host("test-host").
		Device("mlx5_0").
		Port(20000).
		ForLatencyTest(true).
		RunInfinitely(false).
		DurationSeconds(5).
		GidIndex(3)

	cmd := builder.String()

	// Should contain GID index
	if !strings.Contains(cmd, " -x 3") {
		t.Error("Command should contain ' -x 3' for GID index")
	}

	// Should use ib_write_lat
	if !strings.Contains(cmd, "ib_write_lat") {
		t.Error("Command should contain 'ib_write_lat'")
	}
}

// TestBandwidthTestStillWorks tests that bandwidth tests still work after adding latency support
func TestBandwidthTestStillWorks(t *testing.T) {
	builder := NewIBWriteBWCommandBuilder()
	builder.Host("test-host").
		Device("mlx5_0").
		Port(20000).
		RunInfinitely(false).
		DurationSeconds(20).
		QueuePairNum(10).
		MessageSize(4096).
		Report(true).
		OutputFileName("bandwidth_report.json")

	cmd := builder.String()

	// Should use ib_write_bw
	if !strings.Contains(cmd, "ib_write_bw") {
		t.Error("Bandwidth command should contain 'ib_write_bw'")
	}

	// Should contain queue pair and message size
	if !strings.Contains(cmd, " -q 10") {
		t.Error("Bandwidth command should contain ' -q 10'")
	}
	if !strings.Contains(cmd, " -m 4096") {
		t.Error("Bandwidth command should contain ' -m 4096'")
	}

	// Should contain bandwidth-specific report flags
	if !strings.Contains(cmd, "--report_gbits") {
		t.Error("Bandwidth command should contain '--report_gbits'")
	}
}
