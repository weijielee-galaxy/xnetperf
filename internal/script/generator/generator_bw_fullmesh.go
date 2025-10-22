package generator

import (
	"fmt"
	"xnetperf/config"
	"xnetperf/internal/tools"
)

type bwFullmeshScriptGenerator struct {
	cfg    *config.Config
	allIPs map[string]string // Optional: for testing, bypasses SSH lookup
}

func NewBwFullmeshScriptGenerator(cfg *config.Config) *bwFullmeshScriptGenerator {
	return &bwFullmeshScriptGenerator{
		cfg: cfg,
	}
}

// WithHostIPs allows injecting host IPs for testing (bypasses SSH lookup)
func (g *bwFullmeshScriptGenerator) WithHostIPs(ips map[string]string) *bwFullmeshScriptGenerator {
	g.allIPs = ips
	return g
}

func (g *bwFullmeshScriptGenerator) GenerateScripts() (*ScriptResult, error) {
	// Check port availability
	if err := g.CheckPortsAvailability(); err != nil {
		return nil, err
	}

	// Get all hosts (server + client)
	allHosts := append(g.cfg.Server.Hostname, g.cfg.Client.Hostname...)

	// Get all HCAs (assuming server and client have same HCAs for fullmesh)
	allHcas := g.cfg.Client.Hca

	// Lookup IPs for all hosts
	var allIPs map[string]string
	if g.allIPs != nil {
		allIPs = g.allIPs
	} else {
		allIPs = make(map[string]string)
		for _, host := range allHosts {
			// For server hosts
			if contains(g.cfg.Server.Hostname, host) {
				serverIps, err := g.cfg.LookupServerHostsIP()
				if err != nil {
					return nil, fmt.Errorf("failed to lookup server IPs: %v", err)
				}
				allIPs[host] = serverIps[host]
				continue
			}
			// For client hosts
			if contains(g.cfg.Client.Hostname, host) {
				clientIps, err := g.cfg.LookupClientHostsIP()
				if err != nil {
					return nil, fmt.Errorf("failed to lookup client IPs: %v", err)
				}
				allIPs[host] = clientIps[host]
			}
		}
	} // Group scripts by host
	// For fullmesh: each host acts as both server and client
	// serverCmdMap: map[serverHost -> []commands] - commands where this host is server
	// clientCmdMap: map[clientHost -> []commands] - commands where this host is client
	serverCmdMap := make(map[string][]string)
	clientCmdMap := make(map[string][]string)

	port := g.cfg.StartPort

	// Generate scripts for each server host with each HCA
	for _, serverHost := range allHosts {
		for _, serverHca := range allHcas {
			// For each potential client (all other hosts)
			for _, clientHost := range allHosts {
				// Skip connection to itself
				if clientHost == serverHost {
					continue
				}

				for _, clientHca := range allHcas {
					// Server command
					serverCmd := tools.NewIBWriteBwCommand().
						Device(serverHca).
						QueuePairs(g.cfg.QpNum).
						MessageSize(g.cfg.MessageSizeBytes).
						Port(port).
						RunInfinitely(g.cfg.Run.Infinitely).
						Duration(g.cfg.Run.DurationSeconds).
						RdmaCm(g.cfg.RdmaCm).
						GidIndex(g.cfg.GidIndex)
					if g.cfg.Report.Enable {
						serverCmd = serverCmd.EnableReport(
							fmt.Sprintf("%s/report_s_%s_%s_%d.json",
								g.cfg.Report.Dir, serverHost, serverHca, port))
					}
					serverCmdMap[serverHost] = append(serverCmdMap[serverHost], serverCmd.String())

					// Client command
					clientCmd := tools.NewIBWriteBwCommand().
						Device(clientHca).
						QueuePairs(g.cfg.QpNum).
						MessageSize(g.cfg.MessageSizeBytes).
						Port(port).
						TargetIP(allIPs[serverHost]).
						RunInfinitely(g.cfg.Run.Infinitely).
						Duration(g.cfg.Run.DurationSeconds).
						RdmaCm(g.cfg.RdmaCm).
						GidIndex(g.cfg.GidIndex)
					if g.cfg.Report.Enable {
						clientCmd = clientCmd.EnableReport(
							fmt.Sprintf("%s/report_c_%s_%s_%d.json",
								g.cfg.Report.Dir, clientHost, clientHca, port))
					}
					clientCmdMap[clientHost] = append(clientCmdMap[clientHost], clientCmd.String())

					port++
				}
			}
		}
	}

	// Build server scripts
	sScripts := BuildHostScriptsFromCmdMap(serverCmdMap)
	// Build client scripts
	cScripts := BuildHostScriptsFromCmdMap(clientCmdMap)

	return &ScriptResult{
		ServerScripts: sScripts,
		ClientScripts: cScripts,
	}, nil
}

func (g *bwFullmeshScriptGenerator) CheckPortsAvailability() error {
	// For fullmesh: all hosts connect to all other hosts
	allHosts := append(g.cfg.Server.Hostname, g.cfg.Client.Hostname...)
	allHcas := g.cfg.Client.Hca

	// Each host (with each HCA) connects to every other host (with each HCA)
	// Formula: len(allHosts) * len(allHcas) * (len(allHosts) - 1) * len(allHcas)
	// Simplified: len(allHosts) * (len(allHosts) - 1) * len(allHcas)^2
	totalConnections := len(allHosts) * (len(allHosts) - 1) * len(allHcas) * len(allHcas)
	requiredPorts := totalConnections
	availablePorts := 65535 - g.cfg.StartPort + 1

	if requiredPorts > availablePorts {
		return fmt.Errorf("not enough available ports starting from %d: required %d, available %d",
			g.cfg.StartPort, requiredPorts, availablePorts)
	}
	return nil
}

// Helper function to check if a slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
