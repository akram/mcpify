package config

import (
	"errors"
	"testing"
)

func TestErrors(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{
			name:     "ErrInvalidTransport",
			err:      ErrInvalidTransport,
			expected: "invalid transport method",
		},
		{
			name:     "ErrInvalidPort",
			err:      ErrInvalidPort,
			expected: "invalid port number",
		},
		{
			name:     "ErrMissingOpenAPISpec",
			err:      ErrMissingOpenAPISpec,
			expected: "OpenAPI spec path is required",
		},
		{
			name:     "ErrInvalidTimeout",
			err:      ErrInvalidTimeout,
			expected: "invalid timeout value",
		},
		{
			name:     "ErrInvalidMaxRetries",
			err:      ErrInvalidMaxRetries,
			expected: "invalid max retries value",
		},
		{
			name:     "ErrInvalidRateLimit",
			err:      ErrInvalidRateLimit,
			expected: "invalid rate limit value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Error() != tt.expected {
				t.Errorf("Expected error message '%s', got '%s'", tt.expected, tt.err.Error())
			}
		})
	}
}

func TestErrorTypes(t *testing.T) {
	// Test that errors are of the correct type
	if !errors.Is(ErrInvalidTransport, ErrInvalidTransport) {
		t.Error("ErrInvalidTransport should be equal to itself")
	}

	if !errors.Is(ErrInvalidPort, ErrInvalidPort) {
		t.Error("ErrInvalidPort should be equal to itself")
	}

	if !errors.Is(ErrMissingOpenAPISpec, ErrMissingOpenAPISpec) {
		t.Error("ErrMissingOpenAPISpec should be equal to itself")
	}

	if !errors.Is(ErrInvalidTimeout, ErrInvalidTimeout) {
		t.Error("ErrInvalidTimeout should be equal to itself")
	}

	if !errors.Is(ErrInvalidMaxRetries, ErrInvalidMaxRetries) {
		t.Error("ErrInvalidMaxRetries should be equal to itself")
	}

	if !errors.Is(ErrInvalidRateLimit, ErrInvalidRateLimit) {
		t.Error("ErrInvalidRateLimit should be equal to itself")
	}
}

func TestErrorUniqueness(t *testing.T) {
	// Test that all errors are unique
	errors := []error{
		ErrInvalidTransport,
		ErrInvalidPort,
		ErrMissingOpenAPISpec,
		ErrInvalidTimeout,
		ErrInvalidMaxRetries,
		ErrInvalidRateLimit,
	}

	for i, err1 := range errors {
		for j, err2 := range errors {
			if i != j && err1.Error() == err2.Error() {
				t.Errorf("Errors %d and %d have the same message: %s", i, j, err1.Error())
			}
		}
	}
}
