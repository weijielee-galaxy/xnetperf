package runner

import (
	"fmt"
	"log/slog"
	"sync"
	"xnetperf/config"
	"xnetperf/internal/script"
	"xnetperf/pkg/tools"
	"xnetperf/pkg/tools/logger"
)

type runner struct {
	cfg    *config.Config
	logger *slog.Logger
}

func New(cfg *config.Config) *runner {
	return &runner{
		cfg:    cfg,
		logger: logger.GetLogger().With("module", "RUN"),
	}
}

func (r *runner) Run(testType script.TestType) error {
	r.logger.Info("Starting network test run")

	executor := script.NewExecutor(r.cfg, testType)
	if executor == nil {
		r.logger.Error("Unsupported stream type for v1 execute workflow. Aborting.")
		return fmt.Errorf("Unsupported stream type for v1 execute workflow")
	}

	if r.cfg.Report.Enable {
		cleanupRemoteReportFiles(r.cfg)
	}

	err := executor.Execute()
	if err != nil {
		r.logger.Error("Run step failed: %v. Aborting workflow.", slog.Any("error", err))
		return fmt.Errorf("Run step failed: %v. Aborting workflow.", err)
	}

	r.logger.Info("Network test run completed successfully")
	return nil
}

// RunResult 运行测试的结果
type RunResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Error   string `json:"error,omitempty"`
}

func (r *runner) RunAndGetResult(testType script.TestType) (RunResult, error) {
	err := r.Run(testType)
	if err != nil {
		return RunResult{
			Success: false,
			Message: "Test run failed",
			Error:   err.Error(),
		}, err
	}

	return RunResult{
		Success: true,
		Message: "Test scripts distributed and started successfully",
	}, nil
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
