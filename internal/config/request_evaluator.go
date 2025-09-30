package config

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/PaesslerAG/jsonpath"
)

// RequestContext represents the full HTTP request context for evaluation
type RequestContext struct {
	Headers map[string]string      `json:"headers"`
	Query   map[string]string      `json:"query"`
	Form    map[string]string      `json:"form"`
	Body    interface{}            `json:"body,omitempty"`
	Method  string                 `json:"method"`
	Path    string                 `json:"path"`
	RawData map[string]interface{} `json:"raw_data,omitempty"` // For additional context
}

// RequestEvaluator handles evaluation of JSONPath expressions against request context
type RequestEvaluator struct{}

// NewRequestEvaluator creates a new request evaluator
func NewRequestEvaluator() *RequestEvaluator {
	return &RequestEvaluator{}
}

// EvaluateHeaders processes headers and evaluates valueFrom expressions
func (e *RequestEvaluator) EvaluateHeaders(headers HeadersConfig, requestContext RequestContext) (map[string]string, error) {
	result := make(map[string]string)

	// Process each header
	for _, item := range headers {
		if item.Header.Value != "" {
			// Static value
			result[item.Header.Name] = item.Header.Value
		} else if item.Header.ValueFrom != "" {
			// Dynamic value - evaluate JSONPath
			value, err := e.evaluateValueFrom(item.Header.ValueFrom, requestContext)
			if err != nil {
				return nil, fmt.Errorf("failed to evaluate header %s: %w", item.Header.Name, err)
			}
			if value != "" {
				result[item.Header.Name] = value
			}
		}
	}

	return result, nil
}

// evaluateValueFrom evaluates a JSONPath expression against the request context
func (e *RequestEvaluator) evaluateValueFrom(expression string, requestContext RequestContext) (string, error) {
	// Convert the expression to use the correct JSONPath syntax
	jsonPathExpr := e.convertExpressionToJSONPath(expression)

	// Convert context to JSON for evaluation
	contextJSON, err := json.Marshal(requestContext)
	if err != nil {
		return "", fmt.Errorf("failed to marshal context: %w", err)
	}

	var contextData interface{}
	if err := json.Unmarshal(contextJSON, &contextData); err != nil {
		return "", fmt.Errorf("failed to unmarshal context: %w", err)
	}

	// Check if this is a nested expression that needs special handling
	if e.hasNestedPath(expression) {
		return e.evaluateNestedExpression(expression, contextData)
	}

	// Evaluate the JSONPath expression
	result, err := jsonpath.Get(jsonPathExpr, contextData)
	if err != nil {
		// If the path doesn't exist, return empty string instead of error
		if strings.Contains(err.Error(), "unknown key") || strings.Contains(err.Error(), "not found") {
			return "", nil
		}
		return "", fmt.Errorf("failed to evaluate JSONPath expression '%s': %w", expression, err)
	}

	// Convert result to string
	switch v := result.(type) {
	case string:
		return v, nil
	case nil:
		return "", nil
	default:
		// Convert other types to string
		resultJSON, err := json.Marshal(v)
		if err != nil {
			return "", fmt.Errorf("failed to convert result to string: %w", err)
		}
		return string(resultJSON), nil
	}
}

// convertExpressionToJSONPath converts our custom expression format to JSONPath
func (e *RequestEvaluator) convertExpressionToJSONPath(expression string) string {
	// Handle expressions like:
	// request.headers['key'] -> $.headers["key"]
	// request.query['param'] -> $.query["param"]
	// request.form['field'] -> $.form["field"]
	// request.headers['key'].nested -> $.headers["key"].nested

	// Remove 'request.' prefix if present
	if len(expression) > 8 && expression[:8] == "request." {
		expression = expression[8:]
	}

	// Handle different request data types
	if strings.HasPrefix(expression, "headers[") {
		return e.convertHeaderExpression(expression)
	} else if strings.HasPrefix(expression, "query[") {
		return e.convertQueryExpression(expression)
	} else if strings.HasPrefix(expression, "form[") {
		return e.convertFormExpression(expression)
	} else if strings.HasPrefix(expression, "body.") {
		return e.convertBodyExpression(expression)
	}

	// Fallback: assume it's already a JSONPath expression
	return expression
}

