package generator

import (
	"fmt"
	"xnetperf/config"
)

type bwFullmeshScriptGenerator struct {
	*ScriptGenerator
	cfg     *config.Config
	hostIPs map[string]string
}

func NewBwFullmeshScriptGenerator(cfg *config.Config, hostIPs map[string]string) *bwFullmeshScriptGenerator {
	return &bwFullmeshScriptGenerator{
		ScriptGenerator: new(ScriptGenerator),
		cfg:             cfg,
		hostIPs:         hostIPs,
	}
}

func (g *bwFullmeshScriptGenerator) GenerateScripts() (*ScriptResult, error) {
	// Check port availability
	if err := g.CheckPortsAvailability(); err != nil {
		return nil, err
	}

	// Group scripts by host
	serverCmdMap := make(map[string][]string)
	clientCmdMap := make(map[string][]string)

	port := g.cfg.StartPort
	for _, sHost := range g.cfg.Server.Hostname {
		for _, sHca := range g.cfg.Server.Hca {
			for _, cHost := range g.cfg.Client.Hostname {
				for _, cHca := range g.cfg.Client.Hca {
					sasFile := fmt.Sprintf("%s/report_s_%s_%s_%d.json",
						g.cfg.Report.Dir, sHost, sHca, port)
					sasCmd := g.buildIbWriteBwCommand(g.cfg, sHca, port, "", sasFile)
					serverCmdMap[sHost] = append(serverCmdMap[sHost], sasCmd)

					cacFile := fmt.Sprintf("%s/report_c_%s_%s_%d.json",
						g.cfg.Report.Dir, cHost, cHca, port)
					cacCmd := g.buildIbWriteBwCommand(g.cfg, cHca, port, g.hostIPs[sHost], cacFile)
					clientCmdMap[cHost] = append(clientCmdMap[cHost], cacCmd)

					casFile := fmt.Sprintf("%s/report_s_%s_%s_%d.json",
						g.cfg.Report.Dir, cHost, cHca, port)
					casCmd := g.buildIbWriteBwCommand(g.cfg, cHca, port, "", casFile)
					serverCmdMap[cHost] = append(serverCmdMap[cHost], casCmd)

					sacFile := fmt.Sprintf("%s/report_c_%s_%s_%d.json",
						g.cfg.Report.Dir, sHost, sHca, port)
					sacCmd := g.buildIbWriteBwCommand(g.cfg, sHca, port, g.hostIPs[cHost], sacFile)
					clientCmdMap[sHost] = append(clientCmdMap[sHost], sacCmd)

					port++
				}
			}
		}
	}

	// Build server scripts
	sScripts := BuildHostScriptsFromCmdMap(serverCmdMap, g.cfg.SSH.User)
	// Build client scripts
	cScripts := BuildHostScriptsFromCmdMap(clientCmdMap, g.cfg.SSH.User)

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
