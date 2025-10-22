package tools

import (
	"fmt"
	"strings"
)

// SSHWrapper wraps a command for remote execution via SSH
type SSHWrapper struct {
	host           string
	privateKeyPath string
	command        string
	background     bool
	redirectOutput string
	sleepAfter     string
}

// NewSSHWrapper creates a new SSH command wrapper
func NewSSHWrapper(host string) *SSHWrapper {
	return &SSHWrapper{
		host:           host,
		background:     false,
		redirectOutput: "",
		sleepAfter:     "",
	}
}

// PrivateKey sets the SSH private key path for authentication
func (w *SSHWrapper) PrivateKey(path string) *SSHWrapper {
	w.privateKeyPath = path
	return w
}

// Command sets the command to execute remotely
func (w *SSHWrapper) Command(cmd string) *SSHWrapper {
	w.command = cmd
	return w
}

// Background runs the command in background mode
func (w *SSHWrapper) Background(enable bool) *SSHWrapper {
	w.background = enable
	return w
}

// RedirectOutput sets output redirection (e.g., ">/dev/null 2>&1")
func (w *SSHWrapper) RedirectOutput(redirect string) *SSHWrapper {
	w.redirectOutput = redirect
	return w
}

// SleepAfter adds a sleep command after execution (e.g., "0.02" for 20ms)
func (w *SSHWrapper) SleepAfter(duration string) *SSHWrapper {
	w.sleepAfter = duration
	return w
}

// Build generates the complete SSH command string
func (w *SSHWrapper) Build() string {
	var cmd strings.Builder

	// SSH command with optional private key
	cmd.WriteString("ssh")
	if w.privateKeyPath != "" {
		cmd.WriteString(fmt.Sprintf(" -i %s", w.privateKeyPath))
	}
	cmd.WriteString(fmt.Sprintf(" %s", w.host))

	// Remote command in single quotes
	cmd.WriteString(" '")
	cmd.WriteString(w.command)

	// Output redirection
	if w.redirectOutput != "" {
		cmd.WriteString(fmt.Sprintf(" %s", w.redirectOutput))
	}

	// Background mode
	if w.background {
		cmd.WriteString(" &")
	}

	cmd.WriteString("'")

	// Sleep after command
	if w.sleepAfter != "" {
		cmd.WriteString(fmt.Sprintf("; sleep %s", w.sleepAfter))
	}

	return cmd.String()
}

// String returns the built SSH command string
func (w *SSHWrapper) String() string {
	return w.Build()
}

// WrapIBCommand is a convenience method to wrap an IBCommand
func (w *SSHWrapper) WrapIBCommand(ibCmd *IBCommand) *SSHWrapper {
	w.command = ibCmd.Build()
	return w
}
