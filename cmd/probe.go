package cmd

import (
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"

	"xnetperf/config"

	"github.com/spf13/cobra"
)

var (
	probeInterval int
	oneShot       bool
)

var probeCmd = &cobra.Command{
	Use:   "probe",
	Short: "Probe ib_write_bw processes status on remote hosts",
	Long: `Probe and monitor ib_write_bw processes on all configured hosts.
By default, probes every 5 seconds until all processes complete.
Can be configured to run once or with custom intervals.

Examples:
  # Monitor with default 5s interval until completion
  xnetperf probe

  # Check once and exit
  xnetperf probe --once

  # Monitor with 10s interval
  xnetperf probe --interval 10`,
	Run: runProbe,
}

func init() {
	probeCmd.Flags().IntVar(&probeInterval, "interval", 5, "Probe interval in seconds")
	probeCmd.Flags().BoolVar(&oneShot, "once", false, "Probe once and exit without waiting")
}

func runProbe(cmd *cobra.Command, args []string) {
	cfg, err := config.LoadConfig(cfgFile)
	if err != nil {
		fmt.Printf("Error reading config: %v\n", err)
		return
	}

	execProbeCommand(cfg)
}

func execProbeCommand(cfg *config.Config) {

	// 获取所有主机列表
	allHosts := make(map[string]bool)
	for _, host := range cfg.Server.Hostname {
		allHosts[host] = true
	}
	for _, host := range cfg.Client.Hostname {
		allHosts[host] = true
	}

	if len(allHosts) == 0 {
		fmt.Println("No hosts configured in config file")
		return
	}

	fmt.Printf("Probing ib_write_bw processes on %d hosts...\n", len(allHosts))
	fmt.Printf("Probe interval: %d seconds\n", probeInterval)
	if oneShot {
		fmt.Println("Mode: One-shot probe")
	} else {
		fmt.Println("Mode: Continuous monitoring until all processes complete")
	}
	fmt.Println()

	for {
		results := probeAllHosts(allHosts)
		displayProbeResults(results)

		// 如果是一次性探测，直接退出
		if oneShot {
			break
		}

		// 检查是否所有进程都已完成
		allCompleted := true
		for _, result := range results {
			if result.ProcessCount > 0 {
				allCompleted = false
				break
			}
		}

		if allCompleted {
			fmt.Println("✅ All ib_write_bw processes have completed!")
			break
		}

		// 等待下一次探测
		fmt.Printf("Waiting %d seconds for next probe...\n\n", probeInterval)
		time.Sleep(time.Duration(probeInterval) * time.Second)
	}
}

type ProbeResult struct {
	Hostname     string
	ProcessCount int
	Processes    []string
	Error        string
	Status       string
}

func probeAllHosts(hosts map[string]bool) []ProbeResult {
	var results []ProbeResult
	var mu sync.Mutex
	var wg sync.WaitGroup

	for hostname := range hosts {
		wg.Add(1)
		go func(host string) {
			defer wg.Done()
			result := probeHost(host)

			mu.Lock()
			results = append(results, result)
			mu.Unlock()
		}(hostname)
	}

	wg.Wait()
	return results
}

