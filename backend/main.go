package main

import (
	"log"
	"net/http"
	"strings"

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

	log.Println("Server starting on :8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
