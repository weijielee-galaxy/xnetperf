package cmd

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"xnetperf/internal/service/connectivity"

	"github.com/spf13/cobra"
)

var checkConnCmd = &cobra.Command{
	Use:   "check-conn",
	Short: "Check bidirectional connectivity between all HCA pairs",
	Long: `Check bidirectional network connectivity between all configured HCA pairs.

This command performs a comprehensive connectivity test by:
1. Running client->server latency tests (all clients to all servers)
2. Running server->client latency tests (swapped roles for reverse direction)
3. Analyzing connectivity status based on successful latency measurements
4. Displaying a summary of connected, disconnected, and error pairs

The test uses short-duration (5s) incast latency tests to quickly verify
connectivity without long-running performance tests.

Requirements:
  - Report generation must be enabled in config (report.enable: true)
  - Config will be temporarily modified during testing and restored after

Examples:
  # Check connectivity with default config
  xnetperf check-conn

  # Check with custom config file
  xnetperf check-conn -c /path/to/config.yaml`,
	Run: runCheckConn,
}

func runCheckConn(cmd *cobra.Command, args []string) {
	cfg := GetConfig()

	// Only v1 version is supported for connectivity check
	if cfg.Version != "v1" {
		fmt.Println("❌ Connectivity check is only available in v1 mode")
		fmt.Println("   Please set 'version: v1' in your config file")
		os.Exit(1)
	}

	// Check if report generation is enabled
	if !cfg.Report.Enable {
		fmt.Println("❌ Report generation must be enabled for connectivity check")
		fmt.Println("   Please set 'report.enable: true' in your config file")
		os.Exit(1)
	}

	fmt.Println("🔍 Starting bidirectional connectivity check...")
	fmt.Println(fmt.Sprintf("   Timeout: %d seconds per test direction", connectivity.GetConnectivityTestTimeout()))
	fmt.Println()

	checker := connectivity.New(cfg)
	summary, err := checker.CheckConnectivity()
	if err != nil {
		fmt.Printf("❌ Connectivity check failed: %v\n", err)
		os.Exit(1)
	}

	// Display results
	displayConnectivityResults(summary)

	// Exit with appropriate status
	if summary.ErrorPairs > 0 || summary.DisconnectedPairs > 0 {
		os.Exit(1)
	}
}

func displayConnectivityResults(summary *connectivity.ConnectivitySummary) {
	fmt.Println(strings.Repeat("=", 100))
	fmt.Println("  📊 Connectivity Check Results")
	fmt.Println(strings.Repeat("=", 100))
	fmt.Println()

	// Display summary statistics
	fmt.Printf("📈 Summary:\n")
	fmt.Printf("   Total HCA pairs tested:  %d\n", summary.TotalPairs)
	fmt.Printf("   ✅ Connected pairs:       %d\n", summary.ConnectedPairs)
	fmt.Printf("   ❌ Disconnected pairs:    %d\n", summary.DisconnectedPairs)
	fmt.Printf("   ⚠️  Error pairs:           %d\n", summary.ErrorPairs)
	fmt.Println()

	// Build bidirectional connectivity map and sort
	connPairs := buildAndSortConnectivityPairs(summary.Results)

	// Calculate dynamic column widths
	hostWidth, hcaWidth := calculateColumnWidths(connPairs)

	// Display all connectivity details
	fmt.Println("🔗 Bidirectional Connectivity Details:")
	fmt.Println()

	// Print table header with dynamic widths
	printTableHeader(hostWidth, hcaWidth)

	var lastSourceHost string
	var lastHostPair string // Track source+target combination

	for i, pair := range connPairs {
		sourceHost := pair.SourceHost
		sourceHCA := pair.SourceHCA
		targetHost := pair.TargetHost
		targetHCA := pair.TargetHCA

		currentHostPair := fmt.Sprintf("%s->%s", sourceHost, targetHost)

		// Determine if we need to print source host (merge cells)
		var sourceHostStr string
		if sourceHost != lastSourceHost {
			sourceHostStr = sourceHost
			lastSourceHost = sourceHost
			lastHostPair = "" // Reset host pair when source changes
		} else if currentHostPair != lastHostPair {
			// Same source host but different target host, show source host again
			sourceHostStr = sourceHost
		} else {
			sourceHostStr = ""
		}

		// Determine if we need to print target host (merge cells based on host pair)
		var targetHostStr string
		if currentHostPair != lastHostPair {
			targetHostStr = targetHost
			lastHostPair = currentHostPair
		} else {
			targetHostStr = ""
		}

		// Display the connection (4 rows)
		displayConnectivityRow(sourceHostStr, sourceHCA, targetHCA, targetHostStr, pair.Forward, pair.Backward, hostWidth, hcaWidth)

		// Add separator between rows (except last one)
		if i < len(connPairs)-1 {
			nextPair := connPairs[i+1]
			nextHostPair := fmt.Sprintf("%s->%s", nextPair.SourceHost, nextPair.TargetHost)

			// Check if next row has different source host (major separator)
			if nextPair.SourceHost != sourceHost {
				printTableSeparator(hostWidth, hcaWidth, true)
			} else if nextHostPair != currentHostPair {
				// Same source host but different target host (also use major separator)
				printTableSeparator(hostWidth, hcaWidth, true)
			} else {
				// Same host pair, just different HCA (minor separator)
				printTableSeparator(hostWidth, hcaWidth, false)
			}
		}
	}

	printTableFooter(hostWidth, hcaWidth)

	fmt.Println()

	// Overall status
	if summary.ConnectedPairs == summary.TotalPairs {
		fmt.Println("✅ All HCA pairs are connected! Network connectivity is healthy.")
	} else {
		fmt.Println("⚠️  Some HCA pairs have connectivity issues. Please check the details above.")
	}
	fmt.Println()
}

