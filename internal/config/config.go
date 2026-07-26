package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	HTTPAddr      string
	GroqAPIKey    string
	GroqModelName string
}

func Load() (Config, error) {
	_ = godotenv.Load()

	c := Config{
		HTTPAddr:      getenv("HTTP_ADDR", ":8080"),
		GroqAPIKey:    getenv("GROQ_API_KEY", ""),
		GroqModelName: getenv("GROQ_MODEL_NAME", "openai/gpt-oss-120b"),
	}

	if c.GroqAPIKey == "" {
		return Config{}, fmt.Errorf("config: GROQ_API_KEY is required")
	}

	return c, nil
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
