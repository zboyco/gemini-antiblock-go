package config

import (
	"os"
	"strconv"
	"time"
)

// Config holds all configuration values
type Config struct {
	UpstreamURLBase           string
	MaxConsecutiveRetries     int
	DebugMode                 bool
	RetryDelayMs              time.Duration
	SwallowThoughtsAfterRetry bool
	Port                      string

	// HTTP Client Configuration
	HTTPTimeout         time.Duration
	HTTPIdleConnTimeout time.Duration
	HTTPMaxIdleConns    int
	HTTPMaxConnsPerHost int
	JSONBufferSize      int

	// Stream Processing Configuration
	SSEBufferSize int
}

// LoadConfig loads configuration from environment variables
func LoadConfig() *Config {
	return &Config{
		UpstreamURLBase:           getEnvString("UPSTREAM_URL_BASE", "https://generativelanguage.googleapis.com"),
		MaxConsecutiveRetries:     getEnvInt("MAX_CONSECUTIVE_RETRIES", 100),
		DebugMode:                 getEnvBool("DEBUG_MODE", true),
		RetryDelayMs:              time.Duration(getEnvInt("RETRY_DELAY_MS", 750)) * time.Millisecond,
		SwallowThoughtsAfterRetry: getEnvBool("SWALLOW_THOUGHTS_AFTER_RETRY", true),
		Port:                      getEnvString("PORT", "8080"),

		// HTTP Client Configuration
		HTTPTimeout:         time.Duration(getEnvInt("HTTP_TIMEOUT_SECONDS", 30)) * time.Second,
		HTTPIdleConnTimeout: time.Duration(getEnvInt("HTTP_IDLE_CONN_TIMEOUT_SECONDS", 90)) * time.Second,
		HTTPMaxIdleConns:    getEnvInt("HTTP_MAX_IDLE_CONNS", 100),
		HTTPMaxConnsPerHost: getEnvInt("HTTP_MAX_CONNS_PER_HOST", 10),
		JSONBufferSize:      getEnvInt("JSON_BUFFER_SIZE", 4096),

		// Stream Processing Configuration
		SSEBufferSize: getEnvInt("SSE_BUFFER_SIZE", 100),
	}
}

func getEnvString(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}
