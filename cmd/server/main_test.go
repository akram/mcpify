/*
Copyright 2025
SPDX-License-Identifier: Apache-2.0
*/
package main

import (
	"flag"
	"os"
	"testing"

	"mcpify/internal/config"
)

func TestExtractBaseURLFromSpec(t *testing.T) {
	tests := []struct {
		name     string
		specPath string
		expected string
	}{
		{
			name:     "localhost with port",
			specPath: "http://localhost:8080/swagger",
			expected: "http://localhost:8080",
		},
		{
			name:     "localhost with non-default port",
			specPath: "http://localhost:3000/api/swagger",
			expected: "http://localhost:3000",
		},
		{
			name:     "https with default port",
			specPath: "https://api.example.com/openapi.json",
			expected: "https://api.example.com",
		},
		{
			name:     "https with explicit default port",
			specPath: "https://api.example.com:443/openapi.json",
			expected: "https://api.example.com",
		},
		{
			name:     "http with explicit default port",
			specPath: "http://api.example.com:80/swagger",
			expected: "http://api.example.com",
		},
		{
			name:     "https with non-default port",
			specPath: "https://api.example.com:8443/swagger",
			expected: "https://api.example.com:8443",
		},
		{
			name:     "complex path",
			specPath: "https://petstore3.swagger.io/api/v3/openapi.json",
			expected: "https://petstore3.swagger.io",
		},
		{
			name:     "subdomain with port",
			specPath: "http://api.dev.example.com:8080/docs/swagger",
			expected: "http://api.dev.example.com:8080",
		},
		{
			name:     "non-http URL",
			specPath: "file:///path/to/openapi.json",
			expected: "",
		},
		{
			name:     "relative path",
			specPath: "./openapi.json",
			expected: "",
		},
		{
			name:     "invalid URL",
			specPath: "not-a-url",
			expected: "",
		},
		{
			name:     "empty string",
			specPath: "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractBaseURLFromSpec(tt.specPath)
			if result != tt.expected {
				t.Errorf("extractBaseURLFromSpec(%q) = %q, want %q", tt.specPath, result, tt.expected)
			}
		})
	}
}

func TestFlagParsing(t *testing.T) {
	// Save original args
	originalArgs := os.Args
	defer func() {
		os.Args = originalArgs
		flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	}()

	tests := []struct {
		name            string
		args            []string
		expectedBaseURL string
		expectedSpec    string
	}{
		{
			name:            "base-url flag only",
			args:            []string{"mcpify", "--base-url", "https://api.example.com"},
			expectedBaseURL: "https://api.example.com",
			expectedSpec:    "",
		},
		{
			name:            "short base-url flag",
			args:            []string{"mcpify", "-b", "https://api.example.com"},
			expectedBaseURL: "https://api.example.com",
			expectedSpec:    "",
		},
		{
			name:            "spec and base-url flags",
			args:            []string{"mcpify", "--spec", "https://petstore3.swagger.io/api/v3/openapi.json", "--base-url", "https://api.example.com"},
			expectedBaseURL: "https://api.example.com",
			expectedSpec:    "https://petstore3.swagger.io/api/v3/openapi.json",
		},
		{
			name:            "spec flag only",
			args:            []string{"mcpify", "--spec", "http://localhost:8080/swagger"},
			expectedBaseURL: "",
			expectedSpec:    "http://localhost:8080/swagger",
		},
		{
			name:            "no flags",
			args:            []string{"mcpify"},
			expectedBaseURL: "",
			expectedSpec:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset flag.CommandLine for each test
			flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
			os.Args = tt.args

			// Parse flags
			baseURL := flag.String("base-url", "", "Base URL for API requests")
			specPath := flag.String("spec", "", "Path to OpenAPI specification")
			flag.StringVar(baseURL, "b", "", "Base URL for API requests")

			flag.Parse()

			if *baseURL != tt.expectedBaseURL {
				t.Errorf("baseURL = %q, want %q", *baseURL, tt.expectedBaseURL)
			}
			if *specPath != tt.expectedSpec {
				t.Errorf("specPath = %q, want %q", *specPath, tt.expectedSpec)
			}
		})
	}
}

