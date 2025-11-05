package server

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"xnetperf/config"
	"xnetperf/internal/script"
	"xnetperf/internal/service/analyze"
	"xnetperf/internal/service/collect"
	"xnetperf/internal/service/precheck"
	"xnetperf/internal/service/probe"
	runnerservice "xnetperf/internal/service/runner"

	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v3"
)

const (
	DefaultConfigFile = "config.yaml"
	ConfigsDir        = "configs"
)

// ConfigFileInfo 配置文件信息
type ConfigFileInfo struct {
	Name        string `json:"name"`         // 文件名
	Path        string `json:"path"`         // 文件路径
	IsDefault   bool   `json:"is_default"`   // 是否为默认配置
	IsDeletable bool   `json:"is_deletable"` // 是否可删除
}

// ConfigService 配置文件服务
type ConfigService struct {
}

// NewConfigService 创建配置文件服务
func NewConfigService() *ConfigService {
	return &ConfigService{}
}

// ListConfigs 获取配置文件列表
func (s *ConfigService) ListConfigs(c *gin.Context) {
	var configs []ConfigFileInfo

	// 添加默认配置文件
	if _, err := os.Stat(DefaultConfigFile); err == nil {
		configs = append(configs, ConfigFileInfo{
			Name:        DefaultConfigFile,
			Path:        DefaultConfigFile,
			IsDefault:   true,
			IsDeletable: false,
		})
	}

	// 扫描 configs 目录
	if _, err := os.Stat(ConfigsDir); err == nil {
		files, err := os.ReadDir(ConfigsDir)
		if err == nil {
			for _, file := range files {
				if !file.IsDir() {
					// 只扫描 yaml 和 yml 文件
					if strings.HasSuffix(file.Name(), ".yaml") || strings.HasSuffix(file.Name(), ".yml") {
						configs = append(configs, ConfigFileInfo{
							Name:        file.Name(),
							Path:        filepath.Join(ConfigsDir, file.Name()),
							IsDefault:   false,
							IsDeletable: true,
						})
					}
				}
			}
		}
	}

	c.JSON(200, Success(configs))
}

// GetConfig 获取指定配置文件内容
func (s *ConfigService) GetConfig(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		c.JSON(400, Error(400, "配置文件名不能为空"))
		return
	}

	// 构建文件路径
	var filePath string
	if name == DefaultConfigFile {
		filePath = DefaultConfigFile
	} else {
		filePath = filepath.Join(ConfigsDir, name)
	}

	// 读取配置文件
	cfg, err := config.LoadConfig(filePath)
	if err != nil {
		c.JSON(404, Error(404, fmt.Sprintf("配置文件不存在或读取失败: %v", err)))
		return
	}

	c.JSON(200, Success(cfg))
}

// PreviewConfig 预览配置文件（返回 YAML 格式）
func (s *ConfigService) PreviewConfig(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		c.JSON(400, Error(400, "配置文件名不能为空"))
		return
	}

	// 构建文件路径
	var filePath string
	if name == DefaultConfigFile {
		filePath = DefaultConfigFile
	} else {
		filePath = filepath.Join(ConfigsDir, name)
	}

	// 读取配置文件原始内容
	data, err := os.ReadFile(filePath)
	if err != nil {
		c.JSON(404, Error(404, fmt.Sprintf("配置文件不存在或读取失败: %v", err)))
		return
	}

	c.JSON(200, Success(gin.H{"yaml": string(data)}))
}

