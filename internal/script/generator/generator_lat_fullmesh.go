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

func (g *latFullmeshScriptGenerator) GenerateScripts() (*ScriptResult, error) {
	// Check port availability
	if err := g.CheckPortsAvailability(); err != nil {
		return nil, err
	}

	serverCmdMap := make(map[string][]string) // map: serverHost -> []commands
	clientCmdMap := make(map[string][]string) // map: clientHost -> []commands
	// 构建所有 host_hca 组合
	type hostHca struct {
		host string
		hca  string
	}

	var allCombos []hostHca

	// 添加 Server 配置中的所有 host_hca
	for _, host := range g.cfg.Server.Hostname {
		for _, hca := range g.cfg.Server.Hca {
			allCombos = append(allCombos, hostHca{host: host, hca: hca})
		}
	}

	// 添加 Client 配置中的所有 host_hca
	for _, host := range g.cfg.Client.Hostname {
		for _, hca := range g.cfg.Client.Hca {
			allCombos = append(allCombos, hostHca{host: host, hca: hca})
		}
	}

	// 去重（同一个 host_hca 可能在 Server 和 Client 都出现）
	comboMap := make(map[string]hostHca)
	for _, combo := range allCombos {
		key := fmt.Sprintf("%s_%s", combo.host, combo.hca)
		comboMap[key] = combo
	}

	// 转换回切片
	allCombos = make([]hostHca, 0, len(comboMap))
	for _, combo := range comboMap {
		allCombos = append(allCombos, combo)
	}
	port := g.cfg.StartPort
	// 所有 host_hca 组合互相连接
	for _, combo1 := range allCombos {
		for _, combo2 := range allCombos {
			// 跳过同一个 HCA 自己到自己
			if combo1.host == combo2.host && combo1.hca == combo2.hca {
				continue
			}
			// combo1 -> combo2
			serverFile := fmt.Sprintf("%s/latency_fullmesh_s_%s_%s_from_%s_%s_p%d.json",
				g.cfg.Report.Dir, combo1.host, combo1.hca, combo2.host, combo2.hca, port)
			serverCmd := g.buildIbWriteLatCommand(g.cfg, combo1.hca, port, "", serverFile)
			serverCmdMap[combo1.host] = append(serverCmdMap[combo1.host], serverCmd)

			clientFile := fmt.Sprintf("%s/latency_fullmesh_c_%s_%s_to_%s_%s_p%d.json",
				g.cfg.Report.Dir, combo2.host, combo2.hca, combo1.host, combo1.hca, port)
			clientCmd := g.buildIbWriteLatCommand(g.cfg, combo2.hca, port, g.hostIPs[combo1.host], clientFile)
			clientCmdMap[combo2.host] = append(clientCmdMap[combo2.host], clientCmd)

			port++
		}
	}

	sScripts := BuildHostScriptsFromCmdMap(serverCmdMap)
	cScripts := BuildHostScriptsFromCmdMap(clientCmdMap)

	return &ScriptResult{
		ServerScripts: sScripts,
		ClientScripts: cScripts,
	}, nil
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
// Deprecated: previous fullmesh implementation
// does not cover all host_hca combinations, only server to client and vice versa
func (g *latFullmeshScriptGenerator) GenerateScripts0() (*ScriptResult, error) {
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
	totalConnections := len(allHosts) * (len(allHosts) - 1) * len(allHcas) * len(allHcas) * 2
	requiredPorts := totalConnections
	availablePorts := 65535 - g.cfg.StartPort + 1

	if requiredPorts > availablePorts {
		return fmt.Errorf("not enough available ports starting from %d: required %d, available %d",
			g.cfg.StartPort, requiredPorts, availablePorts)
	}
	return nil
}
