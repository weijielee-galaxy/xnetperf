package generator

import (
	"fmt"
	"xnetperf/config"
	"xnetperf/internal/tools"
)

type latIncastScriptGenerator struct {
	cfg *config.Config
	*ScriptGenerator
	hostIPs map[string]string
}

func NewLatIncastScriptGenerator(cfg *config.Config, hostIPs map[string]string) *latIncastScriptGenerator {
	return &latIncastScriptGenerator{
		ScriptGenerator: new(ScriptGenerator),
		cfg:             cfg,
		hostIPs:         hostIPs,
	}
}

func (g *latIncastScriptGenerator) GenerateScripts() (*ScriptResult, error) {
	// Check port availability
	if err := g.CheckPortsAvailability(); err != nil {
		return nil, err
	}

	serverCmdMap := make(map[string][]string) // map: serverHost -> []commands
	clientCmdMap := make(map[string][]string) // map: clientHost -> []commands

	for _, sHost := range g.cfg.Server.Hostname {
		port := g.cfg.StartPort
		for _, sHca := range g.cfg.Server.Hca {
			for _, cHost := range g.cfg.Client.Hostname {
				for _, cHca := range g.cfg.Client.Hca {
					serverCmd := tools.NewIBWriteLatCommand().
						Device(sHca).
						Port(port).
						RunInfinitely(false).
						Duration(5).
						RdmaCm(g.cfg.RdmaCm).
						GidIndex(g.cfg.GidIndex)
					if g.cfg.Report.Enable {
						serverCmd = serverCmd.EnableReport(fmt.Sprintf("%s/latency_incast_s_%s_%s_from_%s_%s_p%d.json",
							g.cfg.Report.Dir, sHost, sHca, cHost, cHca, port))
					}
					serverCmdMap[sHost] = append(serverCmdMap[sHost], serverCmd.String())

					clientCmd := tools.NewIBWriteLatCommand().
						Device(cHca).
						Port(port).
						TargetIP(g.hostIPs[sHost]).
						RunInfinitely(false).
						Duration(5).
						RdmaCm(g.cfg.RdmaCm).
						GidIndex(g.cfg.GidIndex)

					if g.cfg.Report.Enable {
						clientCmd = clientCmd.EnableReport(
							fmt.Sprintf("%s/latency_incast_c_%s_%s_to_%s_%s_p%d.json",
								g.cfg.Report.Dir, cHost, cHca, sHost, sHca, port))
					}

					clientCmdMap[cHost] = append(clientCmdMap[cHost], clientCmd.String())
					port++
				}
			}
		}
	}

	sScripts := BuildHostScriptsFromCmdMap(serverCmdMap)
	cScripts := BuildHostScriptsFromCmdMap(clientCmdMap)

	return &ScriptResult{
		ServerScripts: sScripts,
		ClientScripts: cScripts,
	}, nil
}

func (g *latIncastScriptGenerator) CheckPortsAvailability() error {
	totalConnections := len(g.cfg.Server.Hostname) * len(g.cfg.Server.Hca) * len(g.cfg.Client.Hostname) * len(g.cfg.Client.Hca)
	requiredPorts := totalConnections
	availablePorts := 65535 - g.cfg.StartPort + 1

	if requiredPorts > availablePorts {
		return fmt.Errorf("not enough available ports starting from %d: required %d, available %d",
			g.cfg.StartPort, requiredPorts, availablePorts)
	}
	return nil
}
