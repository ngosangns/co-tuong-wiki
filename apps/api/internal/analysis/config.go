package analysis

import (
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultEngineTimeMS = 500
	maxEngineTimeMS     = 10000
	engineTimeoutSlack  = 3 * time.Second
)

type Config struct {
	Kind          string
	Path          string
	DefaultTimeMS int
	MaxTimeMS     int
	DefaultDepth  int
	MaxDepth      int
	TimeoutSlack  time.Duration
}

func ConfigFromEnv() Config {
	return Config{
		Kind:          strings.ToLower(env("ENGINE_KIND", "uci")),
		Path:          os.Getenv("ENGINE_PATH"),
		DefaultTimeMS: envInt("ENGINE_DEFAULT_TIME_MS", defaultEngineTimeMS),
		MaxTimeMS:     envInt("ENGINE_MAX_TIME_MS", maxEngineTimeMS),
		DefaultDepth:  envInt("ENGINE_DEFAULT_DEPTH", 8),
		MaxDepth:      envInt("ENGINE_MAX_DEPTH", 12),
		TimeoutSlack:  engineTimeoutSlack,
	}
}

func env(key string, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func envInt(key string, fallback int) int {
	value, err := strconv.Atoi(os.Getenv(key))
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}
