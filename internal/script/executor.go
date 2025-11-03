package script

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"xnetperf/config"
	"xnetperf/internal/script/generator"
	"xnetperf/internal/tools"

	"golang.org/x/sync/errgroup"
)

// TestMode 定义测试模式常量
type TestMode string

const (
	ModeBwFullmesh   TestMode = "bw_fullmesh"
	ModeBwIncast     TestMode = "bw_incast"
	ModeBwP2P        TestMode = "bw_p2p"
	ModeBwLocaltest  TestMode = "bw_localtest"
	ModeLatFullmesh  TestMode = "lat_fullmesh"
	ModeLatIncast    TestMode = "lat_incast"
	ModeLatP2P       TestMode = "lat_p2p"
	ModeLatLocaltest TestMode = "lat_localtest"
)

type TestType string

const (
	TestTypeBandwidth TestType = "bandwidth"
	TestTypeLatency   TestType = "latency"
)

func (testType TestType) String() string {
	return string(testType)
}

func (testType TestType) Command() string {
	switch testType {
	case TestTypeBandwidth:
		return "ib_write_bw"
	case TestTypeLatency:
		return "ib_write_lat"
	default:
		return ""
	}
}

// Executor 负责脚本的生成和执行编排
type Executor struct {
	cfg      *config.Config
	mode     TestMode
	TestType TestType
	timeout  time.Duration // TODO
}

// NewExecutor 创建执行器
func NewExecutor(cfg *config.Config, testType TestType) *Executor {
	// 基本验证
	if cfg == nil {
		return nil
	}
	mode := parseTestMode(testType, cfg)
	if !IsValidMode(mode) {
		return nil
	}

	return &Executor{
		cfg:      cfg,
		mode:     mode,
		TestType: testType,
		timeout:  10 * time.Minute,
	}
}

func parseTestMode(testType TestType, cfg *config.Config) TestMode {
	switch testType {
	case TestTypeBandwidth:
		switch cfg.StreamType {
		case config.FullMesh:
			return ModeBwFullmesh
		case config.InCast:
			return ModeBwIncast
		case config.P2P:
			return ModeBwP2P
		case config.LocalTest:
			return ModeBwLocaltest
		}
	case TestTypeLatency:
		switch cfg.StreamType {
		case config.FullMesh:
			return ModeLatFullmesh
		case config.InCast:
			return ModeLatIncast
		case config.P2P:
			return ModeLatP2P
		case config.LocalTest:
			return ModeLatLocaltest
		}
	}
	return ""
}

// GenerateScripts 生成脚本
func (e *Executor) GenerateScripts() (*generator.ScriptResult, error) {
	// 1. 查询IP地址
	hostIPs, err := e.lookupIPs()
	if err != nil {
		return nil, fmt.Errorf("failed to lookup IPs: %v", err)
	}

	// 2. 根据模式创建对应的generator
	var result *generator.ScriptResult
	switch e.mode {
	case ModeBwFullmesh:
		gen := generator.NewBwFullmeshScriptGenerator(e.cfg, hostIPs)
		result, err = gen.GenerateScripts()
	case ModeBwIncast:
		gen := generator.NewBwIncastScriptGenerator(e.cfg, hostIPs)
		result, err = gen.GenerateScripts()
	case ModeBwP2P:
		gen := generator.NewBwP2PScriptGenerator(e.cfg, hostIPs)
		result, err = gen.GenerateScripts()
	case ModeBwLocaltest:
		gen := generator.NewBwLocaltestScriptGenerator(e.cfg, hostIPs)
		result, err = gen.GenerateScripts()
	case ModeLatFullmesh:
		gen := generator.NewLatFullmeshScriptGenerator(e.cfg, hostIPs)
		result, err = gen.GenerateScripts()
	case ModeLatIncast:
		gen := generator.NewLatIncastScriptGenerator(e.cfg, hostIPs)
		result, err = gen.GenerateScripts()
	case ModeLatP2P:
		// TODO: 实现lat p2p generator
		return nil, fmt.Errorf("lat_p2p mode not implemented yet")
	case ModeLatLocaltest:
		// TODO: 实现lat localtest generator
		return nil, fmt.Errorf("lat_localtest mode not implemented yet")
	default:
		return nil, fmt.Errorf("unknown mode: %s", e.mode)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to generate scripts: %v", err)
	}

	return result, nil
}

