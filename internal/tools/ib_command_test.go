package tools_test

import (
	"testing"

	"xnetperf/internal/tools"
)

func TestIBWriteBwCommand(t *testing.T) {
	tests := []struct {
		name     string
		builder  func() *tools.IBCommand
		expected string
	}{
		{
			name: "Basic server command",
			builder: func() *tools.IBCommand {
				return tools.NewIBWriteBwCommand().
					Device("mlx5_0").
					Port(18515).
					RunInfinitely(true).
					AsServer()
			},
			expected: "ib_write_bw -d mlx5_0 --run_infinitely -p 18515",
		},
		{
			name: "Client command with target IP",
			builder: func() *tools.IBCommand {
				return tools.NewIBWriteBwCommand().
					Device("mlx5_0").
					Port(18515).
					RunInfinitely(true).
					AsClient("192.168.1.100")
			},
			expected: "ib_write_bw -d mlx5_0 --run_infinitely -p 18515 192.168.1.100",
		},
		{
			name: "Bandwidth test with queue pairs and message size",
			builder: func() *tools.IBCommand {
				return tools.NewIBWriteBwCommand().
					Device("mlx5_0").
					QueuePairs(8).
					MessageSize(65536).
					Port(18515).
					RunInfinitely(true).
					AsServer()
			},
			expected: "ib_write_bw -d mlx5_0 --run_infinitely -q 8 -m 65536 -p 18515",
		},
		{
			name: "Bandwidth test with duration and report",
			builder: func() *tools.IBCommand {
				return tools.NewIBWriteBwCommand().
					Device("mlx5_0").
					RunInfinitely(false).
					Duration(10).
					Port(18515).
					EnableReport("/tmp/report.json").
					AsClient("192.168.1.100")
			},
			expected: "ib_write_bw -d mlx5_0 -D 10 -p 18515 192.168.1.100 --report_gbits --out_json --out_json_file /tmp/report.json",
		},
		{
			name: "Bidirectional test with RDMA CM and GID index",
			builder: func() *tools.IBCommand {
				return tools.NewIBWriteBwCommand().
					Device("mlx5_0").
					Port(18515).
					Bidirectional(true).
					RdmaCm(true).
					GidIndex(3).
					RunInfinitely(true).
					AsClient("192.168.1.100")
			},
			expected: "ib_write_bw -d mlx5_0 --run_infinitely -p 18515 -b -R -x 3 192.168.1.100",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := tt.builder()
			result := cmd.Build()
			if result != tt.expected {
				t.Errorf("Expected:\n%s\nGot:\n%s", tt.expected, result)
			}
		})
	}
}

func TestIBWriteLatCommand(t *testing.T) {
	tests := []struct {
		name     string
		builder  func() *tools.IBCommand
		expected string
	}{
		{
			name: "Basic latency server command",
			builder: func() *tools.IBCommand {
				return tools.NewIBWriteLatCommand().
					Device("mlx5_0").
					Port(18515).
					AsServer()
			},
			expected: "ib_write_lat -d mlx5_0 -p 18515",
		},
		{
			name: "Latency client with duration",
			builder: func() *tools.IBCommand {
				return tools.NewIBWriteLatCommand().
					Device("mlx5_0").
					Duration(5).
					Port(18515).
					AsClient("192.168.1.100")
			},
			expected: "ib_write_lat -d mlx5_0 -D 5 -p 18515 192.168.1.100",
		},
		{
			name: "Latency test with report",
			builder: func() *tools.IBCommand {
				return tools.NewIBWriteLatCommand().
					Device("mlx5_0").
					Duration(10).
					Port(18515).
					EnableReport("/tmp/latency.json").
					AsClient("192.168.1.100")
			},
			expected: "ib_write_lat -d mlx5_0 -D 10 -p 18515 192.168.1.100 --out_json --out_json_file /tmp/latency.json",
		},
		{
			name: "Latency test should ignore queue pairs and message size",
			builder: func() *tools.IBCommand {
				return tools.NewIBWriteLatCommand().
					Device("mlx5_0").
					QueuePairs(8).      // Should be ignored
					MessageSize(65536). // Should be ignored
					Port(18515).
					AsServer()
			},
			expected: "ib_write_lat -d mlx5_0 -p 18515",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := tt.builder()
			result := cmd.Build()
			if result != tt.expected {
				t.Errorf("Expected:\n%s\nGot:\n%s", tt.expected, result)
			}
		})
	}
}

