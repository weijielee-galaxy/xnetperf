package connectivity

import (
	"fmt"
	"log/slog"
	"os"

	"xnetperf/config"
	"xnetperf/internal/script"
	"xnetperf/internal/service/collect"
	"xnetperf/internal/service/lat"
)

const (
	connectivityTestTimeoutSeconds = 600
)

// GetConnectivityTestTimeout returns the connectivity test timeout in seconds
func GetConnectivityTestTimeout() int {
	return connectivityTestTimeoutSeconds
}

// Checker manages connectivity testing workflow
type Checker struct {
	cfg    *config.Config
	logger *slog.Logger
}

// New creates a new connectivity checker instance
func New(cfg *config.Config) *Checker {
	return &Checker{
		cfg:    cfg,
		logger: slog.Default().With("module", "CONNECTIVITY"),
	}
}

// CheckConnectivity performs bidirectional connectivity testing using 2 incast latency tests
// This tests all client->server and server->client HCA connectivity
func (c *Checker) CheckConnectivity() (*ConnectivitySummary, error) {
	c.logger.Info("Starting connectivity check")

	// Save original config
	originalStreamType := c.cfg.StreamType
	originalDuration := c.cfg.Run.DurationSeconds
	originalInfinitely := c.cfg.Run.Infinitely

	// Set config for connectivity testing (use incast mode with short duration)
	c.cfg.StreamType = config.InCast
	c.cfg.Run.DurationSeconds = 5 // Short test duration
	c.cfg.Run.Infinitely = false

	// Restore original config after testing
	defer func() {
		c.cfg.StreamType = originalStreamType
		c.cfg.Run.DurationSeconds = originalDuration
		c.cfg.Run.Infinitely = originalInfinitely
	}()

	// Step 1: Run client->server connectivity test (normal incast)
	c.logger.Info("Running client->server connectivity test")
	if err := c.runConnectivityTest("client-to-server"); err != nil {
		return nil, fmt.Errorf("client->server test failed: %w", err)
	}

	// Monitor test progress with timeout
	if err := c.monitorTestProgress(connectivityTestTimeoutSeconds); err != nil {
		return nil, fmt.Errorf("failed to monitor client->server test: %w", err)
	}

	// Collect reports
	if err := c.collectReports(); err != nil {
		return nil, fmt.Errorf("failed to collect client->server reports: %w", err)
	}

	// Parse client->server results
	clientToServerResults, err := c.parseConnectivityResults("reports")
	if err != nil {
		return nil, fmt.Errorf("failed to parse client->server results: %w", err)
	}

	// Step 2: Swap client/server roles for server->client connectivity test
	c.logger.Info("Running server->client connectivity test")
	c.swapClientServer()

	if err := c.runConnectivityTest("server-to-client"); err != nil {
		c.swapClientServer() // Restore roles
		return nil, fmt.Errorf("server->client test failed: %w", err)
	}

	// Monitor test progress with timeout
	if err := c.monitorTestProgress(connectivityTestTimeoutSeconds); err != nil {
		c.swapClientServer() // Restore roles
		return nil, fmt.Errorf("failed to monitor server->client test: %w", err)
	}

	// Collect reports
	if err := c.collectReports(); err != nil {
		c.swapClientServer() // Restore roles
		return nil, fmt.Errorf("failed to collect server->client reports: %w", err)
	}

	// Parse server->client results
	serverToClientResults, err := c.parseConnectivityResults("reports")
	if err != nil {
		c.swapClientServer() // Restore roles
		return nil, fmt.Errorf("failed to parse server->client results: %w", err)
	}

	// Restore client/server roles
	c.swapClientServer()

	// Merge results
	allResults := append(clientToServerResults, serverToClientResults...)

	// Build summary
	summary := c.buildSummary(allResults)

	c.logger.Info("Connectivity check completed",
		"total_pairs", summary.TotalPairs,
		"connected", summary.ConnectedPairs,
		"disconnected", summary.DisconnectedPairs,
		"errors", summary.ErrorPairs)

	return summary, nil
}

