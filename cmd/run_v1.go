package cmd

import (
	"fmt"
	"os"
	"xnetperf/config"
	"xnetperf/stream"
)

func execRunCommandV1(cfg *config.Config) {
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

	ret, err := stream.GenerateScriptsV1(cfg)
	if err != nil {
		fmt.Printf("❌ Failed to generate scripts: %v\n", err)
		os.Exit(1)
	}
	stream.DistributeAndRunScriptsV1(ret, cfg)
}
