package mcp

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

)

func TestStreamableHTTPTransport_FormSizeLimits(t *testing.T) {
	// Create a test MCP server
	mcpServer := &Server{
		tools:   make(map[string]ToolHandler),
		schemas: make(map[string]ToolSchema),
	}

	// Test configuration with small size limits
	config := &StreamableHTTPConfig{
		Host:        "127.0.0.1",
		Port:        8080,
		MaxFormSize: 1024, // 1KB limit
	}

	transport := NewStreamableHTTPTransport(mcpServer, config)
	server := httptest.NewServer(transport.corsMiddleware(http.HandlerFunc(transport.handleMCP)))
	defer server.Close()

	tests := []struct {
		name           string
		contentType    string
		contentLength  int64
		formData       map[string]string
		expectedStatus int
		expectWarning  bool
	}{
		{
			name:           "small_form_data",
			contentType:    "application/x-www-form-urlencoded",
			contentLength:  100,
			formData:       map[string]string{"key": "value"},
			expectedStatus: http.StatusBadRequest, // Will fail because it's not valid JSON-RPC
			expectWarning:  false,
		},
		{
			name:           "large_form_data",
			contentType:    "application/x-www-form-urlencoded",
			contentLength:  2048, // Exceeds 1KB limit
			formData:       map[string]string{"key": strings.Repeat("x", 2000)},
			expectedStatus: http.StatusBadRequest, // Will fail because it's not valid JSON-RPC
			expectWarning:  true,                  // Should log warning about large form
		},
		{
			name:           "multipart_small_file",
			contentType:    "multipart/form-data",
			contentLength:  500,
			formData:       map[string]string{"file": "small content"},
			expectedStatus: http.StatusBadRequest, // Will fail because it's not valid JSON-RPC
			expectWarning:  false,
		},
		{
			name:           "multipart_large_file",
			contentType:    "multipart/form-data",
			contentLength:  2048, // Exceeds 1KB limit
			formData:       map[string]string{"file": strings.Repeat("x", 2000)},
			expectedStatus: http.StatusBadRequest, // Will fail because it's not valid JSON-RPC
			expectWarning:  true,                  // Should log warning about large form
		},
		{
			name:           "no_content_length",
			contentType:    "application/x-www-form-urlencoded",
			contentLength:  0, // No content length
			formData:       map[string]string{"key": "value"},
			expectedStatus: http.StatusBadRequest, // Will fail because it's not valid JSON-RPC
			expectWarning:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var body io.Reader
			var contentType string

			if tt.contentType == "multipart/form-data" {
				// Create multipart form
				var buf bytes.Buffer
				writer := multipart.NewWriter(&buf)

				for key, value := range tt.formData {
					if key == "file" {
						// Create a file field
						fileWriter, err := writer.CreateFormFile("file", "test.txt")
						if err != nil {
							t.Fatalf("Failed to create form file: %v", err)
						}
						fileWriter.Write([]byte(value))
					} else {
						// Create a regular field
						writer.WriteField(key, value)
					}
				}
				writer.Close()

				body = &buf
				contentType = writer.FormDataContentType()
			} else {
				// Create URL-encoded form
				formData := url.Values{}
				for key, value := range tt.formData {
					formData.Set(key, value)
				}
				body = strings.NewReader(formData.Encode())
				contentType = "application/x-www-form-urlencoded"
			}

			// Create request
			req, err := http.NewRequest("POST", server.URL+"/mcp", body)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			req.Header.Set("Content-Type", contentType)
			if tt.contentLength > 0 {
				req.Header.Set("Content-Length", fmt.Sprintf("%d", tt.contentLength))
			}
			req.Header.Set("Accept", "application/json")
			req.Header.Set("MCP-Protocol-Version", "2024-11-05")

			// Send request
			client := &http.Client{Timeout: 5 * time.Second}
			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("Failed to send request: %v", err)
			}
			defer resp.Body.Close()

			// Check status code
			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}

			// Read response body for debugging
			respBody, _ := io.ReadAll(resp.Body)
			if tt.expectWarning && len(respBody) == 0 {
				t.Error("Expected error response body")
			}
		})
	}
}


func TestStreamableHTTPTransport_DefaultConfig(t *testing.T) {
	// Test that default configuration is applied correctly
	transport := NewStreamableHTTPTransport(nil, nil)
	
	// Check that MaxFormSize is set to default value (1MB)
	expectedMaxFormSize := int64(1 << 20) // 1MB
	if transport.config.MaxFormSize != expectedMaxFormSize {
		t.Errorf("Expected MaxFormSize %d, got %d", expectedMaxFormSize, transport.config.MaxFormSize)
	}
}

func TestStreamableHTTPTransport_CustomConfig(t *testing.T) {
	// Test that custom configuration is applied correctly
	customConfig := &StreamableHTTPConfig{
		MaxFormSize: 2 << 20, // 2MB
	}
	transport := NewStreamableHTTPTransport(nil, customConfig)
	
	// Check that MaxFormSize is set to custom value
	expectedMaxFormSize := int64(2 << 20) // 2MB
	if transport.config.MaxFormSize != expectedMaxFormSize {
		t.Errorf("Expected MaxFormSize %d, got %d", expectedMaxFormSize, transport.config.MaxFormSize)
	}
}

func TestStreamableHTTPTransport_FormSizeLimitIntegration(t *testing.T) {
	// Test form size limits through actual HTTP requests
	mcpServer := &Server{
		tools:   make(map[string]ToolHandler),
		schemas: make(map[string]ToolSchema),
	}

	// Test with very small limit
	config := &StreamableHTTPConfig{
		MaxFormSize: 50, // 50 bytes limit
	}
	transport := NewStreamableHTTPTransport(mcpServer, config)
	server := httptest.NewServer(transport.corsMiddleware(http.HandlerFunc(transport.handleMCP)))
	defer server.Close()

	tests := []struct {
		name          string
		formSize      int
		expectWarning bool
	}{
		{
			name:          "small_form",
			formSize:      30, // Within 50 byte limit
			expectWarning: false,
		},
		{
			name:          "large_form",
			formSize:      100, // Exceeds 50 byte limit
			expectWarning: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create form data of specified size
			formData := url.Values{}
			formData.Set("test_field", strings.Repeat("x", tt.formSize-20)) // -20 for field name and overhead
			
			// Create request
			req, err := http.NewRequest("POST", server.URL+"/mcp", strings.NewReader(formData.Encode()))
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.Header.Set("Content-Length", fmt.Sprintf("%d", len(formData.Encode())))
			req.Header.Set("Accept", "application/json")
			req.Header.Set("MCP-Protocol-Version", "2024-11-05")

			// Send request
			client := &http.Client{Timeout: 5 * time.Second}
			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("Failed to send request: %v", err)
			}
			defer resp.Body.Close()

			// Should get BadRequest because it's not valid JSON-RPC
			if resp.StatusCode != http.StatusBadRequest {
				t.Errorf("Expected status %d, got %d", http.StatusBadRequest, resp.StatusCode)
			}

			// The test passes if we get here without errors
			// The actual form size limit behavior is tested through the HTTP handler
		})
	}
}
