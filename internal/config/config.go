package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port           string
	OpenAIAPIKey   string
	QdrantHost     string
	QdrantPort     string
	RedisAddr      string
}

// LoadConfig reads the .env file and populates the Config struct.
func LoadConfig() *Config {
	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: No .env file found, relying on system environment variables")
	}

	openAIKey := os.Getenv("OPENAI_API_KEY")
	if openAIKey == "" {
		log.Fatal("Fatal: OPENAI_API_KEY environment variable is required")
	}

	return &Config{
		Port:           getEnvOrDefault("PORT", "8080"),
		OpenAIAPIKey:   openAIKey,
		QdrantHost:     getEnvOrDefault("QDRANT_HOST", "localhost"),
		QdrantPort:     getEnvOrDefault("QDRANT_PORT", "6334"),
		RedisAddr:      getEnvOrDefault("REDIS_ADDR", "localhost:6379"),
	}
}

func getEnvOrDefault(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}