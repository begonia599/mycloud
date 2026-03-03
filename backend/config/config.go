package config

import (
	"log"
	"os"
)

type Config struct {
	DBHost    string
	DBPort    string
	DBUser    string
	DBPass    string
	DBName    string
	DBSSLMode string
	JWTSecret string
	AdminUser string
	AdminPass string
	UploadDir string
}

func Load() *Config {
	dbPass := os.Getenv("DB_PASSWORD")
	if dbPass == "" {
		log.Fatal("DB_PASSWORD environment variable is required")
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET environment variable is required")
	}

	adminPass := getEnv("ADMIN_PASS", "admin123")
	if adminPass == "admin123" {
		log.Println("WARNING: using default ADMIN_PASS — change it in production!")
	}

	return &Config{
		DBHost:    getEnv("DB_HOST", "localhost"),
		DBPort:    getEnv("DB_PORT", "5432"),
		DBUser:    getEnv("DB_USER", "clouddisk"),
		DBPass:    dbPass,
		DBName:    getEnv("DB_NAME", "clouddisk"),
		DBSSLMode: getEnv("DB_SSLMODE", "disable"),
		JWTSecret: jwtSecret,
		AdminUser: getEnv("ADMIN_USER", "admin"),
		AdminPass: adminPass,
		UploadDir: getEnv("UPLOAD_DIR", "./uploads"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
