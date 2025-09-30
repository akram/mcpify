package config

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHeaderEvaluator_EvaluateHeaders(t *testing.T) {
	evaluator := NewHeaderEvaluator()

	tests := []struct {
		name           string
		headers        HeadersConfig
		requestHeaders http.Header
		expected       map[string]string
		wantErr        bool
	}{
		{
			name: "static headers only",
			headers: HeadersConfig{
				{Header: HeaderConfig{Name: "User-Agent", Value: "MCPify/1.0.0"}},
				{Header: HeaderConfig{Name: "Accept", Value: "application/json"}},
			},
			requestHeaders: http.Header{},
			expected: map[string]string{
				"User-Agent": "MCPify/1.0.0",
				"Accept":     "application/json",
			},
			wantErr: false,
		},
		{
			name: "dynamic headers with JSONPath",
			headers: HeadersConfig{
				{Header: HeaderConfig{Name: "User-Agent", Value: "MCPify/1.0.0"}},
				{Header: HeaderConfig{Name: "Authorization", ValueFrom: "request.headers['x-mcpify-provider-data'].apikey"}},
				{Header: HeaderConfig{Name: "X-API-Key", ValueFrom: "request.headers['x-mcpify-provider-data'].auth.api_key"}},
			},
			requestHeaders: http.Header{
				"X-Mcpify-Provider-Data": []string{`{"apikey": "sk-1234567890", "auth": {"api_key": "key-abc123"}}`},
			},
			expected: map[string]string{
				"User-Agent":    "MCPify/1.0.0",
				"Authorization": "sk-1234567890",
				"X-API-Key":     "key-abc123",
			},
			wantErr: false,
		},
		{
			name: "mixed static and dynamic headers",
			headers: HeadersConfig{
				{Header: HeaderConfig{Name: "User-Agent", Value: "MCPify/1.0.0"}},
				{Header: HeaderConfig{Name: "Authorization", ValueFrom: "request.headers['authorization']"}},
				{Header: HeaderConfig{Name: "Content-Type", Value: "application/json"}},
			},
			requestHeaders: http.Header{
				"Authorization": []string{"Bearer token123"},
			},
			expected: map[string]string{
				"User-Agent":    "MCPify/1.0.0",
				"Authorization": "Bearer token123",
				"Content-Type":  "application/json",
			},
			wantErr: false,
		},
		{
			name: "invalid JSONPath expression",
			headers: HeadersConfig{
				{Header: HeaderConfig{Name: "Test-Header", ValueFrom: "invalid.jsonpath[expression"}},
			},
			requestHeaders: http.Header{},
			expected:       nil,
			wantErr:        true,
		},
		{
			name: "missing header in JSONPath",
			headers: HeadersConfig{
				{Header: HeaderConfig{Name: "Test-Header", ValueFrom: "request.headers['missing-header'].value"}},
			},
			requestHeaders: http.Header{},
			expected:       map[string]string{},
			wantErr:        false,
		},
		{
			name: "complex JSONPath with nested objects",
			headers: HeadersConfig{
				{Header: HeaderConfig{Name: "X-Client-ID", ValueFrom: "request.headers['x-mcpify-provider-data'].client.id"}},
				{Header: HeaderConfig{Name: "X-Client-Secret", ValueFrom: "request.headers['x-mcpify-provider-data'].client.secret"}},
			},
			requestHeaders: http.Header{
				"X-Mcpify-Provider-Data": []string{`{"client": {"id": "client123", "secret": "secret456"}}`},
			},
			expected: map[string]string{
				"X-Client-ID":     "client123",
				"X-Client-Secret": "secret456",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := evaluator.EvaluateHeaders(tt.headers, tt.requestHeaders)

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

func TestHeaderEvaluator_evaluateValueFrom(t *testing.T) {
	evaluator := NewHeaderEvaluator()

	tests := []struct {
		name       string
		expression string
		context    HeaderRequestContext
		expected   string
		wantErr    bool
	}{
		{
			name:       "simple header access",
			expression: "request.headers['authorization']",
			context: HeaderRequestContext{
				Headers: map[string]string{
					"authorization": "Bearer token123",
				},
			},
			expected: "Bearer token123",
			wantErr:  false,
		},
		{
			name:       "nested JSONPath",
			expression: "request.headers['x-mcpify-provider-data'].apikey",
			context: HeaderRequestContext{
				Headers: map[string]string{
					"x-mcpify-provider-data": `{"apikey": "sk-1234567890"}`,
				},
			},
			expected: "sk-1234567890",
			wantErr:  false,
		},
		{
			name:       "complex nested JSONPath",
			expression: "request.headers['x-mcpify-provider-data'].auth.api_key",
			context: HeaderRequestContext{
				Headers: map[string]string{
					"x-mcpify-provider-data": `{"auth": {"api_key": "key-abc123"}}`,
				},
			},
			expected: "key-abc123",
			wantErr:  false,
		},
		{
			name:       "missing header",
			expression: "request.headers['missing-header']",
			context: HeaderRequestContext{
				Headers: map[string]string{},
			},
			expected: "",
			wantErr:  false,
		},
		{
			name:       "invalid JSONPath",
			expression: "invalid.jsonpath[expression",
			context: HeaderRequestContext{
				Headers: map[string]string{},
			},
			expected: "",
			wantErr:  true,
		},
		{
			name:       "non-string result",
			expression: "request.headers['x-mcpify-provider-data'].count",
			context: HeaderRequestContext{
				Headers: map[string]string{
					"x-mcpify-provider-data": `{"count": 42}`,
				},
			},
			expected: "42",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := evaluator.evaluateValueFrom(tt.expression, tt.context)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestHeaderRequestContext_JSONSerialization(t *testing.T) {
	headerContext := HeaderRequestContext{
		Headers: map[string]string{
			"authorization": "Bearer token123",
			"content-type":  "application/json",
		},
	}

	// Test that context can be serialized to JSON
	jsonData, err := json.Marshal(headerContext)
	require.NoError(t, err)

	// Test that JSON can be deserialized back
	var deserialized HeaderRequestContext
	err = json.Unmarshal(jsonData, &deserialized)
	require.NoError(t, err)

	assert.Equal(t, headerContext.Headers, deserialized.Headers)
}