// ConnectivityPair represents a pair of HCAs with their bidirectional connectivity
type ConnectivityPair struct {
	SourceHost string
	SourceHCA  string
	TargetHost string
	TargetHCA  string
	Forward    *connectivity.ConnectivityResult
	Backward   *connectivity.ConnectivityResult
}

// calculateColumnWidths calculates the optimal column widths for hosts and HCAs
func calculateColumnWidths(pairs []ConnectivityPair) (int, int) {
	maxHostLen := 8 // minimum width for "Host"
	maxHCALen := 4  // minimum width for "HCA"

	for _, pair := range pairs {
		if len(pair.SourceHost) > maxHostLen {
			maxHostLen = len(pair.SourceHost)
		}
		if len(pair.TargetHost) > maxHostLen {
			maxHostLen = len(pair.TargetHost)
		}
		if len(pair.SourceHCA) > maxHCALen {
			maxHCALen = len(pair.SourceHCA)
		}
		if len(pair.TargetHCA) > maxHCALen {
			maxHCALen = len(pair.TargetHCA)
		}
	}

	return maxHostLen, maxHCALen
}

// printTableHeader prints the table header with dynamic column widths
func printTableHeader(hostWidth, hcaWidth int) {
	// Top border
	fmt.Printf("┌%s┬%s┬───────────────────────────────────────┬%s┬%s┐\n",
		strings.Repeat("─", hostWidth+2),
		strings.Repeat("─", hcaWidth+2),
		strings.Repeat("─", hcaWidth+2),
		strings.Repeat("─", hostWidth+2))

	// Header row 1
	fmt.Printf("│ %-*s │ %-*s │           Connectivity                │ %-*s │ %-*s │\n",
		hostWidth, "Source", hcaWidth, "", hcaWidth, "", hostWidth, "Target")

	// Header row 2
	fmt.Printf("│ %-*s │ %-*s │                                       │ %-*s │ %-*s │\n",
		hostWidth, "Host", hcaWidth, "HCA", hcaWidth, "HCA", hostWidth, "Host")

	// Separator
	fmt.Printf("├%s┼%s┼───────────────────────────────────────┼%s┼%s┤\n",
		strings.Repeat("─", hostWidth+2),
		strings.Repeat("─", hcaWidth+2),
		strings.Repeat("─", hcaWidth+2),
		strings.Repeat("─", hostWidth+2))
}

