package v0

import (
	"fmt"
	"os"
	"sync"
	"xnetperf/config"
	"xnetperf/pkg/tools"
	"xnetperf/stream"
)

// ExecRunCommand executes the run command with precheck and script generation
// Deprecated: This function is part of the v0 internal package and is retained for backward compatibility.
func ExecRunCommand(cfg *config.Config) {
	// 在运行测试前先进行网卡状态检查
	fmt.Println("🔍 Performing network card precheck before starting tests...")
	success := ExecPrecheckCommand(cfg)
	if !success {
		fmt.Printf("❌ Precheck failed! Network cards are not ready. Please fix the issues before running tests.\n")
		os.Exit(1)
	}
	fmt.Println("✅ Precheck passed! All network cards are healthy. Proceeding with tests...")

	// 在运行测试前清理远程主机上的旧JSON报告文件
	if cfg.Report.Enable {
		cleanupRemoteReportFiles(cfg)
	}

	err := stream.GenerateScripts(cfg)
	if err != nil {
		fmt.Printf("❌ Failed to generate scripts: %v\n", err)
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
			cmd := tools.BuildSSHCommand(host, rmCmd, cfg.SSH.PrivateKey, cfg.SSH.User)

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
