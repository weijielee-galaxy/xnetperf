package stream

import (
	"fmt"
	"os"
	"strings"
	"xnetperf/config"
)

// 假设所有的服务器都有同样的HCA数目
func GenerateIncastScripts(cfg *config.Config) {
	// 清空streamScript文件夹内容
	ClearStreamScriptDir(cfg)
	// 生成incast脚本
	totalPort := len(cfg.Client.Hostname) * len(cfg.Client.Hca) * len(cfg.Server.Hostname) * len(cfg.Server.Hca)
	fmt.Println("Total Ports Needed:", totalPort)
	if totalPort > (65535 - cfg.StartPort + 1) {
		fmt.Printf("Error: Not enough ports available. Required: %d, Available: %d\n", totalPort, 65535-cfg.StartPort+1)
		return
	}

	// 根据配置文件，生成在每个server的每个HCA上监听的脚本
	port := cfg.StartPort
	for _, sHost := range cfg.Server.Hostname {
		// 3. Run the command and capture the combined output (stdout and stderr).
		hostIP, err := getHostIP(sHost, cfg.SSH.PrivateKey, cfg.SSH.User, cfg.NetworkInterface)
		if err != nil {
			// If command fails, return the output for debugging and a detailed error.
			fmt.Printf("Error executing command on %s: %v\nOutput: %s\n", sHost, err, string(hostIP))
		}

		for _, sHca := range cfg.Server.Hca { // 对于每一个server的HCA，其要启动多少个监听脚本取决于client的HCA和client的hostname数量

			serverScriptFileName := fmt.Sprintf("%s/%s_%s_server_incast.sh", cfg.OutputDir(), sHost, sHca)
			clientScriptFileName := fmt.Sprintf("%s/%s_%s_client_incast.sh", cfg.OutputDir(), sHost, sHca)

			serverScriptContent := strings.Builder{}
			clientScriptContent := strings.Builder{}

			for _, cHost := range cfg.Client.Hostname {
				for _, cHca := range cfg.Client.Hca {
					// 使用命令构建器创建服务器命令
					serverCmd := NewIBWriteBWCommandBuilder(true).
						Host(sHost).
						Device(sHca).
						QueuePairNum(cfg.QpNum).
						MessageSize(cfg.MessageSizeBytes).
						Port(port).
						RunInfinitely(cfg.Run.Infinitely).
						DurationSeconds(cfg.Run.DurationSeconds).
						RdmaCm(cfg.RdmaCm).
						GidIndex(cfg.GidIndex).
						Report(cfg.Report.Enable).
						OutputFileName(fmt.Sprintf("%s/report_s_%s_%s_%d.json", cfg.Report.Dir, sHost, sHca, port)).
						SSHPrivateKey(cfg.SSH.PrivateKey).
						ServerCommand() // 服务器命令不需要目标IP

					// 使用命令构建器创建客户端命令
					clientCmd := NewIBWriteBWCommandBuilder(true).
						Host(cHost).
						Device(cHca).
						QueuePairNum(cfg.QpNum).
						MessageSize(cfg.MessageSizeBytes).
						Port(port).
						TargetIP(strings.TrimSpace(string(hostIP))).
						RunInfinitely(cfg.Run.Infinitely).
						DurationSeconds(cfg.Run.DurationSeconds).
						RdmaCm(cfg.RdmaCm).
						GidIndex(cfg.GidIndex).
						Report(cfg.Report.Enable).
						OutputFileName(fmt.Sprintf("%s/report_c_%s_%s_%d.json", cfg.Report.Dir, cHost, cHca, port)).
						SSHPrivateKey(cfg.SSH.PrivateKey).
						ClientCommand() // 客户端命令有更长的睡眠时间

					serverScriptContent.WriteString(serverCmd.String() + "\n")
					clientScriptContent.WriteString(clientCmd.String() + "\n")
					port++
				}
			}
			err := os.WriteFile(serverScriptFileName, []byte(serverScriptContent.String()), 0755)
			if err != nil {
				fmt.Printf("Error writing script file '%s': %v\n", serverScriptFileName, err)
				return
			}
			err = os.WriteFile(clientScriptFileName, []byte(clientScriptContent.String()), 0755)
			if err != nil {
				fmt.Printf("Error writing script file '%s': %v\n", serverScriptFileName, err)
				return
			}
		}
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
func GenerateIncastScriptsV1(cfg *config.Config) *ScriptResult {
	sCmdMap := make(map[string][]string) // map: serverHost -> []commands
	cCmdMap := make(map[string][]string) // map: clientHost -> []commands
	for _, sHost := range cfg.Server.Hostname {
		port := 20000
		for _, sHca := range cfg.Server.Hca {
			for _, cHost := range cfg.Client.Hostname {
				for _, cHca := range cfg.Client.Hca {
					serverCmd := NewIBWriteBWCommandBuilder(false).
						Host(sHost).
						Device(sHca).
						QueuePairNum(cfg.QpNum).
						MessageSize(cfg.MessageSizeBytes).
						Port(port).
						RunInfinitely(cfg.Run.Infinitely).
						DurationSeconds(cfg.Run.DurationSeconds).
						RdmaCm(cfg.RdmaCm).
						GidIndex(cfg.GidIndex).
						Report(cfg.Report.Enable).
						OutputFileName(fmt.Sprintf("%s/report_s_%s_%s_%d.json", cfg.Report.Dir, sHost, sHca, port)).
						SSHPrivateKey(cfg.SSH.PrivateKey).
						SleepTime("").
						ServerCommand() // 服务器命令不需要目标IP
					sCmdMap[sHost] = append(sCmdMap[sHost], serverCmd.String())

					clientCmd := NewIBWriteBWCommandBuilder(false).
						Host(cHost).
						Device(cHca).
						QueuePairNum(cfg.QpNum).
						MessageSize(cfg.MessageSizeBytes).
						Port(port).
						// TargetIP(strings.TrimSpace(string(hostIP))). // TODO
						TargetIP(sHost).
						RunInfinitely(cfg.Run.Infinitely).
						DurationSeconds(cfg.Run.DurationSeconds).
						RdmaCm(cfg.RdmaCm).
						GidIndex(cfg.GidIndex).
						Report(cfg.Report.Enable).
						OutputFileName(fmt.Sprintf("%s/report_c_%s_%s_%d.json", cfg.Report.Dir, cHost, cHca, port)).
						SSHPrivateKey(cfg.SSH.PrivateKey).
						SleepTime("").
						ClientCommand()
					cCmdMap[cHost] = append(cCmdMap[cHost], clientCmd.String())

					port++
				}
			}
		}
	}

	sScripts := make([]*HostScript, 0)
	for host, cmds := range sCmdMap {
		sCmds := []string{}
		for _, cmd := range cmds {
			// fmt.Println("\t", cmd)
			sCmds = append(sCmds, "( "+cmd+" )")
		}

		sScripts = append(sScripts, &HostScript{
			Host:    host,
			Command: strings.Join(sCmds, delimiter),
		})
	}

	cScripts := make([]*HostScript, 0)
	for host, cmds := range cCmdMap {
		cCmds := []string{}
		for _, cmd := range cmds {
			cCmds = append(cCmds, "( "+cmd+" )")
		}
		cScripts = append(cScripts, &HostScript{
			Host:    host,
			Command: strings.Join(cCmds, delimiter),
		})
	}

	return &ScriptResult{
		ServerScripts: sScripts,
		ClientScripts: cScripts,
	}
}

const delimiter = " && \\ \n"

type ScriptResult struct {
	ServerScripts []*HostScript
	ClientScripts []*HostScript
}

type HostScript struct {
	Host    string
	Command string
}
