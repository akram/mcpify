/*
Copyright 2025
SPDX-License-Identifier: Apache-2.0
*/
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"mcpify/internal/config"
	"mcpify/internal/handlers"
	"mcpify/internal/openapi"
	"mcpify/internal/types"
	"mcpify/pkg/mcp"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

func main() {
	// Parse command line flags with combined long/short form help
	transport := flag.String("transport", "", "Transport method (stdio, http)")
	port := flag.Int("port", 0, "Port for HTTP transport")
	host := flag.String("host", "", "Host for HTTP transport")
	configPath := flag.String("config", "", "Path to configuration file")
	specPath := flag.String("spec", "", "Path to OpenAPI specification (local file or URL)")
	baseURL := flag.String("base-url", "", "Base URL for API requests (defaults to domain from spec URL)")
	debug := flag.Bool("debug", false, "Enable debug logging for API requests and responses")

	// Add short flag aliases
	flag.StringVar(transport, "t", "", "Transport method (stdio, http)")
	flag.IntVar(port, "p", 0, "Port for HTTP transport")
	flag.StringVar(host, "h", "", "Host for HTTP transport")
	flag.StringVar(configPath, "c", "", "Path to configuration file")
	flag.StringVar(specPath, "s", "", "Path to OpenAPI specification (local file or URL)")
	flag.StringVar(baseURL, "b", "", "Base URL for API requests (defaults to domain from spec URL)")
	flag.BoolVar(debug, "d", false, "Enable debug logging for API requests and responses")

	// Customize flag usage to show both long and short forms on same line
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  -b, --base-url string\n")
		fmt.Fprintf(os.Stderr, "        Base URL for API requests (defaults to domain from spec URL)\n")
		fmt.Fprintf(os.Stderr, "  -c, --config string\n")
		fmt.Fprintf(os.Stderr, "        Path to configuration file\n")
		fmt.Fprintf(os.Stderr, "  -h, --host string\n")
		fmt.Fprintf(os.Stderr, "        Host for HTTP transport\n")
		fmt.Fprintf(os.Stderr, "  -p, --port int\n")
		fmt.Fprintf(os.Stderr, "        Port for HTTP transport\n")
		fmt.Fprintf(os.Stderr, "  -s, --spec string\n")
		fmt.Fprintf(os.Stderr, "        Path to OpenAPI specification (local file or URL)\n")
		fmt.Fprintf(os.Stderr, "  -t, --transport string\n")
		fmt.Fprintf(os.Stderr, "        Transport method (stdio, http)\n")
		fmt.Fprintf(os.Stderr, "  --help\n")
		fmt.Fprintf(os.Stderr, "        Show this help message\n")
	}

	flag.Parse()

	// Load configuration
	loader := config.NewLoader()
	cfg, err := loader.Load(*configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Override configuration with command line flags and log warnings
	if *transport != "" {
		if cfg.Server.Transport != "" && cfg.Server.Transport != *transport {
			log.Printf("WARNING: Overriding config transport '%s' with command line value '%s'", cfg.Server.Transport, *transport)
		}
		cfg.Server.Transport = *transport
	}
	if *host != "" {
		if cfg.Server.HTTP.Host != "" && cfg.Server.HTTP.Host != *host {
			log.Printf("WARNING: Overriding config host '%s' with command line value '%s'", cfg.Server.HTTP.Host, *host)
		}
		cfg.Server.HTTP.Host = *host
	}
	if *port != 0 {
		if cfg.Server.HTTP.Port != 0 && cfg.Server.HTTP.Port != *port {
			log.Printf("WARNING: Overriding config port %d with command line value %d", cfg.Server.HTTP.Port, *port)
		}
		cfg.Server.HTTP.Port = *port
	}
	if *specPath != "" {
		if cfg.OpenAPI.SpecPath != "" && cfg.OpenAPI.SpecPath != *specPath {
			log.Printf("WARNING: Overriding config spec_path '%s' with command line value '%s'", cfg.OpenAPI.SpecPath, *specPath)
		}
		cfg.OpenAPI.SpecPath = *specPath
	}
	if *baseURL != "" {
		if cfg.OpenAPI.BaseURL != "" && cfg.OpenAPI.BaseURL != *baseURL {
			log.Printf("WARNING: Overriding config base_url '%s' with command line value '%s'", cfg.OpenAPI.BaseURL, *baseURL)
		}
		cfg.OpenAPI.BaseURL = *baseURL
	}
	if *debug {
		cfg.OpenAPI.Debug = true
	}

	// Set default base URL from spec URL if not provided
	if cfg.OpenAPI.BaseURL == "" && cfg.OpenAPI.SpecPath != "" {
		if extractedBaseURL := extractBaseURLFromSpec(cfg.OpenAPI.SpecPath); extractedBaseURL != "" {
			cfg.OpenAPI.BaseURL = extractedBaseURL
			log.Printf("Using base URL extracted from spec: %s", cfg.OpenAPI.BaseURL)
		}
	}

	// Validate final configuration
	if err := cfg.Validate(); err != nil {
		log.Fatalf("Configuration validation failed: %v", err)
	}

	// Create MCP server
	server := mcp.NewServer()

	// Parse OpenAPI specification and generate tools
	parser := openapi.NewParser(&cfg.OpenAPI)
	apiTools, err := parser.ParseSpec()
	if err != nil {
		log.Fatalf("Failed to parse OpenAPI specification: %v", err)
	}

	log.Printf("Parsing OpenAPI spec from %s", cfg.OpenAPI.SpecPath)

	// Create API handler
	apiHandler := handlers.NewAPIHandler(&cfg.OpenAPI)

	// Register tools from OpenAPI specification
	registerAPITools(server, apiTools, apiHandler)
	log.Printf("Successfully parsed OpenAPI spec, generated %d tools", len(apiTools))

	// Log configuration summary
	log.Printf("=== MCPify Configuration Summary ===")
	log.Printf("OpenAPI Spec: %s", cfg.OpenAPI.SpecPath)
	log.Printf("Base URL: %s", cfg.OpenAPI.BaseURL)
	log.Printf("Transport: %s", cfg.Server.Transport)
	if cfg.Server.Transport == "http" {
		log.Printf("HTTP Server: %s:%d", cfg.Server.HTTP.Host, cfg.Server.HTTP.Port)
	}
	log.Printf("=====================================")

	// Start server based on transport
	switch cfg.Server.Transport {
	case "stdio":
		log.Println("Starting mcpify server with stdio transport...")
		if err := server.Run(); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	case "http":
		startHTTPServerWithConfig(server, cfg)
	default:
		log.Fatalf("Unknown transport: %s", cfg.Server.Transport)
	}
}