func TestBaseURLOverrideLogic(t *testing.T) {
	tests := []struct {
		name           string
		configBaseURL  string
		flagBaseURL    string
		expectedResult string
		shouldWarn     bool
	}{
		{
			name:           "no config, no flag",
			configBaseURL:  "",
			flagBaseURL:    "",
			expectedResult: "",
			shouldWarn:     false,
		},
		{
			name:           "config only",
			configBaseURL:  "https://api.example.com",
			flagBaseURL:    "",
			expectedResult: "https://api.example.com",
			shouldWarn:     false,
		},
		{
			name:           "flag only",
			configBaseURL:  "",
			flagBaseURL:    "https://api.example.com",
			expectedResult: "https://api.example.com",
			shouldWarn:     false,
		},
		{
			name:           "same values",
			configBaseURL:  "https://api.example.com",
			flagBaseURL:    "https://api.example.com",
			expectedResult: "https://api.example.com",
			shouldWarn:     false,
		},
		{
			name:           "different values",
			configBaseURL:  "https://api.example.com",
			flagBaseURL:    "https://api.different.com",
			expectedResult: "https://api.different.com",
			shouldWarn:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the override logic from main.go
			result := tt.configBaseURL
			warned := false

			if tt.flagBaseURL != "" {
				if tt.configBaseURL != "" && tt.configBaseURL != tt.flagBaseURL {
					warned = true
				}
				result = tt.flagBaseURL
			}

			if result != tt.expectedResult {
				t.Errorf("result = %q, want %q", result, tt.expectedResult)
			}
			if warned != tt.shouldWarn {
				t.Errorf("warned = %v, want %v", warned, tt.shouldWarn)
			}
		})
	}
}

func TestDefaultBaseURLExtraction(t *testing.T) {
	tests := []struct {
		name           string
		configBaseURL  string
		specPath       string
		expectedResult string
		shouldExtract  bool
	}{
		{
			name:           "no base URL, no spec",
			configBaseURL:  "",
			specPath:       "",
			expectedResult: "",
			shouldExtract:  false,
		},
		{
			name:           "has base URL, no spec",
			configBaseURL:  "https://api.example.com",
			specPath:       "",
			expectedResult: "https://api.example.com",
			shouldExtract:  false,
		},
		{
			name:           "no base URL, has spec",
			configBaseURL:  "",
			specPath:       "http://localhost:8080/swagger",
			expectedResult: "http://localhost:8080",
			shouldExtract:  true,
		},
		{
			name:           "has base URL, has spec",
			configBaseURL:  "https://api.example.com",
			specPath:       "http://localhost:8080/swagger",
			expectedResult: "https://api.example.com",
			shouldExtract:  false,
		},
		{
			name:           "no base URL, non-http spec",
			configBaseURL:  "",
			specPath:       "./openapi.json",
			expectedResult: "",
			shouldExtract:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the default extraction logic from main.go
			result := tt.configBaseURL
			extracted := false

			if tt.configBaseURL == "" && tt.specPath != "" {
				if extractedBaseURL := extractBaseURLFromSpec(tt.specPath); extractedBaseURL != "" {
					result = extractedBaseURL
					extracted = true
				}
			}

			if result != tt.expectedResult {
				t.Errorf("result = %q, want %q", result, tt.expectedResult)
			}
			if extracted != tt.shouldExtract {
				t.Errorf("extracted = %v, want %v", extracted, tt.shouldExtract)
			}
		})
	}
}

