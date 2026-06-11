package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

// Config holds all application configuration loaded from the environment.
type Config struct {
	ServerPort       string
	GinMode          string
	DBHost           string
	DBPort           string
	DBUser           string
	DBPassword       string
	DBName           string
	DBSSLMode        string
	CORSAllowOrigins string
	DefaultEndpoint  string
}

// Load reads the .env file (if present) and returns the populated Config.
func Load() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, relying on system environment variables")
	}

	return &Config{
		ServerPort:       getEnv("SERVER_PORT", "8080"),
		GinMode:          getEnv("GIN_MODE", "debug"),
		DBHost:           getEnv("DB_HOST", "localhost"),
		DBPort:           getEnv("DB_PORT", "5432"),
		DBUser:           getEnv("DB_USER", "postgres"),
		DBPassword:       getEnv("DB_PASSWORD", "postgres"),
		DBName:           getEnv("DB_NAME", "go_api"),
		DBSSLMode:        getEnv("DB_SSLMODE", "disable"),
		CORSAllowOrigins: getEnv("CORS_ALLOW_ORIGINS", "*"),
		DefaultEndpoint:  getEnv("DEFAULT_ENDPOINT", ""),
	}
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok && value != "" {
		return value
	}
	return fallback
}
