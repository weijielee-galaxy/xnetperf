package lat

import (
	"fmt"
	"sort"
	"strings"

	"xnetperf/config"
)

// ANSI color codes
const (
	colorRed   = "\033[31m"
	colorReset = "\033[0m"
)

// latencyThreshold is the threshold in microseconds for marking latency as high (red)
const latencyThreshold = 4.0

// displayLatencyMatrix displays the N×N latency matrix in table format for fullmesh mode
func displayLatencyMatrix(latencyData []LatencyData) {
	if len(latencyData) == 0 {
		fmt.Println("⚠️  No latency data to display")
		return
	}

	// Build matrix structure and collect unique hosts/HCAs
	sourceHostHCAs := make(map[string][]string)   // host -> []hca
	targetHostHCAs := make(map[string][]string)   // host -> []hca
	matrix := make(map[string]map[string]float64) // "host:hca" -> "host:hca" -> latency

	for _, data := range latencyData {
		// Track source hosts and HCAs
		if sourceHostHCAs[data.SourceHost] == nil {
			sourceHostHCAs[data.SourceHost] = []string{}
		}
		found := false
		for _, hca := range sourceHostHCAs[data.SourceHost] {
			if hca == data.SourceHCA {
				found = true
				break
			}
		}
		if !found {
			sourceHostHCAs[data.SourceHost] = append(sourceHostHCAs[data.SourceHost], data.SourceHCA)
		}

		// Track target hosts and HCAs
		if targetHostHCAs[data.TargetHost] == nil {
			targetHostHCAs[data.TargetHost] = []string{}
		}
		found = false
		for _, hca := range targetHostHCAs[data.TargetHost] {
			if hca == data.TargetHCA {
				found = true
				break
			}
		}
		if !found {
			targetHostHCAs[data.TargetHost] = append(targetHostHCAs[data.TargetHost], data.TargetHCA)
		}

		// Build matrix
		sourceKey := fmt.Sprintf("%s:%s", data.SourceHost, data.SourceHCA)
		targetKey := fmt.Sprintf("%s:%s", data.TargetHost, data.TargetHCA)
		if matrix[sourceKey] == nil {
			matrix[sourceKey] = make(map[string]float64)
		}
		matrix[sourceKey][targetKey] = data.AvgLatencyUs
	}

	// Sort hosts and their HCAs
	var sourceHosts []string
	for host := range sourceHostHCAs {
		sourceHosts = append(sourceHosts, host)
		sort.Strings(sourceHostHCAs[host])
	}
	sort.Strings(sourceHosts)

	var targetHosts []string
	for host := range targetHostHCAs {
		targetHosts = append(targetHosts, host)
		sort.Strings(targetHostHCAs[host])
	}
	sort.Strings(targetHosts)

	// Calculate column widths - no truncation, use actual max length
	hostColWidth := 10 // Minimum width for hostname column
	for _, host := range sourceHosts {
		if len(host) > hostColWidth {
			hostColWidth = len(host)
		}
	}
	for _, host := range targetHosts {
		if len(host) > hostColWidth {
			hostColWidth = len(host)
		}
	}

	hcaColWidth := 10 // Minimum width for HCA column
	for _, hcas := range sourceHostHCAs {
		for _, hca := range hcas {
			if len(hca) > hcaColWidth {
				hcaColWidth = len(hca)
			}
		}
	}
	for _, hcas := range targetHostHCAs {
		for _, hca := range hcas {
			if len(hca) > hcaColWidth {
				hcaColWidth = len(hca)
			}
		}
	}

	valueColWidth := 12 // Width for latency values (e.g., "123.45 μs")

	// Print title
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("📊 Latency Matrix (Average Latency in microseconds)")
	fmt.Println(strings.Repeat("=", 80))

	// Count total target columns
	totalTargetCols := 0
	for _, host := range targetHosts {
		totalTargetCols += len(targetHostHCAs[host])
	}

	// Print top border
	fmt.Printf("┌%s┬%s┬",
		strings.Repeat("─", hostColWidth+2),
		strings.Repeat("─", hcaColWidth+2))
	for i, targetHost := range targetHosts {
		numHCAs := len(targetHostHCAs[targetHost])
		width := numHCAs*valueColWidth + (numHCAs-1)*3 + 2
		if i < len(targetHosts)-1 {
			fmt.Printf("%s┬", strings.Repeat("─", width))
		} else {
			fmt.Printf("%s┐\n", strings.Repeat("─", width))
		}
	}

	// Print first header row (target hostnames)
	fmt.Printf("│%*s│%*s│",
		hostColWidth+2, " ",
		hcaColWidth+2, " ")
	for i, targetHost := range targetHosts {
		numHCAs := len(targetHostHCAs[targetHost])
		width := numHCAs*valueColWidth + (numHCAs-1)*3
		displayHost := targetHost
		// No truncation - expand width if needed
		if len(targetHost) > width {
			width = len(targetHost)
		}
		if i < len(targetHosts)-1 {
			fmt.Printf(" %-*s │", width, displayHost)
		} else {
			fmt.Printf(" %-*s │\n", width, displayHost)
		}
	}

	// Print separator between hostname row and HCA row
	fmt.Printf("│%*s│%*s├",
		hostColWidth+2, " ",
		hcaColWidth+2, " ")
	for i, targetHost := range targetHosts {
		hcas := targetHostHCAs[targetHost]
		for j := range hcas {
			if j < len(hcas)-1 {
				fmt.Printf("%s┬", strings.Repeat("─", valueColWidth+2))
			} else {
				if i < len(targetHosts)-1 {
					fmt.Printf("%s┼", strings.Repeat("─", valueColWidth+2))
				} else {
					fmt.Printf("%s┤\n", strings.Repeat("─", valueColWidth+2))
				}
			}
		}
	}

	// Print second header row (target HCAs)
	fmt.Printf("│%*s│%*s│",
		hostColWidth+2, " ",
		hcaColWidth+2, " ")
	for i, targetHost := range targetHosts {
		hcas := targetHostHCAs[targetHost]
		for j, hca := range hcas {
			displayHCA := hca
			// Calculate actual width needed for this HCA column
			actualWidth := valueColWidth
			if len(hca) > valueColWidth {
				actualWidth = len(hca)
			}
			if j < len(hcas)-1 || i < len(targetHosts)-1 {
				fmt.Printf(" %-*s │", actualWidth, displayHCA)
			} else {
				fmt.Printf(" %-*s │\n", actualWidth, displayHCA)
			}
		}
	}

	// Print header separator
	fmt.Printf("├%s┼%s┼",
		strings.Repeat("─", hostColWidth+2),
		strings.Repeat("─", hcaColWidth+2))
	for i := 0; i < totalTargetCols; i++ {
		if i < totalTargetCols-1 {
			fmt.Printf("%s┼", strings.Repeat("─", valueColWidth+2))
		} else {
			fmt.Printf("%s┤\n", strings.Repeat("─", valueColWidth+2))
		}
	}

	// Print data rows
	rowIdx := 0
	for _, sourceHost := range sourceHosts {
		sourceHCAs := sourceHostHCAs[sourceHost]
		for hcaIdx, sourceHCA := range sourceHCAs {
			// Print hostname in first column (only for first HCA of this host) - no truncation
			if hcaIdx == 0 {
				displayHost := sourceHost
				fmt.Printf("│ %-*s │", hostColWidth, displayHost)
			} else {
				fmt.Printf("│%*s│", hostColWidth+2, " ")
			}

			// Print HCA in second column - no truncation
			displayHCA := sourceHCA
			fmt.Printf(" %-*s │", hcaColWidth, displayHCA)

			// Print latency values
			for _, targetHost := range targetHosts {
				targetHCAs := targetHostHCAs[targetHost]
				for _, targetHCA := range targetHCAs {
					sourceKey := fmt.Sprintf("%s:%s", sourceHost, sourceHCA)
					targetKey := fmt.Sprintf("%s:%s", targetHost, targetHCA)
					latency := matrix[sourceKey][targetKey]

					if latency > 0 {
						valueStr := fmt.Sprintf("%.2f μs", latency)
						if latency > latencyThreshold {
							// Mark high latency in red
							fmt.Printf(" %s%*s%s │", colorRed, valueColWidth, valueStr, colorReset)
						} else {
							fmt.Printf(" %*s │", valueColWidth, valueStr)
						}
					} else {
						// Check if this is self-to-self (diagonal)
						if sourceHost == targetHost && sourceHCA == targetHCA {
							// Self-to-self: display "-" without red color
							fmt.Printf(" %*s │", valueColWidth, "-")
						} else {
							// Missing data: display red "∞" to indicate test failure/unreachable
							fmt.Printf(" %s%*s%s │", colorRed, valueColWidth, "∞", colorReset)
						}
					}
				}
			}
			fmt.Println()

			rowIdx++

			// Print row separator
			needsSeparator := false
			isLastHCAOfHost := hcaIdx == len(sourceHCAs)-1
			isLastHost := sourceHost == sourceHosts[len(sourceHosts)-1]

			if !isLastHost || !isLastHCAOfHost {
				if isLastHCAOfHost {
					// Separator between different hosts (with left border crossing hostname column)
					fmt.Printf("├%s┼%s┼",
						strings.Repeat("─", hostColWidth+2),
						strings.Repeat("─", hcaColWidth+2))
					needsSeparator = true
				} else {
					// Separator within same host (hostname column stays empty)
					fmt.Printf("│%*s├%s┼",
						hostColWidth+2, " ",
						strings.Repeat("─", hcaColWidth+2))
					needsSeparator = true
				}

				if needsSeparator {
					for i := 0; i < totalTargetCols; i++ {
						if i < totalTargetCols-1 {
							fmt.Printf("%s┼", strings.Repeat("─", valueColWidth+2))
						} else {
							fmt.Printf("%s┤\n", strings.Repeat("─", valueColWidth+2))
						}
					}
				}
			}
		}
	}

	// Print bottom border
	fmt.Printf("└%s┴%s┴",
		strings.Repeat("─", hostColWidth+2),
		strings.Repeat("─", hcaColWidth+2))
	for i := 0; i < totalTargetCols; i++ {
		if i < totalTargetCols-1 {
			fmt.Printf("%s┴", strings.Repeat("─", valueColWidth+2))
		} else {
			fmt.Printf("%s┘\n", strings.Repeat("─", valueColWidth+2))
		}
	}

	// Calculate and display statistics
	displayStatistics(latencyData)
}

