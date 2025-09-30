package config

import (
	"encoding/json"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRequestEvaluator_EvaluateHeaders(t *testing.T) {
	evaluator := NewRequestEvaluator()

	tests := []struct {
		name           string
		headers        HeadersConfig
		requestContext RequestContext
		expected       map[string]string
		wantErr        bool
	}{
		{
			name: "static headers only",
			headers: HeadersConfig{
				{Header: HeaderConfig{Name: "User-Agent", Value: "MCPify/1.0.0"}},
				{Header: HeaderConfig{Name: "Accept", Value: "application/json"}},
			},
			requestContext: NewRequestContextFromMap(
				map[string]string{},
				map[string]string{},
				map[string]string{},
				"GET", "/api/test",
			),
			expected: map[string]string{
				"User-Agent": "MCPify/1.0.0",
				"Accept":     "application/json",
			},
			wantErr: false,
		},
		{
			name: "dynamic headers from request headers",
			headers: HeadersConfig{
				{Header: HeaderConfig{Name: "User-Agent", Value: "MCPify/1.0.0"}},
				{Header: HeaderConfig{Name: "Authorization", ValueFrom: "request.headers['authorization']"}},
				{Header: HeaderConfig{Name: "X-Forwarded-For", ValueFrom: "request.headers['x-forwarded-for']"}},
			},
			requestContext: NewRequestContextFromMap(
				map[string]string{
					"Authorization":   "Bearer token123",
					"X-Forwarded-For": "192.168.1.1",
				},
				map[string]string{},
				map[string]string{},
				"GET", "/api/test",
			),
			expected: map[string]string{
				"User-Agent":      "MCPify/1.0.0",
				"Authorization":   "Bearer token123",
				"X-Forwarded-For": "192.168.1.1",
			},
			wantErr: false,
		},
		{
			name: "dynamic headers from query parameters",
			headers: HeadersConfig{
				{Header: HeaderConfig{Name: "User-Agent", Value: "MCPify/1.0.0"}},
				{Header: HeaderConfig{Name: "X-API-Key", ValueFrom: "request.query['apikey']"}},
				{Header: HeaderConfig{Name: "X-Client-ID", ValueFrom: "request.query['client_id']"}},
			},
			requestContext: NewRequestContextFromMap(
				map[string]string{},
				map[string]string{
					"apikey":    "sk-1234567890",
					"client_id": "client-abc123",
				},
				map[string]string{},
				"GET", "/api/test",
			),
			expected: map[string]string{
				"User-Agent":  "MCPify/1.0.0",
				"X-API-Key":   "sk-1234567890",
				"X-Client-ID": "client-abc123",
			},
			wantErr: false,
		},
		{
			name: "dynamic headers from form data",
			headers: HeadersConfig{
				{Header: HeaderConfig{Name: "User-Agent", Value: "MCPify/1.0.0"}},
				{Header: HeaderConfig{Name: "X-Form-Token", ValueFrom: "request.form['token']"}},
				{Header: HeaderConfig{Name: "X-Form-User", ValueFrom: "request.form['user_id']"}},
			},
			requestContext: NewRequestContextFromMap(
				map[string]string{},
				map[string]string{},
				map[string]string{
					"token":   "form-token-xyz",
					"user_id": "user-123",
				},
				"POST", "/api/test",
			),
			expected: map[string]string{
				"User-Agent":   "MCPify/1.0.0",
				"X-Form-Token": "form-token-xyz",
				"X-Form-User":  "user-123",
			},
			wantErr: false,
		},
		{
			name: "nested JSON extraction from headers",
			headers: HeadersConfig{
				{Header: HeaderConfig{Name: "User-Agent", Value: "MCPify/1.0.0"}},
				{Header: HeaderConfig{Name: "Authorization", ValueFrom: "request.headers['x-mcpify-provider-data'].apikey"}},
				{Header: HeaderConfig{Name: "X-API-Key", ValueFrom: "request.headers['x-mcpify-provider-data'].auth.api_key"}},
			},
			requestContext: NewRequestContextFromMap(
				map[string]string{
					"X-Mcpify-Provider-Data": `{"apikey": "sk-1234567890", "auth": {"api_key": "key-abc123"}}`,
				},
				map[string]string{},
				map[string]string{},
				"GET", "/api/test",
			),
			expected: map[string]string{
				"User-Agent":    "MCPify/1.0.0",
				"Authorization": "sk-1234567890",
				"X-API-Key":     "key-abc123",
			},
			wantErr: false,
		},
		{
			name: "nested JSON extraction from query parameters",
			headers: HeadersConfig{
				{Header: HeaderConfig{Name: "User-Agent", Value: "MCPify/1.0.0"}},
				{Header: HeaderConfig{Name: "X-Client-ID", ValueFrom: "request.query['client_data'].id"}},
				{Header: HeaderConfig{Name: "X-Client-Secret", ValueFrom: "request.query['client_data'].secret"}},
			},
			requestContext: NewRequestContextFromMap(
				map[string]string{},
				map[string]string{
					"client_data": `{"id": "client123", "secret": "secret456"}`,
				},
				map[string]string{},
				"GET", "/api/test",
			),
			expected: map[string]string{
				"User-Agent":      "MCPify/1.0.0",
				"X-Client-ID":     "client123",
				"X-Client-Secret": "secret456",
			},
			wantErr: false,
		},
		{
			name: "mixed static and dynamic headers",
			headers: HeadersConfig{
				{Header: HeaderConfig{Name: "User-Agent", Value: "MCPify/1.0.0"}},
				{Header: HeaderConfig{Name: "Authorization", ValueFrom: "request.headers['authorization']"}},
				{Header: HeaderConfig{Name: "Content-Type", Value: "application/json"}},
				{Header: HeaderConfig{Name: "X-API-Key", ValueFrom: "request.query['apikey']"}},
			},
			requestContext: NewRequestContextFromMap(
				map[string]string{
					"Authorization": "Bearer token123",
				},
				map[string]string{
					"apikey": "sk-1234567890",
				},
				map[string]string{},
				"POST", "/api/test",
			),
			expected: map[string]string{
				"User-Agent":    "MCPify/1.0.0",
				"Authorization": "Bearer token123",
				"Content-Type":  "application/json",
				"X-API-Key":     "sk-1234567890",
			},
			wantErr: false,
		},
		{
			name: "missing values return empty strings",
			headers: HeadersConfig{
				{Header: HeaderConfig{Name: "User-Agent", Value: "MCPify/1.0.0"}},
				{Header: HeaderConfig{Name: "Missing-Header", ValueFrom: "request.headers['missing-header']"}},
				{Header: HeaderConfig{Name: "Missing-Query", ValueFrom: "request.query['missing-param']"}},
			},
			requestContext: NewRequestContextFromMap(
				map[string]string{},
				map[string]string{},
				map[string]string{},
				"GET", "/api/test",
			),
			expected: map[string]string{
				"User-Agent": "MCPify/1.0.0",
			},
			wantErr: false,
		},
		{
			name: "invalid JSONPath expression",
			headers: HeadersConfig{
				{Header: HeaderConfig{Name: "Test-Header", ValueFrom: "invalid.jsonpath[expression"}},
			},
			requestContext: NewRequestContextFromMap(
				map[string]string{},
				map[string]string{},
				map[string]string{},
				"GET", "/api/test",
			),
			expected: nil,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := evaluator.EvaluateHeaders(tt.headers, tt.requestContext)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestRequestEvaluator_evaluateValueFrom(t *testing.T) {
	evaluator := NewRequestEvaluator()

	tests := []struct {
		name           string
		expression     string
		requestContext RequestContext
		expected       string
		wantErr        bool
	}{
		{
			name:       "simple header access",
			expression: "request.headers['authorization']",
			requestContext: NewRequestContextFromMap(
				map[string]string{"Authorization": "Bearer token123"},
				map[string]string{},
				map[string]string{},
				"GET", "/api/test",
			),
			expected: "Bearer token123",
			wantErr:  false,
		},
		{
			name:       "simple query parameter access",
			expression: "request.query['apikey']",
			requestContext: NewRequestContextFromMap(
				map[string]string{},
				map[string]string{"apikey": "sk-1234567890"},
				map[string]string{},
				"GET", "/api/test",
			),
			expected: "sk-1234567890",
			wantErr:  false,
		},
		{
			name:       "simple form data access",
			expression: "request.form['token']",
			requestContext: NewRequestContextFromMap(
				map[string]string{},
				map[string]string{},
				map[string]string{"token": "form-token-xyz"},
				"POST", "/api/test",
			),
			expected: "form-token-xyz",
			wantErr:  false,
		},
		{
			name:       "nested JSONPath from header",
			expression: "request.headers['x-mcpify-provider-data'].apikey",
			requestContext: NewRequestContextFromMap(
				map[string]string{
					"X-Mcpify-Provider-Data": `{"apikey": "sk-1234567890"}`,
				},
				map[string]string{},
				map[string]string{},
				"GET", "/api/test",
			),
			expected: "sk-1234567890",
			wantErr:  false,
		},
		{
			name:       "complex nested JSONPath from header",
			expression: "request.headers['x-mcpify-provider-data'].auth.api_key",
			requestContext: NewRequestContextFromMap(
				map[string]string{
					"X-Mcpify-Provider-Data": `{"auth": {"api_key": "key-abc123"}}`,
				},
				map[string]string{},
				map[string]string{},
				"GET", "/api/test",
			),
			expected: "key-abc123",
			wantErr:  false,
		},
		{
			name:       "nested JSONPath from query parameter",
			expression: "request.query['client_data'].id",
			requestContext: NewRequestContextFromMap(
				map[string]string{},
				map[string]string{
					"client_data": `{"id": "client123", "secret": "secret456"}`,
				},
				map[string]string{},
				"GET", "/api/test",
			),
			expected: "client123",
			wantErr:  false,
		},
		{
			name:       "missing header",
			expression: "request.headers['missing-header']",
			requestContext: NewRequestContextFromMap(
				map[string]string{},
				map[string]string{},
				map[string]string{},
				"GET", "/api/test",
			),
			expected: "",
			wantErr:  false,
		},
		{
			name:       "missing query parameter",
			expression: "request.query['missing-param']",
			requestContext: NewRequestContextFromMap(
				map[string]string{},
				map[string]string{},
				map[string]string{},
				"GET", "/api/test",
			),
			expected: "",
			wantErr:  false,
		},
		{
			name:       "invalid JSONPath",
			expression: "invalid.jsonpath[expression",
			requestContext: NewRequestContextFromMap(
				map[string]string{},
				map[string]string{},
				map[string]string{},
				"GET", "/api/test",
			),
			expected: "",
			wantErr:  true,
		},
		{
			name:       "non-string result",
			expression: "request.headers['x-mcpify-provider-data'].count",
			requestContext: NewRequestContextFromMap(
				map[string]string{
					"X-Mcpify-Provider-Data": `{"count": 42}`,
				},
				map[string]string{},
				map[string]string{},
				"GET", "/api/test",
			),
			expected: "42",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := evaluator.evaluateValueFrom(tt.expression, tt.requestContext)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestRequestEvaluator_convertExpressionToJSONPath(t *testing.T) {
	evaluator := NewRequestEvaluator()

	tests := []struct {
		name       string
		expression string
		expected   string
	}{
		{
			name:       "header expression",
			expression: "request.headers['authorization']",
			expected:   `$.headers["authorization"]`,
		},
		{
			name:       "header expression with nested path",
			expression: "request.headers['x-mcpify-provider-data'].apikey",
			expected:   `$.headers["x-mcpify-provider-data"].apikey`,
		},
		{
			name:       "query parameter expression",
			expression: "request.query['apikey']",
			expected:   `$.query["apikey"]`,
		},
		{
			name:       "query parameter with nested path",
			expression: "request.query['client_data'].id",
			expected:   `$.query["client_data"].id`,
		},
		{
			name:       "form data expression",
			expression: "request.form['token']",
			expected:   `$.form["token"]`,
		},
		{
			name:       "form data with nested path",
			expression: "request.form['user_data'].name",
			expected:   `$.form["user_data"].name`,
		},
		{
			name:       "body expression",
			expression: "request.body.user.id",
			expected:   `$.body.user.id`,
		},
		{
			name:       "expression without request prefix",
			expression: "headers['authorization']",
			expected:   `$.headers["authorization"]`,
		},
		{
			name:       "double quotes in key",
			expression: `request.headers["authorization"]`,
			expected:   `$.headers["authorization"]`,
		},
		{
			name:       "special characters in key",
			expression: "request.headers['x-mcpify-provider-data']",
			expected:   `$.headers["x-mcpify-provider-data"]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evaluator.convertExpressionToJSONPath(tt.expression)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNewRequestContextFromHTTP(t *testing.T) {
	// Test with URL values
	headers := map[string][]string{
		"Authorization": {"Bearer token123"},
		"Content-Type":  {"application/json"},
	}

	query := url.Values{
		"apikey":    {"sk-1234567890"},
		"client_id": {"client-abc123"},
	}

	form := url.Values{
		"token":   {"form-token-xyz"},
		"user_id": {"user-123"},
	}

	ctx := NewRequestContextFromHTTP(headers, query, form, "POST", "/api/test")

	assert.Equal(t, "Bearer token123", ctx.Headers["authorization"])
	assert.Equal(t, "application/json", ctx.Headers["content-type"])
	assert.Equal(t, "sk-1234567890", ctx.Query["apikey"])
	assert.Equal(t, "client-abc123", ctx.Query["client_id"])
	assert.Equal(t, "form-token-xyz", ctx.Form["token"])
	assert.Equal(t, "user-123", ctx.Form["user_id"])
	assert.Equal(t, "POST", ctx.Method)
	assert.Equal(t, "/api/test", ctx.Path)
}

func TestRequestContext_JSONSerialization(t *testing.T) {
	ctx := NewRequestContextFromMap(
		map[string]string{
			"Authorization": "Bearer token123",
			"Content-Type":  "application/json",
		},
		map[string]string{
			"apikey": "sk-1234567890",
		},
		map[string]string{
			"token": "form-token-xyz",
		},
		"POST", "/api/test",
	)

	// Test JSON serialization
	jsonData, err := json.Marshal(ctx)
	assert.NoError(t, err)

	// Test JSON deserialization
	var deserializedCtx RequestContext
	err = json.Unmarshal(jsonData, &deserializedCtx)
	assert.NoError(t, err)

	assert.Equal(t, ctx.Headers, deserializedCtx.Headers)
	assert.Equal(t, ctx.Query, deserializedCtx.Query)
	assert.Equal(t, ctx.Form, deserializedCtx.Form)
	assert.Equal(t, ctx.Method, deserializedCtx.Method)
	assert.Equal(t, ctx.Path, deserializedCtx.Path)
}

func TestRequestEvaluator_hasNestedPath(t *testing.T) {
	evaluator := NewRequestEvaluator()

	tests := []struct {
		name       string
		expression string
		expected   bool
	}{
		{
			name:       "header expression with nested path",
			expression: "request.headers['x-mcpify-provider-data'].apikey",
			expected:   true,
		},
		{
			name:       "query expression with nested path",
			expression: "request.query['client_data'].id",
			expected:   true,
		},
		{
			name:       "form expression with nested path",
			expression: "request.form['user_data'].name",
			expected:   true,
		},
		{
			name:       "header expression without nested path",
			expression: "request.headers['authorization']",
			expected:   false,
		},
		{
			name:       "query expression without nested path",
			expression: "request.query['apikey']",
			expected:   false,
		},
		{
			name:       "form expression without nested path",
			expression: "request.form['token']",
			expected:   false,
		},
		{
			name:       "complex nested path",
			expression: "request.headers['x-mcpify-provider-data'].auth.api_key",
			expected:   true,
		},
		{
			name:       "deeply nested path",
			expression: "request.query['data'].user.profile.settings.theme",
			expected:   true,
		},
		{
			name:       "expression without brackets",
			expression: "request.body.user.id",
			expected:   false,
		},
		{
			name:       "expression with brackets but no dot after",
			expression: "request.headers['authorization']extra",
			expected:   false,
		},
		{
			name:       "expression with brackets and space after",
			expression: "request.headers['authorization'] ",
			expected:   false,
		},
		{
			name:       "expression with brackets at end",
			expression: "request.headers['authorization']",
			expected:   false,
		},
		{
			name:       "empty expression",
			expression: "",
			expected:   false,
		},
		{
			name:       "expression with only brackets",
			expression: "[]",
			expected:   false,
		},
		{
			name:       "expression with brackets and dot at end",
			expression: "request.headers['authorization'].",
			expected:   true,
		},
		{
			name:       "expression with multiple brackets - first one",
			expression: "request.headers['key1'].nested['key2']",
			expected:   true,
		},
		{
			name:       "expression without request prefix",
			expression: "headers['authorization'].apikey",
			expected:   true,
		},
		{
			name:       "expression with double quotes",
			expression: `request.headers["authorization"].apikey`,
			expected:   true,
		},
		{
			name:       "expression with special characters in key",
			expression: "request.headers['x-mcpify-provider-data'].apikey",
			expected:   true,
		},
		{
			name:       "expression with numbers in nested path",
			expression: "request.query['data'].item123.value",
			expected:   true,
		},
		{
			name:       "expression with underscores in nested path",
			expression: "request.form['user_data'].first_name",
			expected:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evaluator.hasNestedPath(tt.expression)
			assert.Equal(t, tt.expected, result, "hasNestedPath(%q) = %v, expected %v", tt.expression, result, tt.expected)
		})
	}
}

// TestRequestEvaluator_hasNestedPath_EdgeCases tests edge cases and boundary conditions
func TestRequestEvaluator_hasNestedPath_EdgeCases(t *testing.T) {
	evaluator := NewRequestEvaluator()

	tests := []struct {
		name       string
		expression string
		expected   bool
	}{
		{
			name:       "single character after bracket",
			expression: "request.headers['key'].a",
			expected:   true,
		},
		{
			name:       "bracket at beginning",
			expression: "['key'].value",
			expected:   true,
		},
		{
			name:       "bracket at end with dot",
			expression: "request.headers['key'].",
			expected:   true,
		},
		{
			name:       "multiple dots after bracket",
			expression: "request.headers['key']...",
			expected:   true,
		},
		{
			name:       "bracket with empty key",
			expression: "request.headers[''].value",
			expected:   true,
		},
		{
			name:       "nested brackets",
			expression: "request.headers['key[0]'].value",
			expected:   false, // The first ']' found is at position of 'key[0]', not the outer bracket
		},
		{
			name:       "bracket with spaces",
			expression: "request.headers[ 'key' ].value",
			expected:   true, // The function finds the first ']' and checks what follows
		},
		{
			name:       "unicode characters in nested path",
			expression: "request.headers['key'].café.value",
			expected:   true,
		},
		{
			name:       "very long nested path",
			expression: "request.headers['key'].a.b.c.d.e.f.g.h.i.j.k.l.m.n.o.p.q.r.s.t.u.v.w.x.y.z",
			expected:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evaluator.hasNestedPath(tt.expression)
			assert.Equal(t, tt.expected, result, "hasNestedPath(%q) = %v, expected %v", tt.expression, result, tt.expected)
		})
	}
}

// TestRequestEvaluator_hasNestedPath_Performance tests performance with various input sizes
func TestRequestEvaluator_hasNestedPath_Performance(t *testing.T) {
	evaluator := NewRequestEvaluator()

	// Test with various input sizes
	testCases := []struct {
		name  string
		input string
	}{
		{
			name:  "short expression",
			input: "headers['key'].value",
		},
		{
			name:  "medium expression",
			input: "request.headers['x-mcpify-provider-data'].auth.api_key",
		},
		{
			name:  "long expression",
			input: "request.headers['very-long-header-name-with-many-characters'].deeply.nested.path.with.many.segments",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test multiple times to ensure consistent performance
			for i := 0; i < 1000; i++ {
				result := evaluator.hasNestedPath(tc.input)
				assert.True(t, result, "Expected nested path to be detected")
			}
		})
	}
}

// TestRequestEvaluator_hasNestedPath_Regression tests for regression issues
func TestRequestEvaluator_hasNestedPath_Regression(t *testing.T) {
	evaluator := NewRequestEvaluator()

	// These are specific cases that might have caused issues in the past
	regressionTests := []struct {
		name       string
		expression string
		expected   bool
		reason     string
	}{
		{
			name:       "bracket with no content",
			expression: "request.headers[''].value",
			expected:   true,
			reason:     "Empty key should still allow nested path",
		},
		{
			name:       "bracket with whitespace only",
			expression: "request.headers[' '].value",
			expected:   true,
			reason:     "Whitespace-only key should still allow nested path",
		},
		{
			name:       "multiple consecutive dots",
			expression: "request.headers['key']...value",
			expected:   true,
			reason:     "Multiple dots should still be detected as nested path",
		},
		{
			name:       "bracket at very end",
			expression: "request.headers['key']",
			expected:   false,
			reason:     "Bracket at end should not have nested path",
		},
		{
			name:       "bracket followed by non-dot character",
			expression: "request.headers['key']x",
			expected:   false,
			reason:     "Non-dot character after bracket should not be nested path",
		},
	}

	for _, tt := range regressionTests {
		t.Run(tt.name, func(t *testing.T) {
			result := evaluator.hasNestedPath(tt.expression)
			assert.Equal(t, tt.expected, result, "Regression test failed: %s. hasNestedPath(%q) = %v, expected %v", tt.reason, tt.expression, result, tt.expected)
		})
	}
}

func TestRequestEvaluator_evaluateNestedExpression(t *testing.T) {
	evaluator := NewRequestEvaluator()

	tests := []struct {
		name          string
		expression    string
		contextData   interface{}
		expected      string
		wantErr       bool
		errorContains string
	}{
		{
			name:       "valid JSON string with nested path",
			expression: "request.headers['x-mcpify-provider-data'].apikey",
			contextData: map[string]interface{}{
				"headers": map[string]interface{}{
					"x-mcpify-provider-data": `{"apikey": "sk-1234567890"}`,
				},
			},
			expected: "sk-1234567890",
			wantErr:  false,
		},
		{
			name:       "valid JSON string with complex nested path",
			expression: "request.headers['x-mcpify-provider-data'].auth.api_key",
			contextData: map[string]interface{}{
				"headers": map[string]interface{}{
					"x-mcpify-provider-data": `{"auth": {"api_key": "key-abc123"}}`,
				},
			},
			expected: "key-abc123",
			wantErr:  false,
		},
		{
			name:       "non-JSON string result",
			expression: "request.headers['authorization'].extra",
			contextData: map[string]interface{}{
				"headers": map[string]interface{}{
					"authorization": "Bearer token123",
				},
			},
			expected: "Bearer token123",
			wantErr:  false,
		},
		{
			name:       "non-string result (number)",
			expression: "request.query['count'].value",
			contextData: map[string]interface{}{
				"query": map[string]interface{}{
					"count": 42,
				},
			},
			expected: "42",
			wantErr:  false,
		},
		{
			name:       "non-string result (boolean)",
			expression: "request.form['enabled'].value",
			contextData: map[string]interface{}{
				"form": map[string]interface{}{
					"enabled": true,
				},
			},
			expected: "true",
			wantErr:  false,
		},
		{
			name:       "non-string result (object)",
			expression: "request.headers['data'].value",
			contextData: map[string]interface{}{
				"headers": map[string]interface{}{
					"data": map[string]interface{}{
						"key": "value",
					},
				},
			},
			expected: `{"key":"value"}`,
			wantErr:  false,
		},
		{
			name:       "invalid JSON string",
			expression: "request.headers['invalid-json'].key",
			contextData: map[string]interface{}{
				"headers": map[string]interface{}{
					"invalid-json": `{"key": "value"`, // Missing closing brace
				},
			},
			expected: `{"key": "value"`, // Should return as-is since it's not valid JSON
			wantErr:  false,
		},
		{
			name:       "missing base path",
			expression: "request.headers['missing'].key",
			contextData: map[string]interface{}{
				"headers": map[string]interface{}{
					"other": "value",
				},
			},
			expected: "",
			wantErr:  false,
		},
		{
			name:       "JSONPath evaluation error",
			expression: "request.headers['data'].invalid[path",
			contextData: map[string]interface{}{
				"headers": map[string]interface{}{
					"data": `{"key": "value"}`,
				},
			},
			expected:      "",
			wantErr:       true,
			errorContains: "failed to evaluate nested JSONPath",
		},
		{
			name:       "nil result",
			expression: "request.headers['null-value'].key",
			contextData: map[string]interface{}{
				"headers": map[string]interface{}{
					"null-value": nil,
				},
			},
			expected: "null",
			wantErr:  false,
		},
		{
			name:       "empty string result",
			expression: "request.headers['empty'].key",
			contextData: map[string]interface{}{
				"headers": map[string]interface{}{
					"empty": "",
				},
			},
			expected: "",
			wantErr:  false,
		},
		{
			name:       "JSON string with no nested path",
			expression: "request.headers['json-data']",
			contextData: map[string]interface{}{
				"headers": map[string]interface{}{
					"json-data": `{"key": "value"}`,
				},
			},
			expected: `{"key": "value"}`,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := evaluator.evaluateNestedExpression(tt.expression, tt.contextData)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestRequestEvaluator_extractBasePath(t *testing.T) {
	evaluator := NewRequestEvaluator()

	tests := []struct {
		name       string
		expression string
		expected   string
	}{
		{
			name:       "header expression with nested path",
			expression: "request.headers['x-mcpify-provider-data'].apikey",
			expected:   `$.headers["x-mcpify-provider-data"]`,
		},
		{
			name:       "query expression with nested path",
			expression: "request.query['client_data'].id",
			expected:   `$.query["client_data"]`,
		},
		{
			name:       "form expression with nested path",
			expression: "request.form['user_data'].name",
			expected:   `$.form["user_data"]`,
		},
		{
			name:       "body expression with nested path",
			expression: "request.body.user.profile.name",
			expected:   `$.body.user.profile.name`,
		},
		{
			name:       "expression without brackets",
			expression: "request.body.user.id",
			expected:   `$.body.user.id`,
		},
		{
			name:       "expression with double quotes",
			expression: `request.headers["authorization"].token`,
			expected:   `$.headers["authorization"]`,
		},
		{
			name:       "expression with special characters",
			expression: "request.headers['x-mcpify-provider-data'].auth.api_key",
			expected:   `$.headers["x-mcpify-provider-data"]`,
		},
		{
			name:       "expression without request prefix",
			expression: "headers['authorization'].token",
			expected:   `$.headers["authorization"]`,
		},
		{
			name:       "complex nested expression",
			expression: "request.query['data'].user.profile.settings.theme",
			expected:   `$.query["data"]`,
		},
		{
			name:       "empty expression",
			expression: "",
			expected:   "",
		},
		{
			name:       "expression with only brackets",
			expression: "[]",
			expected:   "[]",
		},
		{
			name:       "expression with multiple brackets",
			expression: "request.headers['key[0]'].value",
			expected:   `$.headers["key[0"]`, // The first ']' found is at position of 'key[0]', not the outer bracket
		},
		{
			name:       "expression with spaces in key",
			expression: "request.headers['key with spaces'].value",
			expected:   `$.headers["key with spaces"]`,
		},
		{
			name:       "expression with unicode in key",
			expression: "request.headers['café'].value",
			expected:   `$.headers["café"]`,
		},
		{
			name:       "expression with numbers in key",
			expression: "request.query['param123'].value",
			expected:   `$.query["param123"]`,
		},
		{
			name:       "expression with underscores in key",
			expression: "request.form['user_data'].value",
			expected:   `$.form["user_data"]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evaluator.extractBasePath(tt.expression)
			assert.Equal(t, tt.expected, result, "extractBasePath(%q) = %v, expected %v", tt.expression, result, tt.expected)
		})
	}
}

// TestRequestEvaluator_extractFromJSONString_EdgeCases tests edge cases for extractFromJSONString
func TestRequestEvaluator_extractFromJSONString_EdgeCases(t *testing.T) {
	evaluator := NewRequestEvaluator()

	tests := []struct {
		name          string
		jsonStr       string
		expression    string
		expected      string
		wantErr       bool
		errorContains string
	}{
		{
			name:       "valid JSON with nested path",
			jsonStr:    `{"apikey": "sk-1234567890", "auth": {"api_key": "key-abc123"}}`,
			expression: "request.headers['data'].apikey",
			expected:   "sk-1234567890",
			wantErr:    false,
		},
		{
			name:       "valid JSON with complex nested path",
			jsonStr:    `{"auth": {"api_key": "key-abc123", "secret": "secret456"}}`,
			expression: "request.headers['data'].auth.api_key",
			expected:   "key-abc123",
			wantErr:    false,
		},
		{
			name:       "valid JSON with no nested path",
			jsonStr:    `{"key": "value"}`,
			expression: "request.headers['data']",
			expected:   `{"key": "value"}`,
			wantErr:    false,
		},
		{
			name:       "valid JSON with missing nested path",
			jsonStr:    `{"key": "value"}`,
			expression: "request.headers['data'].missing",
			expected:   "",
			wantErr:    false,
		},
		{
			name:       "valid JSON with nested path to non-string value",
			jsonStr:    `{"count": 42, "enabled": true}`,
			expression: "request.headers['data'].count",
			expected:   "42",
			wantErr:    false,
		},
		{
			name:       "valid JSON with nested path to boolean value",
			jsonStr:    `{"count": 42, "enabled": true}`,
			expression: "request.headers['data'].enabled",
			expected:   "true",
			wantErr:    false,
		},
		{
			name:       "valid JSON with nested path to object value",
			jsonStr:    `{"user": {"name": "John", "age": 30}}`,
			expression: "request.headers['data'].user",
			expected:   `{"age":30,"name":"John"}`,
			wantErr:    false,
		},
		{
			name:       "valid JSON with nested path to null value",
			jsonStr:    `{"key": "value", "null_value": null}`,
			expression: "request.headers['data'].null_value",
			expected:   "",
			wantErr:    false,
		},
		{
			name:          "invalid JSON string",
			jsonStr:       `{"key": "value"`, // Missing closing brace
			expression:    "request.headers['data'].key",
			expected:      "",
			wantErr:       true,
			errorContains: "failed to parse JSON string",
		},
		{
			name:          "completely invalid JSON",
			jsonStr:       "not json at all",
			expression:    "request.headers['data'].key",
			expected:      "",
			wantErr:       true,
			errorContains: "failed to parse JSON string",
		},
		{
			name:       "empty JSON string",
			jsonStr:    "",
			expression: "request.headers['data'].key",
			expected:   "",
			wantErr:    true,
		},
		{
			name:       "JSON with array",
			jsonStr:    `{"items": [1, 2, 3], "name": "test"}`,
			expression: "request.headers['data'].name",
			expected:   "test",
			wantErr:    false,
		},
		{
			name:       "JSON with nested array access",
			jsonStr:    `{"items": [{"id": 1, "name": "first"}, {"id": 2, "name": "second"}]}`,
			expression: "request.headers['data'].items[0].name",
			expected:   "first",
			wantErr:    false,
		},
		{
			name:       "JSON with deeply nested structure",
			jsonStr:    `{"level1": {"level2": {"level3": {"value": "deep"}}}}`,
			expression: "request.headers['data'].level1.level2.level3.value",
			expected:   "deep",
			wantErr:    false,
		},
		{
			name:       "JSON with special characters in values",
			jsonStr:    `{"message": "Hello, \"world\"!", "unicode": "café"}`,
			expression: "request.headers['data'].message",
			expected:   "Hello, \"world\"!",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := evaluator.extractFromJSONString(tt.jsonStr, tt.expression)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}
