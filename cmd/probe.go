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

	// 获取测试持续时间和开始时间
	testDuration := cfg.Run.DurationSeconds
	testStartTime := time.Now()
	testEndTime := testStartTime.Add(time.Duration(testDuration) * time.Second)

	fmt.Printf("Probing ib_write_bw processes on %d hosts...\n", len(allHosts))
	fmt.Printf("Probe interval: %d seconds\n", probeInterval)
	if testDuration > 0 {
		fmt.Printf("Test duration: %d seconds (estimated completion: %s)\n", testDuration, testEndTime.Format("15:04:05"))
	}
	if oneShot {
		fmt.Println("Mode: One-shot probe")
	} else {
		fmt.Println("Mode: Continuous monitoring until all processes complete")
	}
	fmt.Println()

	isFirstProbe := true

	for {
		results := probeAllHosts(allHosts)

		// 计算倒计时
		remainingTime := testEndTime.Sub(time.Now())

		// 如果不是第一次探测，清除之前的输出
		if !isFirstProbe && !oneShot {
			clearPreviousOutput()
		}

		displayProbeResults(results, remainingTime, testDuration)
		isFirstProbe = false

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

		// 检查是否已经超过测试时间
		if testDuration > 0 && remainingTime <= 0 {
			fmt.Println("⏰ Test duration completed!")
			break
		}

		// 等待下一次探测
		nextProbeIn := probeInterval
		if testDuration > 0 && remainingTime.Seconds() < float64(probeInterval) && remainingTime.Seconds() > 0 {
			nextProbeIn = int(remainingTime.Seconds()) + 1
		}

		if !oneShot {
			fmt.Printf("Next probe in %d seconds...\n", nextProbeIn)
		}
		time.Sleep(time.Duration(nextProbeIn) * time.Second)
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

// clearPreviousOutput clears the previous output using ANSI escape sequences
func clearPreviousOutput() {
	// 移动光标到上方并清除内容
	// 这里估算需要清除大约20行的内容（表格+摘要信息）
	fmt.Print("\033[20A") // 向上移动20行
	fmt.Print("\033[J")   // 清除从光标到屏幕结尾的内容
}

func displayProbeResults(results []ProbeResult, remainingTime time.Duration, testDuration int) {
	// 显示当前时间和倒计时信息
	currentTime := time.Now().Format("15:04:05")
	fmt.Printf("=== Probe Results (%s) ===\n", currentTime)

	if testDuration > 0 {
		if remainingTime > 0 {
			fmt.Printf("⏰ Time remaining: %02d:%02d (%.0f seconds)\n",
				int(remainingTime.Minutes()), int(remainingTime.Seconds())%60, remainingTime.Seconds())
		} else {
			fmt.Printf("⏰ Test duration completed!\n")
		}
	}

	// 增加列宽：Status和Details列各增加1/3
	// 原来Status是11个字符，Details是11个字符
	// 增加1/3后：Status约15个字符，Details约15个字符
	fmt.Println("┌─────────────────────┬─────────────────┬──────────────┬─────────────────┐")
	fmt.Println("│ Hostname            │ Status          │ Process Count│ Details         │")
	fmt.Println("├─────────────────────┼─────────────────┼──────────────┼─────────────────┤")

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

		fmt.Printf("│ %-19s │ %-15s │ %12d │ %-15s │\n",
			result.Hostname, statusIcon, result.ProcessCount, details)

		// 如果有错误，在下一行显示错误信息
		if result.Error != "" {
			errorMsg := result.Error
			// 如果错误信息太长，截断它
			if len(errorMsg) > 15 {
				errorMsg = errorMsg[:12] + "..."
			}
			fmt.Printf("│ %-19s │ %-15s │ %12s │ %-15s │\n",
				"", "Error:", "", errorMsg)
		}
	}

	fmt.Println("└─────────────────────┴─────────────────┴──────────────┴─────────────────┘")

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
