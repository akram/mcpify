package config

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestHeaderConfig_UnmarshalYAML(t *testing.T) {
	tests := []struct {
		name     string
		yamlData string
		expected HeaderConfig
		wantErr  bool
	}{
		{
			name: "static header value",
			yamlData: `
name: "User-Agent"
value: "MCPify/1.0.0"
`,
			expected: HeaderConfig{
				Name:  "User-Agent",
				Value: "MCPify/1.0.0",
			},
			wantErr: false,
		},
		{
			name: "dynamic header with valueFrom",
			yamlData: `
name: "Authorization"
valueFrom: "request.headers['x-mcpify-provider-data'].apikey"
`,
			expected: HeaderConfig{
				Name:      "Authorization",
				ValueFrom: "request.headers['x-mcpify-provider-data'].apikey",
			},
			wantErr: false,
		},
		{
			name: "header with both value and valueFrom (should error)",
			yamlData: `
name: "Test-Header"
value: "static-value"
valueFrom: "request.headers['dynamic']"
`,
			expected: HeaderConfig{},
			wantErr:  true,
		},
		{
			name: "header with neither value nor valueFrom (should error)",
			yamlData: `
name: "Test-Header"
`,
			expected: HeaderConfig{},
			wantErr:  true,
		},
		{
			name: "complex JSONPath expression",
			yamlData: `
name: "X-API-Key"
valueFrom: "request.headers['x-mcpify-provider-data'].auth.api_key"
`,
			expected: HeaderConfig{
				Name:      "X-API-Key",
				ValueFrom: "request.headers['x-mcpify-provider-data'].auth.api_key",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var header HeaderConfig
			err := yaml.Unmarshal([]byte(tt.yamlData), &header)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, header)
			}
		})
	}
}

func TestHeaderConfig_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		jsonData string
		expected HeaderConfig
		wantErr  bool
	}{
		{
			name:     "static header value",
			jsonData: `{"name": "User-Agent", "value": "MCPify/1.0.0"}`,
			expected: HeaderConfig{
				Name:  "User-Agent",
				Value: "MCPify/1.0.0",
			},
			wantErr: false,
		},
		{
			name:     "dynamic header with valueFrom",
			jsonData: `{"name": "Authorization", "valueFrom": "request.headers['x-mcpify-provider-data'].apikey"}`,
			expected: HeaderConfig{
				Name:      "Authorization",
				ValueFrom: "request.headers['x-mcpify-provider-data'].apikey",
			},
			wantErr: false,
		},
		{
			name:     "header with both value and valueFrom (should error)",
			jsonData: `{"name": "Test-Header", "value": "static-value", "valueFrom": "request.headers['dynamic']"}`,
			expected: HeaderConfig{},
			wantErr:  true,
		},
		{
			name:     "header with neither value nor valueFrom (should error)",
			jsonData: `{"name": "Test-Header"}`,
			expected: HeaderConfig{},
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var header HeaderConfig
			err := json.Unmarshal([]byte(tt.jsonData), &header)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, header)
			}
		})
	}
}

func TestHeaderConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		header  HeaderConfig
		wantErr bool
	}{
		{
			name: "valid static header",
			header: HeaderConfig{
				Name:  "User-Agent",
				Value: "MCPify/1.0.0",
			},
			wantErr: false,
		},
		{
			name: "valid dynamic header",
			header: HeaderConfig{
				Name:      "Authorization",
				ValueFrom: "request.headers['x-mcpify-provider-data'].apikey",
			},
			wantErr: false,
		},
		{
			name: "invalid: both value and valueFrom",
			header: HeaderConfig{
				Name:      "Test-Header",
				Value:     "static-value",
				ValueFrom: "request.headers['dynamic']",
			},
			wantErr: true,
		},
		{
			name: "invalid: neither value nor valueFrom",
			header: HeaderConfig{
				Name: "Test-Header",
			},
			wantErr: true,
		},
		{
			name: "invalid: empty name",
			header: HeaderConfig{
				Name:  "",
				Value: "some-value",
			},
			wantErr: true,
		},
		{
			name: "invalid: empty value",
			header: HeaderConfig{
				Name:  "Test-Header",
				Value: "",
			},
			wantErr: true,
		},
		{
			name: "invalid: empty valueFrom",
			header: HeaderConfig{
				Name:      "Test-Header",
				ValueFrom: "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.header.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestHeadersConfig_UnmarshalYAML(t *testing.T) {
	yamlData := `
- header:
    name: "User-Agent"
    value: "MCPify/1.0.0"
- header:
    name: "x-mcpify-provider-data"
    valueFrom: "request.headers['x-mcpify-other-data']"
- header:
    name: "Authorization"
    valueFrom: "request.headers['x-mcpify-other-data'].apikey"
`

	var headers HeadersConfig
	err := yaml.Unmarshal([]byte(yamlData), &headers)
	require.NoError(t, err)

	expected := HeadersConfig{
		{Header: HeaderConfig{Name: "User-Agent", Value: "MCPify/1.0.0"}},
		{Header: HeaderConfig{Name: "x-mcpify-provider-data", ValueFrom: "request.headers['x-mcpify-other-data']"}},
		{Header: HeaderConfig{Name: "Authorization", ValueFrom: "request.headers['x-mcpify-other-data'].apikey"}},
	}

	assert.Equal(t, expected, headers)
}

func TestHeadersConfig_UnmarshalJSON(t *testing.T) {
	jsonData := `[
		{"header": {"name": "User-Agent", "value": "MCPify/1.0.0"}},
		{"header": {"name": "x-mcpify-provider-data", "valueFrom": "request.headers['x-mcpify-other-data']"}},
		{"header": {"name": "Authorization", "valueFrom": "request.headers['x-mcpify-other-data'].apikey"}}
	]`

	var headers HeadersConfig
	err := json.Unmarshal([]byte(jsonData), &headers)
	require.NoError(t, err)

	expected := HeadersConfig{
		{Header: HeaderConfig{Name: "User-Agent", Value: "MCPify/1.0.0"}},
		{Header: HeaderConfig{Name: "x-mcpify-provider-data", ValueFrom: "request.headers['x-mcpify-other-data']"}},
		{Header: HeaderConfig{Name: "Authorization", ValueFrom: "request.headers['x-mcpify-other-data'].apikey"}},
	}

	assert.Equal(t, expected, headers)
}

func TestHeadersConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		headers HeadersConfig
		wantErr bool
	}{
		{
			name: "valid headers",
			headers: HeadersConfig{
				{Header: HeaderConfig{Name: "User-Agent", Value: "MCPify/1.0.0"}},
				{Header: HeaderConfig{Name: "Authorization", ValueFrom: "request.headers['auth']"}},
			},
			wantErr: false,
		},
		{
			name: "invalid header in list",
			headers: HeadersConfig{
				{Header: HeaderConfig{Name: "User-Agent", Value: "MCPify/1.0.0"}},
				{Header: HeaderConfig{Name: "Invalid", Value: "value", ValueFrom: "request.headers['auth']"}},
			},
			wantErr: true,
		},
		{
			name: "duplicate header names",
			headers: HeadersConfig{
				{Header: HeaderConfig{Name: "User-Agent", Value: "MCPify/1.0.0"}},
				{Header: HeaderConfig{Name: "User-Agent", Value: "Another/1.0.0"}},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.headers.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestOpenAPIConfig_ValidateHeaders(t *testing.T) {
	tests := []struct {
		name    string
		config  OpenAPIConfig
		wantErr bool
	}{
		{
			name: "valid config with new headers format",
			config: OpenAPIConfig{
				Auth: AuthConfig{
					Type: "bearer",
					Headers: HeadersConfig{
						{Header: HeaderConfig{Name: "X-Auth-Version", Value: "v2"}},
					},
				},
				Headers: HeadersConfig{
					{Header: HeaderConfig{Name: "User-Agent", Value: "MCPify/1.0.0"}},
					{Header: HeaderConfig{Name: "Authorization", ValueFrom: "request.headers['x-mcpify-provider-data'].apikey"}},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid config with duplicate header names",
			config: OpenAPIConfig{
				Auth: AuthConfig{
					Type: "bearer",
					Headers: HeadersConfig{
						{Header: HeaderConfig{Name: "Authorization", Value: "Bearer token"}},
					},
				},
				Headers: HeadersConfig{
					{Header: HeaderConfig{Name: "Authorization", ValueFrom: "request.headers['auth']"}},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := (&tt.config).Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