func startHTTPServerWithConfig(server *mcp.Server, cfg *config.Config) {
	// Configure MCP-compliant streamable HTTP transport from config
	httpConfig := &mcp.StreamableHTTPConfig{
		Host:           cfg.Server.HTTP.Host,
		Port:           cfg.Server.HTTP.Port,
		SessionTimeout: cfg.Server.HTTP.SessionTimeout,
		MaxConnections: cfg.Server.HTTP.MaxConnections,
		CORSEnabled:    cfg.Server.HTTP.CORS.Enabled,
		CORSOrigins:    cfg.Server.HTTP.CORS.Origins,
	}

	// Create MCP-compliant streamable HTTP transport
	httpTransport := mcp.NewStreamableHTTPTransport(server, httpConfig)

	// Setup graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Channel to listen for interrupt signals
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	// Start server in a goroutine
	go func() {
		log.Printf("Starting mcpify server with MCP streamable HTTP transport on %s:%d...",
			cfg.Server.HTTP.Host, cfg.Server.HTTP.Port)

		if err := httpTransport.Start(); err != nil {
			log.Printf("HTTP server error: %v", err)
			cancel()
		}
	}()

	// Wait for shutdown signal
	select {
	case <-c:
		log.Println("Received shutdown signal...")
	case <-ctx.Done():
		log.Println("Server context cancelled...")
	}

	// Create a timeout context for shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// Graceful shutdown
	if err := httpTransport.Stop(shutdownCtx); err != nil {
		log.Printf("Error during shutdown: %v", err)
	} else {
		log.Println("Server shut down gracefully")
	}
}

