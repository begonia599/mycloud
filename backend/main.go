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

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.Load()
	database.Init(cfg)

	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		AllowCredentials: true,
	}))

	// Set max upload size (100MB)
	r.MaxMultipartMemory = 100 << 20

	authHandler := &handlers.AuthHandler{Config: cfg}
	fileHandler := &handlers.FileHandler{Config: cfg}
	shareHandler := &handlers.ShareHandler{Config: cfg}

	api := r.Group("/api")
	{
		api.POST("/auth/login", authHandler.Login)

		// Public share endpoints
		api.GET("/s/:code", shareHandler.GetShareInfo)
		api.POST("/s/:code/verify", shareHandler.VerifyShare)
		api.GET("/s/:code/download/:fileId", shareHandler.Download)

		// Protected endpoints
		admin := api.Group("")
		admin.Use(middleware.AuthRequired(cfg))
		{
			admin.POST("/files/upload", fileHandler.Upload)
			admin.POST("/files/upload/init", fileHandler.InitUpload)
			admin.POST("/files/upload/chunk", fileHandler.UploadChunk)
			admin.GET("/files/upload/status", fileHandler.UploadStatus)
			admin.POST("/files/upload/complete", fileHandler.CompleteUpload)
			admin.GET("/files", fileHandler.List)
			admin.DELETE("/files/:id", fileHandler.Delete)

			admin.POST("/shares", shareHandler.Create)
			admin.GET("/shares", shareHandler.List)
			admin.DELETE("/shares/:id", shareHandler.Delete)
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
