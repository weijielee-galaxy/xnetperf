package tools

// CommandType represents the type of IB command
type CommandType string

const (
	// IBWriteBw represents the ib_write_bw command for bandwidth testing
	IBWriteBw CommandType = "ib_write_bw"

	// IBWriteLat represents the ib_write_lat command for latency testing
	IBWriteLat CommandType = "ib_write_lat"
)

// String returns the string representation of the command type
func (ct CommandType) String() string {
	return string(ct)
}

// IsLatencyTest returns true if this is a latency test command
func (ct CommandType) IsLatencyTest() bool {
	return ct == IBWriteLat
}

// IsBandwidthTest returns true if this is a bandwidth test command
func (ct CommandType) IsBandwidthTest() bool {
	return ct == IBWriteBw
}
