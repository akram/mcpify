package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestEndToEnd_ConfigLoading tests the complete configuration loading pipeline
func TestEndToEnd_ConfigLoading(t *testing.T) {
	// Test loading all provided config files
	configFiles := []string{
		"config.crossref.yaml",
		"config.muse.yaml",
		"config.sample.json",
	}

	loader := NewLoader()

	for _, configFile := range configFiles {
		t.Run(configFile, func(t *testing.T) {
			configPath := filepath.Join("..", "..", configFile)

			// Load configuration
			config, err := loader.Load(configPath)
			if err != nil {
				t.Fatalf("Failed to load %s: %v", configFile, err)
			}

			if config == nil {
				t.Fatal("Expected config, got nil")
			}

			// Validate configuration
			if err := config.Validate(); err != nil {
				t.Fatalf("Config validation failed for %s: %v", configFile, err)
			}

			// Test that all required fields are present
			if config.Server.Transport == "" {
				t.Error("Server transport should not be empty")
			}

			if config.OpenAPI.SpecPath == "" {
				t.Error("OpenAPI spec path should not be empty")
			}

			if config.OpenAPI.BaseURL == "" {
				t.Error("OpenAPI base URL should not be empty")
			}

			// Test that timeouts are reasonable
			if config.OpenAPI.Timeout < time.Second {
				t.Error("OpenAPI timeout should be at least 1 second")
			}

			if config.Server.HTTP.SessionTimeout < time.Minute {
				t.Error("Session timeout should be at least 1 minute")
			}

			// Test that ports are valid
			if config.Server.HTTP.Port < 1 || config.Server.HTTP.Port > 65535 {
				t.Errorf("Port %d is not valid", config.Server.HTTP.Port)
			}

			// Test that rate limiting is reasonable
			if config.Security.RateLimiting.RequestsPerMinute < 1 {
				t.Error("Rate limit should be at least 1 request per minute")
			}
		})
	}
}

