package tools

import (
	"fmt"
	"os/exec"
)

// BuildSSHCommand builds an ssh command with optional user and private key
func BuildSSHCommand(hostname, remoteCmd, sshKeyPath string, user string) *exec.Cmd {
	// Construct host string (user@host or just host)
	host := hostname
	if user != "" {
		host = fmt.Sprintf("%s@%s", user, hostname)
	}

	if sshKeyPath != "" {
		return exec.Command("ssh", "-i", sshKeyPath, "-o", "StrictHostKeyChecking=no", "-o", "LogLevel=ERROR", host, remoteCmd)
	}
	return exec.Command("ssh", "-o", "StrictHostKeyChecking=no", "-o", "LogLevel=ERROR", host, remoteCmd)
}
