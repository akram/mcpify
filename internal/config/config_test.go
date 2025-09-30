package config

import (
	"testing"
	"time"
)

func TestDefault(t *testing.T) {
	config := Default()

	// Test server defaults
	if config.Server.Transport != "stdio" {
		t.Errorf("Expected default transport to be 'stdio', got '%s'", config.Server.Transport)
	}

	if config.Server.HTTP.Host != "127.0.0.1" {
		t.Errorf("Expected default host to be '127.0.0.1', got '%s'", config.Server.HTTP.Host)
	}

	if config.Server.HTTP.Port != 8080 {
		t.Errorf("Expected default port to be 8080, got %d", config.Server.HTTP.Port)
	}

	if config.Server.HTTP.SessionTimeout != 5*time.Minute {
		t.Errorf("Expected default session timeout to be 5m, got %v", config.Server.HTTP.SessionTimeout)
	}

	if config.Server.HTTP.MaxConnections != 100 {
		t.Errorf("Expected default max connections to be 100, got %d", config.Server.HTTP.MaxConnections)
	}

	if !config.Server.HTTP.CORS.Enabled {
		t.Error("Expected default CORS to be enabled")
	}

	expectedOrigins := []string{"http://localhost:3000", "http://127.0.0.1:3000"}
	if len(config.Server.HTTP.CORS.Origins) != len(expectedOrigins) {
		t.Errorf("Expected %d CORS origins, got %d", len(expectedOrigins), len(config.Server.HTTP.CORS.Origins))
	}

	// Test logging defaults
	if config.Logging.Level != "info" {
		t.Errorf("Expected default log level to be 'info', got '%s'", config.Logging.Level)
	}

	if config.Logging.Format != "json" {
		t.Errorf("Expected default log format to be 'json', got '%s'", config.Logging.Format)
	}

	if config.Logging.Output != "stdout" {
		t.Errorf("Expected default log output to be 'stdout', got '%s'", config.Logging.Output)
	}

	// Test OpenAPI defaults
	if config.OpenAPI.Timeout != 30*time.Second {
		t.Errorf("Expected default timeout to be 30s, got %v", config.OpenAPI.Timeout)
	}

	if config.OpenAPI.MaxRetries != 3 {
		t.Errorf("Expected default max retries to be 3, got %d", config.OpenAPI.MaxRetries)
	}

	if config.OpenAPI.Auth.Type != "none" {
		t.Errorf("Expected default auth type to be 'none', got '%s'", config.OpenAPI.Auth.Type)
	}

	if config.OpenAPI.Headers == nil {
		t.Error("Expected default headers to be initialized")
	}

	// Test security defaults
	if !config.Security.RateLimiting.Enabled {
		t.Error("Expected default rate limiting to be enabled")
	}

	if config.Security.RateLimiting.RequestsPerMinute != 100 {
		t.Errorf("Expected default requests per minute to be 100, got %d", config.Security.RateLimiting.RequestsPerMinute)
	}

	if config.Security.RequestSizeLimit != "1MB" {
		t.Errorf("Expected default request size limit to be '1MB', got '%s'", config.Security.RequestSizeLimit)
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
		errType error
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
			errType: ErrInvalidTransport,
		},
		{
			name: "invalid port - too low",
			config: &Config{
				Server: ServerConfig{
					Transport: "http",
					HTTP: HTTPConfig{
						Port: 0,
					},
				},
			},
			wantErr: true,
			errType: ErrInvalidPort,
		},
		{
			name: "invalid port - too high",
			config: &Config{
				Server: ServerConfig{
					Transport: "http",
					HTTP: HTTPConfig{
						Port: 65536,
					},
				},
			},
			wantErr: true,
			errType: ErrInvalidPort,
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
			errType: ErrMissingOpenAPISpec,
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
			errType: ErrInvalidTimeout,
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
			errType: ErrInvalidMaxRetries,
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
			errType: ErrInvalidRateLimit,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error, got nil")
					return
				}
				if tt.errType != nil && err != tt.errType {
					t.Errorf("Expected error %v, got %v", tt.errType, err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
			}
		})
	}
}

func TestConfigStructs(t *testing.T) {
	// Test that all config structs can be instantiated
	config := &Config{
		Server: ServerConfig{
			Transport: "http",
			HTTP: HTTPConfig{
				Host:           "localhost",
				Port:           3000,
				SessionTimeout: 10 * time.Minute,
				MaxConnections: 50,
				CORS: CORSConfig{
					Enabled: true,
					Origins: []string{"http://localhost:3000"},
				},
			},
		},
		Logging: LoggingConfig{
			Level:  "debug",
			Format: "text",
			Output: "stderr",
		},
		OpenAPI: OpenAPIConfig{
			SpecPath:     "https://api.example.com/openapi.json",
			BaseURL:      "https://api.example.com",
			Timeout:      60 * time.Second,
			MaxRetries:   5,
			ToolPrefix:   "test_",
			ExcludePaths: []string{"/health"},
			IncludePaths: []string{"/api/v1/*"},
			Auth: AuthConfig{
				Type:       "bearer",
				Token:      "test-token",
				Username:   "user",
				Password:   "pass",
				APIKey:     "key",
				APIKeyName: "X-API-Key",
				APIKeyIn:   "header",
				Headers: HeadersConfig{
					{Header: HeaderConfig{Name: "X-Custom", Value: "value"}},
				},
			},
			Headers: HeadersConfig{
				{Header: HeaderConfig{Name: "User-Agent", Value: "Test/1.0"}},
			},
		},
		Security: SecurityConfig{
			RateLimiting: RateLimitingConfig{
				Enabled:           false,
				RequestsPerMinute: 200,
			},
			RequestSizeLimit: "2MB",
		},
	}

	// Validate the config
	if err := config.Validate(); err != nil {
		t.Errorf("Config validation failed: %v", err)
	}

	// Test individual struct access
	if config.Server.Transport != "http" {
		t.Errorf("Expected transport 'http', got '%s'", config.Server.Transport)
	}

	if config.OpenAPI.Auth.Type != "bearer" {
		t.Errorf("Expected auth type 'bearer', got '%s'", config.OpenAPI.Auth.Type)
	}

	if config.Security.RateLimiting.Enabled {
		t.Error("Expected rate limiting to be disabled")
	}
}
