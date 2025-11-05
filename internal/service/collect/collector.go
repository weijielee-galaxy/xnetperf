package collect

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"xnetperf/config"
	"xnetperf/pkg/tools"
)

type Collector struct {
	cfg    *config.Config
	logger *slog.Logger
}

func New(cfg *config.Config) *Collector {
	return &Collector{
		cfg:    cfg,
		logger: slog.Default().With("module", "COLLECT"),
	}
}

func (c *Collector) DoCollect(cleanupRemote bool) error {
	c.logger.Info("Starting collection of report files", "cleanup_remote", cleanupRemote)
	// 创建本地reports目录
	reportsDir := "reports"

	// Remove existing reports directory if it exists
	if _, err := os.Stat(reportsDir); err == nil {
		err = os.RemoveAll(reportsDir)
		if err != nil {
			fmt.Printf("Error removing existing reports directory: %v\n", err)
			return err
		}
		fmt.Printf("Removed existing reports directory\n")
	}

	// Create new reports directory
	err := os.MkdirAll(reportsDir, 0755)
	if err != nil {
		fmt.Printf("Error creating reports directory: %v\n", err)
		return err
	}

	// 获取所有主机列表
	allHosts := make(map[string]bool)
	for _, host := range c.cfg.Server.Hostname {
		allHosts[host] = true
	}
	for _, host := range c.cfg.Client.Hostname {
		allHosts[host] = true
	}

	var wg sync.WaitGroup
	fmt.Printf("Collecting reports from %d hosts...\n", len(allHosts))

	for hostname := range allHosts {
		wg.Add(1)
		go func(host string) {
			defer wg.Done()
			collectFromHost(host, c.cfg.Report.Dir, reportsDir, c.cfg.SSH.PrivateKey, c.cfg.SSH.User, cleanupRemote)
		}(hostname)
	}

	wg.Wait()
	fmt.Printf("Report collection completed. Files saved to '%s' directory.\n", reportsDir)
	c.logger.Info("Collection process completed successfully")
	return nil
}

func collectFromHost(hostname, remoteDir, localBaseDir, sshKeyPath, user string, cleanupRemote bool) int {
	// 为每个主机创建本地子目录
	hostDir := filepath.Join(localBaseDir, hostname)
	err := os.MkdirAll(hostDir, 0755)
	if err != nil {
		fmt.Printf("Error creating directory for host %s: %v\n", hostname, err)
		return 0
	}

	fmt.Printf("-> Collecting reports from %s...\n", hostname)

	// 使用scp收集属于当前主机的JSON报告文件（按主机名匹配）
	// scp hostname:remoteDir/*hostname*.json localDir/
	scpCmd := fmt.Sprintf("%s/*%s*.json", remoteDir, hostname)
	var tmpHost string
	if user != "" && !strings.Contains(hostname, "@") {
		tmpHost = fmt.Sprintf("%s@%s", user, hostname)
	}
	cmd := exec.Command("scp", fmt.Sprintf("%s:%s", tmpHost, scpCmd), hostDir+"/")
	if sshKeyPath != "" {
		cmd = exec.Command("scp", "-i", sshKeyPath, fmt.Sprintf("%s:%s", tmpHost, scpCmd), hostDir+"/")
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		// 检查是否是因为没有匹配的文件
		if string(output) != "" {
			fmt.Printf("   [WARNING] ⚠️  %s: %s\n", hostname, string(output))
		} else {
			fmt.Printf("   [WARNING] ⚠️  %s: No report files found or scp failed: %v\n", hostname, err)
		}
		return 0
	}

	// 计算收集到的文件数量
	files, err := filepath.Glob(filepath.Join(hostDir, "*.json"))
	if err != nil {
		fmt.Printf("   [ERROR] ❌ %s: Error counting files: %v\n", hostname, err)
		return 0
	}

	if len(files) > 0 {
		fmt.Printf("   [SUCCESS] ✅ %s: Collected %d report files\n", hostname, len(files))

		// 仅在启用cleanup标志时清理远程主机上的报告文件
		if cleanupRemote {
			cleanupRemoteFiles(hostname, remoteDir, sshKeyPath, user)
		}
	} else {
		fmt.Printf("   [INFO] ℹ️  %s: No report files found\n", hostname)
	}

	return len(files)
}

