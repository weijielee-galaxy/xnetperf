package generator

import (
	"fmt"
	"xnetperf/config"
	"xnetperf/internal/tools"
)

type bwP2PScriptGenerator struct {
	*ScriptGenerator
	cfg     *config.Config
	hostIPs map[string]string
}

func NewBwP2PScriptGenerator(cfg *config.Config, hostIPs map[string]string) *bwP2PScriptGenerator {
	return &bwP2PScriptGenerator{
		ScriptGenerator: new(ScriptGenerator),
		cfg:             cfg,
		hostIPs:         hostIPs,
	}
}

func (g *bwP2PScriptGenerator) GenerateScripts() (*ScriptResult, error) {
	// Check port availability
	if err := g.CheckPortsAvailability(); err != nil {
		return nil, err
	}

	// P2P: 按索引一对一配对
	serverCmdMap := make(map[string][]string) // key: serverHost
	clientCmdMap := make(map[string][]string) // key: clientHost

	port := g.cfg.StartPort

	// 外层循环：遍历host pairs（按索引配对）
	for hostIndex, serverHost := range g.cfg.Server.Hostname {
		clientHost := g.cfg.Client.Hostname[hostIndex]

		// 内层循环：遍历HCA pairs（staggered配对）
		for hcaIndex, serverHca := range g.cfg.Server.Hca {
			// Staggered HCA pairing to avoid same-index connections
			clientHcaIndex := (hcaIndex + 1) % len(g.cfg.Client.Hca)
			clientHca := g.cfg.Client.Hca[clientHcaIndex]

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
					fmt.Sprintf("%s/report_%s_%s_%d.json",
						g.cfg.Report.Dir, serverHost, serverHca, port))
			}

			serverCmdMap[serverHost] = append(serverCmdMap[serverHost], serverCmd.String())

			// Client command
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
					fmt.Sprintf("%s/report_%s_%s_%d.json",
						g.cfg.Report.Dir, clientHost, clientHca, port))
			}

			clientCmdMap[clientHost] = append(clientCmdMap[clientHost], clientCmd.String())

			port++
		}
	}

	// 按host构建脚本
	sScripts := BuildHostScriptsFromCmdMap(serverCmdMap)
	cScripts := BuildHostScriptsFromCmdMap(clientCmdMap)

	return &ScriptResult{
		ServerScripts: sScripts,
		ClientScripts: cScripts,
	}, nil
}

func (g *bwP2PScriptGenerator) CheckPortsAvailability() error {
	// P2P: 总连接数 = host pairs数 × HCA pairs数
	// host pairs = len(Server.Hostname) (因为一对一配对)
	// HCA pairs = len(Server.Hca) (因为一对一配对)
	totalConnections := len(g.cfg.Server.Hostname) * len(g.cfg.Server.Hca)
	requiredPorts := totalConnections
	availablePorts := 65535 - g.cfg.StartPort + 1

	if requiredPorts > availablePorts {
		return fmt.Errorf("not enough available ports starting from %d: required %d, available %d",
			g.cfg.StartPort, requiredPorts, availablePorts)
	}
	return nil
}
