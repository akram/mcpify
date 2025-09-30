# Dynamic Header Forwarding

mcpify supports dynamic header forwarding using JSONPath expressions to extract values from incoming request headers and forward them to backend APIs.

## Configuration Format

Headers can be configured in two ways:

### 1. Static Headers
```yaml
headers:
  - header:
      name: "User-Agent"
      value: "MCPify/1.0.0"
```

### 2. Dynamic Headers
```yaml
headers:
  - header:
      name: "Authorization"
      valueFrom: "request.headers['authorization']"
  - header:
      name: "X-API-Key"
      valueFrom: "request.headers['x-mcpify-provider-data'].apikey"
```

## JSONPath Expressions

The `valueFrom` field supports JSONPath expressions to extract values from incoming request headers:

### Simple Header Access
```yaml
valueFrom: "request.headers['authorization']"
```
Extracts the value of the `authorization` header directly.

### Nested JSON Extraction
```yaml
valueFrom: "request.headers['x-mcpify-provider-data'].apikey"
```
Extracts the `apikey` field from a JSON object in the `x-mcpify-provider-data` header.

### Complex Nested Objects
```yaml
valueFrom: "request.headers['x-mcpify-provider-data'].auth.api_key"
```
Extracts nested values from complex JSON structures.

## Configuration Locations

Dynamic headers can be configured in two places:

### 1. General Headers
```yaml
openapi:
  headers:
    - header:
        name: "User-Agent"
        value: "MCPify/1.0.0"
    - header:
        name: "Authorization"
        valueFrom: "request.headers['authorization']"
```

### 2. Authentication Headers
```yaml
openapi:
  auth:
    type: "bearer"
    token: "static-token"
    headers:
      - header:
          name: "X-API-Key"
          valueFrom: "request.headers['x-mcpify-provider-data'].apikey"
```

## Use Cases

### 1. Token Forwarding
Forward Bearer tokens from incoming requests:
```yaml
headers:
  - header:
      name: "Authorization"
      valueFrom: "request.headers['authorization']"
```

### 2. API Key Extraction
Extract API keys from JSON payloads:
```yaml
headers:
  - header:
      name: "X-API-Key"
      valueFrom: "request.headers['x-mcpify-provider-data'].apikey"
```

### 3. Multi-Provider Authentication
Handle different authentication methods:
```yaml
auth:
  headers:
    - header:
        name: "X-Provider-Auth"
        valueFrom: "request.headers['x-mcpify-provider-data'].provider.auth"
    - header:
        name: "X-Client-ID"
        valueFrom: "request.headers['x-mcpify-provider-data'].client.id"
```

## Example Request Flow

1. **Incoming Request**:
   ```
   POST /mcp
   Authorization: Bearer user-token-123
   X-Mcpify-Provider-Data: {"apikey": "sk-1234567890", "client": {"id": "client123"}}
   ```

2. **Configuration**:
   ```yaml
   headers:
     - header:
         name: "Authorization"
         valueFrom: "request.headers['authorization']"
     - header:
         name: "X-API-Key"
         valueFrom: "request.headers['x-mcpify-provider-data'].apikey"
     - header:
         name: "X-Client-ID"
         valueFrom: "request.headers['x-mcpify-provider-data'].client.id"
   ```

3. **Outgoing Request**:
   ```
   POST https://api.example.com/endpoint
   Authorization: Bearer user-token-123
   X-API-Key: sk-1234567890
   X-Client-ID: client123
   ```

## Error Handling

- If a JSONPath expression fails to evaluate, the header is skipped
- If a header value is empty or null, the header is not added
- Invalid JSONPath expressions are logged as warnings

## Backward Compatibility

The old header format is still supported:
```yaml
headers:
  "User-Agent": "MCPify/1.0.0"
  "Accept": "application/json"
```

This is automatically converted to the new format internally.
