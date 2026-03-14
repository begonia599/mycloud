package config

import (
	"log"
	"os"
)

type Config struct {
	// Database (网盘业务表：shares, files 等)
	DBHost    string
	DBPort    string
	DBUser    string
	DBPass    string
	DBName    string
	DBSSLMode string

	// 统一后端地址（内部通信）
	PlatformURL string

	// 统一后端公网地址（给前端用，图片直链等）
	PlatformPublicURL string

	// 本地文件存储目录
	UploadDir string
}

func Load() *Config {
	dbPass := os.Getenv("DB_PASSWORD")
	if dbPass == "" {
		log.Fatal("DB_PASSWORD environment variable is required")
	}

	platformURL := getEnv("PLATFORM_URL", "http://localhost:8080")

	platformPublicURL := getEnv("PLATFORM_PUBLIC_URL", platformURL)

	return &Config{
		DBHost:            getEnv("DB_HOST", "localhost"),
		DBPort:            getEnv("DB_PORT", "5432"),
		DBUser:            getEnv("DB_USER", "clouddisk"),
		DBPass:            dbPass,
		DBName:            getEnv("DB_NAME", "clouddisk"),
		DBSSLMode:         getEnv("DB_SSLMODE", "disable"),
		PlatformURL:       platformURL,
		PlatformPublicURL: platformPublicURL,
		UploadDir:         getEnv("UPLOAD_DIR", "./uploads"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
