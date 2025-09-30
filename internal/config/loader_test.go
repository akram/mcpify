package config

import (
	"os"
	"testing"
	"time"
)

func TestNewLoader(t *testing.T) {
	loader := NewLoader()
	if loader == nil {
		t.Error("Expected loader to be created, got nil")
	}
}

func TestLoad_EmptyPath(t *testing.T) {
	loader := NewLoader()
	config, err := loader.Load("")

	if err != nil {
		t.Errorf("Expected no error for empty path, got %v", err)
	}

	if config == nil {
		t.Error("Expected default config, got nil")
		return
	}

	// Should return default config
	defaultConfig := Default()
	if config.Server.Transport != defaultConfig.Server.Transport {
		t.Errorf("Expected default transport, got %s", config.Server.Transport)
	}
}

func TestLoad_NonExistentFile(t *testing.T) {
	loader := NewLoader()
	config, err := loader.Load("nonexistent.yaml")

	if err == nil {
		t.Error("Expected error for non-existent file, got nil")
	}

	if config != nil {
		t.Error("Expected nil config for non-existent file")
	}

	expectedErr := "configuration file not found: nonexistent.yaml"
	if err.Error() != expectedErr {
		t.Errorf("Expected error '%s', got '%s'", expectedErr, err.Error())
	}
}

func TestLoad_UnsupportedFormat(t *testing.T) {
	// Create a temporary file with unsupported extension
	tmpFile, err := os.CreateTemp("", "test_config.*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer func() {
		_ = os.Remove(tmpFile.Name())
	}()

	loader := NewLoader()
	config, err := loader.Load(tmpFile.Name())

	if err == nil {
		t.Error("Expected error for unsupported format, got nil")
	}

	if config != nil {
		t.Error("Expected nil config for unsupported format")
	}

	expectedErr := "unsupported configuration file format: .txt"
	if err.Error() != expectedErr {
		t.Errorf("Expected error '%s', got '%s'", expectedErr, err.Error())
	}
}

