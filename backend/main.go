package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"clouddisk/config"
	"clouddisk/database"
	"clouddisk/handlers"
	"clouddisk/middleware"

	"github.com/begonia599/myplatform/sdk"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.Load()
	database.Init(cfg)

	// 初始化统一后端 SDK 客户端
	platform := sdk.New(&sdk.Config{
		BaseURL: cfg.PlatformURL,
	})

	// 注册网盘权限到核心平台
	registerCloudPermissions(platform)

	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders: []string{"Origin", "Content-Type", "Authorization"},
	}))

	// Security headers
	r.Use(func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; font-src 'self'")
		c.Next()
	})

	// Set max upload size (100MB)
	r.MaxMultipartMemory = 100 << 20

	authHandler := &handlers.AuthHandler{Platform: platform}
	fileHandler := &handlers.FileHandler{Config: cfg}
	shareHandler := &handlers.ShareHandler{Config: cfg}
	imageHandler := &handlers.ImageHandler{Platform: platform}

	api := r.Group("/api")
	{
		// Auth 代理端点（转发到统一后端）
		api.POST("/auth/login", authHandler.Login)
		api.POST("/auth/refresh", authHandler.Refresh)

		// 公开分享端点
		api.GET("/s/:code", shareHandler.GetShareInfo)
		api.POST("/s/:code/verify", shareHandler.VerifyShare)
		api.POST("/s/:code/download-token", shareHandler.IssueDownloadToken)
		api.GET("/s/:code/download/:fileId", shareHandler.Download)

		// 需要认证的端点（通过 SDK 验证）
		auth := api.Group("")
		auth.Use(middleware.AuthRequired(platform))
		{
			// 文件操作
			auth.POST("/files/upload", middleware.RequirePermission(platform, "cloud.file", "upload"), fileHandler.Upload)
			auth.POST("/files/upload/init", middleware.RequirePermission(platform, "cloud.file", "upload"), fileHandler.InitUpload)
			auth.POST("/files/upload/chunk", middleware.RequirePermission(platform, "cloud.file", "upload"), fileHandler.UploadChunk)
			auth.GET("/files/upload/status", middleware.RequirePermission(platform, "cloud.file", "read"), fileHandler.UploadStatus)
			auth.POST("/files/upload/complete", middleware.RequirePermission(platform, "cloud.file", "upload"), fileHandler.CompleteUpload)
			auth.GET("/files", middleware.RequirePermission(platform, "cloud.file", "read"), fileHandler.List)
			auth.DELETE("/files/:id", middleware.RequirePermission(platform, "cloud.file", "delete"), fileHandler.Delete)

			// 分享管理
			auth.POST("/shares", middleware.RequirePermission(platform, "cloud.share", "create"), shareHandler.Create)
			auth.GET("/shares", middleware.RequirePermission(platform, "cloud.share", "read"), shareHandler.List)
			auth.DELETE("/shares/:id", middleware.RequirePermission(platform, "cloud.share", "delete"), shareHandler.Delete)

			// 图床管理（SDK 代理到 myplatform）
			auth.POST("/images/upload", imageHandler.Upload)
			auth.GET("/images", imageHandler.List)
			auth.DELETE("/images/:id", imageHandler.Delete)
			auth.PATCH("/images/:id/visibility", imageHandler.ToggleVisibility)
			auth.GET("/images/platform-url", imageHandler.PlatformURL)
		}
	}

	// Serve embedded frontend static files
	staticFS := frontendAssets()
	fileServer := http.FileServer(staticFS)

	r.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path

		// Try to serve static file (JS, CSS, images, etc.)
		if strings.Contains(path, ".") {
			fileServer.ServeHTTP(c.Writer, c.Request)
			return
		}

		// SPA fallback: serve index.html for all routes without file extensions
		c.Request.URL.Path = "/"
		fileServer.ServeHTTP(c.Writer, c.Request)
	})

	// Start goroutine to clean up stale chunked upload temp directories (older than 24h)
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		tmpDir := filepath.Join(cfg.UploadDir, "tmp")
		for range ticker.C {
			entries, err := os.ReadDir(tmpDir)
			if err != nil {
				continue
			}
			for _, e := range entries {
				if !e.IsDir() {
					continue
				}
				info, err := e.Info()
				if err != nil {
					continue
				}
				if time.Since(info.ModTime()) > 24*time.Hour {
					os.RemoveAll(filepath.Join(tmpDir, e.Name()))
					log.Printf("Cleaned up stale upload: %s", e.Name())
				}
			}
		}
	}()

	log.Println("Server starting on :8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func registerCloudPermissions(platform *sdk.Client) {
	err := platform.Permission.RegisterPermissions("cloud", []sdk.ResourceDef{
		{Resource: "cloud.file", Actions: []string{"upload", "read", "delete"}},
		{Resource: "cloud.share", Actions: []string{"create", "read", "delete"}},
	})
	if err != nil {
		log.Printf("Warning: failed to register cloud permissions: %v", err)
	} else {
		log.Println("Cloud permissions registered with platform")
	}
}