// displayLatencyMatrixIncast displays the client×server latency matrix for incast mode
func displayLatencyMatrixIncast(latencyData []LatencyData, cfg *config.Config) {
	if len(latencyData) == 0 {
		fmt.Println("⚠️  No latency data to display")
		return
	}

	// Build matrix structure for incast mode (clients → servers)
	clientHostHCAs := make(map[string][]string)   // client host -> []hca
	serverHostHCAs := make(map[string][]string)   // server host -> []hca
	matrix := make(map[string]map[string]float64) // "client:hca" -> "server:hca" -> latency

	// Separate clients and servers based on config
	clientHostSet := make(map[string]bool)
	for _, host := range cfg.Client.Hostname {
		clientHostSet[host] = true
	}

	for _, data := range latencyData {
		// Determine if this is client→server or vice versa based on config
		isClientToServer := clientHostSet[data.SourceHost]

		if isClientToServer {
			// Track client (source) hosts and HCAs
			if clientHostHCAs[data.SourceHost] == nil {
				clientHostHCAs[data.SourceHost] = []string{}
			}
			found := false
			for _, hca := range clientHostHCAs[data.SourceHost] {
				if hca == data.SourceHCA {
					found = true
					break
				}
			}
			if !found {
				clientHostHCAs[data.SourceHost] = append(clientHostHCAs[data.SourceHost], data.SourceHCA)
			}

			// Track server (target) hosts and HCAs
			if serverHostHCAs[data.TargetHost] == nil {
				serverHostHCAs[data.TargetHost] = []string{}
			}
			found = false
			for _, hca := range serverHostHCAs[data.TargetHost] {
				if hca == data.TargetHCA {
					found = true
					break
				}
			}
			if !found {
				serverHostHCAs[data.TargetHost] = append(serverHostHCAs[data.TargetHost], data.TargetHCA)
			}

			// Build matrix
			clientKey := fmt.Sprintf("%s:%s", data.SourceHost, data.SourceHCA)
			serverKey := fmt.Sprintf("%s:%s", data.TargetHost, data.TargetHCA)
			if matrix[clientKey] == nil {
				matrix[clientKey] = make(map[string]float64)
			}
			matrix[clientKey][serverKey] = data.AvgLatencyUs
		}
	}

	// Sort hosts and their HCAs
	var clientHosts []string
	for host := range clientHostHCAs {
		clientHosts = append(clientHosts, host)
		sort.Strings(clientHostHCAs[host])
	}
	sort.Strings(clientHosts)

	var serverHosts []string
	for host := range serverHostHCAs {
		serverHosts = append(serverHosts, host)
		sort.Strings(serverHostHCAs[host])
	}
	sort.Strings(serverHosts)

	// Calculate column widths - no truncation, use actual max length
	hostColWidth := 10
	for _, host := range clientHosts {
		if len(host) > hostColWidth {
			hostColWidth = len(host)
		}
	}
	for _, host := range serverHosts {
		if len(host) > hostColWidth {
			hostColWidth = len(host)
		}
	}

	hcaColWidth := 10
	for _, hcas := range clientHostHCAs {
		for _, hca := range hcas {
			if len(hca) > hcaColWidth {
				hcaColWidth = len(hca)
			}
		}
	}
	for _, hcas := range serverHostHCAs {
		for _, hca := range hcas {
			if len(hca) > hcaColWidth {
				hcaColWidth = len(hca)
			}
		}
	}

	valueColWidth := 12

	// Print title
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("📊 Latency Matrix - INCAST Mode (Client → Server)")
	fmt.Println("   Average Latency in microseconds")
	fmt.Println(strings.Repeat("=", 80))

	// Print top border
	fmt.Printf("┌%s┬%s┬",
		strings.Repeat("─", hostColWidth+2),
		strings.Repeat("─", hcaColWidth+2))
	for i, serverHost := range serverHosts {
		numHCAs := len(serverHostHCAs[serverHost])
		width := numHCAs*valueColWidth + (numHCAs-1)*3 + 2
		if i < len(serverHosts)-1 {
			fmt.Printf("%s┬", strings.Repeat("─", width))
		} else {
			fmt.Printf("%s┐\n", strings.Repeat("─", width))
		}
	}

	// Print first header row (server hostnames)
	fmt.Printf("│%*s│%*s│",
		hostColWidth+2, " ",
		hcaColWidth+2, " ")
	for i, serverHost := range serverHosts {
		numHCAs := len(serverHostHCAs[serverHost])
		width := numHCAs*valueColWidth + (numHCAs-1)*3
		displayHost := serverHost
		// No truncation - expand width if needed
		if len(serverHost) > width {
			width = len(serverHost)
		}
		if i < len(serverHosts)-1 {
			fmt.Printf(" %-*s │", width, displayHost)
		} else {
			fmt.Printf(" %-*s │\n", width, displayHost)
		}
	}

	// Print separator between hostname row and HCA row
	fmt.Printf("│%*s│%*s├",
		hostColWidth+2, " ",
		hcaColWidth+2, " ")
	for i, serverHost := range serverHosts {
		hcas := serverHostHCAs[serverHost]
		for j := range hcas {
			if j < len(hcas)-1 {
				fmt.Printf("%s┬", strings.Repeat("─", valueColWidth+2))
			} else {
				if i < len(serverHosts)-1 {
					fmt.Printf("%s┼", strings.Repeat("─", valueColWidth+2))
				} else {
					fmt.Printf("%s┤\n", strings.Repeat("─", valueColWidth+2))
				}
			}
		}
	}

	// Print second header row (server HCAs)
	fmt.Printf("│%*s│%*s│",
		hostColWidth+2, " ",
		hcaColWidth+2, " ")
	for i, serverHost := range serverHosts {
		hcas := serverHostHCAs[serverHost]
		for j, hca := range hcas {
			displayHCA := hca
			// Calculate actual width needed for this HCA column
			actualWidth := valueColWidth
			if len(hca) > valueColWidth {
				actualWidth = len(hca)
			}
			if j < len(hcas)-1 || i < len(serverHosts)-1 {
				fmt.Printf(" %-*s │", actualWidth, displayHCA)
			} else {
				fmt.Printf(" %-*s │\n", actualWidth, displayHCA)
			}
		}
	}

	// Print header separator
	fmt.Printf("├%s┼%s┼",
		strings.Repeat("─", hostColWidth+2),
		strings.Repeat("─", hcaColWidth+2))
	for i, serverHost := range serverHosts {
		numHCAs := len(serverHostHCAs[serverHost])
		for j := 0; j < numHCAs; j++ {
			if j < numHCAs-1 {
				fmt.Printf("%s┼", strings.Repeat("─", valueColWidth+2))
			} else {
				if i < len(serverHosts)-1 {
					fmt.Printf("%s┼", strings.Repeat("─", valueColWidth+2))
				} else {
					fmt.Printf("%s┤\n", strings.Repeat("─", valueColWidth+2))
				}
			}
		}
	}

	// Print data rows (clients)
	for clientHostIdx, clientHostName := range clientHosts {
		hcas := clientHostHCAs[clientHostName]
		for hcaIdx, clientHCA := range hcas {
			// Print client hostname (only on first HCA row) - no truncation
			if hcaIdx == 0 {
				displayHost := clientHostName
				fmt.Printf("│ %-*s │", hostColWidth, displayHost)
			} else {
				fmt.Printf("│%*s│", hostColWidth+2, " ")
			}

			// Print client HCA - no truncation
			displayHCA := clientHCA
			fmt.Printf(" %-*s │", hcaColWidth, displayHCA)

			// Print latency values for all servers
			clientKey := fmt.Sprintf("%s:%s", clientHostName, clientHCA)
			for _, serverHost := range serverHosts {
				serverHcas := serverHostHCAs[serverHost]
				for _, serverHCA := range serverHcas {
					serverKey := fmt.Sprintf("%s:%s", serverHost, serverHCA)
					latency := matrix[clientKey][serverKey]
					if latency > 0 {
						valueStr := fmt.Sprintf("%.2f μs", latency)
						if latency > latencyThreshold {
							// Mark high latency in red
							fmt.Printf(" %s%*s%s │", colorRed, valueColWidth, valueStr, colorReset)
						} else {
							fmt.Printf(" %*s │", valueColWidth, valueStr)
						}
					} else {
						// In incast mode, client and server are separate, so missing data is always a failure
						// Display red "∞" to indicate test failure/unreachable
						fmt.Printf(" %s%*s%s │", colorRed, valueColWidth, "∞", colorReset)
					}
				}
			}
			fmt.Println()

			// Print row separator
			isLastHCA := hcaIdx == len(hcas)-1
			isLastHost := clientHostIdx == len(clientHosts)-1

			if isLastHost && isLastHCA {
				// Last row - bottom border
				fmt.Printf("└%s┴%s┴",
					strings.Repeat("─", hostColWidth+2),
					strings.Repeat("─", hcaColWidth+2))
				for i, serverHost := range serverHosts {
					numHCAs := len(serverHostHCAs[serverHost])
					for j := 0; j < numHCAs; j++ {
						if j < numHCAs-1 {
							fmt.Printf("%s┴", strings.Repeat("─", valueColWidth+2))
						} else {
							if i < len(serverHosts)-1 {
								fmt.Printf("%s┴", strings.Repeat("─", valueColWidth+2))
							} else {
								fmt.Printf("%s┘\n", strings.Repeat("─", valueColWidth+2))
							}
						}
					}
				}
			} else if isLastHCA {
				// End of host group - use crossing separator
				fmt.Printf("├%s┼%s┼",
					strings.Repeat("─", hostColWidth+2),
					strings.Repeat("─", hcaColWidth+2))
				for i, serverHost := range serverHosts {
					numHCAs := len(serverHostHCAs[serverHost])
					for j := 0; j < numHCAs; j++ {
						if j < numHCAs-1 {
							fmt.Printf("%s┼", strings.Repeat("─", valueColWidth+2))
						} else {
							if i < len(serverHosts)-1 {
								fmt.Printf("%s┼", strings.Repeat("─", valueColWidth+2))
							} else {
								fmt.Printf("%s┤\n", strings.Repeat("─", valueColWidth+2))
							}
						}
					}
				}
			} else {
				// Within host group - use non-crossing separator
				fmt.Printf("│%*s├%s┼",
					hostColWidth+2, " ",
					strings.Repeat("─", hcaColWidth+2))
				for i, serverHost := range serverHosts {
					numHCAs := len(serverHostHCAs[serverHost])
					for j := 0; j < numHCAs; j++ {
						if j < numHCAs-1 {
							fmt.Printf("%s┼", strings.Repeat("─", valueColWidth+2))
						} else {
							if i < len(serverHosts)-1 {
								fmt.Printf("%s┼", strings.Repeat("─", valueColWidth+2))
							} else {
								fmt.Printf("%s┤\n", strings.Repeat("─", valueColWidth+2))
							}
						}
					}
				}
			}
		}
	}

	// Calculate and display statistics
	displayIncastStatistics(latencyData, clientHostHCAs, serverHostHCAs)
}