// printTableSeparator prints a separator line between rows
func printTableSeparator(hostWidth, hcaWidth int, fullSeparator bool) {
	if fullSeparator {
		// Full separator (between different source hosts)
		fmt.Printf("├%s┼%s┼───────────────────────────────────────┼%s┼%s┤\n",
			strings.Repeat("─", hostWidth+2),
			strings.Repeat("─", hcaWidth+2),
			strings.Repeat("─", hcaWidth+2),
			strings.Repeat("─", hostWidth+2))
	} else {
		// Partial separator (within same source host)
		fmt.Printf("│%s├%s┼───────────────────────────────────────┼%s┤%s│\n",
			strings.Repeat(" ", hostWidth+2),
			strings.Repeat("─", hcaWidth+2),
			strings.Repeat("─", hcaWidth+2),
			strings.Repeat(" ", hostWidth+2))
	}
}

// printTableFooter prints the table footer
func printTableFooter(hostWidth, hcaWidth int) {
	fmt.Printf("└%s┴%s┴───────────────────────────────────────┴%s┴%s┘\n",
		strings.Repeat("─", hostWidth+2),
		strings.Repeat("─", hcaWidth+2),
		strings.Repeat("─", hcaWidth+2),
		strings.Repeat("─", hostWidth+2))
}

// buildAndSortConnectivityPairs builds and sorts connectivity pairs
func buildAndSortConnectivityPairs(results []connectivity.ConnectivityResult) []ConnectivityPair {
	// First, build bidirectional map
	connMap := buildBidirectionalConnectivityMap(results)

	// Convert map to slice for sorting
	var pairs []ConnectivityPair
	for _, directions := range connMap {
		forward := directions[0]
		backward := directions[1]

		if forward == nil && backward == nil {
			continue
		}

		var sourceHost, sourceHCA, targetHost, targetHCA string
		if forward != nil {
			sourceHost = forward.SourceHost
			sourceHCA = forward.SourceHCA
			targetHost = forward.TargetHost
			targetHCA = forward.TargetHCA
		} else {
			sourceHost = backward.TargetHost
			sourceHCA = backward.TargetHCA
			targetHost = backward.SourceHost
			targetHCA = backward.SourceHCA
		}

		pairs = append(pairs, ConnectivityPair{
			SourceHost: sourceHost,
			SourceHCA:  sourceHCA,
			TargetHost: targetHost,
			TargetHCA:  targetHCA,
			Forward:    forward,
			Backward:   backward,
		})
	}

	// Sort by: source host -> target host -> source HCA -> target HCA
	sort.Slice(pairs, func(i, j int) bool {
		if pairs[i].SourceHost != pairs[j].SourceHost {
			return pairs[i].SourceHost < pairs[j].SourceHost
		}
		if pairs[i].TargetHost != pairs[j].TargetHost {
			return pairs[i].TargetHost < pairs[j].TargetHost
		}
		if pairs[i].SourceHCA != pairs[j].SourceHCA {
			return pairs[i].SourceHCA < pairs[j].SourceHCA
		}
		return pairs[i].TargetHCA < pairs[j].TargetHCA
	})

	return pairs
}

// buildBidirectionalConnectivityMap groups connectivity results by HCA pairs
func buildBidirectionalConnectivityMap(results []connectivity.ConnectivityResult) map[string][2]*connectivity.ConnectivityResult {
	connMap := make(map[string][2]*connectivity.ConnectivityResult)

	for i := range results {
		result := &results[i]

		// Create a normalized pair key (always smaller first for consistency)
		source := fmt.Sprintf("%s:%s", result.SourceHost, result.SourceHCA)
		target := fmt.Sprintf("%s:%s", result.TargetHost, result.TargetHCA)

		var pairKey string
		var isForward bool
		if source < target {
			pairKey = fmt.Sprintf("%s<->%s", source, target)
			isForward = true
		} else {
			pairKey = fmt.Sprintf("%s<->%s", target, source)
			isForward = false
		}

		pair := connMap[pairKey]
		if isForward {
			pair[0] = result
		} else {
			pair[1] = result
		}
		connMap[pairKey] = pair
	}

	return connMap
}

// displayBidirectionalConnection displays a bidirectional connection with arrows
func displayBidirectionalConnection(forward, backward *connectivity.ConnectivityResult) {
	displayBidirectionalConnectionRow(forward, backward)
}