// runConnectivityTest executes a single connectivity test
func (c *Checker) runConnectivityTest(testName string) error {
	c.logger.Info("Executing connectivity test", "test", testName)

	executor := script.NewExecutor(c.cfg, script.TestTypeLatency)
	if executor == nil {
		return fmt.Errorf("failed to create executor for connectivity test")
	}

	if err := executor.Execute(); err != nil {
		return fmt.Errorf("executor failed: %w", err)
	}

	return nil
}

// monitorTestProgress monitors test progress with timeout
func (c *Checker) monitorTestProgress(timeoutSeconds int) error {
	c.logger.Info("Monitoring test progress", "timeout_seconds", timeoutSeconds)

	latRunner := lat.New(c.cfg)
	if err := latRunner.MonitorProgressWithTimeout(timeoutSeconds); err != nil {
		return fmt.Errorf("monitor progress failed: %w", err)
	}

	return nil
}

// collectReports collects latency report files
func (c *Checker) collectReports() error {
	if !c.cfg.Report.Enable {
		c.logger.Warn("Report generation is disabled, skipping collect")
		return fmt.Errorf("report generation must be enabled for connectivity testing")
	}

	c.logger.Info("Collecting connectivity test reports")

	collector := collect.New(c.cfg)
	var cleanupRemote = true
	if err := collector.DoCollect(cleanupRemote); err != nil {
		return fmt.Errorf("error during report collection: %w", err)
	}

	return nil
}

// parseConnectivityResults parses latency reports and converts to connectivity results
func (c *Checker) parseConnectivityResults(reportsDir string) ([]ConnectivityResult, error) {
	c.logger.Info("Parsing connectivity results", "dir", reportsDir)

	// Check if reports directory exists
	if _, err := os.Stat(reportsDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("reports directory not found: %s", reportsDir)
	}

	// Use latency runner to parse reports (code reuse!)
	latRunner := lat.New(c.cfg)
	// 这里ParseLatencyReportsFromDir没有依赖cfg，否则这个cfg要传进来，避免swaped cfg影响
	latencyData, err := latRunner.ParseLatencyReportsFromDir(reportsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to parse latency reports: %w", err)
	}

	// Convert latency data to connectivity results
	var results []ConnectivityResult
	for _, data := range latencyData {
		result := ConnectivityResult{
			SourceHost:   data.SourceHost,
			SourceHCA:    data.SourceHCA,
			TargetHost:   data.TargetHost,
			TargetHCA:    data.TargetHCA,
			Connected:    true, // If we got latency data, connection succeeded
			AvgLatencyUs: data.AvgLatencyUs,
			MinLatencyUs: data.MinLatencyUs,
			MaxLatencyUs: data.MaxLatencyUs,
		}
		results = append(results, result)
	}

	c.logger.Info("Parsed connectivity results", "count", len(results))
	return results, nil
}

// swapClientServer swaps client and server roles for bidirectional testing
func (c *Checker) swapClientServer() {
	c.logger.Info("Swapping client and server roles")
	c.cfg.Client.Hostname, c.cfg.Server.Hostname = c.cfg.Server.Hostname, c.cfg.Client.Hostname
	c.cfg.Client.Hca, c.cfg.Server.Hca = c.cfg.Server.Hca, c.cfg.Client.Hca
}

// buildSummary creates a connectivity summary from individual results
func (c *Checker) buildSummary(results []ConnectivityResult) *ConnectivitySummary {
	summary := &ConnectivitySummary{
		Results: results,
	}

	for _, result := range results {
		summary.TotalPairs++
		if result.Error != "" {
			summary.ErrorPairs++
		} else if result.Connected {
			summary.ConnectedPairs++
		} else {
			summary.DisconnectedPairs++
		}
	}

	return summary
}
