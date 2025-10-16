package cmd

import (
	"fmt"
	"os"
	"sync"
	"xnetperf/config"
	"xnetperf/stream"

	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run network test",
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := config.LoadConfig(cfgFile)
		if err != nil {
			fmt.Printf("Error reading config: %v\n", err)
			os.Exit(1)
		}
		execRunCommand(cfg)
	},
}

func execRunCommand(cfg *config.Config) {
	// 在运行测试前先进行网卡状态检查
	fmt.Println("🔍 Performing network card precheck before starting tests...")
	success := execPrecheckCommand(cfg)
	if !success {
		fmt.Printf("❌ Precheck failed! Network cards are not ready. Please fix the issues before running tests.\n")
		os.Exit(1)
	}
	fmt.Println("✅ Precheck passed! All network cards are healthy. Proceeding with tests...")

	// 在运行测试前清理远程主机上的旧JSON报告文件
	if cfg.Report.Enable {
		cleanupRemoteReportFiles(cfg)
	}

	switch cfg.StreamType {
	case config.FullMesh:
		stream.GenerateFullMeshScript(cfg)
	case config.InCast:
		stream.GenerateIncastScripts(cfg)
	case config.P2P:
		err := stream.GenerateP2PScripts(cfg)
		if err != nil {
			fmt.Printf("❌ Error generating P2P scripts: %v\n", err)
			os.Exit(1)
		}
	default:
		fmt.Printf("Invalid stream_type '%s' in config. Supported types: fullmesh, incast, p2p\n", cfg.StreamType)
		os.Exit(1)
	}
	stream.DistributeAndRunScripts(cfg)
}

func cleanupRemoteReportFiles(cfg *config.Config) {
	fmt.Println("Cleaning up old report files on remote hosts before starting tests...")

	// 获取所有主机列表
	allHosts := make(map[string]bool)
	for _, host := range cfg.Server.Hostname {
		allHosts[host] = true
	}
	for _, host := range cfg.Client.Hostname {
		allHosts[host] = true
	}

	var wg sync.WaitGroup

	for hostname := range allHosts {
		wg.Add(1)
		go func(host string) {
			defer wg.Done()

			// 删除远程主机上属于当前主机的JSON报告文件（按主机名安全匹配）
			rmCmd := fmt.Sprintf("rm -f %s/*%s*.json", cfg.Report.Dir, host)
			cmd := buildSSHCommand(host, rmCmd, cfg.SSH.PrivateKey)

			output, err := cmd.CombinedOutput()
			if err != nil {
				fmt.Printf("   [WARNING] ⚠️  %s: Failed to cleanup old reports: %v\n", host, err)
				if len(output) > 0 {
					fmt.Printf("   [WARNING] ⚠️  %s: SSH output: %s\n", host, string(output))
				}
			} else {
				fmt.Printf("   [CLEANUP] 🧹 %s: Old report files cleaned\n", host)
			}
		}(hostname)
	}

	wg.Wait()
	fmt.Println()
}
