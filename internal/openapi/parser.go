package openapi

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"

	"mcpify/internal/config"
	"mcpify/internal/types"

	"github.com/getkin/kin-openapi/openapi2"
	"github.com/getkin/kin-openapi/openapi2conv"
	"github.com/getkin/kin-openapi/openapi3"
)

// Parser handles OpenAPI specification parsing and tool generation
type Parser struct {
	config    *config.OpenAPIConfig
	client    *http.Client
	evaluator *config.RequestEvaluator
}

// NewParser creates a new OpenAPI parser
func NewParser(cfg *config.OpenAPIConfig) *Parser {
	return &Parser{
		config: cfg,
		client: &http.Client{
			Timeout: cfg.Timeout,
		},
		evaluator: config.NewRequestEvaluator(),
	}
}

// ParseSpec parses an OpenAPI specification and returns generated tools
func (p *Parser) ParseSpec() ([]types.APITool, error) {
	log.Printf("Starting to parse OpenAPI spec")
	// Load OpenAPI spec
	spec, err := p.loadSpec()
	if err != nil {
		return nil, fmt.Errorf("failed to load OpenAPI spec: %w", err)
	}
	log.Printf("Successfully loaded spec, starting tool generation")

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

	log.Printf("Loading OpenAPI spec from: %s", p.config.SpecPath)

	// Check if spec path is a URL
	if strings.HasPrefix(p.config.SpecPath, "http://") || strings.HasPrefix(p.config.SpecPath, "https://") {
		content, err = p.loadFromURL(p.config.SpecPath)
	} else {
		content, err = p.loadFromFile(p.config.SpecPath)
	}

	if err != nil {
		return nil, err
	}

	log.Printf("Successfully loaded spec, content length: %d bytes", len(content))

	// Check if it's Swagger 2.0 first
	var swagger2Spec openapi2.T
	swaggerErr := swagger2Spec.UnmarshalJSON(content)
	log.Printf("Swagger 2.0 unmarshal error: %v", swaggerErr)
	log.Printf("Swagger version: %s", swagger2Spec.Swagger)

	var spec *openapi3.T
	if swaggerErr == nil && swagger2Spec.Swagger == "2.0" {
		log.Printf("Detected Swagger 2.0 spec, converting to OpenAPI 3.x")
		// Convert Swagger 2.0 to OpenAPI 3.x
		spec, err = p.convertSwagger2ToOpenAPI3(&swagger2Spec)
		if err != nil {
			return nil, fmt.Errorf("failed to convert Swagger 2.0 to OpenAPI 3.x: %w", err)
		}
		log.Printf("Swagger 2.0 conversion succeeded")
	} else {
		log.Printf("Trying to parse as OpenAPI 3.x")
		// Try to parse as OpenAPI 3.x
		loader := openapi3.NewLoader()
		loader.IsExternalRefsAllowed = true

		spec, err = loader.LoadFromData(content)
		if err != nil {
			log.Printf("OpenAPI 3.x parsing failed: %v", err)
			return nil, fmt.Errorf("failed to parse OpenAPI spec: %w", err)
		}
		log.Printf("OpenAPI 3.x parsing succeeded")
	}

	// Skip validation for converted specs
	log.Printf("Skipping validation for spec")
	// if err := spec.Validate(loader.Context); err != nil {
	//	return nil, fmt.Errorf("OpenAPI spec validation failed: %w", err)
	// }

	return spec, nil
}

// convertSwagger2ToOpenAPI3 converts a Swagger 2.0 spec to OpenAPI 3.x using kin-openapi
func (p *Parser) convertSwagger2ToOpenAPI3(swagger2 *openapi2.T) (*openapi3.T, error) {
	log.Printf("Converting Swagger 2.0 spec with title: %s, version: %s", swagger2.Info.Title, swagger2.Info.Version)

	// Use the official kin-openapi conversion function
	spec, err := openapi2conv.ToV3(swagger2)
	if err != nil {
		return nil, fmt.Errorf("failed to convert Swagger 2.0 to OpenAPI 3.x: %w", err)
	}

	log.Printf("Conversion completed successfully using kin-openapi")
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

	// Add custom headers (static and dynamic)
	evaluatedHeaders, err := p.evaluateHeaders(p.config.Headers, req.Header)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate headers: %w", err)
	}

	for name, value := range evaluatedHeaders {
		req.Header.Set(name, value)
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

	// Add custom auth headers (static and dynamic)
	evaluatedAuthHeaders, err := p.evaluateHeaders(p.config.Auth.Headers, req.Header)
	if err != nil {
		// Log error but continue - don't fail the request
		// TODO: Add proper logging
		log.Printf("Warning: failed to evaluate auth headers: %v", err)
	} else {
		for name, value := range evaluatedAuthHeaders {
			req.Header.Set(name, value)
		}
	}
}

