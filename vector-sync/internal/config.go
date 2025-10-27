package internal

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	// Required API keys
	PineconeAPIKey string
	NotesDir       string
	PineconeHost   string
	EmbeddingUrl   string
}

// LoadConfig loads configuration from .env file and environment variables
func LoadConfig() (*Config, error) {
	// Try to load .env file (optional - don't fail if it doesn't exist)
	if err := godotenv.Load(); err != nil {
		fmt.Printf("Info: .env file not found, using environment variables only\n")
	}

	config := &Config{
		PineconeAPIKey: getEnvRequired("PINECONE_API_KEY"),
		PineconeHost:   getEnvRequired("PINECONE_HOST"),
		NotesDir:       getEnvRequired("NOTES_DIR"),
		EmbeddingUrl:   os.Getenv("EMBEDDING_URL"),
	}

	if err := config.validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return config, nil
}

func (c *Config) validate() error {
	if c.PineconeAPIKey == "" {
		return fmt.Errorf("PINECONE_API_KEY is required")
	}
	if c.PineconeHost == "" {
		return fmt.Errorf("PINECONE_HOST is required")
	}
	if c.NotesDir == "" {
		return fmt.Errorf("NOTES_DIR is required")
	}
	if c.EmbeddingUrl == "" {
		c.EmbeddingUrl = "http://localhost:8000/embed" // default value
	}
	return nil
}

// Helper function
func getEnvRequired(key string) string {
	value := os.Getenv(key)
	if value == "" {
		panic(fmt.Sprintf("Required environment variable %s is not set", key))
	}
	return value
}
