package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"mcpify/internal/config"
	"mcpify/internal/types"
)

// APIHandler handles HTTP requests to external APIs
type APIHandler struct {
	config    *config.OpenAPIConfig
	client    *http.Client
	evaluator *config.RequestEvaluator
}

// NewAPIHandler creates a new API handler
func NewAPIHandler(cfg *config.OpenAPIConfig) *APIHandler {
	return &APIHandler{
		config: cfg,
		client: &http.Client{
			Timeout: cfg.Timeout,
		},
		evaluator: config.NewRequestEvaluator(),
	}
}

// HandleAPICall handles an API call based on the tool configuration
func (h *APIHandler) HandleAPICall(tool types.APITool, params map[string]interface{}, requestContext config.RequestContext) (interface{}, error) {
	// Log tool and parameters for debugging
	if h.config.Debug {
		log.Printf("DEBUG: Tool: %s (%s %s)", tool.Name, tool.Method, tool.Path)
		log.Printf("DEBUG: Tool description: %s", tool.Description)
		log.Printf("DEBUG: Parameters received: %+v", params)
		log.Printf("DEBUG: Request context: %+v", requestContext)
	}

	// Build the request URL
	requestURL, err := h.buildRequestURL(tool, params)
	if err != nil {
		return nil, fmt.Errorf("failed to build request URL: %w", err)
	}

	// Create HTTP request
	req, err := h.createRequest(tool, requestURL, params)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add authentication headers
	h.addAuthHeaders(req, requestContext)

	// Add custom headers (static and dynamic)
	// Convert headers map to http.Header for evaluation
	evaluatedHeaders, err := h.evaluator.EvaluateHeaders(h.config.Headers, requestContext)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate headers: %w", err)
	}

	for name, value := range evaluatedHeaders {
		req.Header.Set(name, value)
	}

	// Log request details for debugging
	if h.config.Debug {
		log.Printf("DEBUG: Making %s request to: %s", req.Method, req.URL.String())
		log.Printf("DEBUG: Request headers: %+v", req.Header)
		if req.Body != nil {
			// Read the body to log it, then recreate it
			bodyBytes, _ := io.ReadAll(req.Body)
			log.Printf("DEBUG: Request body: %s", string(bodyBytes))
			req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		}
	}

	// Make the request with retries
	var resp *http.Response
	for attempt := 0; attempt <= h.config.MaxRetries; attempt++ {
		if h.config.Debug && attempt > 0 {
			log.Printf("DEBUG: Retry attempt %d/%d", attempt, h.config.MaxRetries)
		}
		resp, err = h.client.Do(req)
		if err == nil {
			if h.config.Debug && attempt > 0 {
				log.Printf("DEBUG: Request succeeded on attempt %d", attempt+1)
			}
			break
		}
		if attempt < h.config.MaxRetries {
			if h.config.Debug {
				log.Printf("DEBUG: Request failed (attempt %d): %v, retrying in %d seconds", attempt+1, err, attempt+1)
			}
			time.Sleep(time.Duration(attempt+1) * time.Second)
		}
	}

	if err != nil {
		return nil, fmt.Errorf("failed to make request after %d attempts: %w", h.config.MaxRetries+1, err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Log response details for debugging
	if h.config.Debug {
		log.Printf("DEBUG: Response status: %d", resp.StatusCode)
		log.Printf("DEBUG: Response headers: %+v", resp.Header)
		log.Printf("DEBUG: Response body: %s", string(body))
	}

	// Handle response based on status code
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response body
	var result interface{}
	if len(body) > 0 {
		// Try to parse as JSON
		if err := json.Unmarshal(body, &result); err != nil {
			// If not JSON, return as string - this is valid for APIs that return plain text
			result = string(body)
		}
	}

	// Convert headers to a serializable map
	headers := make(map[string]string)
	for name, values := range resp.Header {
		if len(values) > 0 {
			headers[name] = values[0] // Take the first value
		}
	}

	return map[string]interface{}{
		"status_code": resp.StatusCode,
		"headers":     headers,
		"body":        result,
	}, nil
}

// buildRequestURL builds the complete request URL
func (h *APIHandler) buildRequestURL(tool types.APITool, params map[string]interface{}) (string, error) {
	// Start with base URL
	baseURL := h.config.BaseURL
	if baseURL == "" {
		return "", fmt.Errorf("base URL not configured")
	}

	// Ensure base URL ends with /
	if !strings.HasSuffix(baseURL, "/") {
		baseURL += "/"
	}

	// Remove leading / from path
	path := strings.TrimPrefix(tool.Path, "/")

	// Build URL
	requestURL := baseURL + path

	// Replace path parameters
	for _, param := range tool.Parameters {
		if param.In == "path" {
			paramValue, exists := params[param.Name]
			if !exists && param.Required {
				return "", fmt.Errorf("required path parameter '%s' not provided", param.Name)
			}
			if exists {
				placeholder := "{" + param.Name + "}"
				requestURL = strings.ReplaceAll(requestURL, placeholder, fmt.Sprintf("%v", paramValue))
			}
		}
	}

	// Add query parameters
	queryParams := url.Values{}
	for _, param := range tool.Parameters {
		if param.In == "query" {
			paramValue, exists := params[param.Name]
			if exists {
				queryParams.Add(param.Name, fmt.Sprintf("%v", paramValue))
			} else if param.Required {
				return "", fmt.Errorf("required query parameter '%s' not provided", param.Name)
			}
		}
	}

	// Add API key as query parameter if configured
	if h.config.Auth.Type == "api_key" && h.config.Auth.APIKeyIn == "query" {
		queryParams.Add(h.config.Auth.APIKeyName, h.config.Auth.APIKey)
	}

	// Append query parameters to URL
	if len(queryParams) > 0 {
		requestURL += "?" + queryParams.Encode()
	}

	return requestURL, nil
}

// createRequest creates an HTTP request
func (h *APIHandler) createRequest(tool types.APITool, requestURL string, params map[string]interface{}) (*http.Request, error) {
	var body io.Reader
	var contentType string

	// Handle request body for POST, PUT, PATCH methods
	if tool.RequestBody != nil && (tool.Method == "POST" || tool.Method == "PUT" || tool.Method == "PATCH") {
		// Look for body parameter in params
		if bodyData, exists := params["body"]; exists {
			switch v := bodyData.(type) {
			case string:
				body = strings.NewReader(v)
				contentType = "text/plain"
			case map[string]interface{}, []interface{}:
				jsonData, err := json.Marshal(v)
				if err != nil {
					return nil, fmt.Errorf("failed to marshal request body: %w", err)
				}
				body = bytes.NewReader(jsonData)
				contentType = "application/json"
			default:
				body = strings.NewReader(fmt.Sprintf("%v", v))
				contentType = "text/plain"
			}
		}
	}

	// Create request
	req, err := http.NewRequest(tool.Method, requestURL, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set content type if we have a body
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	// Add header parameters
	for _, param := range tool.Parameters {
		if param.In == "header" {
			paramValue, exists := params[param.Name]
			if exists {
				req.Header.Set(param.Name, fmt.Sprintf("%v", paramValue))
			} else if param.Required {
				return nil, fmt.Errorf("required header parameter '%s' not provided", param.Name)
			}
		}
	}

	return req, nil
}

// addAuthHeaders adds authentication headers to the request
func (h *APIHandler) addAuthHeaders(req *http.Request, requestContext config.RequestContext) {
	switch h.config.Auth.Type {
	case "bearer":
		if h.config.Auth.Token != "" {
			req.Header.Set("Authorization", "Bearer "+h.config.Auth.Token)
		}
	case "basic":
		if h.config.Auth.Username != "" && h.config.Auth.Password != "" {
			req.SetBasicAuth(h.config.Auth.Username, h.config.Auth.Password)
		}
	case "api_key":
		if h.config.Auth.APIKey != "" && h.config.Auth.APIKeyName != "" && h.config.Auth.APIKeyIn == "header" {
			req.Header.Set(h.config.Auth.APIKeyName, h.config.Auth.APIKey)
		}
	}

	// Add custom auth headers (static and dynamic)
	evaluatedAuthHeaders, err := h.evaluator.EvaluateHeaders(h.config.Auth.Headers, requestContext)
	if err != nil {
		// Log error but continue - don't fail the request
		log.Printf("Warning: failed to evaluate auth headers: %v", err)
	} else {
		for name, value := range evaluatedAuthHeaders {
			req.Header.Set(name, value)
		}
	}
}
