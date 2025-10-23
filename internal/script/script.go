package script

import (
	"fmt"
	"os/exec"
	"time"
	"xnetperf/config"
	"xnetperf/internal/script/generator"
	"xnetperf/internal/tools"
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

// Executor 负责脚本的生成和执行编排
type Executor struct {
	cfg     *config.Config
	mode    TestMode
	timeout time.Duration // TODO
}

// NewExecutor 创建执行器
func NewExecutor(cfg *config.Config, mode TestMode) *Executor {
	// 基本验证
	if cfg == nil {
		return nil
	}
	if !IsValidMode(mode) {
		return nil
	}

	return &Executor{
		cfg:  cfg,
		mode: mode,
	}
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
	for _, script := range result.ServerScripts {
		if err := e.executeRemote(script); err != nil {
			return fmt.Errorf("failed to execute server script on %s: %v", script.Host, err)
		}
	}

	// 3. 等待服务端启动
	// TODO probe server status instead of fixed sleep
	fmt.Println("Waiting for servers to start...")
	time.Sleep(time.Duration(e.cfg.WaitingTimeSeconds) * time.Second)

	// 4. 执行客户端脚本
	fmt.Println("Starting client processes...")
	for _, script := range result.ClientScripts {
		if err := e.executeRemote(script); err != nil {
			return fmt.Errorf("failed to execute client script on %s: %v", script.Host, err)
		}
	}

	fmt.Println("All scripts executed successfully")
	return nil
}

func (e *Executor) waitingForServerStart(sHosts []string) {
	// TODO probe server status instead of fixed sleep
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
