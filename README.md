# MCPify - Universal MCP Server for OpenAPI APIs

<div align="center">
  <img src="logo.png" alt="MCPify Logo" width="200">
</div>
MCPify is a universal Model Context Protocol (MCP) server that automatically converts any OpenAPI/Swagger-enabled API into MCP tools. It reads OpenAPI specification files and dynamically registers tools for each endpoint, making any REST API accessible through the MCP protocol.

## Features

- **Universal API Support**: Works with any OpenAPI 3.0+ specification
- **Automatic Tool Generation**: Converts API endpoints into MCP tools automatically
- **Multiple Transport Modes**: Supports both stdio and HTTP transports
- **Authentication Support**: Bearer tokens, Basic auth, API keys, and custom headers
- **Flexible Configuration**: YAML/JSON configuration with command-line overrides
- **Path Filtering**: Include/exclude specific API paths
- **Retry Logic**: Configurable retry attempts for failed requests
- **CORS Support**: Built-in CORS handling for web clients
- **Session Management**: MCP-compliant session handling for HTTP transport

## Installation

### Prerequisites

- Go 1.21 or later
- OpenAPI 3.0+ specification file

### Build from Source

```bash
git clone https://github.com/akram/mcpify.git
cd mcpify
go build -o mcpify ./cmd/server
```

## Quick Start

1. **Create a configuration file** (`config.yaml`):

```yaml
server:
  transport: "http"
  http:
    host: "127.0.0.1"
    port: 9090

openapi:
  spec_path: "https://api.example.com/openapi.json"
  base_url: "https://api.example.com"
  auth:
    type: "bearer"
    token: "your-api-token"
  tool_prefix: "example"
```

2. **Run the server**:

```bash
./mcpify --config config.yaml
```

3. **Connect with an MCP client** to `http://127.0.0.1:9090/mcp`

## Configuration

### Server Configuration

```yaml
server:
  transport: "http"  # "stdio" or "http"
  http:
    host: "127.0.0.1"
    port: 9090  # Default port
    session_timeout: "5m"
    max_connections: 100
    cors:
      enabled: true
      origins:
        - "http://localhost:3000"
        - "http://127.0.0.1:3000"
```

### OpenAPI Configuration

```yaml
openapi:
  spec_path: "path/to/openapi.json"  # Local file or URL
  base_url: "https://api.example.com"
  timeout: "30s"
  max_retries: 3
  # tool_prefix: "api"  # Optional, defaults to empty
  
  # Authentication
  auth:
    type: "bearer"  # "none", "bearer", "basic", "api_key"
    token: "your-bearer-token"
    # For basic auth:
    # username: "user"
    # password: "pass"
    # For API key:
    # api_key: "key"
    # api_key_name: "X-API-Key"
    # api_key_in: "header"  # "header" or "query"
  
  # Custom headers
  headers:
    "User-Agent": "MCPify/1.0.0"
    "Accept": "application/json"
  
  # Path filtering
  exclude_paths:
    - "/health"
    - "/metrics"
  include_paths:
    - "/api/v1/*"
```

### Logging Configuration

```yaml
logging:
  level: "info"  # "debug", "info", "warn", "error"
  format: "json"  # "json" or "text"
  output: "stdout"
```

### Security Configuration

```yaml
security:
  rate_limiting:
    enabled: true
    requests_per_minute: 100
  request_size_limit: "1MB"
```

## Command Line Options

```bash
./mcpify [options]

Options:
  --base-url, -b string
        Base URL for API requests (defaults to domain from spec URL)
  --config, -c string
        Path to configuration file
  --transport, -t string
        Transport method (stdio, http)
  --host, -h string
        Host for HTTP transport
  --port, -p int
        Port for HTTP transport (default: 9090)
  --spec, -s string
        Path to OpenAPI specification (local file or URL)
```

### Command Line Precedence

Command line arguments take precedence over configuration file values. When a parameter is specified both in the config file and via command line with different values, mcpify will log a warning and use the command line value.

**Example:**
```bash
# Config file has port: 8080, but command line overrides it
./mcpify --config config.yaml --port 9090
# Output: WARNING: Overriding config port 8080 with command line value 9090
```

## Examples

### Example 1: GitHub API

```yaml
openapi:
  spec_path: "https://raw.githubusercontent.com/github/rest-api-description/main/descriptions/api.github.com/api.github.com.json"
  base_url: "https://api.github.com"
  auth:
    type: "bearer"
    token: "ghp_your_github_token"
  tool_prefix: "github"
  headers:
    "Accept": "application/vnd.github.v3+json"
```