// Execute 生成并执行脚本
func (e *Executor) Execute() error {
	// 1. 生成脚本
	result, err := e.GenerateScripts()
	if err != nil {
		return err
	}

	// 2. 执行服务端脚本
	fmt.Println("Starting server processes...")
	var eg errgroup.Group
	for _, script := range result.ServerScripts {
		script := script // capture loop variable
		eg.Go(func() error {
			if err := e.executeRemote(script); err != nil {
				return fmt.Errorf("failed to execute server script on %s: %v", script.Host, err)
			}
			return nil
		})
	}
	if err := eg.Wait(); err != nil {
		return err
	}

	// 3. 等待服务端启动
	e.waitingForServerStart(result.ServerScripts)

	// 4. 执行客户端脚本
	fmt.Println("Starting client processes...")
	var clientEg errgroup.Group
	for _, script := range result.ClientScripts {
		script := script // capture loop variable
		clientEg.Go(func() error {
			if err := e.executeRemote(script); err != nil {
				return fmt.Errorf("failed to execute client script on %s: %v", script.Host, err)
			}
			return nil
		})
	}
	if err := clientEg.Wait(); err != nil {
		return err
	}

	fmt.Println("All scripts executed successfully")
	return nil
}

func (e *Executor) waitingForServerStart(sHosts []*generator.HostScript) {
	if len(sHosts) == 0 {
		return
	}

	// 构建期望的进程数映射
	expectedProcesses := make(map[string]int)
	for _, script := range sHosts {
		expectedProcesses[script.Host] = script.CommandCount
	}

	fmt.Println("Waiting for server processes to start...")
	fmt.Printf("Expected processes: %v\n\n", expectedProcesses)

	startTime := time.Now()
	probeInterval := 1 * time.Second
	ticker := time.NewTicker(probeInterval)
	defer ticker.Stop()

	for {
		// 检查超时
		if time.Since(startTime) > e.timeout {
			fmt.Printf("⚠️  Timeout after %v - some servers may not have started\n", e.timeout)
			return
		}

		// 探测所有服务器
		allReady := true
		fmt.Printf("=== Probe Status (%s, elapsed: %v) ===\n",
			time.Now().Format("15:04:05"),
			time.Since(startTime).Round(time.Second))

		for _, script := range sHosts {
			count := e.probeProcessCount(script.Host)
			expected := expectedProcesses[script.Host]

			status := "⏳"
			if count >= expected {
				status = "✅"
			} else {
				allReady = false
			}

			fmt.Printf("%s %s: %d/%d processes\n", status, script.Host, count, expected)
		}
		fmt.Println()

		if allReady {
			fmt.Printf("✅ All server processes started successfully (took %v)\n\n",
				time.Since(startTime).Round(time.Second))
			return
		}

		// 等待下一次探测
		<-ticker.C
	}
}

// probeProcessCount 探测指定主机上的 ib_write_bw 进程数量
func (e *Executor) probeProcessCount(hostname string) int {
	sshKeyPath := e.cfg.SSH.PrivateKey

	if !strings.Contains(hostname, "@") && e.cfg.SSH.User != "" {
		hostname = fmt.Sprintf("%s@%s", e.cfg.SSH.User, hostname)
	}

	command := fmt.Sprintf("ps aux | grep %s | grep -v grep | wc -l", e.TestType.Command())
	// 构建SSH命令
	var cmd *exec.Cmd
	if sshKeyPath != "" {
		cmd = exec.Command("ssh",
			"-i", sshKeyPath,
			"-o", "StrictHostKeyChecking=no",
			"-o", "UserKnownHostsFile=/dev/null",
			"-o", "LogLevel=ERROR",
			hostname,
			command)
	} else {
		cmd = exec.Command("ssh",
			"-o", "StrictHostKeyChecking=no",
			"-o", "UserKnownHostsFile=/dev/null",
			"-o", "LogLevel=ERROR",
			hostname,
			command)
	}

	output, err := cmd.Output()
	if err != nil {
		return 0
	}

	var count int
	fmt.Sscanf(string(output), "%d", &count)
	return count
}

