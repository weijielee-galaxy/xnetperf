package generator

import (
	"fmt"
	"xnetperf/config"
	"xnetperf/internal/tools"
)

type bwLocaltestScriptGenerator struct {
	*ScriptGenerator
	cfg     *config.Config
	hostIPs map[string]string
}

func NewBwLocaltestScriptGenerator(cfg *config.Config, hostIPs map[string]string) *bwLocaltestScriptGenerator {
	return &bwLocaltestScriptGenerator{
		ScriptGenerator: new(ScriptGenerator),
		cfg:             cfg,
		hostIPs:         hostIPs,
	}
}

func (g *bwLocaltestScriptGenerator) GenerateScripts() (*ScriptResult, error) {
	// Check port availability
	if err := g.CheckPortsAvailability(); err != nil {
		return nil, err
	}

	// Localtest: 使用server配置的hosts和HCAs
	// 每个host既作为server又作为client，包括自己打自己
	hosts := g.cfg.Server.Hostname
	hcas := g.cfg.Server.Hca

	// Group scripts by host (not host_hca)
	// Key format: "hostname"
	serverCmdMap := make(map[string][]string)
	clientCmdMap := make(map[string][]string)

	port := g.cfg.StartPort

	// 外层循环：遍历所有"server"角色的host_hca组合
	for _, serverHost := range hosts {
		for _, serverHca := range hcas {
			// 内层循环：与所有"client"角色的host_hca组合建立连接（包括自己）
			for _, clientHost := range hosts {
				for _, clientHca := range hcas {
					// Server command (在serverHost上执行)
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

					// Client command (在clientHost上执行)
					clientCmd := tools.NewIBWriteBwCommand().
						Device(clientHca).
						QueuePairs(g.cfg.QpNum).
						MessageSize(g.cfg.MessageSizeBytes).
						Port(port).
						TargetIP(g.hostIPs[serverHost]).
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

	// Build scripts grouped by host
	sScripts := BuildHostScriptsFromCmdMap(serverCmdMap, g.cfg.SSH.User)
	cScripts := BuildHostScriptsFromCmdMap(clientCmdMap, g.cfg.SSH.User)

	return &ScriptResult{
		ServerScripts: sScripts,
		ClientScripts: cScripts,
	}, nil
}

func (g *bwLocaltestScriptGenerator) CheckPortsAvailability() error {
	// Localtest: 每个host_hca与所有host_hca（包括自己）建立连接
	// 总连接数 = (hosts数 * HCAs数)^2
	numHostHcaCombinations := len(g.cfg.Server.Hostname) * len(g.cfg.Server.Hca)
	totalConnections := numHostHcaCombinations * numHostHcaCombinations
	requiredPorts := totalConnections
	availablePorts := 65535 - g.cfg.StartPort + 1

	if requiredPorts > availablePorts {
		return fmt.Errorf("not enough available ports starting from %d: required %d, available %d",
			g.cfg.StartPort, requiredPorts, availablePorts)
	}
	return nil
}