func registerAPITools(server *mcp.Server, apiTools []types.APITool, apiHandler *handlers.APIHandler) {
	for _, tool := range apiTools {
		// Create tool handler
		handler := func(tool types.APITool) func(params map[string]interface{}, requestContext config.RequestContext) (interface{}, error) {
			return func(params map[string]interface{}, requestContext config.RequestContext) (interface{}, error) {
				return apiHandler.HandleAPICall(tool, params, requestContext)
			}
		}(tool)

		// Generate input schema from OpenAPI parameters
		inputSchema := generateInputSchema(tool)

		// Register tool
		server.RegisterTool(
			tool.Name,
			tool.Description,
			inputSchema,
			handler,
		)

		log.Printf("Registered tool: %s (%s %s)", tool.Name, tool.Method, tool.Path)
	}
}

func generateInputSchema(tool types.APITool) map[string]interface{} {
	properties := make(map[string]interface{})
	required := []string{}

	// Add parameters
	for _, param := range tool.Parameters {
		// Add parameter location as a property
		properties[param.Name] = map[string]interface{}{
			"type":        getParameterType(param),
			"description": param.Description + " (in " + param.In + ")",
		}

		if param.Required {
			required = append(required, param.Name)
		}
	}

	// Add request body if present
	if tool.RequestBody != nil {
		// Use the actual request body schema from OpenAPI spec
		if tool.RequestBody.Content != nil {
			if jsonContent, exists := tool.RequestBody.Content["application/json"]; exists {
				// Check if this is a resolved schema (from our new schema resolution)
				if contentMap, ok := jsonContent.(map[string]interface{}); ok {
					if schema, hasSchema := contentMap["schema"]; hasSchema {
						// Use the resolved schema
						properties["body"] = schema
					} else {
						// Fallback to the content itself
						properties["body"] = jsonContent
					}
				} else {
					// Fallback to the content itself
					properties["body"] = jsonContent
				}
			} else {
				// Fallback to generic object if no JSON content type found
				properties["body"] = map[string]interface{}{
					"type":        "object",
					"description": "Request body data",
				}
			}
		} else {
			// Fallback to generic object if no content defined
			properties["body"] = map[string]interface{}{
				"type":        "object",
				"description": "Request body data",
			}
		}

		// Add body to required fields if the request body is required
		if tool.RequestBody.Required {
			required = append(required, "body")
		}
	}

	// Handle Swagger 2.0 body parameters (parameters with in: "body")
	// These should be treated as request body parameters
	for _, param := range tool.Parameters {
		if param.In == "body" {
			// This is a body parameter from Swagger 2.0, use the parameter name
			paramSchema := map[string]interface{}{
				"type":        "object",
				"description": param.Description,
			}

			// Try to use the actual schema if available
			if param.Schema != nil {
				if schemaMap, ok := param.Schema.(map[string]interface{}); ok {
					paramSchema = schemaMap
				}
			}

			properties[param.Name] = paramSchema

			if param.Required {
				required = append(required, param.Name)
			}
		}
	}

	finalSchema := map[string]interface{}{
		"type":       "object",
		"properties": properties,
		"required":   required,
	}

	return finalSchema
}

func getParameterType(param types.OpenAPIParameter) string {
	// Default to string type
	paramType := "string"

	// Try to extract type from schema
	if param.Schema != nil {
		if schemaMap, ok := param.Schema.(map[string]interface{}); ok {
			if typeVal, exists := schemaMap["type"]; exists {
				if typeStr, ok := typeVal.(string); ok {
					paramType = typeStr
				}
			}
		}
	}

	return paramType
}

// extractBaseURLFromSpec extracts the base URL (domain) from a spec URL
// For example: http://localhost:8080/swagger -> http://localhost:8080
func extractBaseURLFromSpec(specPath string) string {
	// Only process HTTP/HTTPS URLs
	if !strings.HasPrefix(specPath, "http://") && !strings.HasPrefix(specPath, "https://") {
		return ""
	}

	// Parse the URL
	parsedURL, err := url.Parse(specPath)
	if err != nil {
		return ""
	}

	// Reconstruct the base URL with scheme and host
	// Only include port if it's not the default port
	if parsedURL.Port() != "" {
		// Check if it's a non-default port
		if (parsedURL.Scheme == "http" && parsedURL.Port() != "80") ||
			(parsedURL.Scheme == "https" && parsedURL.Port() != "443") {
			baseURL := fmt.Sprintf("%s://%s:%s", parsedURL.Scheme, parsedURL.Hostname(), parsedURL.Port())
			return baseURL
		}
	}

	// Use hostname without port for default ports
	baseURL := fmt.Sprintf("%s://%s", parsedURL.Scheme, parsedURL.Hostname())

	return baseURL
}