// convertHeaderExpression converts header expressions to JSONPath
func (e *RequestEvaluator) convertHeaderExpression(expression string) string {
	// Find the opening bracket
	openBracket := strings.Index(expression, "[")
	if openBracket == -1 {
		return expression
	}

	// Find the closing bracket
	closeBracket := -1
	for i := openBracket + 1; i < len(expression); i++ {
		if expression[i] == ']' {
			closeBracket = i
			break
		}
	}

	if closeBracket == -1 {
		return expression
	}

	// Extract the key (remove quotes)
	key := expression[openBracket+1 : closeBracket]
	key = strings.Trim(key, "'\"")

	// Get remaining path after the bracket
	remaining := expression[closeBracket+1:]

	// Convert to JSONPath format
	jsonPath := fmt.Sprintf("$.headers[\"%s\"]", strings.ToLower(key))

	// Add nested path if present
	if len(remaining) > 0 && remaining[0] == '.' {
		jsonPath += remaining
	}

	return jsonPath
}

// convertQueryExpression converts query parameter expressions to JSONPath
func (e *RequestEvaluator) convertQueryExpression(expression string) string {
	// Find the opening bracket
	openBracket := strings.Index(expression, "[")
	if openBracket == -1 {
		return expression
	}

	// Find the closing bracket
	closeBracket := -1
	for i := openBracket + 1; i < len(expression); i++ {
		if expression[i] == ']' {
			closeBracket = i
			break
		}
	}

	if closeBracket == -1 {
		return expression
	}

	// Extract the key (remove quotes)
	key := expression[openBracket+1 : closeBracket]
	key = strings.Trim(key, "'\"")

	// Get remaining path after the bracket
	remaining := expression[closeBracket+1:]

	// Convert to JSONPath format
	jsonPath := fmt.Sprintf("$.query[\"%s\"]", key)

	// Add nested path if present
	if len(remaining) > 0 && remaining[0] == '.' {
		jsonPath += remaining
	}

	return jsonPath
}

// convertFormExpression converts form data expressions to JSONPath
func (e *RequestEvaluator) convertFormExpression(expression string) string {
	// Find the opening bracket
	openBracket := strings.Index(expression, "[")
	if openBracket == -1 {
		return expression
	}

	// Find the closing bracket
	closeBracket := -1
	for i := openBracket + 1; i < len(expression); i++ {
		if expression[i] == ']' {
			closeBracket = i
			break
		}
	}

	if closeBracket == -1 {
		return expression
	}

	// Extract the key (remove quotes)
	key := expression[openBracket+1 : closeBracket]
	key = strings.Trim(key, "'\"")

	// Get remaining path after the bracket
	remaining := expression[closeBracket+1:]

	// Convert to JSONPath format
	jsonPath := fmt.Sprintf("$.form[\"%s\"]", key)

	// Add nested path if present
	if len(remaining) > 0 && remaining[0] == '.' {
		jsonPath += remaining
	}

	return jsonPath
}

// convertBodyExpression converts request body expressions to JSONPath
func (e *RequestEvaluator) convertBodyExpression(expression string) string {
	// Remove 'body.' prefix and convert to JSONPath
	bodyPath := expression[5:] // Remove "body."
	result := "$.body." + bodyPath
	return result
}

// isJSONString checks if a string is valid JSON
func (e *RequestEvaluator) isJSONString(s string) bool {
	var js json.RawMessage
	return json.Unmarshal([]byte(s), &js) == nil
}

// extractFromJSONString extracts a value from a JSON string using the nested path
func (e *RequestEvaluator) extractFromJSONString(jsonStr, originalExpression string) (string, error) {
	// Parse the JSON string
	var jsonData interface{}
	if err := json.Unmarshal([]byte(jsonStr), &jsonData); err != nil {
		return "", fmt.Errorf("failed to parse JSON string: %w", err)
	}

	// Extract the nested path from the original expression
	nestedPath := e.extractNestedPath(originalExpression)
	if nestedPath == "" {
		// No nested path, return the whole JSON as string
		return jsonStr, nil
	}

	// Evaluate the nested path
	result, err := jsonpath.Get("$"+nestedPath, jsonData)
	if err != nil {
		// If the path doesn't exist, return empty string instead of error
		if strings.Contains(err.Error(), "unknown key") {
			return "", nil
		}
		return "", fmt.Errorf("failed to evaluate nested JSONPath: %w", err)
	}

	// Convert result to string
	switch v := result.(type) {
	case string:
		return v, nil
	case nil:
		return "", nil
	default:
		// Convert other types to string
		resultJSON, err := json.Marshal(v)
		if err != nil {
			return "", fmt.Errorf("failed to convert result to string: %w", err)
		}
		return string(resultJSON), nil
	}
}

