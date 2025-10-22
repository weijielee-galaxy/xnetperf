package stream

import (
	"fmt"
	"os"
	"strings"
	"xnetperf/config"
)

// 假设所有的服务器都有同样的HCA数目
// 只从配置文件中获取server的hostname和HCA，生成localtest脚本
func GenerateLocaltestScript(cfg *config.Config) {
	// 清空streamScript文件夹内容
	ClearStreamScriptDir(cfg)
	// 生成localtest脚本
	// 双份server
	totalPort := len(cfg.Server.Hostname) * len(cfg.Server.Hca) * len(cfg.Server.Hostname) * len(cfg.Server.Hca) * 2
	fmt.Println("Total Ports Needed:", totalPort)
	if totalPort > (65535 - cfg.StartPort + 1) {
		fmt.Printf("Error: Not enough ports available. Required: %d, Available: %d\n", totalPort, 65535-cfg.StartPort+1)
		return
	}
	srvHosts := cfg.Server.Hostname
	cliHosts := cfg.Server.Hostname // 本地测试，client和server使用同一组hostname, 复制一份server当做client

	srvHcas := cfg.Server.Hca
	cliHcas := cfg.Server.Hca // 本地测试，client和server使用同一组HCA, 复制一份server当做client

	allHosts := append(srvHosts, cliHosts...)

	num := 1
	// port := cfg.StartPort

	for _, Server := range allHosts {
		port := cfg.StartPort
		// 3. Run the command and capture the combined output (stdout and stderr).
		output, err := getHostIP(Server, cfg.SSH.PrivateKey, cfg.NetworkInterface)
		if err != nil {
			// If command fails, return the output for debugging and a detailed error.
			fmt.Printf("Error executing command on %s: %v\nOutput: %s\n", Server, err, string(output))
		}
		fmt.Println("Output from", Server, ":", string(output))
		for _, hcaServer := range cliHcas {

			serverScriptContent := strings.Builder{}
			clientScriptContent := strings.Builder{}

			serverScriptFileName := fmt.Sprintf("%s/%s_%s_server_localtest.sh", cfg.OutputDir(), Server, hcaServer)
			// serverScriptContent := "#!/bin/bash\n"

			clientScriptFileName := fmt.Sprintf("%s/%s_%s_client_localtest.sh", cfg.OutputDir(), Server, hcaServer)
			// clientScriptContent := "#!/bin/bash\n"

			fmt.Println("Generating scripts for Server:", Server, "Client HCA:", hcaServer)
			for _, allHost := range allHosts {
				// if allHost == Server {
				// 	continue
				// }
				for _, hcaClient := range srvHcas {
					fmt.Println("num:", num, "Server HCA:", Server, "Server HCA:", hcaClient, port)
					fmt.Println("num:", num, "client HCA:", allHost, "Client HCA:", hcaClient, port, Server)

					// 使用命令构建器创建服务器命令
					serverCmd := NewIBWriteBWCommandBuilder(true).
						Host(Server).
						Device(hcaServer).
						MessageSize(cfg.MessageSizeBytes).
						Port(port).
						QueuePairNum(cfg.QpNum).
						RunInfinitely(cfg.Run.Infinitely).
						DurationSeconds(cfg.Run.DurationSeconds).
						RdmaCm(cfg.RdmaCm).
						GidIndex(cfg.GidIndex).
						Report(cfg.Report.Enable).
						OutputFileName(fmt.Sprintf("%s/report_s_%s_%s_%d.json", cfg.Report.Dir, Server, hcaServer, port)).
						SSHPrivateKey(cfg.SSH.PrivateKey).
						ServerCommand() // 服务器命令不需要目标IP

					// 使用命令构建器创建客户端命令
					clientCmd := NewIBWriteBWCommandBuilder(true).
						Host(allHost).
						Device(hcaClient).
						MessageSize(cfg.MessageSizeBytes).
						Port(port).
						QueuePairNum(cfg.QpNum).
						TargetIP(strings.TrimSpace(string(output))).
						RunInfinitely(cfg.Run.Infinitely).
						DurationSeconds(cfg.Run.DurationSeconds).
						RdmaCm(cfg.RdmaCm).
						GidIndex(cfg.GidIndex).
						Report(cfg.Report.Enable).
						OutputFileName(fmt.Sprintf("%s/report_c_%s_%s_%d.json", cfg.Report.Dir, allHost, hcaClient, port)).
						SSHPrivateKey(cfg.SSH.PrivateKey).
						ClientCommand() // 客户端命令有更长的睡眠时间

					serverScriptContent.WriteString(serverCmd.String() + "\n")
					clientScriptContent.WriteString(clientCmd.String() + "\n")
					port++
					num++
				}
			}
			// Write the complete scriptContent to the file after the loops
			err := os.WriteFile(serverScriptFileName, []byte(serverScriptContent.String()), 0755)
			if err != nil {
				fmt.Printf("Error writing script file '%s': %v\n", serverScriptFileName, err)
				return
			}

			// Write the complete scriptContent to the file after the loops
			err = os.WriteFile(clientScriptFileName, []byte(clientScriptContent.String()), 0755)
			if err != nil {
				fmt.Printf("Error writing script file '%s': %v\n", serverScriptFileName, err)
				return
			}
		}
	}
}
