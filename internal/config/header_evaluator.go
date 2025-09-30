package config

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/PaesslerAG/jsonpath"
)

// HeaderRequestContext represents the context available for header evaluation
type HeaderRequestContext struct {
	Headers map[string]string `json:"headers"`
}

// HeaderEvaluator handles evaluation of dynamic header values
type HeaderEvaluator struct{}

// NewHeaderEvaluator creates a new header evaluator
func NewHeaderEvaluator() *HeaderEvaluator {
	return &HeaderEvaluator{}
}

// EvaluateHeaders processes headers and evaluates valueFrom expressions
func (e *HeaderEvaluator) EvaluateHeaders(headers HeadersConfig, requestHeaders http.Header) (map[string]string, error) {
	result := make(map[string]string)

	// Create request context from HTTP headers
	headerContext := HeaderRequestContext{
		Headers: make(map[string]string),
	}

	// Convert HTTP headers to map (normalize to lowercase for case-insensitive matching)
	for name, values := range requestHeaders {
		if len(values) > 0 {
			headerContext.Headers[strings.ToLower(name)] = values[0] // Take first value
		}
	}

	// Process each header
	for _, item := range headers {
		if item.Header.Value != "" {
			// Static value
			result[item.Header.Name] = item.Header.Value
		} else if item.Header.ValueFrom != "" {
			// Dynamic value - evaluate JSONPath
			value, err := e.evaluateValueFrom(item.Header.ValueFrom, headerContext)
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
func (e *HeaderEvaluator) evaluateValueFrom(expression string, headerContext HeaderRequestContext) (string, error) {
	// Convert the expression to use the correct JSONPath syntax
	// Replace request.headers['key'] with $.headers.key
	jsonPathExpr := e.convertExpressionToJSONPath(expression)

	// Debug: log the expression and JSONPath (remove in production)
	// fmt.Printf("DEBUG: Original expression: %s\n", expression)
	// fmt.Printf("DEBUG: Converted JSONPath: %s\n", jsonPathExpr)

	// Convert context to JSON for evaluation
	contextJSON, err := json.Marshal(headerContext)
	if err != nil {
		return "", fmt.Errorf("failed to marshal context: %w", err)
	}

	var contextData interface{}
	if err := json.Unmarshal(contextJSON, &contextData); err != nil {
		return "", fmt.Errorf("failed to unmarshal context: %w", err)
	}

	// First, try to get the header value as a string
	headerPath := e.extractHeaderPath(expression)
	// fmt.Printf("DEBUG: headerPath: %s\n", headerPath)
	if headerPath != "" {
		headerResult, err := jsonpath.Get(headerPath, contextData)
		if err != nil {
			// If the header doesn't exist, return empty string
			if strings.Contains(err.Error(), "unknown key") {
				return "", nil
			}
			return "", fmt.Errorf("failed to evaluate header path: %w", err)
		}

		// fmt.Printf("DEBUG: headerResult: %v, type: %T\n", headerResult, headerResult)

		// If we got a string value, check if it's JSON and extract nested values
		if headerStr, ok := headerResult.(string); ok {
			// fmt.Printf("DEBUG: headerStr: %s, isJSON: %v\n", headerStr, e.isJSONString(headerStr))
			if e.isJSONString(headerStr) {
				return e.extractFromJSONString(headerStr, expression)
			}
			return headerStr, nil
		}
	}

	// Fall back to the original JSONPath evaluation
	result, err := jsonpath.Get(jsonPathExpr, contextData)
	if err != nil {
		// If the path doesn't exist, return empty string instead of error
		if strings.Contains(err.Error(), "unknown key") {
			return "", nil
		}
		return "", fmt.Errorf("failed to evaluate JSONPath: %w", err)
	}

	// Convert result to string
	switch v := result.(type) {
	case string:
		// If it's a JSON string, try to parse it and extract the nested value
		if e.isJSONString(v) {
			return e.extractFromJSONString(v, expression)
		}
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
func (e *HeaderEvaluator) convertExpressionToJSONPath(expression string) string {
	// For now, handle simple cases like request.headers['key'] -> $.headers["key"]
	// This is a simplified implementation - in production, you might want a more robust parser

	// Remove 'request.' prefix if present
	if len(expression) > 8 && expression[:8] == "request." {
		expression = expression[8:]
	}

	// Handle headers['key'] -> headers["key"]
	if len(expression) > 9 && expression[:9] == "headers['" {
		// Find the closing bracket
		closeBracket := -1
		for i := 9; i < len(expression); i++ {
			if expression[i] == ']' {
				closeBracket = i
				break
			}
		}
		if closeBracket != -1 {
			headerName := expression[9 : closeBracket-1] // Remove the closing quote
			remaining := expression[closeBracket+1:]

			// Convert to JSONPath format with quotes for property names with special characters
			// Normalize header name to lowercase for case-insensitive matching
			jsonPath := "$.headers[\"" + strings.ToLower(headerName) + "\"]"

			// Handle nested properties - convert dot notation to JSONPath
			if len(remaining) > 0 && remaining[0] == '.' {
				// Remove the leading dot and add the remaining path
				nestedPath := remaining[1:]
				jsonPath += "." + nestedPath
			}

			// fmt.Printf("DEBUG: headerName: %s, remaining: %s, jsonPath: %s\n", headerName, remaining, jsonPath)
			return jsonPath
		}
	}

	// If no conversion needed, return as-is
	return expression
}

// isJSONString checks if a string is valid JSON
func (e *HeaderEvaluator) isJSONString(s string) bool {
	var js json.RawMessage
	return json.Unmarshal([]byte(s), &js) == nil
}

// extractFromJSONString extracts a value from a JSON string using the original expression
func (e *HeaderEvaluator) extractFromJSONString(jsonStr, originalExpression string) (string, error) {
	// fmt.Printf("DEBUG: extractFromJSONString called with jsonStr: %s, expression: %s\n", jsonStr, originalExpression)

	// Parse the JSON string
	var jsonData interface{}
	if err := json.Unmarshal([]byte(jsonStr), &jsonData); err != nil {
		return "", fmt.Errorf("failed to parse JSON string: %w", err)
	}

	// Extract the nested path from the original expression
	// e.g., request.headers['x-mcpify-provider-data'].apikey -> .apikey
	nestedPath := e.extractNestedPath(originalExpression)
	// fmt.Printf("DEBUG: nestedPath: %s\n", nestedPath)
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
func (e *HeaderEvaluator) extractNestedPath(expression string) string {
	// fmt.Printf("DEBUG: extractNestedPath called with expression: %s\n", expression)
	// Find the position after the header name
	// e.g., request.headers['x-mcpify-provider-data'].apikey -> .apikey
	// Look for the pattern: request.headers['...']
	headerStart := strings.Index(expression, "headers['")
	if headerStart == -1 {
		// fmt.Printf("DEBUG: no headers[' found\n")
		return ""
	}

	// Find the closing bracket after headers['...']
	closeBracket := -1
	for i := headerStart + 9; i < len(expression); i++ {
		if expression[i] == ']' {
			closeBracket = i
			break
		}
	}
	// fmt.Printf("DEBUG: closeBracket: %d, expression length: %d\n", closeBracket, len(expression))
	if closeBracket != -1 && closeBracket+1 < len(expression) {
		// Return everything after the closing bracket
		result := expression[closeBracket+1:]
		// fmt.Printf("DEBUG: nestedPath result: %s\n", result)
		return result
	}
	// fmt.Printf("DEBUG: returning empty nestedPath\n")
	return ""
}

// extractHeaderPath extracts just the header path from the expression
func (e *HeaderEvaluator) extractHeaderPath(expression string) string {
	// Remove 'request.' prefix if present
	if len(expression) > 8 && expression[:8] == "request." {
		expression = expression[8:]
	}

	// Handle headers['key'] -> $.headers["key"]
	if len(expression) > 9 && expression[:9] == "headers['" {
		// Find the closing bracket
		closeBracket := -1
		for i := 9; i < len(expression); i++ {
			if expression[i] == ']' {
				closeBracket = i
				break
			}
		}
		if closeBracket != -1 {
			headerName := expression[9 : closeBracket-1] // Remove the closing quote
			return "$.headers[\"" + headerName + "\"]"
		}
	}

	return ""
}
