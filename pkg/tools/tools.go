package tools

import "os/exec"

// BuildSSHCommand builds an ssh command with optional private key
func BuildSSHCommand(hostname, remoteCmd, sshKeyPath string) *exec.Cmd {
	if sshKeyPath != "" {
		return exec.Command("ssh", "-i", sshKeyPath, "-o", "StrictHostKeyChecking=no", "-o", "LogLevel=ERROR", hostname, remoteCmd)
	}
	return exec.Command("ssh", "-o", "StrictHostKeyChecking=no", "-o", "LogLevel=ERROR", hostname, remoteCmd)
}