// extractNestedPath extracts the nested path from the original expression
func (e *RequestEvaluator) extractNestedPath(expression string) string {
	// Find the position after the key
	// e.g., request.headers['x-mcpify-provider-data'].apikey -> .apikey
	// e.g., request.query['data'].nested -> .nested

	// Look for the pattern: [key'] or [key"]
	keyEnd := -1
	for i := 0; i < len(expression); i++ {
		if expression[i] == ']' {
			keyEnd = i
			break
		}
	}

	if keyEnd != -1 && keyEnd+1 < len(expression) {
		// Return everything after the closing bracket
		return expression[keyEnd+1:]
	}

	return ""
}

// NewRequestContextFromHTTP creates a RequestContext from HTTP request data
func NewRequestContextFromHTTP(headers map[string][]string, query url.Values, form url.Values, method, path string) RequestContext {
	ctx := RequestContext{
		Headers: make(map[string]string),
		Query:   make(map[string]string),
		Form:    make(map[string]string),
		Method:  method,
		Path:    path,
	}

	// Convert headers to map (normalize to lowercase for case-insensitive matching)
	for name, values := range headers {
		if len(values) > 0 {
			ctx.Headers[strings.ToLower(name)] = values[0] // Take first value
		}
	}

	// Convert query parameters to map
	for name, values := range query {
		if len(values) > 0 {
			ctx.Query[name] = values[0] // Take first value
		}
	}

	// Convert form data to map
	for name, values := range form {
		if len(values) > 0 {
			ctx.Form[name] = values[0] // Take first value
		}
	}

	return ctx
}

// NewRequestContextFromMap creates a RequestContext from a map (for testing)
func NewRequestContextFromMap(headers, query, form map[string]string, method, path string) RequestContext {
	ctx := RequestContext{
		Headers: make(map[string]string),
		Query:   make(map[string]string),
		Form:    make(map[string]string),
		Method:  method,
		Path:    path,
	}

	// Copy headers (normalize to lowercase)
	for name, value := range headers {
		ctx.Headers[strings.ToLower(name)] = value
	}

	// Copy query parameters
	for name, value := range query {
		ctx.Query[name] = value
	}

	// Copy form data
	for name, value := range form {
		ctx.Form[name] = value
	}

	return ctx
}

// hasNestedPath checks if the expression has a nested path (e.g., .apikey)
func (e *RequestEvaluator) hasNestedPath(expression string) bool {
	// Look for patterns like request.headers['key'].nested
	closeBracket := strings.Index(expression, "]")
	return closeBracket != -1 && closeBracket+1 < len(expression) && expression[closeBracket+1] == '.'
}

// evaluateNestedExpression handles expressions with nested paths
func (e *RequestEvaluator) evaluateNestedExpression(expression string, contextData interface{}) (string, error) {
	// First, get the base value (e.g., the JSON string from the header)
	basePath := e.extractBasePath(expression)
	result, err := jsonpath.Get(basePath, contextData)
	if err != nil {
		if strings.Contains(err.Error(), "unknown key") || strings.Contains(err.Error(), "not found") {
			return "", nil
		}
		return "", fmt.Errorf("failed to evaluate base path '%s': %w", basePath, err)
	}

	// Check if the result is a string (JSON)
	if jsonStr, ok := result.(string); ok && e.isJSONString(jsonStr) {
		// Parse the JSON and extract the nested value
		return e.extractFromJSONString(jsonStr, expression)
	}

	// If it's not a JSON string, return the value as-is
	if str, ok := result.(string); ok {
		return str, nil
	}

	// Convert other types to string
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("failed to convert result to string: %w", err)
	}
	return string(resultJSON), nil
}

// extractBasePath extracts the base path from a nested expression and converts it to JSONPath
func (e *RequestEvaluator) extractBasePath(expression string) string {
	// Find the closing bracket and return everything up to and including it
	closeBracket := strings.Index(expression, "]")
	if closeBracket != -1 {
		baseExpression := expression[:closeBracket+1]
		// Convert the base expression to JSONPath format
		return e.convertExpressionToJSONPath(baseExpression)
	}
	return e.convertExpressionToJSONPath(expression)
}
