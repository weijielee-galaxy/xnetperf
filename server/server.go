package server

import (
	"fmt"
	"io/fs"

	"xnetperf/web"

	"github.com/gin-gonic/gin"
)

// Server HTTP服务器
type Server struct {
	engine        *gin.Engine
	configService *ConfigService
	port          int
}

// NewServer 创建HTTP服务器
func NewServer(port int) *Server {
	gin.SetMode(gin.ReleaseMode)

	server := &Server{
		engine:        gin.Default(),
		configService: NewConfigService(),
		port:          port,
	}

	server.setupRoutes()
	return server
}

// setupRoutes 设置路由
func (s *Server) setupRoutes() {
	// API 分组
	api := s.engine.Group("/api")
	{
		// 配置文件管理API
		configs := api.Group("/configs")
		{
			configs.GET("", s.configService.ListConfigs)                    // 获取配置文件列表
			configs.GET("/:name", s.configService.GetConfig)                // 获取指定配置文件
			configs.POST("", s.configService.CreateConfig)                  // 创建配置文件
			configs.PUT("/:name", s.configService.UpdateConfig)             // 更新配置文件
			configs.DELETE("/:name", s.configService.DeleteConfig)          // 删除配置文件
			configs.POST("/:name/validate", s.configService.ValidateConfig) // 验证配置文件
		}
	}

	// 健康检查
	s.engine.GET("/health", func(c *gin.Context) {
		c.JSON(200, Success(gin.H{"status": "ok"}))
	})

	// 静态文件服务（Web UI）- 放在最后，作为 fallback
	staticFS, err := fs.Sub(web.Static, "static")
	if err != nil {
		panic(err)
	}
	s.engine.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path
		// 如果是根路径或目录，返回 index.html
		if path == "/" || path == "" {
			path = "index.html"
		} else {
			// 去掉开头的 /
			path = path[1:]
		}

		// 尝试读取文件
		data, err := fs.ReadFile(staticFS, path)
		if err != nil {
			// 如果文件不存在，返回 index.html（用于 SPA 路由）
			data, err = fs.ReadFile(staticFS, "index.html")
			if err != nil {
				c.String(404, "Not Found")
				return
			}
			c.Data(200, "text/html; charset=utf-8", data)
			return
		}

		// 根据文件扩展名设置 Content-Type
		contentType := "application/octet-stream"
		if len(path) > 3 && path[len(path)-3:] == ".js" {
			contentType = "application/javascript"
		} else if len(path) > 5 && path[len(path)-5:] == ".html" {
			contentType = "text/html; charset=utf-8"
		} else if len(path) > 4 && path[len(path)-4:] == ".css" {
			contentType = "text/css"
		}

		c.Data(200, contentType, data)
	})
}

// Start 启动服务器
func (s *Server) Start() error {
	addr := fmt.Sprintf(":%d", s.port)
	fmt.Printf("HTTP Server starting on http://localhost%s\n", addr)
	fmt.Println("\n🌐 Web UI:")
	fmt.Printf("  http://localhost%s/\n", addr)
	fmt.Println("\n📡 API Endpoints:")
	fmt.Println("  GET    /health")
	fmt.Println("  GET    /api/configs")
	fmt.Println("  GET    /api/configs/:name")
	fmt.Println("  POST   /api/configs")
	fmt.Println("  PUT    /api/configs/:name")
	fmt.Println("  DELETE /api/configs/:name")
	fmt.Println("  POST   /api/configs/:name/validate")
	return s.engine.Run(addr)
}
