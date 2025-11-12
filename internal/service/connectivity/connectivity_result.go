package connectivity

// ConnectivityResult represents the connectivity status between two HCAs
type ConnectivityResult struct {
	SourceHost   string  `json:"source_host"`
	SourceHCA    string  `json:"source_hca"`
	TargetHost   string  `json:"target_host"`
	TargetHCA    string  `json:"target_hca"`
	Connected    bool    `json:"connected"`       // Whether connectivity is established
	AvgLatencyUs float64 `json:"avg_latency_us"`  // Average latency in microseconds (0 if not connected)
	MinLatencyUs float64 `json:"min_latency_us"`  // Minimum latency in microseconds (0 if not connected)
	MaxLatencyUs float64 `json:"max_latency_us"`  // Maximum latency in microseconds (0 if not connected)
	Error        string  `json:"error,omitempty"` // Error message if connectivity check failed
}

// ConnectivitySummary is the complete connectivity report for API responses
type ConnectivitySummary struct {
	TotalPairs        int                  `json:"total_pairs"`        // Total number of HCA pairs tested
	ConnectedPairs    int                  `json:"connected_pairs"`    // Number of successfully connected pairs
	DisconnectedPairs int                  `json:"disconnected_pairs"` // Number of disconnected pairs
	ErrorPairs        int                  `json:"error_pairs"`        // Number of pairs with errors
	Results           []ConnectivityResult `json:"results"`            // Detailed results for each pair
}
