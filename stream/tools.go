package stream

import "os/exec"

// buildSSHCommand builds an ssh command with optional private key
func buildSSHCommand(hostname, remoteCmd, sshKeyPath string) *exec.Cmd {
	if sshKeyPath != "" {
		return exec.Command("ssh", "-i", sshKeyPath, hostname, remoteCmd)
	}
	return exec.Command("ssh", hostname, remoteCmd)
}
