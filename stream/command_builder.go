package stream

import (
	"fmt"
	"strings"
)

// IBWriteBWCommandBuilder represents a builder for ib_write_bw and ib_write_lat commands
type IBWriteBWCommandBuilder struct {
	host            string
	device          string
	runInfinitely   bool
	durationSeconds int
	queuePairNum    int
	messageSize     int
	port            int
	targetIP        string
	redirectOutput  string
	background      bool
	sleepTime       string
	sshWrapper      bool
	sshPrivateKey   string // SSH private key path
	report          bool
	outputFileName  string
	bidirectional   bool // 新增双向测试参数
	rdmaCm          bool // RDMA CM参数
	gidIndex        int  // GID index for RoCE v2
	latencyTest     bool // 延迟测试模式 (use ib_write_lat instead of ib_write_bw)
}

// NewIBWriteBWCommandBuilder creates a new command builder
func NewIBWriteBWCommandBuilder(sshWrapper bool) *IBWriteBWCommandBuilder {
	return &IBWriteBWCommandBuilder{
		runInfinitely:  true,
		redirectOutput: ">/dev/null 2>&1",
		background:     true,
		sleepTime:      "0.02",
		sshWrapper:     sshWrapper,
	}
}

// Host sets the target host
func (b *IBWriteBWCommandBuilder) Host(host string) *IBWriteBWCommandBuilder {
	b.host = host
	return b
}

// Device sets the InfiniBand device
func (b *IBWriteBWCommandBuilder) Device(device string) *IBWriteBWCommandBuilder {
	b.device = device
	return b
}

// RunInfinitely sets whether to run infinitely
func (b *IBWriteBWCommandBuilder) RunInfinitely(enable bool) *IBWriteBWCommandBuilder {
	b.runInfinitely = enable
	return b
}

// DurationSeconds sets the duration in seconds (only used when runInfinitely is false)
func (b *IBWriteBWCommandBuilder) DurationSeconds(duration int) *IBWriteBWCommandBuilder {
	b.durationSeconds = duration
	return b
}

// QueuePairNum sets the number of queue pairs
func (b *IBWriteBWCommandBuilder) QueuePairNum(qp int) *IBWriteBWCommandBuilder {
	b.queuePairNum = qp
	return b
}

// MessageSize sets the message size in bytes
func (b *IBWriteBWCommandBuilder) MessageSize(size int) *IBWriteBWCommandBuilder {
	b.messageSize = size
	return b
}

// Port sets the port number
func (b *IBWriteBWCommandBuilder) Port(port int) *IBWriteBWCommandBuilder {
	b.port = port
	return b
}

// TargetIP sets the target IP address (for client commands)
func (b *IBWriteBWCommandBuilder) TargetIP(ip string) *IBWriteBWCommandBuilder {
	b.targetIP = ip
	return b
}

// RedirectOutput sets output redirection
func (b *IBWriteBWCommandBuilder) RedirectOutput(redirect string) *IBWriteBWCommandBuilder {
	b.redirectOutput = redirect
	return b
}

// Background sets whether to run in background
func (b *IBWriteBWCommandBuilder) Background(enable bool) *IBWriteBWCommandBuilder {
	b.background = enable
	return b
}

// SleepTime sets the sleep time after command execution
func (b *IBWriteBWCommandBuilder) SleepTime(sleep string) *IBWriteBWCommandBuilder {
	b.sleepTime = sleep
	return b
}

// SSHWrapper sets whether to wrap command in SSH
func (b *IBWriteBWCommandBuilder) SSHWrapper(enable bool) *IBWriteBWCommandBuilder {
	b.sshWrapper = enable
	return b
}

// SSHPrivateKey sets the SSH private key path
func (b *IBWriteBWCommandBuilder) SSHPrivateKey(keyPath string) *IBWriteBWCommandBuilder {
	b.sshPrivateKey = keyPath
	return b
}

// Report sets whether to enable report generation
func (b *IBWriteBWCommandBuilder) Report(enable bool) *IBWriteBWCommandBuilder {
	b.report = enable
	return b
}

// OutputFileName sets the output file name for report (required when report is true)
func (b *IBWriteBWCommandBuilder) OutputFileName(filename string) *IBWriteBWCommandBuilder {
	b.outputFileName = filename
	return b
}

// Bidirectional sets whether to run bidirectional test (adds -b flag)
func (b *IBWriteBWCommandBuilder) Bidirectional(enable bool) *IBWriteBWCommandBuilder {
	b.bidirectional = enable
	return b
}

