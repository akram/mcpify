package config

import "errors"

var (
	ErrInvalidTransport   = errors.New("invalid transport method")
	ErrInvalidPort        = errors.New("invalid port number")
	ErrMissingOpenAPISpec = errors.New("OpenAPI spec path is required")
	ErrInvalidTimeout     = errors.New("invalid timeout value")
	ErrInvalidMaxRetries  = errors.New("invalid max retries value")
	ErrInvalidRateLimit   = errors.New("invalid rate limit value")
)