func cleanupRemoteFiles(hostname, remoteDir, sshKeyPath, user string) {
	fmt.Printf("   [CLEANUP] 🧹 %s: Cleaning up remote report files...\n", hostname)

	// 首先检查远程目录中是否还有属于当前主机的JSON文件
	checkCmd := fmt.Sprintf("ls %s/*%s*.json 2>/dev/null | wc -l", remoteDir, hostname)
	checkExec := tools.BuildSSHCommand(hostname, checkCmd, sshKeyPath, user)

	checkOutput, err := checkExec.CombinedOutput()
	if err != nil {
		fmt.Printf("   [WARNING] ⚠️  %s: Failed to check remote files: %v\n", hostname, err)
		return
	}

	// 如果没有文件需要清理，则跳过
	if string(checkOutput) == "0\n" {
		fmt.Printf("   [CLEANUP] ℹ️  %s: No remote files to cleanup\n", hostname)
		return
	}

	// 使用SSH删除远程主机上属于当前主机的JSON报告文件（安全匹配）
	rmCmd := fmt.Sprintf("rm -f %s/*%s*.json", remoteDir, hostname)
	cmd := tools.BuildSSHCommand(hostname, rmCmd, sshKeyPath, user)

	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("   [WARNING] ⚠️  %s: Failed to cleanup remote files: %v\n", hostname, err)
		if len(output) > 0 {
			fmt.Printf("   [WARNING] ⚠️  %s: SSH output: %s\n", hostname, string(output))
		}
		return
	}

	// 验证清理是否成功
	verifyCmd := fmt.Sprintf("ls %s/*%s*.json 2>/dev/null | wc -l", remoteDir, hostname)
	verifyExec := tools.BuildSSHCommand(hostname, verifyCmd, sshKeyPath, user)

	verifyOutput, err := verifyExec.CombinedOutput()
	if err == nil && string(verifyOutput) == "0\n" {
		fmt.Printf("   [CLEANUP] ✅ %s: Remote files cleaned up successfully\n", hostname)
	} else {
		fmt.Printf("   [WARNING] ⚠️  %s: Cleanup verification failed\n", hostname)
	}
}

// CollectResult 收集结果
type CollectResult struct {
	Success        bool           `json:"success"`
	Message        string         `json:"message"`
	CollectedFiles map[string]int `json:"collected_files"` // hostname -> file count
	Error          string         `json:"error,omitempty"`
}

func (c *Collector) CollectAndGetResult(cfg *config.Config) (*CollectResult, error) {
	result := &CollectResult{
		CollectedFiles: make(map[string]int),
	}

	if !cfg.Report.Enable {
		result.Success = false
		result.Error = "Report is not enabled in config"
		return result, fmt.Errorf("report is not enabled in config")
	}

	// 创建本地reports目录
	reportsDir := "reports"

	// 删除已存在的reports目录
	if _, err := os.Stat(reportsDir); err == nil {
		err = os.RemoveAll(reportsDir)
		if err != nil {
			result.Success = false
			result.Error = fmt.Sprintf("Failed to remove existing reports directory: %v", err)
			return result, err
		}
		fmt.Printf("Removed existing reports directory\n")
	}

	// 创建新的reports目录
	err := os.MkdirAll(reportsDir, 0755)
	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("Failed to create reports directory: %v", err)
		return result, err
	}

	// 获取所有主机列表
	allHosts := make(map[string]bool)
	for _, host := range cfg.Server.Hostname {
		allHosts[host] = true
	}
	for _, host := range cfg.Client.Hostname {
		allHosts[host] = true
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	fmt.Printf("Collecting reports from %d hosts...\n", len(allHosts))

	for hostname := range allHosts {
		wg.Add(1)
		go func(host string) {
			defer wg.Done()
			count := collectFromHost(host, cfg.Report.Dir, reportsDir, cfg.SSH.PrivateKey, cfg.SSH.User, true)
			mu.Lock()
			result.CollectedFiles[host] = count
			mu.Unlock()
		}(hostname)
	}

	wg.Wait()

	result.Success = true
	result.Message = fmt.Sprintf("Report collection completed from %d hosts", len(allHosts))
	fmt.Printf("Report collection completed. Files saved to '%s' directory.\n", reportsDir)

	return result, nil
}