// RdmaCm sets whether to use RDMA CM (adds -R flag)
func (b *IBWriteBWCommandBuilder) RdmaCm(enable bool) *IBWriteBWCommandBuilder {
	b.rdmaCm = enable
	return b
}

// GidIndex sets the GID index for RoCE v2 (adds -x flag)
func (b *IBWriteBWCommandBuilder) GidIndex(index int) *IBWriteBWCommandBuilder {
	b.gidIndex = index
	return b
}

// ForLatencyTest switches to latency test mode (use ib_write_lat instead of ib_write_bw)
func (b *IBWriteBWCommandBuilder) ForLatencyTest(enable bool) *IBWriteBWCommandBuilder {
	b.latencyTest = enable
	return b
}

// String builds and returns the complete command string
func (b *IBWriteBWCommandBuilder) String() string {
	// Validate that outputFileName is provided when report is enabled and not running infinitely
	if b.report && !b.runInfinitely && b.outputFileName == "" {
		panic("outputFileName must be specified when report is enabled and runInfinitely is false")
	}

	var cmd strings.Builder

	if b.sshWrapper {
		if b.sshPrivateKey != "" {
			cmd.WriteString(fmt.Sprintf("ssh -i %s %s '", b.sshPrivateKey, b.host))
		} else {
			cmd.WriteString(fmt.Sprintf("ssh %s '", b.host))
		}
	}

	// Use ib_write_lat for latency tests, ib_write_bw for bandwidth tests
	if b.latencyTest {
		cmd.WriteString("ib_write_lat")
	} else {
		cmd.WriteString("ib_write_bw")
	}

	if b.device != "" {
		cmd.WriteString(fmt.Sprintf(" -d %s", b.device))
	}

	if b.runInfinitely {
		cmd.WriteString(" --run_infinitely")
	} else if b.durationSeconds > 0 {
		cmd.WriteString(fmt.Sprintf(" -D %d", b.durationSeconds))
	}

	// Queue pair and message size parameters are not used in latency tests
	// ib_write_lat tests single message latency
	if !b.latencyTest {
		if b.queuePairNum > 0 {
			cmd.WriteString(fmt.Sprintf(" -q %d", b.queuePairNum))
		}

		if b.messageSize > 0 {
			cmd.WriteString(fmt.Sprintf(" -m %d", b.messageSize))
		}
	}

	if b.port > 0 {
		cmd.WriteString(fmt.Sprintf(" -p %d", b.port))
	}

	if b.bidirectional {
		cmd.WriteString(" -b")
	}

	if b.rdmaCm {
		cmd.WriteString(" -R")
	}

	if b.gidIndex > 0 {
		cmd.WriteString(fmt.Sprintf(" -x %d", b.gidIndex))
	}

	if b.targetIP != "" {
		cmd.WriteString(fmt.Sprintf(" %s", b.targetIP))
	}

	// Add report parameters if report is enabled and not running infinitely
	if b.report && !b.runInfinitely {
		if b.latencyTest {
			// For latency tests, use --out_json and --out_json_file (same as bandwidth)
			cmd.WriteString(" --out_json")
			if b.outputFileName != "" {
				cmd.WriteString(fmt.Sprintf(" --out_json_file %s", b.outputFileName))
			}
		} else {
			// For bandwidth tests, use --report_gbits and --out_json
			cmd.WriteString(" --report_gbits --out_json")
			if b.outputFileName != "" {
				cmd.WriteString(fmt.Sprintf(" --out_json_file %s", b.outputFileName))
			}
		}
	}

	if b.redirectOutput != "" {
		cmd.WriteString(fmt.Sprintf(" %s", b.redirectOutput))
	}

	if b.background {
		cmd.WriteString(" &")
	}

	if b.sshWrapper {
		cmd.WriteString("'")
	}

	if b.sleepTime != "" {
		cmd.WriteString(fmt.Sprintf("; sleep %s", b.sleepTime))
	}

	return cmd.String()
}

// ServerCommand creates a server command (no target IP)
func (b *IBWriteBWCommandBuilder) ServerCommand() *IBWriteBWCommandBuilder {
	return b.TargetIP("")
}

// ClientCommand creates a client command with target IP
func (b *IBWriteBWCommandBuilder) ClientCommand() *IBWriteBWCommandBuilder {
	return b.SleepTime("0.06") // Client commands typically have longer sleep
}
