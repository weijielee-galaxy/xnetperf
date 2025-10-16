package stream

import (
	"strings"
	"testing"
)

func TestSSHPrivateKeyInCommandBuilder(t *testing.T) {
	tests := []struct {
		name           string
		sshPrivateKey  string
		expectedSubstr string
	}{
		{
			name:           "With custom SSH key",
			sshPrivateKey:  "/custom/path/id_rsa",
			expectedSubstr: "ssh -i /custom/path/id_rsa",
		},
		{
			name:           "Without SSH key (default)",
			sshPrivateKey:  "",
			expectedSubstr: "ssh test-host",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := NewIBWriteBWCommandBuilder().
				Host("test-host").
				Device("mlx5_0").
				Port(20000).
				SSHPrivateKey(tt.sshPrivateKey)

			cmd := builder.String()

			if !strings.Contains(cmd, tt.expectedSubstr) {
				t.Errorf("Expected command to contain '%s', got: %s", tt.expectedSubstr, cmd)
			}
		})
	}
}

func TestSSHPrivateKeyWithFullCommand(t *testing.T) {
	builder := NewIBWriteBWCommandBuilder().
		Host("test-server").
		Device("mlx5_1").
		QueuePairNum(10).
		MessageSize(4096).
		Port(20001).
		RunInfinitely(true).
		SSHPrivateKey("~/.ssh/custom_key").
		ServerCommand()

	cmd := builder.String()

	expected := "ssh -i ~/.ssh/custom_key test-server"
	if !strings.HasPrefix(cmd, expected) {
		t.Errorf("Expected command to start with '%s', got: %s", expected, cmd)
	}

	// 验证命令中包含所有必要的参数
	requiredParts := []string{
		"-i ~/.ssh/custom_key",
		"test-server",
		"ib_write_bw",
		"-d mlx5_1",
		"--run_infinitely",
		"-q 10",
		"-m 4096",
		"-p 20001",
	}

	for _, part := range requiredParts {
		if !strings.Contains(cmd, part) {
			t.Errorf("Expected command to contain '%s', got: %s", part, cmd)
		}
	}
}
