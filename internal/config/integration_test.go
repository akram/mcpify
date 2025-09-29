package config

import (
	"path/filepath"
	"testing"
	"time"
)

func TestLoadConfig_CrossrefYAML(t *testing.T) {
	// Test loading the actual config.crossref.yaml file
	configPath := filepath.Join("..", "..", "config.crossref.yaml")

	loader := NewLoader()
	config, err := loader.Load(configPath)

	if err != nil {
		t.Fatalf("Failed to load config.crossref.yaml: %v", err)
	}

	if config == nil {
		t.Fatal("Expected config, got nil")
	}

	// Test server configuration
	if config.Server.Transport != "http" {
		t.Errorf("Expected transport 'http', got '%s'", config.Server.Transport)
	}

	if config.Server.HTTP.Host != "127.0.0.1" {
		t.Errorf("Expected host '127.0.0.1', got '%s'", config.Server.HTTP.Host)
	}

	if config.Server.HTTP.Port != 8080 {
		t.Errorf("Expected port 8080, got %d", config.Server.HTTP.Port)
	}

	if config.Server.HTTP.SessionTimeout != 5*time.Minute {
		t.Errorf("Expected session timeout 5m, got %v", config.Server.HTTP.SessionTimeout)
	}

	if config.Server.HTTP.MaxConnections != 100 {
		t.Errorf("Expected max connections 100, got %d", config.Server.HTTP.MaxConnections)
	}

	if !config.Server.HTTP.CORS.Enabled {
		t.Error("Expected CORS to be enabled")
	}

	expectedOrigins := []string{
		"http://localhost:3000",
		"http://127.0.0.1:3000",
		"https://your-frontend-domain.com",
	}
	if len(config.Server.HTTP.CORS.Origins) != len(expectedOrigins) {
		t.Errorf("Expected %d CORS origins, got %d", len(expectedOrigins), len(config.Server.HTTP.CORS.Origins))
	}

	// Test logging configuration
	if config.Logging.Level != "info" {
		t.Errorf("Expected log level 'info', got '%s'", config.Logging.Level)
	}

	if config.Logging.Format != "json" {
		t.Errorf("Expected log format 'json', got '%s'", config.Logging.Format)
	}

	if config.Logging.Output != "stdout" {
		t.Errorf("Expected log output 'stdout', got '%s'", config.Logging.Output)
	}

	// Test OpenAPI configuration
	if config.OpenAPI.SpecPath != "https://api.crossref.org/swagger-docs" {
		t.Errorf("Expected spec path 'https://api.crossref.org/swagger-docs', got '%s'", config.OpenAPI.SpecPath)
	}

	if config.OpenAPI.BaseURL != "https://api.crossref.org/" {
		t.Errorf("Expected base URL 'https://api.crossref.org/', got '%s'", config.OpenAPI.BaseURL)
	}

	if config.OpenAPI.Timeout != 30*time.Second {
		t.Errorf("Expected timeout 30s, got %v", config.OpenAPI.Timeout)
	}

	if config.OpenAPI.MaxRetries != 3 {
		t.Errorf("Expected max retries 3, got %d", config.OpenAPI.MaxRetries)
	}

	if config.OpenAPI.Auth.Type != "none" {
		t.Errorf("Expected auth type 'none', got '%s'", config.OpenAPI.Auth.Type)
	}

	// Test headers
	if config.OpenAPI.Headers["User-Agent"] != "MCPify/1.0.0" {
		t.Errorf("Expected User-Agent 'MCPify/1.0.0', got '%s'", config.OpenAPI.Headers["User-Agent"])
	}

	if config.OpenAPI.Headers["Accept"] != "application/vnd.oai.openapi+json" {
		t.Errorf("Expected Accept 'application/vnd.oai.openapi+json', got '%s'", config.OpenAPI.Headers["Accept"])
	}

	if config.OpenAPI.Headers["Content-Type"] != "application/json" {
		t.Errorf("Expected Content-Type 'application/json', got '%s'", config.OpenAPI.Headers["Content-Type"])
	}

	// Test exclude paths
	expectedExcludePaths := []string{
		"/health",
		"/metrics",
		"/docs",
		"/swagger*",
		"/openapi*",
	}
	if len(config.OpenAPI.ExcludePaths) != len(expectedExcludePaths) {
		t.Errorf("Expected %d exclude paths, got %d", len(expectedExcludePaths), len(config.OpenAPI.ExcludePaths))
	}

	// Test security configuration
	if !config.Security.RateLimiting.Enabled {
		t.Error("Expected rate limiting to be enabled")
	}

	if config.Security.RateLimiting.RequestsPerMinute != 100 {
		t.Errorf("Expected requests per minute 100, got %d", config.Security.RateLimiting.RequestsPerMinute)
	}

	if config.Security.RequestSizeLimit != "1MB" {
		t.Errorf("Expected request size limit '1MB', got '%s'", config.Security.RequestSizeLimit)
	}

	// Test validation
	if err := config.Validate(); err != nil {
		t.Errorf("Config validation failed: %v", err)
	}
}

