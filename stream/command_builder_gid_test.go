package stream

import (
	"strings"
	"testing"
)

func TestGidIndexParameter(t *testing.T) {
	tests := []struct {
		name     string
		gidIndex int
		want     string
	}{
		{
			name:     "GID index 3",
			gidIndex: 3,
			want:     " -x 3",
		},
		{
			name:     "GID index 0 (should not add -x)",
			gidIndex: 0,
			want:     "",
		},
		{
			name:     "GID index 5",
			gidIndex: 5,
			want:     " -x 5",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewIBWriteBWCommandBuilder().
				Device("mlx5_0").
				Port(20000).
				GidIndex(tt.gidIndex).
				SSHWrapper(false).
				ServerCommand()

			result := cmd.String()

			if tt.gidIndex > 0 {
				if !strings.Contains(result, tt.want) {
					t.Errorf("Expected command to contain %q, got: %s", tt.want, result)
				}
			} else {
				if strings.Contains(result, " -x ") {
					t.Errorf("Expected command NOT to contain -x flag, got: %s", result)
				}
			}
		})
	}
}

func TestGidIndexWithRdmaCm(t *testing.T) {
	// Test that gid_index and rdma_cm can be used together
	cmd := NewIBWriteBWCommandBuilder().
		Device("mlx5_0").
		Port(20000).
		RdmaCm(true).
		GidIndex(3).
		SSHWrapper(false).
		ServerCommand()

	result := cmd.String()

	if !strings.Contains(result, " -R") {
		t.Errorf("Expected command to contain -R flag, got: %s", result)
	}

	if !strings.Contains(result, " -x 3") {
		t.Errorf("Expected command to contain -x 3 flag, got: %s", result)
	}
}

func TestGidIndexInFullCommand(t *testing.T) {
	// Test GID index in a complete command with all parameters
	cmd := NewIBWriteBWCommandBuilder().
		Host("test-host").
		Device("mlx5_0").
		QueuePairNum(10).
		MessageSize(4096).
		Port(20000).
		DurationSeconds(10).
		RdmaCm(false).
		GidIndex(3).
		SSHPrivateKey("~/.ssh/id_rsa").
		ServerCommand()

	result := cmd.String()

	// Verify all key parameters are present
	expectedParts := []string{
		"ssh -i ~/.ssh/id_rsa test-host",
		"ib_write_bw",
		"-d mlx5_0",
		"--run_infinitely",
		"-q 10",
		"-m 4096",
		"-p 20000",
		"-x 3",
	}

	for _, part := range expectedParts {
		if !strings.Contains(result, part) {
			t.Errorf("Expected command to contain %q, got: %s", part, result)
		}
	}
}
