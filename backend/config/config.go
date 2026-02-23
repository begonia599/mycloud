package config

import "os"

type Config struct {
	DBHost    string
	DBPort    string
	DBUser    string
	DBPass    string
	DBName    string
	JWTSecret string
	AdminUser string
	AdminPass string
	UploadDir string
}

func Load() *Config {
	return &Config{
		DBHost:    getEnv("DB_HOST", "localhost"),
		DBPort:    getEnv("DB_PORT", "5432"),
		DBUser:    getEnv("DB_USER", "clouddisk"),
		DBPass:    getEnv("DB_PASSWORD", "clouddisk"),
		DBName:    getEnv("DB_NAME", "clouddisk"),
		JWTSecret: getEnv("JWT_SECRET", "change-me-in-production"),
		AdminUser: getEnv("ADMIN_USER", "admin"),
		AdminPass: getEnv("ADMIN_PASS", "admin123"),
		UploadDir: getEnv("UPLOAD_DIR", "./uploads"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
