package config

import (
	"time"
)

// Config represents the complete server configuration
type Config struct {
	Server   ServerConfig   `yaml:"server" json:"server"`
	Logging  LoggingConfig  `yaml:"logging" json:"logging"`
	OpenAPI  OpenAPIConfig  `yaml:"openapi" json:"openapi"`
	Security SecurityConfig `yaml:"security" json:"security"`
}

// ServerConfig contains server-specific configuration
type ServerConfig struct {
	Transport string     `yaml:"transport" json:"transport"`
	HTTP      HTTPConfig `yaml:"http" json:"http"`
}

// HTTPConfig contains MCP-compliant HTTP transport configuration
type HTTPConfig struct {
	Host           string        `yaml:"host" json:"host"`
	Port           int           `yaml:"port" json:"port"`
	SessionTimeout time.Duration `yaml:"session_timeout" json:"session_timeout"`
	MaxConnections int           `yaml:"max_connections" json:"max_connections"`
	CORS           CORSConfig    `yaml:"cors" json:"cors"`
}

// CORSConfig contains CORS configuration
type CORSConfig struct {
	Enabled bool     `yaml:"enabled" json:"enabled"`
	Origins []string `yaml:"origins" json:"origins"`
}

// LoggingConfig contains logging configuration
type LoggingConfig struct {
	Level  string `yaml:"level" json:"level"`
	Format string `yaml:"format" json:"format"`
	Output string `yaml:"output" json:"output"`
}

// OpenAPIConfig contains OpenAPI-specific configuration
type OpenAPIConfig struct {
	SpecPath     string            `yaml:"spec_path" json:"spec_path"`
	BaseURL      string            `yaml:"base_url" json:"base_url"`
	Auth         AuthConfig        `yaml:"auth" json:"auth"`
	Headers      map[string]string `yaml:"headers" json:"headers"`
	Timeout      time.Duration     `yaml:"timeout" json:"timeout"`
	MaxRetries   int               `yaml:"max_retries" json:"max_retries"`
	ToolPrefix   string            `yaml:"tool_prefix" json:"tool_prefix"`
	ExcludePaths []string          `yaml:"exclude_paths" json:"exclude_paths"`
	IncludePaths []string          `yaml:"include_paths" json:"include_paths"`
}

// AuthConfig contains authentication configuration
type AuthConfig struct {
	Type       string            `yaml:"type" json:"type"` // "none", "bearer", "basic", "api_key"
	Token      string            `yaml:"token" json:"token"`
	Username   string            `yaml:"username" json:"username"`
	Password   string            `yaml:"password" json:"password"`
	APIKey     string            `yaml:"api_key" json:"api_key"`
	APIKeyName string            `yaml:"api_key_name" json:"api_key_name"`
	APIKeyIn   string            `yaml:"api_key_in" json:"api_key_in"` // "header", "query"
	Headers    map[string]string `yaml:"headers" json:"headers"`
}

// SecurityConfig contains security configuration
type SecurityConfig struct {
	RateLimiting     RateLimitingConfig `yaml:"rate_limiting" json:"rate_limiting"`
	RequestSizeLimit string             `yaml:"request_size_limit" json:"request_size_limit"`
}

// RateLimitingConfig contains rate limiting configuration
type RateLimitingConfig struct {
	Enabled           bool `yaml:"enabled" json:"enabled"`
	RequestsPerMinute int  `yaml:"requests_per_minute" json:"requests_per_minute"`
}

// Default returns a configuration with default values
func Default() *Config {
	return &Config{
		Server: ServerConfig{
			Transport: "stdio",
			HTTP: HTTPConfig{
				Host:           "127.0.0.1",
				Port:           8080,
				SessionTimeout: 5 * time.Minute,
				MaxConnections: 100,
				CORS: CORSConfig{
					Enabled: true,
					Origins: []string{"http://localhost:3000", "http://127.0.0.1:3000"},
				},
			},
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "json",
			Output: "stdout",
		},
		OpenAPI: OpenAPIConfig{
			SpecPath:   "",
			BaseURL:    "",
			Timeout:    30 * time.Second,
			MaxRetries: 3,
			ToolPrefix: "",
			Auth: AuthConfig{
				Type: "none",
			},
			Headers: make(map[string]string),
		},
		Security: SecurityConfig{
			RateLimiting: RateLimitingConfig{
				Enabled:           true,
				RequestsPerMinute: 100,
			},
			RequestSizeLimit: "1MB",
		},
	}
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.Server.Transport != "stdio" && c.Server.Transport != "http" {
		return ErrInvalidTransport
	}

	if c.Server.HTTP.Port < 1 || c.Server.HTTP.Port > 65535 {
		return ErrInvalidPort
	}

	if c.OpenAPI.SpecPath == "" {
		return ErrMissingOpenAPISpec
	}

	if c.OpenAPI.Timeout < 1*time.Second {
		return ErrInvalidTimeout
	}

	if c.OpenAPI.MaxRetries < 0 {
		return ErrInvalidMaxRetries
	}

	if c.Security.RateLimiting.RequestsPerMinute < 1 {
		return ErrInvalidRateLimit
	}

	return nil
}
