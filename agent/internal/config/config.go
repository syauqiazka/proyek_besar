package config

import (
	"fmt"
	"os"
)

type Config struct {
	ServerID     string
	APIKey       string
	BackendURL   string
	ScanInterval string // Cron format: "0 * * * *" = hourly
	MountPaths   []string
}

func Load() (*Config, error) {
	serverID := os.Getenv("HERMES_SERVER_ID")
	if serverID == "" {
		return nil, fmt.Errorf("HERMES_SERVER_ID environment variable is required")
	}

	apiKey := os.Getenv("HERMES_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("HERMES_API_KEY environment variable is required")
	}

	backendURL := os.Getenv("HERMES_BACKEND_URL")
	if backendURL == "" {
		backendURL = "http://localhost:3000/api/v1"
	}

	scanInterval := os.Getenv("HERMES_SCAN_INTERVAL")
	if scanInterval == "" {
		scanInterval = "0 * * * *" // Default: every hour
	}

	// Default mount paths for Linux servers
	mountPaths := []string{"/", "/var", "/home"}
	
	return &Config{
		ServerID:     serverID,
		APIKey:       apiKey,
		BackendURL:   backendURL,
		ScanInterval: scanInterval,
		MountPaths:   mountPaths,
	}, nil
}