// displayConnectivityRow displays a single connectivity row in the table (4 lines)
func displayConnectivityRow(sourceHost, sourceHCA, targetHCA, targetHost string, forward, backward *connectivity.ConnectivityResult, hostWidth, hcaWidth int) {
	// Format forward status and latency
	var forwardStatus, forwardLatency string
	if forward != nil && forward.Connected && forward.Error == "" {
		forwardStatus = "Connected"
		forwardLatency = fmt.Sprintf("%.1f us", forward.AvgLatencyUs)
	} else if forward != nil && forward.Error != "" {
		forwardStatus = "Error"
		forwardLatency = truncateString(forward.Error, 15)
	} else {
		forwardStatus = "Disconnected"
		forwardLatency = "-"
	}

	// Format backward status and latency
	var backwardStatus, backwardLatency string
	if backward != nil && backward.Connected && backward.Error == "" {
		backwardStatus = "Connected"
		backwardLatency = fmt.Sprintf("%.1f us", backward.AvgLatencyUs)
	} else if backward != nil && backward.Error != "" {
		backwardStatus = "Error"
		backwardLatency = truncateString(backward.Error, 15)
	} else {
		backwardStatus = "Disconnected"
		backwardLatency = "-"
	}

	// Build the arrows with embedded latency
	forwardArrow := fmt.Sprintf("     ────────── %s─────────────>", forwardLatency)
	backwardArrow := fmt.Sprintf("     <───────── %s──────────────", backwardLatency)

	// Display 4 rows for this connection
	// Row 1: source host/HCA + forward status + target HCA (empty target host)
	fmt.Printf("│ %-*s │ %-*s │  %-37s│ %-*s │ %-*s │\n",
		hostWidth, sourceHost, hcaWidth, sourceHCA, forwardStatus, hcaWidth, targetHCA, hostWidth, "")

	// Row 2: forward arrow with latency
	fmt.Printf("│ %-*s │ %-*s │%-39s│ %-*s │ %-*s │\n",
		hostWidth, "", hcaWidth, "", forwardArrow, hcaWidth, "", hostWidth, "")

	// Row 3: backward arrow with latency
	fmt.Printf("│ %-*s │ %-*s │%-39s│ %-*s │ %-*s │\n",
		hostWidth, "", hcaWidth, "", backwardArrow, hcaWidth, "", hostWidth, "")

	// Row 4: backward status + target host
	fmt.Printf("│ %-*s │ %-*s │%37s  │ %-*s │ %-*s │\n",
		hostWidth, "", hcaWidth, "", backwardStatus, hcaWidth, "", hostWidth, targetHost)
}

// displayBidirectionalConnectionRow displays a single row in the connectivity table
func displayBidirectionalConnectionRow(forward, backward *connectivity.ConnectivityResult) {
	if forward == nil && backward == nil {
		return
	}

	// Determine the two endpoints
	var source, target string
	if forward != nil {
		source = fmt.Sprintf("%s %s", forward.SourceHost, forward.SourceHCA)
		target = fmt.Sprintf("%s %s", forward.TargetHost, forward.TargetHCA)
	} else {
		source = fmt.Sprintf("%s %s", backward.TargetHost, backward.TargetHCA)
		target = fmt.Sprintf("%s %s", backward.SourceHost, backward.SourceHCA)
	}

	// Format forward direction (source -> target)
	var forwardInfo string
	if forward != nil && forward.Connected && forward.Error == "" {
		forwardInfo = fmt.Sprintf("%.1f us", forward.AvgLatencyUs)
	} else if forward != nil && forward.Error != "" {
		forwardInfo = "Error"
	} else {
		forwardInfo = "-"
	}

	// Format backward direction (target -> source)
	var backwardInfo string
	if backward != nil && backward.Connected && backward.Error == "" {
		backwardInfo = fmt.Sprintf("%.1f us", backward.AvgLatencyUs)
	} else if backward != nil && backward.Error != "" {
		backwardInfo = "Error"
	} else {
		backwardInfo = "-"
	}

	// Build the middle section with arrows and latency
	middleSection := fmt.Sprintf("  %10s ───────>", forwardInfo)
	middleSection2 := fmt.Sprintf("  <─────── %-10s", backwardInfo)

	// Print the row (spanning 2 lines for bidirectional arrows)
	fmt.Printf("│ %-22s │%-45s│ %-22s │\n", source, middleSection, "")
	fmt.Printf("│ %-22s │%-45s│ %-22s │\n", "", middleSection2, target)
	fmt.Println("├────────────────────────┼─────────────────────────────────────────────┼────────────────────────┤")
}

// truncateString truncates a string to the specified length
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

// max returns the larger of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