// evaluateHeaders evaluates dynamic headers using the request evaluator
func (p *Parser) evaluateHeaders(headers config.HeadersConfig, requestHeaders http.Header) (map[string]string, error) {
	// Create a minimal request context for OpenAPI spec fetching
	requestContext := config.RequestContext{
		Headers: make(map[string]string),
		Query:   make(map[string]string),
		Form:    make(map[string]string),
		Method:  "GET",
		Path:    "/",
	}

	// Convert HTTP headers to map (normalize to lowercase for case-insensitive matching)
	for name, values := range requestHeaders {
		if len(values) > 0 {
			requestContext.Headers[strings.ToLower(name)] = values[0] // Take first value
		}
	}

	return p.evaluator.EvaluateHeaders(headers, requestContext)
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
		// Resolve schema references if present
		if content.Schema != nil {
			resolvedSchema := p.resolveSchemaRef(content.Schema)
			requestBody.Content[mediaType] = map[string]interface{}{
				"schema": resolvedSchema,
			}
		} else {
			requestBody.Content[mediaType] = content
		}
	}

	return requestBody
}

// resolveSchemaRef resolves a schema reference to its actual schema definition
func (p *Parser) resolveSchemaRef(schemaRef *openapi3.SchemaRef) map[string]interface{} {
	// If the schema reference has a resolved value, use it
	if schemaRef.Value != nil {
		return p.schemaToMap(schemaRef.Value)
	}

	// If it's just a reference without a resolved value, return the reference
	// This handles cases where the reference couldn't be resolved
	if schemaRef.Ref != "" {
		return map[string]interface{}{
			"$ref": schemaRef.Ref,
		}
	}

	// Fallback to empty object
	return map[string]interface{}{
		"type": "object",
	}
}

// schemaToMap converts an OpenAPI schema to a map for JSON serialization
func (p *Parser) schemaToMap(schema *openapi3.Schema) map[string]interface{} {
	result := make(map[string]interface{})

	// Add basic schema properties
	if schema.Type != nil && len(schema.Type.Slice()) > 0 {
		types := schema.Type.Slice()
		if len(types) == 1 {
			result["type"] = types[0]
		} else {
			result["type"] = types
		}
	}
	if schema.Description != "" {
		result["description"] = schema.Description
	}
	if schema.Format != "" {
		result["format"] = schema.Format
	}
	if schema.Example != nil {
		result["example"] = schema.Example
	}

	// Handle array types
	if schema.Type != nil && schema.Type.Is("array") && schema.Items != nil {
		if schema.Items.Value != nil {
			result["items"] = p.schemaToMap(schema.Items.Value)
		} else if schema.Items.Ref != "" {
			result["items"] = map[string]interface{}{
				"$ref": schema.Items.Ref,
			}
		}
	}

	// Handle object properties
	if len(schema.Properties) > 0 {
		properties := make(map[string]interface{})
		for propName, propRef := range schema.Properties {
			if propRef.Value != nil {
				properties[propName] = p.schemaToMap(propRef.Value)
			} else if propRef.Ref != "" {
				properties[propName] = map[string]interface{}{
					"$ref": propRef.Ref,
				}
			}
		}
		result["properties"] = properties
	}

	// Handle required fields
	if len(schema.Required) > 0 {
		result["required"] = schema.Required
	}

	// Handle additional properties
	if schema.AdditionalProperties.Schema != nil {
		if schema.AdditionalProperties.Schema.Value != nil {
			result["additionalProperties"] = p.schemaToMap(schema.AdditionalProperties.Schema.Value)
		} else if schema.AdditionalProperties.Schema.Ref != "" {
			result["additionalProperties"] = map[string]interface{}{
				"$ref": schema.AdditionalProperties.Schema.Ref,
			}
		}
	} else if schema.AdditionalProperties.Has != nil {
		result["additionalProperties"] = *schema.AdditionalProperties.Has
	}

	// Handle enum values
	if len(schema.Enum) > 0 {
		result["enum"] = schema.Enum
	}

	// Handle default value
	if schema.Default != nil {
		result["default"] = schema.Default
	}

	// Handle minimum/maximum for numbers
	if schema.Min != nil {
		result["minimum"] = *schema.Min
	}
	if schema.Max != nil {
		result["maximum"] = *schema.Max
	}

	// Handle minLength/maxLength for strings
	if schema.MinLength > 0 {
		result["minLength"] = schema.MinLength
	}
	if schema.MaxLength != nil {
		result["maxLength"] = *schema.MaxLength
	}

	// Handle pattern for strings
	if schema.Pattern != "" {
		result["pattern"] = schema.Pattern
	}

	return result
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
