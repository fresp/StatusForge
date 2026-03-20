package configs

import (
	"os"
)

type Config struct {
	MongoURI     string
	MongoDBName  string
	RedisAddr    string
	JWTSecret    string
	MFASecretKey string
	Port         string
	AdminEmail   string
	AdminPass    string
	AdminUser    string
	EnableWorker bool
}

func Load() *Config {
	return &Config{
		MongoURI:     getEnv("MONGO_URI", "mongodb://localhost:27017"),
		MongoDBName:  getEnv("MONGO_DB_NAME", "statusplatform"),
		RedisAddr:    getEnv("REDIS_ADDR", "localhost:6379"),
		JWTSecret:    getEnv("JWT_SECRET", "super-secret-jwt-key-change-in-production"),
		MFASecretKey: getEnv("MFA_SECRET_KEY", ""),
		Port:         getEnv("PORT", "8080"),
		AdminEmail:   getEnv("ADMIN_EMAIL", "admin@statusplatform.com"),
		AdminPass:    getEnv("ADMIN_PASSWORD", "admin123"),
		AdminUser:    getEnv("ADMIN_USERNAME", "admin"),
		EnableWorker: getBoolEnv("ENABLE_WORKER", "true"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getBoolEnv(key, fallback string) bool {
	v := os.Getenv(key)
	if v == "" {
		return fallback == "true"
	}
	return v == "true"
}