// CreateConfig 创建新配置文件
func (s *ConfigService) CreateConfig(c *gin.Context) {
	var req struct {
		Name   string        `json:"name" binding:"required"`
		Config config.Config `json:"config" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, Error(400, fmt.Sprintf("请求参数错误: %v", err)))
		return
	}

	// 验证文件名
	if !strings.HasSuffix(req.Name, ".yaml") && !strings.HasSuffix(req.Name, ".yml") {
		c.JSON(400, Error(400, "配置文件必须以 .yaml 或 .yml 结尾"))
		return
	}

	// 不允许创建默认配置文件
	if req.Name == DefaultConfigFile {
		c.JSON(400, Error(400, "不能创建默认配置文件"))
		return
	}

	// 应用默认值到未指定的字段
	req.Config.ApplyDefaults()

	// 确保 configs 目录存在
	if err := os.MkdirAll(ConfigsDir, 0755); err != nil {
		c.JSON(500, Error(500, fmt.Sprintf("创建目录失败: %v", err)))
		return
	}

	// 构建文件路径
	filePath := filepath.Join(ConfigsDir, req.Name)

	// 检查文件是否已存在
	if _, err := os.Stat(filePath); err == nil {
		c.JSON(400, Error(400, "配置文件已存在"))
		return
	}

	// 将配置写入文件
	data, err := yaml.Marshal(&req.Config)
	if err != nil {
		c.JSON(500, Error(500, fmt.Sprintf("配置序列化失败: %v", err)))
		return
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		c.JSON(500, Error(500, fmt.Sprintf("写入文件失败: %v", err)))
		return
	}

	c.JSON(200, SuccessWithMessage("配置文件创建成功", ConfigFileInfo{
		Name:        req.Name,
		Path:        filePath,
		IsDefault:   false,
		IsDeletable: true,
	}))
}

// UpdateConfig 更新指定配置文件
func (s *ConfigService) UpdateConfig(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		c.JSON(400, Error(400, "配置文件名不能为空"))
		return
	}

	var cfg config.Config
	if err := c.ShouldBindJSON(&cfg); err != nil {
		c.JSON(400, Error(400, fmt.Sprintf("请求参数错误: %v", err)))
		return
	}

	// 应用默认值到未指定的字段
	cfg.ApplyDefaults()

	// 构建文件路径
	var filePath string
	if name == DefaultConfigFile {
		filePath = DefaultConfigFile
	} else {
		filePath = filepath.Join(ConfigsDir, name)
	}

	// 检查文件是否存在
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		c.JSON(404, Error(404, "配置文件不存在"))
		return
	}

	// 将配置写入文件
	data, err := yaml.Marshal(&cfg)
	if err != nil {
		c.JSON(500, Error(500, fmt.Sprintf("配置序列化失败: %v", err)))
		return
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		c.JSON(500, Error(500, fmt.Sprintf("写入文件失败: %v", err)))
		return
	}

	c.JSON(200, SuccessWithMessage("配置文件更新成功", nil))
}

// DeleteConfig 删除配置文件
func (s *ConfigService) DeleteConfig(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		c.JSON(400, Error(400, "配置文件名不能为空"))
		return
	}

	// 不允许删除默认配置文件
	if name == DefaultConfigFile {
		c.JSON(400, Error(400, "不能删除默认配置文件"))
		return
	}

	// 构建文件路径
	filePath := filepath.Join(ConfigsDir, name)

	// 检查文件是否存在
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		c.JSON(404, Error(404, "配置文件不存在"))
		return
	}

	// 删除文件
	if err := os.Remove(filePath); err != nil {
		c.JSON(500, Error(500, fmt.Sprintf("删除文件失败: %v", err)))
		return
	}

	c.JSON(200, SuccessWithMessage("配置文件删除成功", nil))
}

// ValidateConfig 验证配置文件是否能正常解析
func (s *ConfigService) ValidateConfig(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		c.JSON(400, Error(400, "配置文件名不能为空"))
		return
	}

	// 构建文件路径
	var filePath string
	if name == DefaultConfigFile {
		filePath = DefaultConfigFile
	} else {
		filePath = filepath.Join(ConfigsDir, name)
	}

	// 检查文件是否存在
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		c.JSON(404, Error(404, "配置文件不存在"))
		return
	}

	// 尝试加载并解析配置文件
	cfg, err := config.LoadConfig(filePath)
	if err != nil {
		c.JSON(400, Error(400, fmt.Sprintf("配置文件解析失败: %v", err)))
		return
	}

	// 验证必要字段
	var validationErrors []string

	// 检查 server 配置
	if len(cfg.Server.Hostname) == 0 {
		validationErrors = append(validationErrors, "server.hostname 不能为空")
	}
	if len(cfg.Server.Hca) == 0 {
		validationErrors = append(validationErrors, "server.hca 不能为空")
	}

	// 检查 client 配置
	if len(cfg.Client.Hostname) == 0 {
		validationErrors = append(validationErrors, "client.hostname 不能为空")
	}
	if len(cfg.Client.Hca) == 0 {
		validationErrors = append(validationErrors, "client.hca 不能为空")
	}

	// 检查 stream_type
	if cfg.StreamType != config.FullMesh && cfg.StreamType != config.InCast && cfg.StreamType != config.P2P {
		validationErrors = append(validationErrors, fmt.Sprintf("stream_type 必须是 fullmesh, incast 或 p2p，当前值: %s", cfg.StreamType))
	}

	// 检查端口号
	if cfg.StartPort <= 0 || cfg.StartPort > 65535 {
		validationErrors = append(validationErrors, fmt.Sprintf("start_port 必须在 1-65535 之间，当前值: %d", cfg.StartPort))
	}

	// 检查队列对数量
	if cfg.QpNum <= 0 {
		validationErrors = append(validationErrors, fmt.Sprintf("qp_num 必须大于 0，当前值: %d", cfg.QpNum))
	}

	// 检查消息大小
	if cfg.MessageSizeBytes <= 0 {
		validationErrors = append(validationErrors, fmt.Sprintf("message_size_bytes 必须大于 0，当前值: %d", cfg.MessageSizeBytes))
	}

	// 检查速度
	if cfg.Speed <= 0 {
		validationErrors = append(validationErrors, fmt.Sprintf("speed 必须大于 0，当前值: %f", cfg.Speed))
	}

	// 检查等待时间
	if cfg.WaitingTimeSeconds < 0 {
		validationErrors = append(validationErrors, fmt.Sprintf("waiting_time_seconds 不能为负数，当前值: %d", cfg.WaitingTimeSeconds))
	}

	// 检查运行时长
	if !cfg.Run.Infinitely && cfg.Run.DurationSeconds <= 0 {
		validationErrors = append(validationErrors, fmt.Sprintf("当 run.infinitely 为 false 时，run.duration_seconds 必须大于 0，当前值: %d", cfg.Run.DurationSeconds))
	}

	// 如果有验证错误，返回错误信息
	if len(validationErrors) > 0 {
		c.JSON(400, gin.H{
			"code":    400,
			"message": "配置文件验证失败",
			"data": gin.H{
				"valid":  false,
				"errors": validationErrors,
			},
		})
		return
	}

	// 验证成功，返回配置信息
	c.JSON(200, gin.H{
		"code":    0,
		"message": "配置文件验证成功",
		"data": gin.H{
			"valid":  true,
			"config": cfg,
		},
	})
}

// PrecheckConfig 执行配置文件的 precheck 检查
func (s *ConfigService) PrecheckConfig(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		c.JSON(400, Error(400, "配置文件名不能为空"))
		return
	}

	// 构建文件路径
	var filePath string
	if name == DefaultConfigFile {
		filePath = DefaultConfigFile
	} else {
		filePath = filepath.Join(ConfigsDir, name)
	}

	// 检查文件是否存在
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		c.JSON(404, Error(404, "配置文件不存在"))
		return
	}

	// 加载并验证配置文件
	cfg, err := config.LoadConfig(filePath)
	if err != nil {
		c.JSON(400, Error(400, fmt.Sprintf("配置文件解析失败: %v", err)))
		return
	}

	// 执行 precheck
	checker := precheck.New(cfg)
	summary, err := checker.DoCheckForAPI(cfg)
	if err != nil {
		c.JSON(500, Error(500, fmt.Sprintf("Precheck 执行失败: %v", err)))
		return
	}

	// 返回检查结果
	c.JSON(200, Success(summary))
}

// RunTest 运行测试（不包含 precheck）
func (s *ConfigService) RunTest(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		c.JSON(400, Error(400, "配置文件名不能为空"))
		return
	}

	// 构建文件路径
	var filePath string
	if name == DefaultConfigFile {
		filePath = DefaultConfigFile
	} else {
		filePath = filepath.Join(ConfigsDir, name)
	}

	// 检查文件是否存在
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		c.JSON(404, Error(404, "配置文件不存在"))
		return
	}

	// 加载配置文件
	cfg, err := config.LoadConfig(filePath)
	if err != nil {
		c.JSON(400, Error(400, fmt.Sprintf("配置文件解析失败: %v", err)))
		return
	}

	// 执行测试
	runner := runnerservice.New(cfg)
	result, err := runner.RunAndGetResult(script.TestTypeBandwidth)
	if err != nil {
		c.JSON(500, Error(500, fmt.Sprintf("测试运行失败: %v", err)))
		return
	}

	// 返回结果
	c.JSON(200, Success(result))
}

// ProbeTest 探测测试状态
func (s *ConfigService) ProbeTest(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		c.JSON(400, Error(400, "配置文件名不能为空"))
		return
	}

	// 构建文件路径
	var filePath string
	if name == DefaultConfigFile {
		filePath = DefaultConfigFile
	} else {
		filePath = filepath.Join(ConfigsDir, name)
	}

	// 检查文件是否存在
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		c.JSON(404, Error(404, "配置文件不存在"))
		return
	}

	// 加载配置文件
	cfg, err := config.LoadConfig(filePath)
	if err != nil {
		c.JSON(400, Error(400, fmt.Sprintf("配置文件解析失败: %v", err)))
		return
	}

	// 执行探测
	prober := probe.New(cfg)
	summary, err := prober.DoProbeAndGetSummary()
	if err != nil {
		c.JSON(500, Error(500, fmt.Sprintf("探测执行失败: %v", err)))
		return
	}

	// 返回结果
	c.JSON(200, Success(summary))
}

// CollectReports 收集测试报告
func (s *ConfigService) CollectReports(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		c.JSON(400, Error(400, "配置文件名不能为空"))
		return
	}

	// 构建文件路径
	var filePath string
	if name == DefaultConfigFile {
		filePath = DefaultConfigFile
	} else {
		filePath = filepath.Join(ConfigsDir, name)
	}

	// 检查文件是否存在
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		c.JSON(404, Error(404, "配置文件不存在"))
		return
	}

	// 加载配置文件
	cfg, err := config.LoadConfig(filePath)
	if err != nil {
		c.JSON(400, Error(400, fmt.Sprintf("配置文件解析失败: %v", err)))
		return
	}

	// 执行收集
	collector := collect.New(cfg)
	result, err := collector.CollectAndGetResult(cfg)
	if err != nil {
		c.JSON(500, Error(500, fmt.Sprintf("报告收集失败: %v", err)))
		return
	}

	// 返回结果
	c.JSON(200, Success(result))
}

// GetReport 获取性能报告
func (s *ConfigService) GetReport(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		c.JSON(400, Error(400, "配置文件名不能为空"))
		return
	}

	// 构建文件路径
	var filePath string
	if name == DefaultConfigFile {
		filePath = DefaultConfigFile
	} else {
		filePath = filepath.Join(ConfigsDir, name)
	}

	// 检查文件是否存在
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		c.JSON(404, Error(404, "配置文件不存在"))
		return
	}

	// 加载配置文件
	cfg, err := config.LoadConfig(filePath)
	if err != nil {
		c.JSON(400, Error(400, fmt.Sprintf("配置文件解析失败: %v", err)))
		return
	}

	// 生成报告
	analyzeer := analyze.New(cfg)
	report, err := analyzeer.GenerateReport()
	if err != nil {
		c.JSON(500, Error(500, fmt.Sprintf("报告生成失败: %v", err)))
		return
	}

	// 返回结果
	c.JSON(200, Success(report))
}
