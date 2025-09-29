package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Loader handles configuration loading from various sources
type Loader struct{}

// NewLoader creates a new configuration loader
func NewLoader() *Loader {
	return &Loader{}
}

// Load loads configuration from a file or returns default config
func (l *Loader) Load(configPath string) (*Config, error) {
	// If no config path provided, return default config
	if configPath == "" {
		return Default(), nil
	}

	// Check if file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("configuration file not found: %s", configPath)
	}

	// Read file content
	content, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read configuration file: %w", err)
	}

	// Determine file format and parse
	ext := strings.ToLower(filepath.Ext(configPath))
	var config Config

	switch ext {
	case ".yaml", ".yml":
		err = yaml.Unmarshal(content, &config)
	case ".json":
		err = json.Unmarshal(content, &config)
	default:
		return nil, fmt.Errorf("unsupported configuration file format: %s", ext)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to parse configuration file: %w", err)
	}

	// Merge with defaults for missing values
	config = l.mergeWithDefaults(config)

	return &config, nil
}

// mergeWithDefaults merges the loaded config with default values
func (l *Loader) mergeWithDefaults(config Config) Config {
	defaults := Default()

	// Merge server config
	if config.Server.Transport == "" {
		config.Server.Transport = defaults.Server.Transport
	}
	if config.Server.HTTP.Host == "" {
		config.Server.HTTP.Host = defaults.Server.HTTP.Host
	}
	if config.Server.HTTP.Port == 0 {
		config.Server.HTTP.Port = defaults.Server.HTTP.Port
	}
	if config.Server.HTTP.SessionTimeout == 0 {
		config.Server.HTTP.SessionTimeout = defaults.Server.HTTP.SessionTimeout
	}
	if config.Server.HTTP.MaxConnections == 0 {
		config.Server.HTTP.MaxConnections = defaults.Server.HTTP.MaxConnections
	}
	// Apply CORS defaults if not explicitly configured
	if len(config.Server.HTTP.CORS.Origins) == 0 {
		config.Server.HTTP.CORS.Enabled = defaults.Server.HTTP.CORS.Enabled
		config.Server.HTTP.CORS.Origins = defaults.Server.HTTP.CORS.Origins
	}

	// Merge logging config
	if config.Logging.Level == "" {
		config.Logging.Level = defaults.Logging.Level
	}
	if config.Logging.Format == "" {
		config.Logging.Format = defaults.Logging.Format
	}
	if config.Logging.Output == "" {
		config.Logging.Output = defaults.Logging.Output
	}

	// Merge OpenAPI config
	if config.OpenAPI.Timeout == 0 {
		config.OpenAPI.Timeout = defaults.OpenAPI.Timeout
	}
	if config.OpenAPI.MaxRetries == 0 {
		config.OpenAPI.MaxRetries = defaults.OpenAPI.MaxRetries
	}
	// ToolPrefix defaults to empty string, no need to override
	if config.OpenAPI.Auth.Type == "" {
		config.OpenAPI.Auth.Type = defaults.OpenAPI.Auth.Type
	}
	if config.OpenAPI.Headers == nil {
		config.OpenAPI.Headers = make(map[string]string)
	}

	// Merge security config
	// Note: For boolean fields, we can't easily detect if they were explicitly set to false
	// So we'll always apply the default if the struct is zero-valued
	if config.Security.RateLimiting.RequestsPerMinute == 0 {
		config.Security.RateLimiting.Enabled = defaults.Security.RateLimiting.Enabled
		config.Security.RateLimiting.RequestsPerMinute = defaults.Security.RateLimiting.RequestsPerMinute
	}
	if config.Security.RequestSizeLimit == "" {
		config.Security.RequestSizeLimit = defaults.Security.RequestSizeLimit
	}

	return config
}
