package generator

import (
	"fmt"
	"xnetperf/config"
	"xnetperf/internal/tools"
)

type bwIncastScriptGenerator struct {
	cfg       *config.Config
	serverIPs map[string]string // Optional: for testing, bypasses SSH lookup
}

func NewBwIncastScriptGenerator(cfg *config.Config) *bwIncastScriptGenerator {
	return &bwIncastScriptGenerator{
		cfg: cfg,
	}
}

// WithServerIPs allows injecting server IPs for testing (bypasses SSH lookup)
func (g *bwIncastScriptGenerator) WithServerIPs(ips map[string]string) *bwIncastScriptGenerator {
	g.serverIPs = ips
	return g
}

func (g *bwIncastScriptGenerator) GenerateScripts() (*ScriptResult, error) {

	// Check port availability
	if err := g.CheckPortsAvailability(); err != nil {
		return nil, err
	}

	// Get server IPs
	var serverIps map[string]string
	var err error
	if g.serverIPs != nil {
		serverIps = g.serverIPs
	} else {
		serverIps, err = g.cfg.LookupServerHostsIP()
		if err != nil {
			return nil, fmt.Errorf("failed to lookup server IPs: %v", err)
		}
	}

	// group scripts by host
	sCmdMap := make(map[string][]string) // map: serverHost -> []commands
	cCmdMap := make(map[string][]string) // map: clientHost -> []commands

	for _, sHost := range g.cfg.Server.Hostname {
		port := g.cfg.StartPort
		for _, sHca := range g.cfg.Server.Hca {
			for _, cHost := range g.cfg.Client.Hostname {
				for _, cHca := range g.cfg.Client.Hca {
					serverCmd := tools.NewIBWriteBwCommand().
						Device(sHca).
						QueuePairs(g.cfg.QpNum).
						MessageSize(g.cfg.MessageSizeBytes).
						Port(port).
						RunInfinitely(g.cfg.Run.Infinitely).
						Duration(g.cfg.Run.DurationSeconds).
						RdmaCm(g.cfg.RdmaCm).
						GidIndex(g.cfg.GidIndex)
					if g.cfg.Report.Enable {
						serverCmd = serverCmd.EnableReport(fmt.Sprintf("%s/report_s_%s_%s_%d.json", g.cfg.Report.Dir, sHost, sHca, port))
					}
					sCmdMap[sHost] = append(sCmdMap[sHost], serverCmd.String())

					clientCmd := tools.NewIBWriteBwCommand().
						Device(cHca).
						QueuePairs(g.cfg.QpNum).
						MessageSize(g.cfg.MessageSizeBytes).
						Port(port).
						TargetIP(serverIps[sHost]).
						RunInfinitely(g.cfg.Run.Infinitely).
						Duration(g.cfg.Run.DurationSeconds).
						RdmaCm(g.cfg.RdmaCm).
						GidIndex(g.cfg.GidIndex)

					if g.cfg.Report.Enable {
						clientCmd = clientCmd.EnableReport(
							fmt.Sprintf("%s/report_c_%s_%s_%d.json", g.cfg.Report.Dir, cHost, cHca, port))
					}

					cCmdMap[cHost] = append(cCmdMap[cHost], clientCmd.String())

					port++
				}
			}
		}
	}

	sScripts := BuildHostScriptsFromCmdMap(sCmdMap)
	cScripts := BuildHostScriptsFromCmdMap(cCmdMap)

	return &ScriptResult{
		ServerScripts: sScripts,
		ClientScripts: cScripts,
	}, nil
}

func (g *bwIncastScriptGenerator) CheckPortsAvailability() error {
	totalConnections := len(g.cfg.Server.Hostname) * len(g.cfg.Server.Hca) * len(g.cfg.Client.Hostname) * len(g.cfg.Client.Hca)
	requiredPorts := totalConnections
	availablePorts := 65535 - g.cfg.StartPort + 1

	if requiredPorts > availablePorts {
		return fmt.Errorf("not enough available ports starting from %d: required %d, available %d",
			g.cfg.StartPort, requiredPorts, availablePorts)
	}
	return nil
}