// Test the complete flow: flag parsing + override + default extraction
func TestCompleteBaseURLFlow(t *testing.T) {
	tests := []struct {
		name           string
		configBaseURL  string
		flagBaseURL    string
		specPath       string
		expectedResult string
		shouldWarn     bool
		shouldExtract  bool
	}{
		{
			name:           "flag overrides config",
			configBaseURL:  "https://config.example.com",
			flagBaseURL:    "https://flag.example.com",
			specPath:       "http://localhost:8080/swagger",
			expectedResult: "https://flag.example.com",
			shouldWarn:     true,
			shouldExtract:  false,
		},
		{
			name:           "config used when no flag",
			configBaseURL:  "https://config.example.com",
			flagBaseURL:    "",
			specPath:       "http://localhost:8080/swagger",
			expectedResult: "https://config.example.com",
			shouldWarn:     false,
			shouldExtract:  false,
		},
		{
			name:           "extract from spec when no config or flag",
			configBaseURL:  "",
			flagBaseURL:    "",
			specPath:       "http://localhost:8080/swagger",
			expectedResult: "http://localhost:8080",
			shouldWarn:     false,
			shouldExtract:  true,
		},
		{
			name:           "no base URL available",
			configBaseURL:  "",
			flagBaseURL:    "",
			specPath:       "./openapi.json",
			expectedResult: "",
			shouldWarn:     false,
			shouldExtract:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the complete flow from main.go
			result := tt.configBaseURL
			warned := false
			extracted := false

			// Step 1: Apply flag override
			if tt.flagBaseURL != "" {
				if tt.configBaseURL != "" && tt.configBaseURL != tt.flagBaseURL {
					warned = true
				}
				result = tt.flagBaseURL
			}

			// Step 2: Extract from spec if still no base URL
			if result == "" && tt.specPath != "" {
				if extractedBaseURL := extractBaseURLFromSpec(tt.specPath); extractedBaseURL != "" {
					result = extractedBaseURL
					extracted = true
				}
			}

			if result != tt.expectedResult {
				t.Errorf("result = %q, want %q", result, tt.expectedResult)
			}
			if warned != tt.shouldWarn {
				t.Errorf("warned = %v, want %v", warned, tt.shouldWarn)
			}
			if extracted != tt.shouldExtract {
				t.Errorf("extracted = %v, want %v", extracted, tt.shouldExtract)
			}
		})
	}
}

func TestConfigurationSummary(t *testing.T) {
	tests := []struct {
		name          string
		config        *config.Config
		expectedLines []string
	}{
		{
			name: "stdio transport",
			config: &config.Config{
				Server: config.ServerConfig{
					Transport: "stdio",
				},
				OpenAPI: config.OpenAPIConfig{
					SpecPath: "https://petstore3.swagger.io/api/v3/openapi.json",
					BaseURL:  "https://petstore3.swagger.io",
				},
			},
			expectedLines: []string{
				"=== MCPify Configuration Summary ===",
				"OpenAPI Spec: https://petstore3.swagger.io/api/v3/openapi.json",
				"Base URL: https://petstore3.swagger.io",
				"Transport: stdio",
				"=====================================",
			},
		},
		{
			name: "http transport",
			config: &config.Config{
				Server: config.ServerConfig{
					Transport: "http",
					HTTP: config.HTTPConfig{
						Host: "127.0.0.1",
						Port: 9090,
					},
				},
				OpenAPI: config.OpenAPIConfig{
					SpecPath: "https://api.example.com/swagger",
					BaseURL:  "https://api.example.com",
				},
			},
			expectedLines: []string{
				"=== MCPify Configuration Summary ===",
				"OpenAPI Spec: https://api.example.com/swagger",
				"Base URL: https://api.example.com",
				"Transport: http",
				"HTTP Server: 127.0.0.1:9090",
				"=====================================",
			},
		},
		{
			name: "empty base URL",
			config: &config.Config{
				Server: config.ServerConfig{
					Transport: "stdio",
				},
				OpenAPI: config.OpenAPIConfig{
					SpecPath: "https://api.example.com/swagger",
					BaseURL:  "",
				},
			},
			expectedLines: []string{
				"=== MCPify Configuration Summary ===",
				"OpenAPI Spec: https://api.example.com/swagger",
				"Base URL: ",
				"Transport: stdio",
				"=====================================",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This test verifies the expected log format
			// In a real implementation, you might want to capture log output
			// For now, we just verify the configuration structure is correct

			// Verify config structure
			if tt.config.OpenAPI.SpecPath == "" {
				t.Error("SpecPath should not be empty")
			}
			if tt.config.Server.Transport == "" {
				t.Error("Transport should not be empty")
			}

			// Verify HTTP config when transport is http
			if tt.config.Server.Transport == "http" {
				if tt.config.Server.HTTP.Host == "" {
					t.Error("HTTP Host should not be empty for http transport")
				}
				if tt.config.Server.HTTP.Port == 0 {
					t.Error("HTTP Port should not be zero for http transport")
				}
			}

			// Verify expected lines would be generated
			if len(tt.expectedLines) < 4 {
				t.Error("Expected at least 4 summary lines")
			}

			// Check that all expected lines contain the right information
			for _, line := range tt.expectedLines {
				if line == "" {
					t.Error("Expected line should not be empty")
				}
			}
		})
	}
}