// lookupIPs 根据模式查询所需的IP地址
func (e *Executor) lookupIPs() (map[string]string, error) {
	hostIPs := make(map[string]string)

	switch e.mode {
	case ModeBwFullmesh, ModeLatFullmesh:
		// Fullmesh需要所有主机的IP
		allHosts := make(map[string]bool)
		for _, host := range e.cfg.Server.Hostname {
			allHosts[host] = true
		}
		for _, host := range e.cfg.Client.Hostname {
			allHosts[host] = true
		}

		// 查询所有主机的IP
		serverIPs, err := e.cfg.LookupServerHostsIP()
		if err != nil {
			return nil, fmt.Errorf("failed to lookup server IPs: %v", err)
		}
		clientIPs, err := e.cfg.LookupClientHostsIP()
		if err != nil {
			return nil, fmt.Errorf("failed to lookup client IPs: %v", err)
		}

		// 合并IP
		for host, ip := range serverIPs {
			hostIPs[host] = ip
		}
		for host, ip := range clientIPs {
			hostIPs[host] = ip
		}

	case ModeBwIncast, ModeLatIncast:
		// Incast只需要server的IP
		serverIPs, err := e.cfg.LookupServerHostsIP()
		if err != nil {
			return nil, fmt.Errorf("failed to lookup server IPs: %v", err)
		}
		hostIPs = serverIPs

	case ModeBwP2P, ModeLatP2P:
		// P2P需要所有server和client的IP
		serverIPs, err := e.cfg.LookupServerHostsIP()
		if err != nil {
			return nil, fmt.Errorf("failed to lookup server IPs: %v", err)
		}
		clientIPs, err := e.cfg.LookupClientHostsIP()
		if err != nil {
			return nil, fmt.Errorf("failed to lookup client IPs: %v", err)
		}

		// 合并IP
		for host, ip := range serverIPs {
			hostIPs[host] = ip
		}
		for host, ip := range clientIPs {
			hostIPs[host] = ip
		}

	case ModeBwLocaltest, ModeLatLocaltest:
		// Localtest只需要server的IP（都是本地测试）
		serverIPs, err := e.cfg.LookupServerHostsIP()
		if err != nil {
			return nil, fmt.Errorf("failed to lookup server IPs: %v", err)
		}
		hostIPs = serverIPs

	default:
		return nil, fmt.Errorf("unknown mode: %s", e.mode)
	}

	return hostIPs, nil
}

// executeRemote 使用SSH执行远程命令
func (e *Executor) executeRemote(script *generator.HostScript) error {
	// 打印执行信息
	fmt.Printf("Executing on %s (%d command(s)):\n%s\n", script.Host, script.CommandCount, script.Command)

	sshWrapper := tools.NewSSHWrapper(script.Host).
		PrivateKey(e.cfg.SSH.PrivateKey).
		User(e.cfg.SSH.User).
		Command(script.Command)
	cmd := exec.Command("bash", "-c", sshWrapper.String())
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("SSH command execution failed: %v", err)
	}
	return nil
}

// GetSupportedModes 返回所有支持的测试模式
func GetSupportedModes() []TestMode {
	return []TestMode{
		ModeBwFullmesh,
		ModeBwIncast,
		ModeBwP2P,
		ModeBwLocaltest,
		ModeLatFullmesh,
		ModeLatIncast,
		ModeLatP2P,
		ModeLatLocaltest,
	}
}

// IsValidMode 检查模式是否有效
func IsValidMode(mode TestMode) bool {
	switch mode {
	case ModeBwFullmesh, ModeBwIncast, ModeBwP2P, ModeBwLocaltest,
		ModeLatFullmesh, ModeLatIncast, ModeLatP2P, ModeLatLocaltest:
		return true
	default:
		return false
	}
}