func TestSSHWrapper(t *testing.T) {
	tests := []struct {
		name     string
		builder  func() *tools.SSHWrapper
		expected string
	}{
		{
			name: "Basic SSH wrapper",
			builder: func() *tools.SSHWrapper {
				return tools.NewSSHWrapper("host1").
					Command("echo hello")
			},
			expected: "ssh host1 'echo hello'",
		},
		{
			name: "SSH with private key",
			builder: func() *tools.SSHWrapper {
				return tools.NewSSHWrapper("host1").
					PrivateKey("/path/to/key").
					Command("echo hello")
			},
			expected: "ssh -i /path/to/key host1 'echo hello'",
		},
		{
			name: "SSH with background and redirect",
			builder: func() *tools.SSHWrapper {
				return tools.NewSSHWrapper("host1").
					Command("ib_write_bw -d mlx5_0").
					Background(true).
					RedirectOutput(">/dev/null 2>&1")
			},
			expected: "ssh host1 'ib_write_bw -d mlx5_0 >/dev/null 2>&1 &'",
		},
		{
			name: "SSH with sleep after",
			builder: func() *tools.SSHWrapper {
				return tools.NewSSHWrapper("host1").
					Command("ib_write_bw -d mlx5_0").
					Background(true).
					SleepAfter("0.02")
			},
			expected: "ssh host1 'ib_write_bw -d mlx5_0 &'; sleep 0.02",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wrapper := tt.builder()
			result := wrapper.Build()
			if result != tt.expected {
				t.Errorf("Expected:\n%s\nGot:\n%s", tt.expected, result)
			}
		})
	}
}

func TestSSHWrapperWithIBCommand(t *testing.T) {
	tests := []struct {
		name     string
		builder  func() string
		expected string
	}{
		{
			name: "Wrap bandwidth command via SSH",
			builder: func() string {
				ibCmd := tools.NewIBWriteBwCommand().
					Device("mlx5_0").
					Port(18515).
					RunInfinitely(true).
					AsServer()

				return tools.NewSSHWrapper("host1").
					PrivateKey("/path/to/key").
					WrapIBCommand(ibCmd).
					Background(true).
					RedirectOutput(">/dev/null 2>&1").
					SleepAfter("0.02").
					Build()
			},
			expected: "ssh -i /path/to/key host1 'ib_write_bw -d mlx5_0 --run_infinitely -p 18515 >/dev/null 2>&1 &'; sleep 0.02",
		},
		{
			name: "Wrap latency command via SSH",
			builder: func() string {
				ibCmd := tools.NewIBWriteLatCommand().
					Device("mlx5_0").
					Duration(10).
					Port(18515).
					EnableReport("/tmp/latency.json").
					AsClient("192.168.1.100")

				return tools.NewSSHWrapper("host1").
					PrivateKey("/path/to/key").
					WrapIBCommand(ibCmd).
					Background(true).
					SleepAfter("0.02").
					Build()
			},
			expected: "ssh -i /path/to/key host1 'ib_write_lat -d mlx5_0 -D 10 -p 18515 192.168.1.100 --out_json --out_json_file /tmp/latency.json &'; sleep 0.02",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.builder()
			if result != tt.expected {
				t.Errorf("Expected:\n%s\nGot:\n%s", tt.expected, result)
			}
		})
	}
}

func TestCommandTypeHelpers(t *testing.T) {
	t.Run("IsLatencyTest", func(t *testing.T) {
		if !tools.IBWriteLat.IsLatencyTest() {
			t.Error("IBWriteLat should be a latency test")
		}
		if tools.IBWriteBw.IsLatencyTest() {
			t.Error("IBWriteBw should not be a latency test")
		}
	})

	t.Run("IsBandwidthTest", func(t *testing.T) {
		if !tools.IBWriteBw.IsBandwidthTest() {
			t.Error("IBWriteBw should be a bandwidth test")
		}
		if tools.IBWriteLat.IsBandwidthTest() {
			t.Error("IBWriteLat should not be a bandwidth test")
		}
	})

	t.Run("String", func(t *testing.T) {
		if tools.IBWriteBw.String() != "ib_write_bw" {
			t.Errorf("Expected 'ib_write_bw', got '%s'", tools.IBWriteBw.String())
		}
		if tools.IBWriteLat.String() != "ib_write_lat" {
			t.Errorf("Expected 'ib_write_lat', got '%s'", tools.IBWriteLat.String())
		}
	})
}
