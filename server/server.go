package server

import (
	"fmt"
	"io/fs"
	"net/http"

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
	// 静态文件服务（Web UI）
	staticFS, err := fs.Sub(web.Static, "static")
	if err != nil {
		panic(err)
	}
	s.engine.StaticFS("/ui", http.FS(staticFS))

	// 根路径重定向到 Web UI
	s.engine.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/ui/")
	})

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
