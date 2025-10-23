package tools

import (
	"fmt"
	"strings"
)

// IBCommand represents an InfiniBand perftest command builder
type IBCommand struct {
	commandType    CommandType
	device         string
	runInfinitely  bool
	durationSec    int
	queuePairNum   int // Only used for bandwidth tests
	messageSize    int // Only used for bandwidth tests
	redirectOutput string
	background     bool
	port           int
	targetIP       string
	bidirectional  bool
	rdmaCm         bool
	gidIndex       int
	report         bool
	outputFileName string
}

// NewIBWriteBwCommand creates a new ib_write_bw command builder
// Default: not running infinitely (user should explicitly set RunInfinitely or Duration)
func NewIBWriteBwCommand() *IBCommand {
	return &IBCommand{
		commandType:    IBWriteBw,
		runInfinitely:  false,
		redirectOutput: ">/dev/null 2>&1",
		// redirectOutput: ">>/root/20000.log 2>&1",
		background: true,
	}
}

// NewIBWriteLatCommand creates a new ib_write_lat command builder
func NewIBWriteLatCommand() *IBCommand {
	return &IBCommand{
		commandType:    IBWriteLat,
		runInfinitely:  false, // Latency tests typically run for a fixed duration
		redirectOutput: ">/dev/null 2>&1",
		background:     true,
	}
}

// Device sets the InfiniBand device (e.g., mlx5_0)
func (c *IBCommand) Device(device string) *IBCommand {
	c.device = device
	return c
}

// RunInfinitely sets whether to run the test infinitely
func (c *IBCommand) RunInfinitely(enable bool) *IBCommand {
	c.runInfinitely = enable
	return c
}

// Duration sets the test duration in seconds (only when not running infinitely)
func (c *IBCommand) Duration(seconds int) *IBCommand {
	c.durationSec = seconds
	return c
}

// QueuePairs sets the number of queue pairs (only for bandwidth tests)
func (c *IBCommand) QueuePairs(num int) *IBCommand {
	c.queuePairNum = num
	return c
}

// MessageSize sets the message size in bytes (only for bandwidth tests)
func (c *IBCommand) MessageSize(bytes int) *IBCommand {
	c.messageSize = bytes
	return c
}

// RedirectOutput sets output redirection
func (b *IBCommand) RedirectOutput(redirect string) *IBCommand {
	b.redirectOutput = redirect
	return b
}

// Background sets whether to run in background
func (b *IBCommand) Background(enable bool) *IBCommand {
	b.background = enable
	return b
}

// Port sets the port number for the test
func (c *IBCommand) Port(port int) *IBCommand {
	c.port = port
	return c
}

// TargetIP sets the target IP address (for client mode)
func (c *IBCommand) TargetIP(ip string) *IBCommand {
	c.targetIP = ip
	return c
}

// Bidirectional enables bidirectional testing
func (c *IBCommand) Bidirectional(enable bool) *IBCommand {
	c.bidirectional = enable
	return c
}

// RdmaCm enables RDMA CM mode
func (c *IBCommand) RdmaCm(enable bool) *IBCommand {
	c.rdmaCm = enable
	return c
}

// GidIndex sets the GID index for RoCE v2
func (c *IBCommand) GidIndex(index int) *IBCommand {
	c.gidIndex = index
	return c
}

// EnableReport enables JSON report output
func (c *IBCommand) EnableReport(outputFile string) *IBCommand {
	c.report = true
	c.outputFileName = outputFile
	return c
}

// AsServer configures the command for server mode (no target IP)
func (c *IBCommand) AsServer() *IBCommand {
	c.targetIP = ""
	return c
}

// AsClient configures the command for client mode (requires target IP to be set separately)
func (c *IBCommand) AsClient(targetIP string) *IBCommand {
	c.targetIP = targetIP
	return c
}

// Build generates the command string
func (c *IBCommand) Build() string {
	var cmd strings.Builder

	// Base command
	cmd.WriteString(c.commandType.String())

	// Device
	if c.device != "" {
		cmd.WriteString(fmt.Sprintf(" -d %s", c.device))
	}

	// Duration mode
	if c.runInfinitely {
		cmd.WriteString(" --run_infinitely")
	} else if c.durationSec > 0 {
		cmd.WriteString(fmt.Sprintf(" -D %d", c.durationSec))
	}

	// Bandwidth-specific parameters (not applicable for latency tests)
	if c.commandType.IsBandwidthTest() {
		if c.queuePairNum > 0 {
			cmd.WriteString(fmt.Sprintf(" -q %d", c.queuePairNum))
		}
		if c.messageSize > 0 {
			cmd.WriteString(fmt.Sprintf(" -m %d", c.messageSize))
		}
	}

	// Port
	if c.port > 0 {
		cmd.WriteString(fmt.Sprintf(" -p %d", c.port))
	}

	// Bidirectional
	if c.bidirectional {
		cmd.WriteString(" -b")
	}

	// RDMA CM
	if c.rdmaCm {
		cmd.WriteString(" -R")
	}

	// GID Index
	if c.gidIndex > 0 {
		cmd.WriteString(fmt.Sprintf(" -x %d", c.gidIndex))
	}

	// Target IP (client mode)
	if c.targetIP != "" {
		cmd.WriteString(fmt.Sprintf(" %s", c.targetIP))
	}

	// Report output
	if c.report && !c.runInfinitely {
		if c.commandType.IsLatencyTest() {
			// Latency test report
			cmd.WriteString(" --out_json")
			if c.outputFileName != "" {
				cmd.WriteString(fmt.Sprintf(" --out_json_file %s", c.outputFileName))
			}
		} else {
			// Bandwidth test report
			cmd.WriteString(" --report_gbits --out_json")
			if c.outputFileName != "" {
				cmd.WriteString(fmt.Sprintf(" --out_json_file %s", c.outputFileName))
			}
		}
	}

	if c.redirectOutput != "" {
		cmd.WriteString(fmt.Sprintf(" %s", c.redirectOutput))
	}

	if c.background {
		cmd.WriteString(" &")
	}

	return cmd.String()
}

// String returns the built command string
func (c *IBCommand) String() string {
	return c.Build()
}
