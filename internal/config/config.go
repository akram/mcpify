package config

import (
	"encoding/json"
	"fmt"
	"time"

	"gopkg.in/yaml.v3"
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

// UnmarshalJSON implements custom JSON unmarshaling for HTTPConfig
func (h *HTTPConfig) UnmarshalJSON(data []byte) error {
	type Alias HTTPConfig
	aux := &struct {
		SessionTimeout string `json:"session_timeout"`
		*Alias
	}{
		Alias: (*Alias)(h),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	if aux.SessionTimeout != "" {
		duration, err := time.ParseDuration(aux.SessionTimeout)
		if err != nil {
			return err
		}
		h.SessionTimeout = duration
	}

	return nil
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

// HeaderConfig represents a single header configuration
type HeaderConfig struct {
	Name      string `yaml:"name" json:"name"`
	Value     string `yaml:"value,omitempty" json:"value,omitempty"`
	ValueFrom string `yaml:"valueFrom,omitempty" json:"valueFrom,omitempty"`
}

// UnmarshalYAML implements custom YAML unmarshaling for HeaderConfig
func (h *HeaderConfig) UnmarshalYAML(value *yaml.Node) error {
	var aux struct {
		Name      string `yaml:"name"`
		Value     string `yaml:"value,omitempty"`
		ValueFrom string `yaml:"valueFrom,omitempty"`
	}

	if err := value.Decode(&aux); err != nil {
		return err
	}

	h.Name = aux.Name
	h.Value = aux.Value
	h.ValueFrom = aux.ValueFrom

	return h.Validate()
}

// UnmarshalJSON implements custom JSON unmarshaling for HeaderConfig
func (h *HeaderConfig) UnmarshalJSON(data []byte) error {
	var aux struct {
		Name      string `json:"name"`
		Value     string `json:"value,omitempty"`
		ValueFrom string `json:"valueFrom,omitempty"`
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	h.Name = aux.Name
	h.Value = aux.Value
	h.ValueFrom = aux.ValueFrom

	return h.Validate()
}

// Validate validates the HeaderConfig
func (h *HeaderConfig) Validate() error {
	if h.Name == "" {
		return fmt.Errorf("header name is required")
	}

	hasValue := h.Value != ""
	hasValueFrom := h.ValueFrom != ""

	if !hasValue && !hasValueFrom {
		return fmt.Errorf("header must have either 'value' or 'valueFrom'")
	}

	if hasValue && hasValueFrom {
		return fmt.Errorf("header cannot have both 'value' and 'valueFrom'")
	}

	return nil
}

// HeaderItem represents a header item in the configuration
type HeaderItem struct {
	Header HeaderConfig `yaml:"header" json:"header"`
}

// HeadersConfig represents a list of header configurations
type HeadersConfig []HeaderItem

// UnmarshalYAML implements custom YAML unmarshaling for HeadersConfig
func (h *HeadersConfig) UnmarshalYAML(value *yaml.Node) error {
	// Try to unmarshal as new format first (array of header items)
	var items []HeaderItem
	if err := value.Decode(&items); err == nil {
		*h = HeadersConfig(items)
		return h.Validate()
	}

	// Fall back to old format (map of string to string)
	var oldFormat map[string]string
	if err := value.Decode(&oldFormat); err != nil {
		return err
	}

	// Convert old format to new format
	items = make([]HeaderItem, 0, len(oldFormat))
	for name, value := range oldFormat {
		items = append(items, HeaderItem{
			Header: HeaderConfig{
				Name:  name,
				Value: value,
			},
		})
	}

	*h = HeadersConfig(items)
	return h.Validate()
}

// UnmarshalJSON implements custom JSON unmarshaling for HeadersConfig
func (h *HeadersConfig) UnmarshalJSON(data []byte) error {
	// Try to unmarshal as new format first (array of header items)
	var items []HeaderItem
	if err := json.Unmarshal(data, &items); err == nil {
		*h = HeadersConfig(items)
		return h.Validate()
	}

	// Fall back to old format (map of string to string)
	var oldFormat map[string]string
	if err := json.Unmarshal(data, &oldFormat); err != nil {
		return err
	}

	// Convert old format to new format
	items = make([]HeaderItem, 0, len(oldFormat))
	for name, value := range oldFormat {
		items = append(items, HeaderItem{
			Header: HeaderConfig{
				Name:  name,
				Value: value,
			},
		})
	}

	*h = HeadersConfig(items)
	return h.Validate()
}

// Validate validates the HeadersConfig
func (h *HeadersConfig) Validate() error {
	seen := make(map[string]bool)
	for _, item := range *h {
		if err := item.Header.Validate(); err != nil {
			return err
		}
		if seen[item.Header.Name] {
			return fmt.Errorf("duplicate header name: %s", item.Header.Name)
		}
		seen[item.Header.Name] = true
	}
	return nil
}

// GetValue returns the value for a header name, or empty string if not found
func (h *HeadersConfig) GetValue(name string) string {
	for _, item := range *h {
		if item.Header.Name == name {
			return item.Header.Value
		}
	}
	return ""
}

// ToMap converts HeadersConfig to a map for backward compatibility
func (h *HeadersConfig) ToMap() map[string]string {
	result := make(map[string]string)
	for _, item := range *h {
		if item.Header.Value != "" {
			result[item.Header.Name] = item.Header.Value
		}
	}
	return result
}

// OpenAPIConfig contains OpenAPI-specific configuration
type OpenAPIConfig struct {
	SpecPath     string        `yaml:"spec_path" json:"spec_path"`
	BaseURL      string        `yaml:"base_url" json:"base_url"`
	Auth         AuthConfig    `yaml:"auth" json:"auth"`
	Headers      HeadersConfig `yaml:"headers" json:"headers"`
	Timeout      time.Duration `yaml:"timeout" json:"timeout"`
	MaxRetries   int           `yaml:"max_retries" json:"max_retries"`
	ToolPrefix   string        `yaml:"tool_prefix" json:"tool_prefix"`
	ExcludePaths []string      `yaml:"exclude_paths" json:"exclude_paths"`
	IncludePaths []string      `yaml:"include_paths" json:"include_paths"`
}

// UnmarshalJSON implements custom JSON unmarshaling for OpenAPIConfig
func (o *OpenAPIConfig) UnmarshalJSON(data []byte) error {
	type Alias OpenAPIConfig
	aux := &struct {
		Timeout string `json:"timeout"`
		*Alias
	}{
		Alias: (*Alias)(o),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	if aux.Timeout != "" {
		duration, err := time.ParseDuration(aux.Timeout)
		if err != nil {
			return err
		}
		o.Timeout = duration
	}

	return nil
}

// AuthConfig contains authentication configuration
type AuthConfig struct {
	Type       string        `yaml:"type" json:"type"` // "none", "bearer", "basic", "api_key"
	Token      string        `yaml:"token" json:"token"`
	Username   string        `yaml:"username" json:"username"`
	Password   string        `yaml:"password" json:"password"`
	APIKey     string        `yaml:"api_key" json:"api_key"`
	APIKeyName string        `yaml:"api_key_name" json:"api_key_name"`
	APIKeyIn   string        `yaml:"api_key_in" json:"api_key_in"` // "header", "query"
	Headers    HeadersConfig `yaml:"headers" json:"headers"`
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
				Type:    "none",
				Headers: HeadersConfig{},
			},
			Headers: HeadersConfig{},
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

	// Validate OpenAPI config
	if err := c.OpenAPI.Validate(); err != nil {
		return err
	}

	return nil
}

// Validate validates the OpenAPIConfig
func (o *OpenAPIConfig) Validate() error {
	// Validate headers
	if err := o.Headers.Validate(); err != nil {
		return fmt.Errorf("invalid headers: %w", err)
	}

	// Validate auth headers
	if err := o.Auth.Headers.Validate(); err != nil {
		return fmt.Errorf("invalid auth headers: %w", err)
	}

	// Check for duplicate header names between auth and general headers
	authHeaderNames := make(map[string]bool)
	for _, item := range o.Auth.Headers {
		authHeaderNames[item.Header.Name] = true
	}

	for _, item := range o.Headers {
		if authHeaderNames[item.Header.Name] {
			return fmt.Errorf("duplicate header name between auth and general headers: %s", item.Header.Name)
		}
	}

	return nil
}