Run with:
```bash
./mcpify --config github-config.yaml
```

### Example 2: Local API with Basic Auth

```yaml
openapi:
  spec_path: "./api-spec.yaml"
  base_url: "http://localhost:3000"
  auth:
    type: "basic"
    username: "admin"
    password: "secret"
  tool_prefix: "local"
```

Run with:
```bash
./mcpify --config local-config.yaml
```

### Example 3: API with Query Parameter Authentication

```yaml
openapi:
  spec_path: "https://api.service.com/openapi.json"
  base_url: "https://api.service.com"
  auth:
    type: "api_key"
    api_key: "your-api-key"
    api_key_name: "api_key"
    api_key_in: "query"
  tool_prefix: "service"
```

Run with:
```bash
./mcpify --config service-config.yaml
```

### Example 4: Command Line Override

```bash
# Override config file values with command line arguments
./mcpify --config config.yaml --host 0.0.0.0 --port 8080 --spec https://api.example.com/openapi.json
```

## Tool Generation

MCPify automatically generates MCP tools based on your OpenAPI specification:

- **Tool Names**: Generated from operation ID or camelCase path + method (e.g., `findPetsByStatus`)
- **Descriptions**: Uses operation summary or description
- **Parameters**: Automatically mapped from OpenAPI parameters
- **Request Bodies**: Supported for POST, PUT, PATCH operations

### Example Generated Tool

For an OpenAPI endpoint:
```yaml
paths:
  /users/{id}:
    get:
      operationId: getUserById
      parameters:
        - name: id
          in: path
          required: true
          schema:
            type: string
```

MCPify generates:
- **Tool Name**: `getUserById` (or with prefix: `api_getUserById`)
- **Description**: From the operation summary/description
- **Parameters**: `id` (required path parameter)

## MCP Protocol Compliance

MCPify implements the full MCP specification:

- **JSON-RPC 2.0**: All requests/responses follow JSON-RPC format
- **Streamable HTTP Transport**: Supports both POST (JSON-RPC) and GET (SSE)
- **Session Management**: Cryptographically secure session IDs
- **Error Handling**: Proper error codes and HTTP status mapping
- **CORS Support**: Configurable cross-origin resource sharing

## Development

### Project Structure

```
mcpify/
├── cmd/server/          # Main server entry point
├── internal/
│   ├── config/         # Configuration management
│   ├── openapi/        # OpenAPI parsing and tool generation
│   ├── handlers/       # HTTP request handlers
│   └── types/          # Type definitions
├── pkg/mcp/            # MCP protocol implementation
└── README.md
```

### Building

```bash
# Build for current platform
go build -o mcpify ./cmd/server

# Build for multiple platforms
GOOS=linux GOARCH=amd64 go build -o mcpify-linux ./cmd/server
GOOS=windows GOARCH=amd64 go build -o mcpify.exe ./cmd/server
GOOS=darwin GOARCH=amd64 go build -o mcpify-macos ./cmd/server
```

### Testing

```bash
# Run tests
go test ./...

# Run with coverage
go test -cover ./...
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests for new functionality
5. Submit a pull request

## License

Apache License 2.0 - see LICENSE file for details.

## Documentation

- **[Dynamic Headers](docs/DYNAMIC_HEADERS.md)**: Configuration guide for dynamic header evaluation
- **[Request Evaluator](docs/REQUEST_EVALUATOR.md)**: Advanced request evaluation and filtering
- **[Implementation Summary](docs/IMPLEMENTATION_SUMMARY.md)**: Technical implementation details

## Support

- **Issues**: Report bugs and request features on GitHub
- **Documentation**: Check the README and configuration examples
- **MCP Protocol**: Learn more at [modelcontextprotocol.io](https://modelcontextprotocol.io)

## About the Logo

The MCPify logo features the **Macropinna microstoma**, commonly known as the barreleye fish. This deep-sea fish is known for its unique transparent head and tubular eyes that can rotate to look upward through its transparent dome. The Macropinna fish symbolizes the project's ability to "see through" API specifications and provide clear, transparent access to any OpenAPI-enabled service through the MCP protocol. Just as the Macropinna can see through its transparent head to spot prey above, MCPify provides clear visibility and access to API endpoints through its universal MCP interface.
