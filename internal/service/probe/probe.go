package probe

import (
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"
	"xnetperf/config"
	"xnetperf/pkg/tools"
)

// ProbeResult 探测结果
type ProbeResult struct {
	Hostname     string   `json:"hostname"`
	ProcessCount int      `json:"process_count"`
	Processes    []string `json:"processes,omitempty"`
	Error        string   `json:"error,omitempty"`
	Status       string   `json:"status"` // RUNNING, COMPLETED, ERROR
}

type Prober struct {
	cfg    *config.Config
	logger *slog.Logger
}

func New(cfg *config.Config) *Prober {
	return &Prober{
		cfg:    cfg,
		logger: slog.Default().With("module", "PROBE"),
	}
}

func (p *Prober) DoProbeWait(probeInterval int) {
	for {
		results, err := p.DoProbe()
		if err != nil {
			p.logger.Error("Probe operation failed", "error", err)
			return
		}

		p.Display(results)

		// 检查是否所有主机的进程都已完成
		allCompleted := true
		for _, result := range results {
			if result.Status == "RUNNING" {
				allCompleted = false
				break
			}
		}

		if allCompleted {
			p.logger.Info("All ib_write_bw processes have completed")
			fmt.Println("✅ All ib_write_bw processes have completed!")
			break
		}

		// 等待下一次探测
		p.logger.Info("Waiting for next probe", "interval_seconds", probeInterval)
		fmt.Printf("Waiting %d seconds for next probe...\n\n", probeInterval)
		time.Sleep(time.Duration(probeInterval) * time.Second)
	}
}

func (p *Prober) DoProbe() ([]ProbeResult, error) {
	p.logger.Info("Starting probe operation")

	// 获取所有主机列表
	allHosts := make(map[string]bool)
	for _, host := range p.cfg.Server.Hostname {
		allHosts[host] = true
	}
	for _, host := range p.cfg.Client.Hostname {
		allHosts[host] = true
	}

	if len(allHosts) == 0 {
		p.logger.Warn("No hosts configured in config file")
		return nil, fmt.Errorf("No hosts found in configuration")
	}

	ret := p.probeAllHosts(allHosts, p.cfg.SSH.PrivateKey, p.cfg.SSH.User)

	p.logger.Info("Probe operation completed successfully")
	return ret, nil
}

func (p *Prober) probeAllHosts(hosts map[string]bool, sshKeyPath, user string) []ProbeResult {
	var results []ProbeResult
	var mu sync.Mutex
	var wg sync.WaitGroup

	for hostname := range hosts {
		wg.Add(1)
		go func(host string) {
			defer wg.Done()
			result := p.probeHost(host, sshKeyPath, user)

			mu.Lock()
			results = append(results, result)
			mu.Unlock()
		}(hostname)
	}

	wg.Wait()
	return results
}

func (p *Prober) probeHost(hostname, sshKeyPath, user string) ProbeResult {
	result := ProbeResult{
		Hostname: hostname,
	}

	// 使用SSH执行ps命令查找ib_write_bw进程
	cmd := tools.BuildSSHCommand(hostname, "ps aux | grep ib_write_bw | grep -v grep", sshKeyPath, user)
	output, err := cmd.CombinedOutput()

	if err != nil {
		// 如果没有找到进程或SSH连接失败
		if strings.Contains(string(output), "") && cmd.ProcessState != nil && cmd.ProcessState.ExitCode() == 1 {
			// ps命令返回1通常表示没有找到匹配的进程
			result.ProcessCount = 0
			result.Status = "COMPLETED"
		} else {
			result.Error = fmt.Sprintf("SSH error: %v", err)
			result.Status = "ERROR"
		}
		return result
	}

	// 解析输出
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) == 1 && lines[0] == "" {
		result.ProcessCount = 0
		result.Status = "COMPLETED"
		return result
	}

	// 过滤和计数有效的进程行
	var processes []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && strings.Contains(line, "ib_write_bw") {
			processes = append(processes, line)
		}
	}

	result.ProcessCount = len(processes)
	result.Processes = processes

	if result.ProcessCount > 0 {
		result.Status = "RUNNING"
	} else {
		result.Status = "COMPLETED"
	}

	return result
}

func (p *Prober) Display(results []ProbeResult) {
	fmt.Printf("=== Probe Results (%s) ===\n", time.Now().Format("15:04:05"))
	fmt.Println("┌─────────────────────┬───────────────┬──────────────┬─────────────────┐")
	fmt.Println("│ Hostname            │ Status        │ Process Count│ Details         │")
	fmt.Println("├─────────────────────┼───────────────┼──────────────┼─────────────────┤")

	for _, result := range results {
		details := ""
		statusIcon := ""

		switch result.Status {
		case "RUNNING":
			statusIcon = "🟡 RUNNING"
			if result.ProcessCount > 0 {
				details = fmt.Sprintf("%d process(es)", result.ProcessCount)
			}
		case "COMPLETED":
			statusIcon = "✅ COMPLETED"
			details = "No processes"
		case "ERROR":
			statusIcon = "❌ ERROR"
			details = "Connection failed"
		}

		fmt.Printf("│ %-19s │ %-12s │ %12d │ %-15s │\n",
			result.Hostname, statusIcon, result.ProcessCount, details)

		// 如果有错误，在下一行显示错误信息
		if result.Error != "" {
			fmt.Printf("│ %-19s │ %-12s │ %12s │ %-15s │\n",
				"", "Error:", "", result.Error)
		}
	}

	fmt.Println("└─────────────────────┴───────────────┴──────────────┴─────────────────┘")

	// 显示总结信息
	running := 0
	completed := 0
	errors := 0
	totalProcesses := 0

	for _, result := range results {
		switch result.Status {
		case "RUNNING":
			running++
			totalProcesses += result.ProcessCount
		case "COMPLETED":
			completed++
		case "ERROR":
			errors++
		}
	}

	fmt.Printf("\nSummary: %d hosts running (%d processes), %d completed, %d errors\n",
		running, totalProcesses, completed, errors)
}