func TestLoadConfig_MuseYAML(t *testing.T) {
	// Test loading the actual config.muse.yaml file
	configPath := filepath.Join("..", "..", "config.muse.yaml")

	loader := NewLoader()
	config, err := loader.Load(configPath)

	if err != nil {
		t.Fatalf("Failed to load config.muse.yaml: %v", err)
	}

	if config == nil {
		t.Fatal("Expected config, got nil")
	}

	// Test server configuration (should be same as crossref)
	if config.Server.Transport != "http" {
		t.Errorf("Expected transport 'http', got '%s'", config.Server.Transport)
	}

	if config.Server.HTTP.Host != "127.0.0.1" {
		t.Errorf("Expected host '127.0.0.1', got '%s'", config.Server.HTTP.Host)
	}

	if config.Server.HTTP.Port != 8080 {
		t.Errorf("Expected port 8080, got %d", config.Server.HTTP.Port)
	}

	// Test OpenAPI configuration specific to Muse
	if config.OpenAPI.SpecPath != "/tmp/muses_openapi.json" {
		t.Errorf("Expected spec path '/tmp/muses_openapi.json', got '%s'", config.OpenAPI.SpecPath)
	}

	if config.OpenAPI.BaseURL != "https://ce.musesframework.io/" {
		t.Errorf("Expected base URL 'https://ce.musesframework.io/', got '%s'", config.OpenAPI.BaseURL)
	}

	// Test authentication configuration
	if config.OpenAPI.Auth.Type != "bearer" {
		t.Errorf("Expected auth type 'bearer', got '%s'", config.OpenAPI.Auth.Type)
	}

	if config.OpenAPI.Auth.Token != "your-bearer-token-here" {
		t.Errorf("Expected auth token 'your-bearer-token-here', got '%s'", config.OpenAPI.Auth.Token)
	}

	// Test custom auth headers
	if config.OpenAPI.Auth.Headers["X-Custom-Auth"] != "custom-value" {
		t.Errorf("Expected X-Custom-Auth 'custom-value', got '%s'", config.OpenAPI.Auth.Headers["X-Custom-Auth"])
	}

	// Test validation
	if err := config.Validate(); err != nil {
		t.Errorf("Config validation failed: %v", err)
	}
}

func TestLoadConfig_SampleJSON(t *testing.T) {
	// Test loading the actual config.sample.json file
	configPath := filepath.Join("..", "..", "config.sample.json")

	loader := NewLoader()
	config, err := loader.Load(configPath)

	if err != nil {
		t.Fatalf("Failed to load config.sample.json: %v", err)
	}

	if config == nil {
		t.Fatal("Expected config, got nil")
	}

	// Test server configuration
	if config.Server.Transport != "http" {
		t.Errorf("Expected transport 'http', got '%s'", config.Server.Transport)
	}

	if config.Server.HTTP.Host != "127.0.0.1" {
		t.Errorf("Expected host '127.0.0.1', got '%s'", config.Server.HTTP.Host)
	}

	if config.Server.HTTP.Port != 8080 {
		t.Errorf("Expected port 8080, got %d", config.Server.HTTP.Port)
	}

	// Test OpenAPI configuration
	if config.OpenAPI.SpecPath != "https://api.example.com/openapi.json" {
		t.Errorf("Expected spec path 'https://api.example.com/openapi.json', got '%s'", config.OpenAPI.SpecPath)
	}

	if config.OpenAPI.BaseURL != "https://api.example.com" {
		t.Errorf("Expected base URL 'https://api.example.com', got '%s'", config.OpenAPI.BaseURL)
	}

	if config.OpenAPI.ToolPrefix != "api" {
		t.Errorf("Expected tool prefix 'api', got '%s'", config.OpenAPI.ToolPrefix)
	}

	// Test authentication configuration
	if config.OpenAPI.Auth.Type != "bearer" {
		t.Errorf("Expected auth type 'bearer', got '%s'", config.OpenAPI.Auth.Type)
	}

	if config.OpenAPI.Auth.Token != "your-bearer-token-here" {
		t.Errorf("Expected auth token 'your-bearer-token-here', got '%s'", config.OpenAPI.Auth.Token)
	}

	// Test custom auth headers
	if config.OpenAPI.Auth.Headers["X-Custom-Auth"] != "custom-value" {
		t.Errorf("Expected X-Custom-Auth 'custom-value', got '%s'", config.OpenAPI.Auth.Headers["X-Custom-Auth"])
	}

	// Test include paths (specific to sample.json)
	expectedIncludePaths := []string{
		"/api/v1/*",
		"/users/*",
		"/posts/*",
	}
	if len(config.OpenAPI.IncludePaths) != len(expectedIncludePaths) {
		t.Errorf("Expected %d include paths, got %d", len(expectedIncludePaths), len(config.OpenAPI.IncludePaths))
	}

	// Test validation
	if err := config.Validate(); err != nil {
		t.Errorf("Config validation failed: %v", err)
	}
}

