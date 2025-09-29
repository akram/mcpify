package openapi

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"unicode"

	"mcpify/internal/config"
	"mcpify/internal/types"

	"github.com/getkin/kin-openapi/openapi3"
)

// Parser handles OpenAPI specification parsing and tool generation
type Parser struct {
	config *config.OpenAPIConfig
	client *http.Client
}

// NewParser creates a new OpenAPI parser
func NewParser(cfg *config.OpenAPIConfig) *Parser {
	return &Parser{
		config: cfg,
		client: &http.Client{
			Timeout: cfg.Timeout,
		},
	}
}

// ParseSpec parses an OpenAPI specification and returns generated tools
func (p *Parser) ParseSpec() ([]types.APITool, error) {
	// Load OpenAPI spec
	spec, err := p.loadSpec()
	if err != nil {
		return nil, fmt.Errorf("failed to load OpenAPI spec: %w", err)
	}

	// Generate tools from spec
	tools, err := p.generateTools(spec)
	if err != nil {
		return nil, fmt.Errorf("failed to generate tools: %w", err)
	}

	return tools, nil
}

// loadSpec loads OpenAPI specification from file or URL
func (p *Parser) loadSpec() (*openapi3.T, error) {
	var content []byte
	var err error

	// fmt.Printf("Loading OpenAPI spec from: %s\n", p.config.SpecPath)

	// Check if spec path is a URL
	if strings.HasPrefix(p.config.SpecPath, "http://") || strings.HasPrefix(p.config.SpecPath, "https://") {
		content, err = p.loadFromURL(p.config.SpecPath)
	} else {
		content, err = p.loadFromFile(p.config.SpecPath)
	}

	if err != nil {
		return nil, err
	}

	// fmt.Printf("Successfully loaded spec, content length: %d bytes\n", len(content))

	// Parse the spec using kin-openapi
	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = true

	spec, err := loader.LoadFromData(content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse OpenAPI spec: %w", err)
	}

	// Validate the spec
	if err := spec.Validate(loader.Context); err != nil {
		return nil, fmt.Errorf("OpenAPI spec validation failed: %w", err)
	}

	return spec, nil
}

// loadFromFile loads OpenAPI spec from a local file
func (p *Parser) loadFromFile(path string) ([]byte, error) {
	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("OpenAPI spec file not found: %s", path)
	}

	// Read file content
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read OpenAPI spec file: %w", err)
	}

	return content, nil
}

// loadFromURL loads OpenAPI spec from a URL
func (p *Parser) loadFromURL(url string) ([]byte, error) {
	// Create request with authentication headers
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add authentication headers
	p.addAuthHeaders(req)

	// Add custom headers
	for key, value := range p.config.Headers {
		req.Header.Set(key, value)
	}

	// Make request
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch OpenAPI spec: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch OpenAPI spec: HTTP %d", resp.StatusCode)
	}

	// Read response body
	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return content, nil
}

// addAuthHeaders adds authentication headers to the request
func (p *Parser) addAuthHeaders(req *http.Request) {
	switch p.config.Auth.Type {
	case "bearer":
		if p.config.Auth.Token != "" {
			req.Header.Set("Authorization", "Bearer "+p.config.Auth.Token)
		}
	case "basic":
		if p.config.Auth.Username != "" && p.config.Auth.Password != "" {
			req.SetBasicAuth(p.config.Auth.Username, p.config.Auth.Password)
		}
	case "api_key":
		if p.config.Auth.APIKey != "" && p.config.Auth.APIKeyName != "" {
			switch p.config.Auth.APIKeyIn {
			case "header":
				req.Header.Set(p.config.Auth.APIKeyName, p.config.Auth.APIKey)
			case "query":
				// This would be handled when building the URL
			}
		}
	}

	// Add custom auth headers
	for key, value := range p.config.Auth.Headers {
		req.Header.Set(key, value)
	}
}

// generateTools generates MCP tools from OpenAPI specification
func (p *Parser) generateTools(spec *openapi3.T) ([]types.APITool, error) {
	var tools []types.APITool

	// fmt.Printf("Generating tools from spec with %d paths\n", len(spec.Paths.Map()))

	// Iterate through all paths and operations
	for path, pathItem := range spec.Paths.Map() {
		// fmt.Printf("Processing path: %s\n", path)
		// Check if path should be excluded
		if p.shouldExcludePath(path) {
			continue
		}

		// Check if path should be included (if include list is specified)
		if !p.shouldIncludePath(path) {
			continue
		}

		// Generate tools for each HTTP method
		operations := []struct {
			method string
			op     *openapi3.Operation
		}{
			{"GET", pathItem.Get},
			{"POST", pathItem.Post},
			{"PUT", pathItem.Put},
			{"DELETE", pathItem.Delete},
			{"PATCH", pathItem.Patch},
		}

		for _, opInfo := range operations {
			if opInfo.op == nil {
				continue
			}

			tool, err := p.generateToolFromOperation(path, opInfo.method, opInfo.op)
			if err != nil {
				return nil, fmt.Errorf("failed to generate tool for %s %s: %w", opInfo.method, path, err)
			}

			tools = append(tools, tool)
		}
	}

	return tools, nil
}

