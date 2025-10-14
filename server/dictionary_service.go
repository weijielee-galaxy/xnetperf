package server

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	DictionaryDir = "dictionary"
	HostnameFile  = "hostnames.txt"
	HCAFile       = "hcas.txt"
)

// DictionaryService 字典服务
type DictionaryService struct{}

// NewDictionaryService 创建字典服务
func NewDictionaryService() *DictionaryService {
	return &DictionaryService{}
}

// readDictionary 读取字典文件
func (s *DictionaryService) readDictionary(filename string) ([]string, error) {
	filePath := filepath.Join(DictionaryDir, filename)

	// 如果文件不存在，返回空列表
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return []string{}, nil
	}

	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var items []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && !strings.HasPrefix(line, "#") {
			items = append(items, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return items, nil
}

// writeDictionary 写入字典文件
func (s *DictionaryService) writeDictionary(filename string, items []string) error {
	// 确保目录存在
	if err := os.MkdirAll(DictionaryDir, 0755); err != nil {
		return err
	}

	filePath := filepath.Join(DictionaryDir, filename)
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	for _, item := range items {
		line := strings.TrimSpace(item)
		if line != "" {
			if _, err := writer.WriteString(line + "\n"); err != nil {
				return err
			}
		}
	}

	return writer.Flush()
}

// GetHostnames 获取主机名列表
func (s *DictionaryService) GetHostnames(c *gin.Context) {
	hostnames, err := s.readDictionary(HostnameFile)
	if err != nil {
		c.JSON(500, Error(500, fmt.Sprintf("读取主机名列表失败: %v", err)))
		return
	}

	c.JSON(200, Success(hostnames))
}

// UpdateHostnames 更新主机名列表
func (s *DictionaryService) UpdateHostnames(c *gin.Context) {
	var req struct {
		Hostnames []string `json:"hostnames" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, Error(400, fmt.Sprintf("请求参数错误: %v", err)))
		return
	}

	// 去重
	hostnameMap := make(map[string]bool)
	var uniqueHostnames []string
	for _, hostname := range req.Hostnames {
		hostname = strings.TrimSpace(hostname)
		if hostname != "" && !hostnameMap[hostname] {
			hostnameMap[hostname] = true
			uniqueHostnames = append(uniqueHostnames, hostname)
		}
	}

	if err := s.writeDictionary(HostnameFile, uniqueHostnames); err != nil {
		c.JSON(500, Error(500, fmt.Sprintf("保存主机名列表失败: %v", err)))
		return
	}

	c.JSON(200, SuccessWithMessage("主机名列表更新成功", uniqueHostnames))
}

// GetHCAs 获取 HCA 列表
func (s *DictionaryService) GetHCAs(c *gin.Context) {
	hcas, err := s.readDictionary(HCAFile)
	if err != nil {
		c.JSON(500, Error(500, fmt.Sprintf("读取 HCA 列表失败: %v", err)))
		return
	}

	c.JSON(200, Success(hcas))
}

// UpdateHCAs 更新 HCA 列表
func (s *DictionaryService) UpdateHCAs(c *gin.Context) {
	var req struct {
		HCAs []string `json:"hcas" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, Error(400, fmt.Sprintf("请求参数错误: %v", err)))
		return
	}

	// 去重
	hcaMap := make(map[string]bool)
	var uniqueHCAs []string
	for _, hca := range req.HCAs {
		hca = strings.TrimSpace(hca)
		if hca != "" && !hcaMap[hca] {
			hcaMap[hca] = true
			uniqueHCAs = append(uniqueHCAs, hca)
		}
	}

	if err := s.writeDictionary(HCAFile, uniqueHCAs); err != nil {
		c.JSON(500, Error(500, fmt.Sprintf("保存 HCA 列表失败: %v", err)))
		return
	}

	c.JSON(200, SuccessWithMessage("HCA 列表更新成功", uniqueHCAs))
}