func TestLoad_YAML(t *testing.T) {
	// Create a temporary YAML file
	yamlContent := `
server:
  transport: "http"
  http:
    host: "0.0.0.0"
    port: 9000
    session_timeout: "10m"
    max_connections: 200
    cors:
      enabled: false
      origins:
        - "https://example.com"

logging:
  level: "debug"
  format: "text"
  output: "stderr"

openapi:
  spec_path: "https://api.example.com/openapi.json"
  base_url: "https://api.example.com"
  timeout: "60s"
  max_retries: 5
  tool_prefix: "test_"
  auth:
    type: "bearer"
    token: "test-token"
    headers:
      "X-Custom": "value"
  headers:
    "User-Agent": "Test/1.0"
  exclude_paths:
    - "/health"
    - "/metrics"
  include_paths:
    - "/api/v1/*"

security:
  rate_limiting:
    enabled: false
    requests_per_minute: 200
  request_size_limit: "2MB"
`

	tmpFile, err := os.CreateTemp("", "test_config.*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer func() {
		_ = os.Remove(tmpFile.Name())
	}()

	if _, err := tmpFile.WriteString(yamlContent); err != nil {
		t.Fatalf("Failed to write YAML content: %v", err)
	}
	_ = tmpFile.Close()

	loader := NewLoader()
	config, err := loader.Load(tmpFile.Name())

	if err != nil {
		t.Errorf("Expected no error loading YAML, got %v", err)
	}

	if config == nil {
		t.Error("Expected config, got nil")
		return
	}

	// Test loaded values
	if config.Server.Transport != "http" {
		t.Errorf("Expected transport 'http', got '%s'", config.Server.Transport)
	}

	if config.Server.HTTP.Host != "0.0.0.0" {
		t.Errorf("Expected host '0.0.0.0', got '%s'", config.Server.HTTP.Host)
	}

	if config.Server.HTTP.Port != 9000 {
		t.Errorf("Expected port 9000, got %d", config.Server.HTTP.Port)
	}

	if config.Server.HTTP.SessionTimeout != 10*time.Minute {
		t.Errorf("Expected session timeout 10m, got %v", config.Server.HTTP.SessionTimeout)
	}

	if config.Server.HTTP.MaxConnections != 200 {
		t.Errorf("Expected max connections 200, got %d", config.Server.HTTP.MaxConnections)
	}

	if config.Server.HTTP.CORS.Enabled {
		t.Error("Expected CORS to be disabled")
	}

	if len(config.Server.HTTP.CORS.Origins) != 1 || config.Server.HTTP.CORS.Origins[0] != "https://example.com" {
		t.Errorf("Expected CORS origins ['https://example.com'], got %v", config.Server.HTTP.CORS.Origins)
	}

	if config.Logging.Level != "debug" {
		t.Errorf("Expected log level 'debug', got '%s'", config.Logging.Level)
	}

	if config.Logging.Format != "text" {
		t.Errorf("Expected log format 'text', got '%s'", config.Logging.Format)
	}

	if config.Logging.Output != "stderr" {
		t.Errorf("Expected log output 'stderr', got '%s'", config.Logging.Output)
	}

	if config.OpenAPI.SpecPath != "https://api.example.com/openapi.json" {
		t.Errorf("Expected spec path 'https://api.example.com/openapi.json', got '%s'", config.OpenAPI.SpecPath)
	}

	if config.OpenAPI.BaseURL != "https://api.example.com" {
		t.Errorf("Expected base URL 'https://api.example.com', got '%s'", config.OpenAPI.BaseURL)
	}

	if config.OpenAPI.Timeout != 60*time.Second {
		t.Errorf("Expected timeout 60s, got %v", config.OpenAPI.Timeout)
	}

	if config.OpenAPI.MaxRetries != 5 {
		t.Errorf("Expected max retries 5, got %d", config.OpenAPI.MaxRetries)
	}

	if config.OpenAPI.ToolPrefix != "test_" {
		t.Errorf("Expected tool prefix 'test_', got '%s'", config.OpenAPI.ToolPrefix)
	}

	if config.OpenAPI.Auth.Type != "bearer" {
		t.Errorf("Expected auth type 'bearer', got '%s'", config.OpenAPI.Auth.Type)
	}

	if config.OpenAPI.Auth.Token != "test-token" {
		t.Errorf("Expected auth token 'test-token', got '%s'", config.OpenAPI.Auth.Token)
	}

	if config.OpenAPI.Auth.Headers.GetValue("X-Custom") != "value" {
		t.Errorf("Expected custom header 'value', got '%s'", config.OpenAPI.Auth.Headers.GetValue("X-Custom"))
	}

	if config.OpenAPI.Headers.GetValue("User-Agent") != "Test/1.0" {
		t.Errorf("Expected User-Agent 'Test/1.0', got '%s'", config.OpenAPI.Headers.GetValue("User-Agent"))
	}

	if len(config.OpenAPI.ExcludePaths) != 2 {
		t.Errorf("Expected 2 exclude paths, got %d", len(config.OpenAPI.ExcludePaths))
	}

	if len(config.OpenAPI.IncludePaths) != 1 {
		t.Errorf("Expected 1 include path, got %d", len(config.OpenAPI.IncludePaths))
	}

	if config.Security.RateLimiting.Enabled {
		t.Error("Expected rate limiting to be disabled")
	}

	if config.Security.RateLimiting.RequestsPerMinute != 200 {
		t.Errorf("Expected requests per minute 200, got %d", config.Security.RateLimiting.RequestsPerMinute)
	}

	if config.Security.RequestSizeLimit != "2MB" {
		t.Errorf("Expected request size limit '2MB', got '%s'", config.Security.RequestSizeLimit)
	}
}

