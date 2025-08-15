package http

import (
	"net/http"

	"gemini-antiblock/config"
	"gemini-antiblock/logger"
)

// ClientManager manages HTTP client with optimized configuration
type ClientManager struct {
	client *http.Client
	config *config.Config
}

// NewClientManager creates a new HTTP client manager with optimized settings
func NewClientManager(cfg *config.Config) *ClientManager {
	logger.LogInfo("Initializing HTTP client manager with optimized settings")
	logger.LogDebug("HTTP Timeout:", cfg.HTTPTimeout)
	logger.LogDebug("HTTP Idle Connection Timeout:", cfg.HTTPIdleConnTimeout)
	logger.LogDebug("HTTP Max Idle Connections:", cfg.HTTPMaxIdleConns)
	logger.LogDebug("HTTP Max Connections Per Host:", cfg.HTTPMaxConnsPerHost)

	// Configure transport with connection pooling and timeouts
	transport := &http.Transport{
		MaxIdleConns:        cfg.HTTPMaxIdleConns,
		MaxConnsPerHost:     cfg.HTTPMaxConnsPerHost,
		IdleConnTimeout:     cfg.HTTPIdleConnTimeout,
		DisableCompression:  false,
		ForceAttemptHTTP2:   true,
		DisableKeepAlives:   false,
		MaxIdleConnsPerHost: cfg.HTTPMaxConnsPerHost,
	}

	// Create client with timeout
	client := &http.Client{
		Transport: transport,
		Timeout:   cfg.HTTPTimeout,
	}

	logger.LogInfo("HTTP client manager initialized successfully")

	return &ClientManager{
		client: client,
		config: cfg,
	}
}

// GetClient returns the configured HTTP client
func (cm *ClientManager) GetClient() *http.Client {
	return cm.client
}

// GetConfig returns the configuration
func (cm *ClientManager) GetConfig() *config.Config {
	return cm.config
}
