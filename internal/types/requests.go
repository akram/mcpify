package types

import (
	"encoding/json"
	"time"
)

// MCPRequest represents a JSON-RPC request
type MCPRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// MCPResponse represents a JSON-RPC response
type MCPResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *MCPError   `json:"error,omitempty"`
}

// MCPError represents a JSON-RPC error
type MCPError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Tool represents an MCP tool
type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

// ListToolsResult represents the result of tools/list
type ListToolsResult struct {
	Tools []Tool `json:"tools"`
}

// CallToolParams represents parameters for tools/call
type CallToolParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

// CallToolResult represents the result of tools/call
type CallToolResult struct {
	Content []ContentBlock `json:"content"`
}

// ContentBlock represents content in a tool result
type ContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// Session represents an MCP session
type Session struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	LastSeen  time.Time `json:"last_seen"`
	Active    bool      `json:"active"`
}

// OpenAPISpec represents the OpenAPI specification
type OpenAPISpec struct {
	OpenAPI string                 `json:"openapi" yaml:"openapi"`
	Info    OpenAPIInfo            `json:"info" yaml:"info"`
	Servers []OpenAPIServer        `json:"servers,omitempty" yaml:"servers,omitempty"`
	Paths   map[string]OpenAPIPath `json:"paths" yaml:"paths"`
}

// OpenAPIInfo represents the info section of OpenAPI spec
type OpenAPIInfo struct {
	Title       string `json:"title" yaml:"title"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
	Version     string `json:"version" yaml:"version"`
}

// OpenAPIServer represents a server in OpenAPI spec
type OpenAPIServer struct {
	URL         string `json:"url" yaml:"url"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
}

// OpenAPIPath represents a path in OpenAPI spec
type OpenAPIPath struct {
	Get    *OpenAPIOperation `json:"get,omitempty" yaml:"get,omitempty"`
	Post   *OpenAPIOperation `json:"post,omitempty" yaml:"post,omitempty"`
	Put    *OpenAPIOperation `json:"put,omitempty" yaml:"put,omitempty"`
	Delete *OpenAPIOperation `json:"delete,omitempty" yaml:"delete,omitempty"`
	Patch  *OpenAPIOperation `json:"patch,omitempty" yaml:"patch,omitempty"`
}

// OpenAPIOperation represents an operation in OpenAPI spec
type OpenAPIOperation struct {
	Summary     string                     `json:"summary,omitempty" yaml:"summary,omitempty"`
	Description string                     `json:"description,omitempty" yaml:"description,omitempty"`
	OperationID string                     `json:"operationId,omitempty" yaml:"operationId,omitempty"`
	Parameters  []OpenAPIParameter         `json:"parameters,omitempty" yaml:"parameters,omitempty"`
	RequestBody *OpenAPIRequestBody        `json:"requestBody,omitempty" yaml:"requestBody,omitempty"`
	Responses   map[string]OpenAPIResponse `json:"responses" yaml:"responses"`
	Tags        []string                   `json:"tags,omitempty" yaml:"tags,omitempty"`
}

// OpenAPIParameter represents a parameter in OpenAPI spec
type OpenAPIParameter struct {
	Name        string      `json:"name" yaml:"name"`
	In          string      `json:"in" yaml:"in"`
	Description string      `json:"description,omitempty" yaml:"description,omitempty"`
	Required    bool        `json:"required,omitempty" yaml:"required,omitempty"`
	Schema      interface{} `json:"schema,omitempty" yaml:"schema,omitempty"`
}

// OpenAPIRequestBody represents a request body in OpenAPI spec
type OpenAPIRequestBody struct {
	Description string                 `json:"description,omitempty" yaml:"description,omitempty"`
	Required    bool                   `json:"required,omitempty" yaml:"required,omitempty"`
	Content     map[string]interface{} `json:"content,omitempty" yaml:"content,omitempty"`
}

// OpenAPIResponse represents a response in OpenAPI spec
type OpenAPIResponse struct {
	Description string                 `json:"description" yaml:"description"`
	Content     map[string]interface{} `json:"content,omitempty" yaml:"content,omitempty"`
}

// APITool represents a tool generated from an OpenAPI endpoint
type APITool struct {
	Name        string
	Description string
	Method      string
	Path        string
	Parameters  []OpenAPIParameter
	RequestBody *OpenAPIRequestBody
	Handler     func(params map[string]interface{}, headers map[string]string) (interface{}, error)
}