func TestLoad_JSON(t *testing.T) {
	// Create a temporary JSON file
	jsonContent := `{
  "server": {
    "transport": "http",
    "http": {
      "host": "localhost",
      "port": 3000,
      "session_timeout": "5m",
      "max_connections": 50,
      "cors": {
        "enabled": true,
        "origins": ["http://localhost:3000", "http://localhost:3001"]
      }
    }
  },
  "logging": {
    "level": "warn",
    "format": "json",
    "output": "stdout"
  },
  "openapi": {
    "spec_path": "https://api.test.com/openapi.json",
    "base_url": "https://api.test.com",
    "timeout": "45s",
    "max_retries": 2,
    "tool_prefix": "json_",
    "auth": {
      "type": "api_key",
      "api_key": "test-key",
      "api_key_name": "X-API-Key",
      "api_key_in": "header"
    },
    "headers": {
      "Accept": "application/json"
    },
    "exclude_paths": ["/health", "/status"],
    "include_paths": ["/api/*"]
  },
  "security": {
    "rate_limiting": {
      "enabled": true,
      "requests_per_minute": 150
    },
    "request_size_limit": "500KB"
  }
}`

	tmpFile, err := os.CreateTemp("", "test_config.*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer func() {
		_ = os.Remove(tmpFile.Name())
	}()

	if _, err := tmpFile.WriteString(jsonContent); err != nil {
		t.Fatalf("Failed to write JSON content: %v", err)
	}
	_ = tmpFile.Close()

	loader := NewLoader()
	config, err := loader.Load(tmpFile.Name())

	if err != nil {
		t.Errorf("Expected no error loading JSON, got %v", err)
	}

	if config == nil {
		t.Error("Expected config, got nil")
		return
	}

	// Test loaded values
	if config.Server.Transport != "http" {
		t.Errorf("Expected transport 'http', got '%s'", config.Server.Transport)
	}

	if config.Server.HTTP.Host != "localhost" {
		t.Errorf("Expected host 'localhost', got '%s'", config.Server.HTTP.Host)
	}

	if config.Server.HTTP.Port != 3000 {
		t.Errorf("Expected port 3000, got %d", config.Server.HTTP.Port)
	}

	if config.Server.HTTP.SessionTimeout != 5*time.Minute {
		t.Errorf("Expected session timeout 5m, got %v", config.Server.HTTP.SessionTimeout)
	}

	if config.Server.HTTP.MaxConnections != 50 {
		t.Errorf("Expected max connections 50, got %d", config.Server.HTTP.MaxConnections)
	}

	if !config.Server.HTTP.CORS.Enabled {
		t.Error("Expected CORS to be enabled")
	}

	if len(config.Server.HTTP.CORS.Origins) != 2 {
		t.Errorf("Expected 2 CORS origins, got %d", len(config.Server.HTTP.CORS.Origins))
	}

	if config.Logging.Level != "warn" {
		t.Errorf("Expected log level 'warn', got '%s'", config.Logging.Level)
	}

	if config.OpenAPI.SpecPath != "https://api.test.com/openapi.json" {
		t.Errorf("Expected spec path 'https://api.test.com/openapi.json', got '%s'", config.OpenAPI.SpecPath)
	}

	if config.OpenAPI.BaseURL != "https://api.test.com" {
		t.Errorf("Expected base URL 'https://api.test.com', got '%s'", config.OpenAPI.BaseURL)
	}

	if config.OpenAPI.Timeout != 45*time.Second {
		t.Errorf("Expected timeout 45s, got %v", config.OpenAPI.Timeout)
	}

	if config.OpenAPI.MaxRetries != 2 {
		t.Errorf("Expected max retries 2, got %d", config.OpenAPI.MaxRetries)
	}

	if config.OpenAPI.ToolPrefix != "json_" {
		t.Errorf("Expected tool prefix 'json_', got '%s'", config.OpenAPI.ToolPrefix)
	}

	if config.OpenAPI.Auth.Type != "api_key" {
		t.Errorf("Expected auth type 'api_key', got '%s'", config.OpenAPI.Auth.Type)
	}

	if config.OpenAPI.Auth.APIKey != "test-key" {
		t.Errorf("Expected API key 'test-key', got '%s'", config.OpenAPI.Auth.APIKey)
	}

	if config.OpenAPI.Auth.APIKeyName != "X-API-Key" {
		t.Errorf("Expected API key name 'X-API-Key', got '%s'", config.OpenAPI.Auth.APIKeyName)
	}

	if config.OpenAPI.Auth.APIKeyIn != "header" {
		t.Errorf("Expected API key in 'header', got '%s'", config.OpenAPI.Auth.APIKeyIn)
	}

	if config.OpenAPI.Headers.GetValue("Accept") != "application/json" {
		t.Errorf("Expected Accept header 'application/json', got '%s'", config.OpenAPI.Headers.GetValue("Accept"))
	}

	if len(config.OpenAPI.ExcludePaths) != 2 {
		t.Errorf("Expected 2 exclude paths, got %d", len(config.OpenAPI.ExcludePaths))
	}

	if len(config.OpenAPI.IncludePaths) != 1 {
		t.Errorf("Expected 1 include path, got %d", len(config.OpenAPI.IncludePaths))
	}

	if !config.Security.RateLimiting.Enabled {
		t.Error("Expected rate limiting to be enabled")
	}

	if config.Security.RateLimiting.RequestsPerMinute != 150 {
		t.Errorf("Expected requests per minute 150, got %d", config.Security.RateLimiting.RequestsPerMinute)
	}

	if config.Security.RequestSizeLimit != "500KB" {
		t.Errorf("Expected request size limit '500KB', got '%s'", config.Security.RequestSizeLimit)
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	// Create a temporary file with invalid YAML
	tmpFile, err := os.CreateTemp("", "test_config.*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer func() {
		_ = os.Remove(tmpFile.Name())
	}()

	invalidYAML := `
server:
  transport: "http"
  http:
    port: invalid_port  # This should cause a parsing error
`

	if _, err := tmpFile.WriteString(invalidYAML); err != nil {
		t.Fatalf("Failed to write invalid YAML content: %v", err)
	}
	_ = tmpFile.Close()

	loader := NewLoader()
	config, err := loader.Load(tmpFile.Name())

	if err == nil {
		t.Error("Expected error for invalid YAML, got nil")
	}

	if config != nil {
		t.Error("Expected nil config for invalid YAML")
	}
}

func TestLoad_InvalidJSON(t *testing.T) {
	// Create a temporary file with invalid JSON
	tmpFile, err := os.CreateTemp("", "test_config.*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer func() {
		_ = os.Remove(tmpFile.Name())
	}()

	invalidJSON := `{
  "server": {
    "transport": "http"
    "http": {  // Missing comma
      "port": 8080
    }
  }
}`

	if _, err := tmpFile.WriteString(invalidJSON); err != nil {
		t.Fatalf("Failed to write invalid JSON content: %v", err)
	}
	_ = tmpFile.Close()

	loader := NewLoader()
	config, err := loader.Load(tmpFile.Name())

	if err == nil {
		t.Error("Expected error for invalid JSON, got nil")
	}

	if config != nil {
		t.Error("Expected nil config for invalid JSON")
	}
}

func TestMergeWithDefaults(t *testing.T) {
	loader := NewLoader()

	// Test partial config that should merge with defaults
	partialConfig := Config{
		Server: ServerConfig{
			Transport: "http",
			HTTP: HTTPConfig{
				Port: 9000,
				// Other fields should be filled with defaults
			},
		},
		Logging: LoggingConfig{
			Level: "debug",
			// Other fields should be filled with defaults
		},
		OpenAPI: OpenAPIConfig{
			SpecPath: "https://api.example.com/openapi.json",
			// Other fields should be filled with defaults
		},
	}

	merged := loader.mergeWithDefaults(partialConfig)

	// Test that defaults were applied
	if merged.Server.HTTP.Host != "127.0.0.1" {
		t.Errorf("Expected default host, got %s", merged.Server.HTTP.Host)
	}

	if merged.Server.HTTP.SessionTimeout != 5*time.Minute {
		t.Errorf("Expected default session timeout, got %v", merged.Server.HTTP.SessionTimeout)
	}

	if merged.Server.HTTP.MaxConnections != 100 {
		t.Errorf("Expected default max connections, got %d", merged.Server.HTTP.MaxConnections)
	}

	if !merged.Server.HTTP.CORS.Enabled {
		t.Error("Expected default CORS to be enabled")
	}

	if len(merged.Server.HTTP.CORS.Origins) == 0 {
		t.Error("Expected default CORS origins")
	}

	if merged.Logging.Format != "json" {
		t.Errorf("Expected default log format, got %s", merged.Logging.Format)
	}

	if merged.Logging.Output != "stdout" {
		t.Errorf("Expected default log output, got %s", merged.Logging.Output)
	}

	if merged.OpenAPI.Timeout != 30*time.Second {
		t.Errorf("Expected default timeout, got %v", merged.OpenAPI.Timeout)
	}

	if merged.OpenAPI.MaxRetries != 3 {
		t.Errorf("Expected default max retries, got %d", merged.OpenAPI.MaxRetries)
	}

	if merged.OpenAPI.Auth.Type != "none" {
		t.Errorf("Expected default auth type, got %s", merged.OpenAPI.Auth.Type)
	}

	if merged.OpenAPI.Headers == nil {
		t.Error("Expected headers to be initialized")
	}

	if !merged.Security.RateLimiting.Enabled {
		t.Error("Expected default rate limiting to be enabled")
	}

	if merged.Security.RateLimiting.RequestsPerMinute != 100 {
		t.Errorf("Expected default requests per minute, got %d", merged.Security.RateLimiting.RequestsPerMinute)
	}

	if merged.Security.RequestSizeLimit != "1MB" {
		t.Errorf("Expected default request size limit, got %s", merged.Security.RequestSizeLimit)
	}

	// Test that provided values were preserved
	if merged.Server.Transport != "http" {
		t.Errorf("Expected preserved transport, got %s", merged.Server.Transport)
	}

	if merged.Server.HTTP.Port != 9000 {
		t.Errorf("Expected preserved port, got %d", merged.Server.HTTP.Port)
	}

	if merged.Logging.Level != "debug" {
		t.Errorf("Expected preserved log level, got %s", merged.Logging.Level)
	}

	if merged.OpenAPI.SpecPath != "https://api.example.com/openapi.json" {
		t.Errorf("Expected preserved spec path, got %s", merged.OpenAPI.SpecPath)
	}
}
