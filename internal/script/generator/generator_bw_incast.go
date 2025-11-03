package generator

import (
	"fmt"
	"xnetperf/config"
	"xnetperf/internal/tools"
)

type bwIncastScriptGenerator struct {
	*ScriptGenerator
	cfg     *config.Config
	hostIPs map[string]string
}

func NewBwIncastScriptGenerator(cfg *config.Config, hostIPs map[string]string) *bwIncastScriptGenerator {
	return &bwIncastScriptGenerator{
		ScriptGenerator: new(ScriptGenerator),
		cfg:             cfg,
		hostIPs:         hostIPs,
	}
}

func (g *bwIncastScriptGenerator) GenerateScripts() (*ScriptResult, error) {

	// Check port availability
	if err := g.CheckPortsAvailability(); err != nil {
		return nil, err
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
						TargetIP(g.hostIPs[sHost]).
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

	sScripts := BuildHostScriptsFromCmdMap(sCmdMap, g.cfg.SSH.User)
	cScripts := BuildHostScriptsFromCmdMap(cCmdMap, g.cfg.SSH.User)

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
