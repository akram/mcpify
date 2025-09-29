package openapi

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"

	"mcpify/internal/config"
	"mcpify/internal/types"

	"github.com/getkin/kin-openapi/openapi2"
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
	fmt.Printf("Starting to parse OpenAPI spec\n")
	// Load OpenAPI spec
	spec, err := p.loadSpec()
	if err != nil {
		return nil, fmt.Errorf("failed to load OpenAPI spec: %w", err)
	}
	fmt.Printf("Successfully loaded spec, starting tool generation\n")

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

	fmt.Printf("Loading OpenAPI spec from: %s\n", p.config.SpecPath)

	// Check if spec path is a URL
	if strings.HasPrefix(p.config.SpecPath, "http://") || strings.HasPrefix(p.config.SpecPath, "https://") {
		content, err = p.loadFromURL(p.config.SpecPath)
	} else {
		content, err = p.loadFromFile(p.config.SpecPath)
	}

	if err != nil {
		return nil, err
	}

	fmt.Printf("Successfully loaded spec, content length: %d bytes\n", len(content))

	// Check if it's Swagger 2.0 first
	var swagger2Spec openapi2.T
	swaggerErr := swagger2Spec.UnmarshalJSON(content)
	fmt.Printf("Swagger 2.0 unmarshal error: %v\n", swaggerErr)
	fmt.Printf("Swagger version: %s\n", swagger2Spec.Swagger)

	var spec *openapi3.T
	if swaggerErr == nil && swagger2Spec.Swagger == "2.0" {
		fmt.Printf("Detected Swagger 2.0 spec, converting to OpenAPI 3.x\n")
		// Convert Swagger 2.0 to OpenAPI 3.x
		spec, err = p.convertSwagger2ToOpenAPI3(&swagger2Spec)
		if err != nil {
			return nil, fmt.Errorf("failed to convert Swagger 2.0 to OpenAPI 3.x: %w", err)
		}
		fmt.Printf("Swagger 2.0 conversion succeeded\n")
	} else {
		fmt.Printf("Trying to parse as OpenAPI 3.x\n")
		// Try to parse as OpenAPI 3.x
		loader := openapi3.NewLoader()
		loader.IsExternalRefsAllowed = true

		spec, err = loader.LoadFromData(content)
		if err != nil {
			fmt.Printf("OpenAPI 3.x parsing failed: %v\n", err)
			return nil, fmt.Errorf("failed to parse OpenAPI spec: %w", err)
		}
		fmt.Printf("OpenAPI 3.x parsing succeeded\n")
	}

	// Skip validation for converted specs
	fmt.Printf("Skipping validation for spec\n")
	// if err := spec.Validate(loader.Context); err != nil {
	//	return nil, fmt.Errorf("OpenAPI spec validation failed: %w", err)
	// }

	return spec, nil
}

// convertSwagger2ToOpenAPI3 converts a Swagger 2.0 spec to OpenAPI 3.x
func (p *Parser) convertSwagger2ToOpenAPI3(swagger2 *openapi2.T) (*openapi3.T, error) {
	fmt.Printf("Converting Swagger 2.0 spec with title: %s, version: %s\n", swagger2.Info.Title, swagger2.Info.Version)
	// Create a basic OpenAPI 3.x spec
	spec := &openapi3.T{
		OpenAPI: "3.0.0",
		Info: &openapi3.Info{
			Title:   swagger2.Info.Title,
			Version: swagger2.Info.Version,
		},
		Paths: &openapi3.Paths{},
	}

	// Convert paths
	fmt.Printf("Converting %d paths\n", len(swagger2.Paths))
	for path, pathItem := range swagger2.Paths {
		openapi3PathItem := &openapi3.PathItem{}

		// Convert operations
		if pathItem.Get != nil {
			openapi3PathItem.Get = p.convertOperation(pathItem.Get)
		}
		if pathItem.Post != nil {
			openapi3PathItem.Post = p.convertOperation(pathItem.Post)
		}
		if pathItem.Put != nil {
			openapi3PathItem.Put = p.convertOperation(pathItem.Put)
		}
		if pathItem.Delete != nil {
			openapi3PathItem.Delete = p.convertOperation(pathItem.Delete)
		}
		if pathItem.Patch != nil {
			openapi3PathItem.Patch = p.convertOperation(pathItem.Patch)
		}

		spec.Paths.Set(path, openapi3PathItem)
	}

	fmt.Printf("Conversion completed, returning spec\n")
	return spec, nil
}

// convertOperation converts a Swagger 2.0 operation to OpenAPI 3.x
func (p *Parser) convertOperation(op *openapi2.Operation) *openapi3.Operation {
	fmt.Printf("Converting operation: %s\n", op.OperationID)
	operation := &openapi3.Operation{
		OperationID: op.OperationID,
		Summary:     op.Summary,
		Description: op.Description,
		Tags:        op.Tags,
	}

	// Convert parameters
	fmt.Printf("Converting %d parameters\n", len(op.Parameters))
	for _, param := range op.Parameters {
		openapi3Param := &openapi3.Parameter{
			Name:        param.Name,
			In:          param.In,
			Description: param.Description,
			Required:    param.Required,
		}

		// Convert schema if present
		if param.Schema != nil {
			openapi3Param.Schema = &openapi3.SchemaRef{
				Value: &openapi3.Schema{
					Type: param.Schema.Value.Type,
				},
			}
		}

		operation.Parameters = append(operation.Parameters, &openapi3.ParameterRef{Value: openapi3Param})
	}

	fmt.Printf("Operation conversion completed\n")
	return operation
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
	defer func() {
		_ = resp.Body.Close()
	}()

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
	// Always generate name from path and method to ensure uniqueness
	// This avoids issues with duplicate operation IDs in the spec
	toolName := p.generateSnakeCaseName(path, method)

	// Add prefix if specified
	if p.config.ToolPrefix != "" {
		return p.config.ToolPrefix + "_" + toolName
	}

	return toolName
}

// generateSnakeCaseName generates a snake_case tool name from path and method
func (p *Parser) generateSnakeCaseName(path, method string) string {
	// Remove leading slash
	path = strings.TrimPrefix(path, "/")

	// Split by path segments
	segments := strings.Split(path, "/")
	var result strings.Builder

	// Add method as first part
	result.WriteString(strings.ToLower(method))

	for _, segment := range segments {
		if segment == "" {
			continue
		}

		// Handle path parameters like {username}
		if strings.HasPrefix(segment, "{") && strings.HasSuffix(segment, "}") {
			paramName := strings.Trim(segment, "{}")
			result.WriteString("_by_" + strings.ToLower(paramName))
		} else {
			// Add segment in lowercase
			result.WriteString("_" + strings.ToLower(segment))
		}
	}

	return result.String()
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
