# Request Evaluator - Production-Ready Dynamic Header Forwarding

The RequestEvaluator is a production-ready system for dynamically extracting values from incoming HTTP requests and forwarding them to backend APIs using JSONPath expressions.

## Features

- **Comprehensive Request Context**: Access headers, query parameters, form data, and request body
- **JSONPath Support**: Extract nested values from JSON strings in any request data
- **Case-Insensitive Matching**: HTTP headers are matched case-insensitively
- **Graceful Error Handling**: Missing values return empty strings instead of errors
- **Production Ready**: Robust parsing, comprehensive testing, and error handling

## Syntax

The RequestEvaluator uses a simple `request.*` syntax to access different parts of the HTTP request:

### Basic Syntax

```yaml
headers:
  - header:
      name: "Header-Name"
      valueFrom: "request.<type>['<key>']"
```

### Supported Request Types

| Type | Description | Example |
|------|-------------|---------|
| `headers` | HTTP request headers | `request.headers['authorization']` |
| `query` | URL query parameters | `request.query['apikey']` |
| `form` | Form data (POST body) | `request.form['user_id']` |
| `body` | Request body (JSON) | `request.body.user.id` |

### Nested JSON Extraction

When a request value contains JSON, you can extract nested values using dot notation:

```yaml
headers:
  - header:
      name: "X-API-Key"
      # Extract 'apikey' from JSON in header
      valueFrom: "request.headers['x-mcpify-provider-data'].apikey"
  - header:
      name: "X-Auth-Token"
      # Extract deeply nested value
      valueFrom: "request.headers['x-mcpify-provider-data'].auth.api_key"
```

## Configuration Examples

### Basic Header Forwarding

```yaml
headers:
  - header:
      name: "Authorization"
      valueFrom: "request.headers['authorization']"
  - header:
      name: "User-Agent"
      value: "MCPify/2.0.0"  # Static value
```

### Query Parameter to Header

```yaml
headers:
  - header:
      name: "X-API-Key"
      valueFrom: "request.query['apikey']"
  - header:
      name: "X-Client-ID"
      valueFrom: "request.query['client_id']"
```

### Form Data to Header

```yaml
headers:
  - header:
      name: "X-User-ID"
      valueFrom: "request.form['user_id']"
  - header:
      name: "X-Session-Token"
      valueFrom: "request.form['session_token']"
```

### Nested JSON Extraction

```yaml
headers:
  - header:
      name: "X-API-Key"
      # From header: X-Mcpify-Provider-Data: {"apikey": "sk-1234567890"}
      valueFrom: "request.headers['x-mcpify-provider-data'].apikey"
  - header:
      name: "X-Auth-Key"
      # From header: X-Mcpify-Provider-Data: {"auth": {"api_key": "key-xyz"}}
      valueFrom: "request.headers['x-mcpify-provider-data'].auth.api_key"
  - header:
      name: "X-Client-Secret"
      # From query: ?client_data={"secret": "secret-123"}
      valueFrom: "request.query['client_data'].secret"
  - header:
      name: "X-User-Name"
      # From form: user_data={"name": "John Doe"}
      valueFrom: "request.form['user_data'].name"
```

### Mixed Static and Dynamic Headers

```yaml
headers:
  - header:
      name: "User-Agent"
      value: "MCPify/2.0.0"  # Static
  - header:
      name: "Authorization"
      valueFrom: "request.headers['authorization']"  # Dynamic
  - header:
      name: "Content-Type"
      value: "application/json"  # Static
  - header:
      name: "X-API-Key"
      valueFrom: "request.query['apikey']"  # Dynamic
```

## Request Context Structure

The RequestEvaluator creates a comprehensive request context:

```json
{
  "headers": {
    "authorization": "Bearer token123",
    "content-type": "application/json",
    "x-mcpify-provider-data": "{\"apikey\": \"sk-1234567890\"}"
  },
  "query": {
    "apikey": "sk-1234567890",
    "client_id": "client-abc123"
  },
  "form": {
    "user_id": "user-123",
    "session_token": "session-xyz"
  },
  "body": {
    "user": {
      "id": "user-123",
      "name": "John Doe"
    }
  },
  "method": "POST",
  "path": "/api/endpoint"
}
```

## Error Handling

The RequestEvaluator handles errors gracefully:

- **Missing Values**: Returns empty string, header is omitted
- **Invalid JSONPath**: Returns empty string, header is omitted
- **Malformed JSON**: Returns empty string, header is omitted
- **Type Conversion**: Automatically converts non-string values to strings

## Case Sensitivity

- **Headers**: Case-insensitive matching (HTTP standard)
- **Query Parameters**: Case-sensitive matching
- **Form Data**: Case-sensitive matching
- **JSON Keys**: Case-sensitive matching

## Performance Considerations

- **Lazy Evaluation**: Values are only extracted when needed
- **Caching**: JSON parsing results are cached per request
- **Minimal Overhead**: Only processes configured headers
- **Memory Efficient**: Reuses request context objects

## Migration from HeaderEvaluator

The new RequestEvaluator is backward compatible with the old HeaderEvaluator:

### Old Syntax (still supported)
```yaml
headers:
  - header:
      name: "Authorization"
      valueFrom: "request.headers['authorization']"
```

### New Syntax (recommended)
```yaml
headers:
  - header:
      name: "Authorization"
      valueFrom: "request.headers['authorization']"
  - header:
      name: "X-API-Key"
      valueFrom: "request.query['apikey']"
  - header:
      name: "X-User-ID"
      valueFrom: "request.form['user_id']"
```

## Testing

The RequestEvaluator includes comprehensive tests covering:

- Static header values
- Dynamic header extraction from all request types
- Nested JSON extraction
- Error handling scenarios
- Case sensitivity
- Type conversion
- Edge cases

## Examples

See `config.request-evaluator.yaml` for a complete example configuration demonstrating all features.

## Best Practices

1. **Use Static Values**: For headers that don't change, use `value` instead of `valueFrom`
2. **Handle Missing Values**: The system gracefully handles missing values
3. **Validate JSON**: Ensure JSON in headers/query/form is valid
4. **Case Sensitivity**: Be aware of case sensitivity differences
5. **Performance**: Only extract values you actually need
6. **Security**: Be careful with sensitive data in headers/query parameters
