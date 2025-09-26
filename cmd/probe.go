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
	var lastResults []ProbeResult

	for {
		results := probeAllHosts(allHosts)
		lastResults = results

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

		// 使用每秒刷新的倒计时
		nextProbeIn := probeInterval
		if testDuration > 0 && remainingTime.Seconds() < float64(probeInterval) && remainingTime.Seconds() > 0 {
			nextProbeIn = int(remainingTime.Seconds()) + 1
		}

		// 每秒刷新倒计时和动态效果
		for i := nextProbeIn; i > 0; i-- {
			if i < nextProbeIn {
				clearPreviousOutput()
				displayProbeResults(lastResults, testEndTime.Sub(time.Now()), testDuration)
				fmt.Printf("Next probe in %d seconds...\n", i)
			} else {
				fmt.Printf("Next probe in %d seconds...\n", i)
			}

			// 在1秒内多次更新动态效果 (每200ms更新一次)
			for j := 0; j < 5; j++ {
				time.Sleep(200 * time.Millisecond)
				if i < nextProbeIn && j < 4 { // 避免在最后一次时重复清除
					clearPreviousOutput()
					displayProbeResults(lastResults, testEndTime.Sub(time.Now()), testDuration)
					fmt.Printf("Next probe in %d seconds...\n", i)
				}
			}

			// 检查是否提前完成
			if testDuration > 0 && testEndTime.Sub(time.Now()) <= 0 {
				break
			}
		}
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
	// 精确计算需要清除的行数：
	// - Probe Results标题行: 1行
	// - 倒计时行: 1行 (如果有)
	// - 表格头部: 3行 (顶部边框、标题行、分隔符)
	// - 表格内容: 最多每个主机1行 (假设最多10行)
	// - 表格底部: 1行
	// - Summary行: 1行
	// - Next probe行: 1行
	// 总计约18行，为了安全起见使用20行

	// 使用更保守的清除方式，避免过度清除
	fmt.Print("\033[2K") // 清除当前行
	fmt.Print("\033[1A") // 向上1行
	for i := 0; i < 15; i++ {
		fmt.Print("\033[2K") // 清除整行
		fmt.Print("\033[1A") // 向上移动1行
	}
	fmt.Print("\033[2K") // 清除最后一行
}

func displayProbeResults(results []ProbeResult, remainingTime time.Duration, testDuration int) {
	// 显示当前时间和倒计时信息
	currentTime := time.Now().Format("15:04:05")
	fmt.Printf("=== Probe Results (%s) ===\n", currentTime)

	if testDuration > 0 {
		if remainingTime > 0 {
			fmt.Printf("Time remaining: %02d:%02d (%.0f seconds)\n",
				int(remainingTime.Minutes()), int(remainingTime.Seconds())%60, remainingTime.Seconds())
		} else {
			fmt.Printf("Test duration completed!\n")
		}
	}

	// 使用纯ASCII字符避免emoji对齐问题
	fmt.Println("┌─────────────────────┬─────────────┬──────────────┬─────────────────┬──────────┐")
	fmt.Println("│ Hostname            │ Status      │ Process Count│ Details         │ Activity │")
	fmt.Println("├─────────────────────┼─────────────┼──────────────┼─────────────────┼──────────┤")

	// 动态效果字符数组 - 更快的刷新频率
	activityChars := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	// 使用毫秒级别的时间戳来获得更快的动画效果
	activityIndex := int(time.Now().UnixNano()/100000000) % len(activityChars)

	for _, result := range results {
		details := ""
		statusText := ""
		activity := " "

		switch result.Status {
		case "RUNNING":
			statusText = "RUNNING"
			activity = activityChars[activityIndex]
			if result.ProcessCount > 0 {
				details = fmt.Sprintf("%d process(es)", result.ProcessCount)
			}
		case "COMPLETED":
			statusText = "COMPLETED"
			activity = "✓"
			details = "No processes"
		case "ERROR":
			statusText = "ERROR"
			activity = "✗"
			details = "Connection failed"
		}

		fmt.Printf("│ %-19s │ %-11s │ %12d │ %-15s │ %-8s │\n",
			result.Hostname, statusText, result.ProcessCount, details, activity)

		// 如果有错误，在下一行显示错误信息
		if result.Error != "" {
			errorMsg := result.Error
			// 如果错误信息太长，截断它
			if len(errorMsg) > 15 {
				errorMsg = errorMsg[:12] + "..."
			}
			fmt.Printf("│ %-19s │ %-11s │ %12s │ %-15s │ %-8s │\n",
				"", "Error:", "", errorMsg, "")
		}
	}

	fmt.Println("└─────────────────────┴─────────────┴──────────────┴─────────────────┴──────────┘")

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
