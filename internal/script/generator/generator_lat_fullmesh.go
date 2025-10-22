package generator

import (
	"fmt"
	"xnetperf/config"
)

type latFullmeshScriptGenerator struct {
	*ScriptGenerator
	cfg     *config.Config
	hostIPs map[string]string
}

func NewLatFullmeshScriptGenerator(cfg *config.Config, hostIPs map[string]string) *latFullmeshScriptGenerator {
	return &latFullmeshScriptGenerator{
		ScriptGenerator: new(ScriptGenerator),
		cfg:             cfg,
		hostIPs:         hostIPs,
	}
}

/*
ServerA
  - mlx5_0 2000 2001 2002 2003

ServerB
  - mlx5_0 2004 2005 2006 2007

ClientA
  - mlx5_0 2000 2004
  - mlx5_1 2001 2005

ClientB
  - mlx5_0 2002 2006
  - mlx5_1 2003 2007
*/
func (g *latFullmeshScriptGenerator) GenerateScripts() (*ScriptResult, error) {
	// Check port availability
	if err := g.CheckPortsAvailability(); err != nil {
		return nil, err
	}

	serverCmdMap := make(map[string][]string) // map: serverHost -> []commands
	clientCmdMap := make(map[string][]string) // map: clientHost -> []commands

	port := g.cfg.StartPort
	/*
		A -> B
		sas means server as server
		cac means client as client

		B -> A
		cas means client as server
		sac means server as client

	*/
	for _, sHost := range g.cfg.Server.Hostname {
		for _, sHca := range g.cfg.Server.Hca {
			for _, cHost := range g.cfg.Client.Hostname {
				for _, cHca := range g.cfg.Client.Hca {

					// A -> B -----------------------------------------------------------------------------------------------------------------
					sasFile := fmt.Sprintf("%s/latency_incast_s_%s_%s_from_%s_%s_p%d.json", g.cfg.Report.Dir, sHost, sHca, cHost, cHca, port)
					sasCmd := g.buildIbWriteLatCommand(g.cfg, sHca, port, "", sasFile)
					serverCmdMap[sHost] = append(serverCmdMap[sHost], sasCmd)

					cacFile := fmt.Sprintf("%s/latency_incast_c_%s_%s_to_%s_%s_p%d.json", g.cfg.Report.Dir, cHost, cHca, sHost, sHca, port)
					cacCmd := g.buildIbWriteLatCommand(g.cfg, cHca, port, g.hostIPs[sHost], cacFile)
					clientCmdMap[cHost] = append(clientCmdMap[cHost], cacCmd)
					// A -> B -----------------------------------------------------------------------------------------------------------------

					// -----------------------------------------------------------------------------------------------------------------------------

					// B -> A -----------------------------------------------------------------------------------------------------------------
					casFile := fmt.Sprintf("%s/latency_incast_s_%s_%s_from_%s_%s_p%d.json", g.cfg.Report.Dir, cHost, cHca, sHost, sHca, port)
					casCmd := g.buildIbWriteLatCommand(g.cfg, cHca, port, "", casFile)
					serverCmdMap[cHost] = append(serverCmdMap[cHost], casCmd)

					sacFile := fmt.Sprintf("%s/latency_incast_c_%s_%s_to_%s_%s_p%d.json", g.cfg.Report.Dir, sHost, sHca, cHost, cHca, port)
					sacCmd := g.buildIbWriteLatCommand(g.cfg, sHca, port, g.hostIPs[cHost], sacFile)
					clientCmdMap[sHost] = append(clientCmdMap[sHost], sacCmd)
					// B -> A -----------------------------------------------------------------------------------------------------------------

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

func (g *latFullmeshScriptGenerator) CheckPortsAvailability() error {
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
