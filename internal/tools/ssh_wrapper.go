package tools

import (
	"fmt"
	"strings"
)

// QuoteType represents the type of quotes to use for SSH command
type QuoteType string

const (
	DoubleQuote QuoteType = "\""
	SingleQuote QuoteType = "'"
)

// SSHWrapper wraps a command for remote execution via SSH
type SSHWrapper struct {
	host           string
	user           string // SSH username (optional, e.g., "root")
	privateKeyPath string
	command        string
	background     bool
	redirectOutput string
	sleepAfter     string
	quoteType      QuoteType
	options        []string // SSH options (e.g., "-o StrictHostKeyChecking=no")
}

// NewSSHWrapper creates a new SSH command wrapper
func NewSSHWrapper(host string, quoteType ...QuoteType) *SSHWrapper {
	qt := DoubleQuote // 默认使用双引号
	if len(quoteType) > 0 {
		qt = quoteType[0]
	}
	return &SSHWrapper{
		host:           host,
		background:     false,
		redirectOutput: "",
		sleepAfter:     "",
		quoteType:      qt,
		options: []string{
			"-o StrictHostKeyChecking=no", // 不询问是否添加 known_hosts
			"-o LogLevel=ERROR",           // 只输出错误日志，抑制 Warning
		},
	}
}

// User sets the SSH username for remote host (e.g., "root")
func (w *SSHWrapper) User(username string) *SSHWrapper {
	w.user = username
	return w
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

// AddOption adds a custom SSH option (e.g., "-o ConnectTimeout=10")
func (w *SSHWrapper) AddOption(option string) *SSHWrapper {
	w.options = append(w.options, option)
	return w
}

// ClearOptions removes all SSH options
func (w *SSHWrapper) ClearOptions() *SSHWrapper {
	w.options = nil
	return w
}

// Build generates the complete SSH command string
func (w *SSHWrapper) Build() string {
	var cmd strings.Builder

	// SSH command with options and optional private key
	cmd.WriteString("ssh")

	// Add SSH options
	for _, opt := range w.options {
		cmd.WriteString(fmt.Sprintf(" %s", opt))
	}

	if w.privateKeyPath != "" {
		cmd.WriteString(fmt.Sprintf(" -i %s", w.privateKeyPath))
	}

	// Construct host (user@host or just host)
	host := w.host
	if w.user != "" {
		host = fmt.Sprintf("%s@%s", w.user, w.host)
	}
	cmd.WriteString(fmt.Sprintf(" %s", host))

	// Remote command with specified quote type
	cmd.WriteString(" ")
	cmd.WriteString(string(w.quoteType))
	cmd.WriteString(w.command)

	// Output redirection
	if w.redirectOutput != "" {
		cmd.WriteString(fmt.Sprintf(" %s", w.redirectOutput))
	}

	// Background mode
	if w.background {
		cmd.WriteString(" &")
	}

	cmd.WriteString(string(w.quoteType))

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