func TestConfigFiles_Consistency(t *testing.T) {
	// Test that all config files can be loaded and have consistent structure
	configFiles := []string{
		filepath.Join("..", "..", "config.crossref.yaml"),
		filepath.Join("..", "..", "config.muse.yaml"),
		filepath.Join("..", "..", "config.sample.json"),
	}

	loader := NewLoader()

	for _, configPath := range configFiles {
		t.Run(filepath.Base(configPath), func(t *testing.T) {
			config, err := loader.Load(configPath)

			if err != nil {
				t.Fatalf("Failed to load %s: %v", configPath, err)
			}

			if config == nil {
				t.Fatal("Expected config, got nil")
			}

			// Test that all configs have required fields
			if config.Server.Transport == "" {
				t.Error("Server transport should not be empty")
			}

			if config.Logging.Level == "" {
				t.Error("Logging level should not be empty")
			}

			if config.OpenAPI.SpecPath == "" {
				t.Error("OpenAPI spec path should not be empty")
			}

			if config.OpenAPI.BaseURL == "" {
				t.Error("OpenAPI base URL should not be empty")
			}

			// Test that all configs validate
			if err := config.Validate(); err != nil {
				t.Errorf("Config validation failed for %s: %v", configPath, err)
			}
		})
	}
}

func TestConfigFiles_AuthenticationTypes(t *testing.T) {
	// Test different authentication types across config files
	tests := []struct {
		configFile string
		authType   string
		hasToken   bool
		hasHeaders bool
	}{
		{
			configFile: "config.crossref.yaml",
			authType:   "none",
			hasToken:   false,
			hasHeaders: false,
		},
		{
			configFile: "config.muse.yaml",
			authType:   "bearer",
			hasToken:   true,
			hasHeaders: true,
		},
		{
			configFile: "config.sample.json",
			authType:   "bearer",
			hasToken:   true,
			hasHeaders: true,
		},
	}

	loader := NewLoader()

	for _, tt := range tests {
		t.Run(tt.configFile, func(t *testing.T) {
			configPath := filepath.Join("..", "..", tt.configFile)
			config, err := loader.Load(configPath)

			if err != nil {
				t.Fatalf("Failed to load %s: %v", configPath, err)
			}

			if config.OpenAPI.Auth.Type != tt.authType {
				t.Errorf("Expected auth type '%s', got '%s'", tt.authType, config.OpenAPI.Auth.Type)
			}

			if tt.hasToken && config.OpenAPI.Auth.Token == "" {
				t.Error("Expected auth token to be present")
			}

			if !tt.hasToken && config.OpenAPI.Auth.Token != "" {
				t.Error("Expected auth token to be empty")
			}

			if tt.hasHeaders && len(config.OpenAPI.Auth.Headers) == 0 {
				t.Error("Expected auth headers to be present")
			}

			if !tt.hasHeaders && len(config.OpenAPI.Auth.Headers) > 0 {
				t.Error("Expected auth headers to be empty")
			}
		})
	}
}

func TestConfigFiles_OpenAPISpecPaths(t *testing.T) {
	// Test that all config files have valid OpenAPI spec paths
	tests := []struct {
		configFile string
		specPath   string
		isURL      bool
	}{
		{
			configFile: "config.crossref.yaml",
			specPath:   "https://api.crossref.org/swagger-docs",
			isURL:      true,
		},
		{
			configFile: "config.muse.yaml",
			specPath:   "/tmp/muses_openapi.json",
			isURL:      false,
		},
		{
			configFile: "config.sample.json",
			specPath:   "https://api.example.com/openapi.json",
			isURL:      true,
		},
	}

	loader := NewLoader()

	for _, tt := range tests {
		t.Run(tt.configFile, func(t *testing.T) {
			configPath := filepath.Join("..", "..", tt.configFile)
			config, err := loader.Load(configPath)

			if err != nil {
				t.Fatalf("Failed to load %s: %v", configPath, err)
			}

			if config.OpenAPI.SpecPath != tt.specPath {
				t.Errorf("Expected spec path '%s', got '%s'", tt.specPath, config.OpenAPI.SpecPath)
			}

			// Test that spec path is not empty
			if config.OpenAPI.SpecPath == "" {
				t.Error("OpenAPI spec path should not be empty")
			}
		})
	}
}