// generateToolFromOperation generates a single MCP tool from an OpenAPI operation
func (p *Parser) generateToolFromOperation(path, method string, operation *openapi3.Operation) (types.APITool, error) {
	// Generate tool name
	toolName := p.generateToolName(path, method, operation)

	// Generate tool description
	description := p.generateToolDescription(operation)

	// Extract parameters
	parameters := p.extractParameters(operation)

	// Extract request body
	requestBody := p.extractRequestBody(operation)

	// Create tool
	tool := types.APITool{
		Name:        toolName,
		Description: description,
		Method:      method,
		Path:        path,
		Parameters:  parameters,
		RequestBody: requestBody,
	}

	return tool, nil
}

// generateToolName generates a unique tool name from path, method, and operation
func (p *Parser) generateToolName(path, method string, operation *openapi3.Operation) string {
	// Use operationId if available
	if operation.OperationID != "" {
		if p.config.ToolPrefix != "" {
			return p.config.ToolPrefix + "_" + operation.OperationID
		}
		return operation.OperationID
	}

	// Generate name from path and method
	// Convert path to camelCase
	pathName := p.pathToCamelCase(path)
	methodName := strings.ToLower(method)

	// Combine method and path
	toolName := methodName + pathName

	// Add prefix if specified
	if p.config.ToolPrefix != "" {
		return p.config.ToolPrefix + toolName
	}

	return toolName
}

// pathToCamelCase converts a path like "/user/{username}" to "UserByUsername"
func (p *Parser) pathToCamelCase(path string) string {
	// Remove leading slash
	path = strings.TrimPrefix(path, "/")

	// Split by path segments
	segments := strings.Split(path, "/")
	var result strings.Builder

	for _, segment := range segments {
		if segment == "" {
			continue
		}

		// Handle path parameters like {username}
		if strings.HasPrefix(segment, "{") && strings.HasSuffix(segment, "}") {
			paramName := strings.Trim(segment, "{}")
			result.WriteString("By" + p.titleCase(paramName))
		} else {
			// Capitalize first letter of each segment
			result.WriteString(p.titleCase(segment))
		}
	}

	return result.String()
}

// titleCase capitalizes the first letter of a string
func (p *Parser) titleCase(s string) string {
	if s == "" {
		return s
	}
	runes := []rune(s)
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}

// generateToolDescription generates a description for the tool
func (p *Parser) generateToolDescription(operation *openapi3.Operation) string {
	if operation.Summary != "" {
		return operation.Summary
	}
	if operation.Description != "" {
		return operation.Description
	}
	return "API endpoint"
}

// extractParameters extracts parameters from OpenAPI operation
func (p *Parser) extractParameters(operation *openapi3.Operation) []types.OpenAPIParameter {
	var parameters []types.OpenAPIParameter

	for _, param := range operation.Parameters {
		if param.Value == nil {
			continue
		}

		parameter := types.OpenAPIParameter{
			Name:        param.Value.Name,
			In:          param.Value.In,
			Description: param.Value.Description,
			Required:    param.Value.Required,
		}

		// Convert schema to interface{} for JSON serialization
		if param.Value.Schema != nil {
			parameter.Schema = param.Value.Schema.Value
		}

		parameters = append(parameters, parameter)
	}

	return parameters
}

// extractRequestBody extracts request body from OpenAPI operation
func (p *Parser) extractRequestBody(operation *openapi3.Operation) *types.OpenAPIRequestBody {
	if operation.RequestBody == nil || operation.RequestBody.Value == nil {
		return nil
	}

	requestBody := &types.OpenAPIRequestBody{
		Description: operation.RequestBody.Value.Description,
		Required:    operation.RequestBody.Value.Required,
		Content:     make(map[string]interface{}),
	}

	// Convert content to interface{} for JSON serialization
	for mediaType, content := range operation.RequestBody.Value.Content {
		requestBody.Content[mediaType] = content
	}

	return requestBody
}

// shouldExcludePath checks if a path should be excluded
func (p *Parser) shouldExcludePath(path string) bool {
	for _, excludePath := range p.config.ExcludePaths {
		if p.matchPath(excludePath, path) {
			return true
		}
	}
	return false
}

// shouldIncludePath checks if a path should be included
func (p *Parser) shouldIncludePath(path string) bool {
	// If no include paths specified, include all
	if len(p.config.IncludePaths) == 0 {
		return true
	}

	// Check if path matches any include pattern
	for _, includePath := range p.config.IncludePaths {
		if p.matchPath(includePath, path) {
			return true
		}
	}
	return false
}

// matchPath matches a pattern against a path, supporting wildcards
func (p *Parser) matchPath(pattern, path string) bool {
	// Simple wildcard matching for URL paths
	if strings.Contains(pattern, "*") {
		// Convert pattern to regex
		regexPattern := strings.ReplaceAll(pattern, "*", ".*")
		regexPattern = "^" + regexPattern + "$"

		matched, err := regexp.MatchString(regexPattern, path)
		if err != nil {
			return false
		}
		return matched
	}

	// Exact match
	return pattern == path
}