// TestEndToEnd_ConfigMerging tests that partial configs are properly merged with defaults
func TestEndToEnd_ConfigMerging(t *testing.T) {
	// Create a minimal config file
	minimalConfig := `
server:
  transport: "http"
  http:
    port: 9000
openapi:
  spec_path: "https://api.example.com/openapi.json"
  base_url: "https://api.example.com"
`

	tmpFile, err := os.CreateTemp("", "minimal_config.*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(minimalConfig); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}
	tmpFile.Close()

	loader := NewLoader()
	config, err := loader.Load(tmpFile.Name())

	if err != nil {
		t.Fatalf("Failed to load minimal config: %v", err)
	}

	// Test that defaults were applied
	if config.Server.HTTP.Host != "127.0.0.1" {
		t.Errorf("Expected default host, got %s", config.Server.HTTP.Host)
	}

	if config.Server.HTTP.SessionTimeout != 5*time.Minute {
		t.Errorf("Expected default session timeout, got %v", config.Server.HTTP.SessionTimeout)
	}

	if config.Server.HTTP.MaxConnections != 100 {
		t.Errorf("Expected default max connections, got %d", config.Server.HTTP.MaxConnections)
	}

	if !config.Server.HTTP.CORS.Enabled {
		t.Error("Expected default CORS to be enabled")
	}

	if config.Logging.Level != "info" {
		t.Errorf("Expected default log level, got %s", config.Logging.Level)
	}

	if config.OpenAPI.Timeout != 30*time.Second {
		t.Errorf("Expected default timeout, got %v", config.OpenAPI.Timeout)
	}

	if config.OpenAPI.MaxRetries != 3 {
		t.Errorf("Expected default max retries, got %d", config.OpenAPI.MaxRetries)
	}

	if config.OpenAPI.Auth.Type != "none" {
		t.Errorf("Expected default auth type, got %s", config.OpenAPI.Auth.Type)
	}

	if !config.Security.RateLimiting.Enabled {
		t.Error("Expected default rate limiting to be enabled")
	}

	if config.Security.RateLimiting.RequestsPerMinute != 100 {
		t.Errorf("Expected default requests per minute, got %d", config.Security.RateLimiting.RequestsPerMinute)
	}

	if config.Security.RequestSizeLimit != "1MB" {
		t.Errorf("Expected default request size limit, got %s", config.Security.RequestSizeLimit)
	}

	// Test that provided values were preserved
	if config.Server.Transport != "http" {
		t.Errorf("Expected preserved transport, got %s", config.Server.Transport)
	}

	if config.Server.HTTP.Port != 9000 {
		t.Errorf("Expected preserved port, got %d", config.Server.HTTP.Port)
	}

	if config.OpenAPI.SpecPath != "https://api.example.com/openapi.json" {
		t.Errorf("Expected preserved spec path, got %s", config.OpenAPI.SpecPath)
	}

	if config.OpenAPI.BaseURL != "https://api.example.com" {
		t.Errorf("Expected preserved base URL, got %s", config.OpenAPI.BaseURL)
	}

	// Test validation
	if err := config.Validate(); err != nil {
		t.Errorf("Config validation failed: %v", err)
	}
}

// TestEndToEnd_ErrorHandling tests error handling across the configuration system
func TestEndToEnd_ErrorHandling(t *testing.T) {
	loader := NewLoader()

	// Test non-existent file
	_, err := loader.Load("nonexistent.yaml")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}

	// Test invalid YAML
	invalidYAML := `
server:
  transport: "http"
  http:
    port: invalid_port
`

	tmpFile, err := os.CreateTemp("", "invalid_config.*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(invalidYAML); err != nil {
		t.Fatalf("Failed to write invalid config: %v", err)
	}
	tmpFile.Close()

	_, err = loader.Load(tmpFile.Name())
	if err == nil {
		t.Error("Expected error for invalid YAML")
	}

	// Test invalid JSON
	invalidJSON := `{
  "server": {
    "transport": "http"
    "http": {
      "port": 8080
    }
  }
}`

	tmpFile2, err := os.CreateTemp("", "invalid_config.*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile2.Name())

	if _, err := tmpFile2.WriteString(invalidJSON); err != nil {
		t.Fatalf("Failed to write invalid config: %v", err)
	}
	tmpFile2.Close()

	_, err = loader.Load(tmpFile2.Name())
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}

	// Test unsupported format
	tmpFile3, err := os.CreateTemp("", "config.*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile3.Name())

	_, err = loader.Load(tmpFile3.Name())
	if err == nil {
		t.Error("Expected error for unsupported format")
	}
}

// TestEndToEnd_ValidationScenarios tests various validation scenarios
func TestEndToEnd_ValidationScenarios(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: &Config{
				Server: ServerConfig{
					Transport: "http",
					HTTP: HTTPConfig{
						Port: 8080,
					},
				},
				OpenAPI: OpenAPIConfig{
					SpecPath:   "https://api.example.com/openapi.json",
					Timeout:    30 * time.Second,
					MaxRetries: 3,
				},
				Security: SecurityConfig{
					RateLimiting: RateLimitingConfig{
						Enabled:           true,
						RequestsPerMinute: 100,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid transport",
			config: &Config{
				Server: ServerConfig{
					Transport: "invalid",
				},
			},
			wantErr: true,
		},
		{
			name: "invalid port",
			config: &Config{
				Server: ServerConfig{
					Transport: "http",
					HTTP: HTTPConfig{
						Port: 0,
					},
				},
			},
			wantErr: true,
		},
		{
			name: "missing OpenAPI spec",
			config: &Config{
				Server: ServerConfig{
					Transport: "http",
					HTTP: HTTPConfig{
						Port: 8080,
					},
				},
				OpenAPI: OpenAPIConfig{
					SpecPath: "",
				},
			},
			wantErr: true,
		},
		{
			name: "invalid timeout",
			config: &Config{
				Server: ServerConfig{
					Transport: "http",
					HTTP: HTTPConfig{
						Port: 8080,
					},
				},
				OpenAPI: OpenAPIConfig{
					SpecPath: "https://api.example.com/openapi.json",
					Timeout:  500 * time.Millisecond,
				},
			},
			wantErr: true,
		},
		{
			name: "invalid max retries",
			config: &Config{
				Server: ServerConfig{
					Transport: "http",
					HTTP: HTTPConfig{
						Port: 8080,
					},
				},
				OpenAPI: OpenAPIConfig{
					SpecPath:   "https://api.example.com/openapi.json",
					Timeout:    30 * time.Second,
					MaxRetries: -1,
				},
			},
			wantErr: true,
		},
		{
			name: "invalid rate limit",
			config: &Config{
				Server: ServerConfig{
					Transport: "http",
					HTTP: HTTPConfig{
						Port: 8080,
					},
				},
				OpenAPI: OpenAPIConfig{
					SpecPath:   "https://api.example.com/openapi.json",
					Timeout:    30 * time.Second,
					MaxRetries: 3,
				},
				Security: SecurityConfig{
					RateLimiting: RateLimitingConfig{
						Enabled:           true,
						RequestsPerMinute: 0,
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
			}
		})
	}
}

// TestEndToEnd_ConfigRoundTrip tests that configs can be loaded and validated consistently
func TestEndToEnd_ConfigRoundTrip(t *testing.T) {
	// Test that all provided config files can be loaded, validated, and used
	configFiles := []string{
		"config.crossref.yaml",
		"config.muse.yaml",
		"config.sample.json",
	}

	loader := NewLoader()

	for _, configFile := range configFiles {
		t.Run(configFile, func(t *testing.T) {
			configPath := filepath.Join("..", "..", configFile)

			// Load config
			config, err := loader.Load(configPath)
			if err != nil {
				t.Fatalf("Failed to load %s: %v", configFile, err)
			}

			// Validate config
			if err := config.Validate(); err != nil {
				t.Fatalf("Config validation failed for %s: %v", configFile, err)
			}

			// Test that config can be used (basic field access)
			if config.Server.Transport == "" {
				t.Error("Server transport should not be empty")
			}

			if config.OpenAPI.SpecPath == "" {
				t.Error("OpenAPI spec path should not be empty")
			}

			if config.OpenAPI.BaseURL == "" {
				t.Error("OpenAPI base URL should not be empty")
			}

			// Test that timeouts are reasonable
			if config.OpenAPI.Timeout < time.Second {
				t.Error("OpenAPI timeout should be at least 1 second")
			}

			if config.Server.HTTP.SessionTimeout < time.Minute {
				t.Error("Session timeout should be at least 1 minute")
			}

			// Test that ports are valid
			if config.Server.HTTP.Port < 1 || config.Server.HTTP.Port > 65535 {
				t.Errorf("Port %d is not valid", config.Server.HTTP.Port)
			}

			// Test that rate limiting is reasonable
			if config.Security.RateLimiting.RequestsPerMinute < 1 {
				t.Error("Rate limit should be at least 1 request per minute")
			}

			// Test that headers are properly initialized
			if config.OpenAPI.Headers == nil {
				t.Error("OpenAPI headers should be initialized")
			}

			// Test that CORS origins are properly set
			if config.Server.HTTP.CORS.Enabled && len(config.Server.HTTP.CORS.Origins) == 0 {
				t.Error("CORS origins should not be empty when CORS is enabled")
			}
		})
	}
}
