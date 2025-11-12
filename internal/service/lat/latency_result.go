package lat

// LatencyData represents a single latency measurement
type LatencyData struct {
	SourceHost   string  `json:"source_host"`
	SourceHCA    string  `json:"source_hca"`
	TargetHost   string  `json:"target_host"`
	TargetHCA    string  `json:"target_hca"`
	AvgLatencyUs float64 `json:"avg_latency_us"` // Average latency in microseconds
	MinLatencyUs float64 `json:"min_latency_us"` // Minimum latency in microseconds
	MaxLatencyUs float64 `json:"max_latency_us"` // Maximum latency in microseconds
}

// LatencySummary is the complete latency report for API responses
type LatencySummary struct {
	StreamType  string                        `json:"stream_type"` // fullmesh or incast
	Matrix      map[string]map[string]float64 `json:"matrix"`      // "host:hca" -> "host:hca" -> latency
	Statistics  LatencyStatistics             `json:"statistics"`
	ClientStats map[string]LatencyStats       `json:"client_stats,omitempty"` // Only for incast mode
	ServerStats map[string]LatencyStats       `json:"server_stats,omitempty"` // Only for incast mode
}

// LatencyStatistics contains global latency statistics
type LatencyStatistics struct {
	MinLatency float64 `json:"min_latency"` // Minimum latency in μs
	MaxLatency float64 `json:"max_latency"` // Maximum latency in μs
	AvgLatency float64 `json:"avg_latency"` // Average latency in μs
	TotalCount int     `json:"total_count"` // Total number of measurements
}

// LatencyStats contains statistics for a specific host/HCA
type LatencyStats struct {
	AvgLatency float64 `json:"avg_latency"` // Average latency in μs
	Count      int     `json:"count"`       // Number of measurements
}

// LatencyReport represents the JSON structure from ib_write_lat
type LatencyReport struct {
	Results struct {
		TAvg float64 `json:"t_avg"` // Average latency in microseconds
		TMin float64 `json:"t_min"` // Minimum latency in microseconds
		TMax float64 `json:"t_max"` // Maximum latency in microseconds
	} `json:"results"`
}

// LatencyProbeResult represents the probe result for ib_write_lat processes
type LatencyProbeResult struct {
	Hostname     string   `json:"hostname"`
	ProcessCount int      `json:"process_count"`
	Processes    []string `json:"processes,omitempty"`
	Error        string   `json:"error,omitempty"`
	Status       string   `json:"status"` // RUNNING, COMPLETED, ERROR
}

// LatencyProbeSummary is the probe summary for API responses
type LatencyProbeSummary struct {
	Timestamp      string               `json:"timestamp"`
	Results        []LatencyProbeResult `json:"results"`
	RunningHosts   int                  `json:"running_hosts"`
	CompletedHosts int                  `json:"completed_hosts"`
	ErrorHosts     int                  `json:"error_hosts"`
	TotalProcesses int                  `json:"total_processes"`
	AllCompleted   bool                 `json:"all_completed"`
}