// displayStatistics calculates and displays statistics for fullmesh mode
func displayStatistics(latencyData []LatencyData) {
	var allLatencies []float64
	for _, data := range latencyData {
		allLatencies = append(allLatencies, data.AvgLatencyUs)
	}

	minLatency := minFloat(allLatencies)
	maxLatency := maxFloat(allLatencies)
	avgLatency := avgFloat(allLatencies)

	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("📈 Latency Statistics:")
	fmt.Printf("  Minimum Latency: %.2f μs\n", minLatency)
	fmt.Printf("  Maximum Latency: %.2f μs\n", maxLatency)
	fmt.Printf("  Average Latency: %.2f μs\n", avgLatency)
	fmt.Printf("  Total Measurements: %d\n", len(latencyData))
	fmt.Println(strings.Repeat("=", 80))
}

// displayIncastStatistics calculates and displays statistics for incast mode
func displayIncastStatistics(latencyData []LatencyData, clientHostHCAs map[string][]string, serverHostHCAs map[string][]string) {
	if len(latencyData) == 0 {
		return
	}

	// Calculate global statistics
	var allLatencies []float64
	for _, data := range latencyData {
		allLatencies = append(allLatencies, data.AvgLatencyUs)
	}

	sort.Float64s(allLatencies)
	minLatency := allLatencies[0]
	maxLatency := allLatencies[len(allLatencies)-1]
	var sum float64
	for _, lat := range allLatencies {
		sum += lat
	}
	avgLatency := sum / float64(len(allLatencies))

	// Calculate per-server statistics
	serverStats := make(map[string][]float64)
	for _, data := range latencyData {
		serverKey := fmt.Sprintf("%s:%s", data.TargetHost, data.TargetHCA)
		serverStats[serverKey] = append(serverStats[serverKey], data.AvgLatencyUs)
	}

	// Calculate per-client statistics
	clientStats := make(map[string][]float64)
	for _, data := range latencyData {
		clientKey := fmt.Sprintf("%s:%s", data.SourceHost, data.SourceHCA)
		clientStats[clientKey] = append(clientStats[clientKey], data.AvgLatencyUs)
	}

	// Print statistics
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("📈 Statistics Summary")
	fmt.Println(strings.Repeat("=", 80))

	// Global statistics
	fmt.Println("\n🌐 Global Statistics:")
	fmt.Printf("   Total measurements: %d\n", len(allLatencies))
	fmt.Printf("   Minimum latency:    %.2f μs\n", minLatency)
	fmt.Printf("   Maximum latency:    %.2f μs\n", maxLatency)
	fmt.Printf("   Average latency:    %.2f μs\n", avgLatency)

	// Per-server statistics
	fmt.Println("\n🖥️  Per-Server Average Latency:")
	var serverHosts []string
	for host := range serverHostHCAs {
		serverHosts = append(serverHosts, host)
	}
	sort.Strings(serverHosts)

	for _, host := range serverHosts {
		for _, hca := range serverHostHCAs[host] {
			serverKey := fmt.Sprintf("%s:%s", host, hca)
			if latencies, ok := serverStats[serverKey]; ok && len(latencies) > 0 {
				var sum float64
				for _, lat := range latencies {
					sum += lat
				}
				avg := sum / float64(len(latencies))
				fmt.Printf("   %-30s  %.2f μs  (%d clients)\n", serverKey, avg, len(latencies))
			}
		}
	}

	// Per-client statistics
	fmt.Println("\n💻 Per-Client Average Latency:")
	var clientHosts []string
	for host := range clientHostHCAs {
		clientHosts = append(clientHosts, host)
	}
	sort.Strings(clientHosts)

	for _, host := range clientHosts {
		for _, hca := range clientHostHCAs[host] {
			clientKey := fmt.Sprintf("%s:%s", host, hca)
			if latencies, ok := clientStats[clientKey]; ok && len(latencies) > 0 {
				var sum float64
				for _, lat := range latencies {
					sum += lat
				}
				avg := sum / float64(len(latencies))
				fmt.Printf("   %-30s  %.2f μs  (%d servers)\n", clientKey, avg, len(latencies))
			}
		}
	}

	fmt.Println(strings.Repeat("=", 80))
}

// Helper functions for statistics
func minFloat(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	min := values[0]
	for _, v := range values {
		if v < min {
			min = v
		}
	}
	return min
}

func maxFloat(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	max := values[0]
	for _, v := range values {
		if v > max {
			max = v
		}
	}
	return max
}

func avgFloat(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}
