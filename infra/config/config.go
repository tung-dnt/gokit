// Package config loads application configuration from environment variables.
package config

import (
	"time"
)

// Config holds all application configuration.
type Config struct {
	Server ServerConfig
}

// ServerConfig holds HTTP server configuration.
type ServerConfig struct {
	Host         string
	Port         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

// Load reads configuration from environment variables with sensible defaults.
// getenv is injected so callers (and tests) can supply their own env source.
func Load(getenv func(string) string) *Config {
	return &Config{
		Server: ServerConfig{
			Host:         getEnv(getenv, "SERVER_HOST", "0.0.0.0"),
			Port:         getEnv(getenv, "SERVER_PORT", "8080"),
			ReadTimeout:  getDurationEnv(getenv, "SERVER_READ_TIMEOUT", 10*time.Second),
			WriteTimeout: getDurationEnv(getenv, "SERVER_WRITE_TIMEOUT", 10*time.Second),
			IdleTimeout:  getDurationEnv(getenv, "SERVER_IDLE_TIMEOUT", 60*time.Second),
		},
	}
}

func getEnv(getenv func(string) string, key, defaultVal string) string {
	if v := getenv(key); v != "" {
		return v
	}
	return defaultVal
}

func getDurationEnv(getenv func(string) string, key string, defaultVal time.Duration) time.Duration {
	v := getenv(key)
	if v == "" {
		return defaultVal
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return defaultVal
	}
	return d
}