func probeHost(hostname string) ProbeResult {
	result := ProbeResult{
		Hostname: hostname,
	}

	// 使用SSH执行ps命令查找ib_write_bw进程
	cmd := exec.Command("ssh", hostname, "ps aux | grep ib_write_bw | grep -v grep")
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

func displayProbeResults(results []ProbeResult) {
	fmt.Printf("=== Probe Results (%s) ===\n", time.Now().Format("15:04:05"))

	// 计算每列的最大宽度
	maxHostnameWidth := len("Hostname")
	maxStatusWidth := len("Status")
	maxProcessCountWidth := len("Process Count")
	maxDetailsWidth := len("Details")

	for _, result := range results {
		if len(result.Hostname) > maxHostnameWidth {
			maxHostnameWidth = len(result.Hostname)
		}

		statusText := ""
		switch result.Status {
		case "RUNNING":
			statusText = "🟡 RUNNING"
		case "COMPLETED":
			statusText = "✅ COMPLETED"
		case "ERROR":
			statusText = "❌ ERROR"
		}

		if len(statusText) > maxStatusWidth {
			maxStatusWidth = len(statusText)
		}

		processCountText := fmt.Sprintf("%d", result.ProcessCount)
		if len(processCountText) > maxProcessCountWidth {
			maxProcessCountWidth = len(processCountText)
		}

		details := ""
		switch result.Status {
		case "RUNNING":
			if result.ProcessCount > 0 {
				details = fmt.Sprintf("%d process(es)", result.ProcessCount)
			}
		case "COMPLETED":
			details = "No processes"
		case "ERROR":
			details = "Connection failed"
		}

		if len(details) > maxDetailsWidth {
			maxDetailsWidth = len(details)
		}
	}

	// 确保最小宽度
	if maxHostnameWidth < 15 {
		maxHostnameWidth = 15
	}
	if maxStatusWidth < 12 {
		maxStatusWidth = 12
	}
	if maxProcessCountWidth < 8 {
		maxProcessCountWidth = 8
	}
	if maxDetailsWidth < 15 {
		maxDetailsWidth = 15
	}

	// 打印表格头部
	headerFormat := fmt.Sprintf("│ %%-%ds │ %%-%ds │ %%%ds │ %%-%ds │\n",
		maxHostnameWidth, maxStatusWidth, maxProcessCountWidth, maxDetailsWidth)
	separatorFormat := fmt.Sprintf("├─%%-%ds─┼─%%-%ds─┼─%%-%ds─┼─%%-%ds─┤\n",
		maxHostnameWidth, maxStatusWidth, maxProcessCountWidth, maxDetailsWidth)
	topBorder := fmt.Sprintf("┌─%s─┬─%s─┬─%s─┬─%s─┐\n",
		strings.Repeat("─", maxHostnameWidth),
		strings.Repeat("─", maxStatusWidth),
		strings.Repeat("─", maxProcessCountWidth),
		strings.Repeat("─", maxDetailsWidth))
	bottomBorder := fmt.Sprintf("└─%s─┴─%s─┴─%s─┴─%s─┘\n",
		strings.Repeat("─", maxHostnameWidth),
		strings.Repeat("─", maxStatusWidth),
		strings.Repeat("─", maxProcessCountWidth),
		strings.Repeat("─", maxDetailsWidth))

	fmt.Print(topBorder)
	fmt.Printf(headerFormat, "Hostname", "Status", "Process Count", "Details")
	fmt.Printf(separatorFormat,
		strings.Repeat("─", maxHostnameWidth),
		strings.Repeat("─", maxStatusWidth),
		strings.Repeat("─", maxProcessCountWidth),
		strings.Repeat("─", maxDetailsWidth))

	// 打印数据行
	dataFormat := fmt.Sprintf("│ %%-%ds │ %%-%ds │ %%%dd │ %%-%ds │\n",
		maxHostnameWidth, maxStatusWidth, maxProcessCountWidth, maxDetailsWidth)

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

		fmt.Printf(dataFormat, result.Hostname, statusIcon, result.ProcessCount, details)

		// 如果有错误，在下一行显示错误信息
		if result.Error != "" {
			errorFormat := fmt.Sprintf("│ %%-%ds │ %%-%ds │ %%%ds │ %%-%ds │\n",
				maxHostnameWidth, maxStatusWidth, maxProcessCountWidth, maxDetailsWidth)
			fmt.Printf(errorFormat, "", "Error:", "", result.Error)
		}
	}

	fmt.Print(bottomBorder)

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

	fmt.Printf("Summary: %d hosts running (%d processes), %d completed, %d errors\n",
		running, totalProcesses, completed, errors)
}
