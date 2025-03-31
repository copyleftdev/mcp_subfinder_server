package mcp

import (
	"context"
	"encoding/base64"
	"fmt"
	"log/slog"
	"strings"

	jsoniter "github.com/json-iterator/go"
	"mcp-subfinder-server/internal/subfinder"
)

// HandleInitialize processes an initialize request
func HandleInitialize(req *Request) Response {
	// Parse and validate params
	var params InitializeParams
	if err := jsoniter.Unmarshal(req.Params, &params); err != nil {
		return Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   ErrParse,
		}
	}

	// Validate protocol version
	if params.ProtocolVersion != SupportedProtocolVersion {
		return Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &RPCError{
				Code:    InvalidParamsCode,
				Message: fmt.Sprintf("Unsupported protocol version: %s. Server supports: %s", params.ProtocolVersion, SupportedProtocolVersion),
			},
		}
	}

	// Return server capabilities
	return Response{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: InitializeResult{
			Name:            "MCP Subfinder Server",
			Version:         "1.0.0",
			ProtocolVersion: SupportedProtocolVersion,
		},
	}
}

// HandleToolsList processes a tools.list request
func HandleToolsList(req *Request) Response {
	// Define the enumerateSubdomains tool with its input schema
	subdomainTool := Tool{
		Name:        "enumerateSubdomains",
		Title:       "Enumerate Subdomains",
		Description: "Discovers subdomains for a given domain using the subfinder tool",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"domain": map[string]interface{}{
					"type":        "string",
					"description": "The base domain to enumerate subdomains for (e.g., example.com)",
				},
				"timeout": map[string]interface{}{
					"type":        "integer",
					"description": "Maximum time in seconds to run enumeration (default: 60)",
					"default":     60,
				},
				"maxDepth": map[string]interface{}{
					"type":        "integer",
					"description": "Maximum depth to explore for subdomain enumeration (default: 1)",
					"default":     1,
				},
				"sourcesFilter": map[string]interface{}{
					"type":        "string",
					"description": "Comma-separated list of sources to use (default: all sources)",
				},
				"excludeSourcesFilter": map[string]interface{}{
					"type":        "string",
					"description": "Comma-separated list of sources to exclude",
				},
				"recursive": map[string]interface{}{
					"type":        "boolean",
					"description": "Enable recursive subdomain discovery (default: false)",
					"default":     false,
				},
			},
			"required": []string{"domain"},
		},
		RequiresAPIKeys: true,
	}

	// Return the list of tools
	return Response{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: ToolsListResult{
			Tools: []Tool{subdomainTool},
		},
	}
}

// HandleToolsCall processes a tools.call request
func HandleToolsCall(ctx context.Context, req *Request, providerConfigPath string, logger *slog.Logger) Response {
	// Parse and validate params
	var params ToolCallParams
	if err := jsoniter.Unmarshal(req.Params, &params); err != nil {
		logger.Error("Failed to parse tools.call params", "error", err)
		return Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   ErrParse,
		}
	}

	// Check if the requested tool is supported
	if params.Name != "enumerateSubdomains" {
		logger.Warn("Tool not found", "requestedTool", params.Name)
		return Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   ErrMethodNotFound,
		}
	}

	// Extract and validate required domain parameter
	domainVal, ok := params.Arguments["domain"]
	if !ok {
		logger.Warn("Missing required domain parameter")
		return Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   ErrInvalidParams,
		}
	}

	domain, ok := domainVal.(string)
	if !ok || domain == "" {
		logger.Warn("Invalid domain parameter", "domain", domainVal)
		return Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   ErrInvalidParams,
		}
	}

	// Parse optional parameters with sensible defaults
	config := subfinder.SubfinderConfig{
		ProviderConfigPath: providerConfigPath,
		Timeout:            60, // Default timeout of 60 seconds
		MaxDepth:           1,  // Default max depth of 1
	}

	// Extract timeout if provided
	if timeoutVal, ok := params.Arguments["timeout"]; ok {
		if timeout, ok := timeoutVal.(float64); ok && timeout > 0 {
			config.Timeout = int(timeout)
			logger.Debug("Using custom timeout", "timeout", config.Timeout)
		} else {
			logger.Warn("Invalid timeout parameter, using default", "providedTimeout", timeoutVal)
		}
	}

	// Extract maxDepth if provided
	if maxDepthVal, ok := params.Arguments["maxDepth"]; ok {
		if maxDepth, ok := maxDepthVal.(float64); ok && maxDepth > 0 {
			config.MaxDepth = int(maxDepth)
			logger.Debug("Using custom maxDepth", "maxDepth", config.MaxDepth)
		} else {
			logger.Warn("Invalid maxDepth parameter, using default", "providedMaxDepth", maxDepthVal)
		}
	}

	// Extract sourcesFilter if provided
	if sourcesFilterVal, ok := params.Arguments["sourcesFilter"]; ok {
		if sourcesFilter, ok := sourcesFilterVal.(string); ok && sourcesFilter != "" {
			config.SourcesFilter = sourcesFilter
			logger.Debug("Using custom sourcesFilter", "sourcesFilter", config.SourcesFilter)
		} else {
			logger.Warn("Invalid sourcesFilter parameter, using default", "providedSourcesFilter", sourcesFilterVal)
		}
	}

	// Extract excludeSourcesFilter if provided
	if excludeSourcesFilterVal, ok := params.Arguments["excludeSourcesFilter"]; ok {
		if excludeSourcesFilter, ok := excludeSourcesFilterVal.(string); ok && excludeSourcesFilter != "" {
			config.ExcludeSourcesFilter = excludeSourcesFilter
			logger.Debug("Using custom excludeSourcesFilter", "excludeSourcesFilter", config.ExcludeSourcesFilter)
		} else {
			logger.Warn("Invalid excludeSourcesFilter parameter, using default", "providedExcludeSourcesFilter", excludeSourcesFilterVal)
		}
	}

	// Extract recursive if provided
	if recursiveVal, ok := params.Arguments["recursive"]; ok {
		if recursive, ok := recursiveVal.(bool); ok {
			config.Recursive = recursive
			logger.Debug("Using custom recursive setting", "recursive", config.Recursive)
		} else {
			logger.Warn("Invalid recursive parameter, using default", "providedRecursive", recursiveVal)
		}
	}

	// Execute the subdomain enumeration
	logger.Info("Running subdomain enumeration", "domain", domain, "config", config)
	subdomains, err := subfinder.RunEnumeration(ctx, domain, config, logger)

	// Prepare result
	var toolCallResult ToolCallResult

	// Handle execution errors
	if err != nil {
		logger.Error("Subdomain enumeration failed", "error", err)
		toolCallResult = ToolCallResult{
			IsError: true,
			Content: []interface{}{
				ContentItem{
					Type: "text",
					Text: fmt.Sprintf("Subdomain enumeration failed: %v", err),
				},
			},
		}
	} else {
		// Format successful results
		resultText := fmt.Sprintf("Found %d subdomains for %s:\n\n%s", 
			len(subdomains), 
			domain,
			strings.Join(subdomains, "\n"),
		)

		// Add simple text content item for CLI interfaces
		toolCallResult = ToolCallResult{
			IsError: false,
			Content: []interface{}{
				ContentItem{
					Type: "text",
					Text: fmt.Sprintf("Successfully enumerated %d subdomains for %s", len(subdomains), domain),
				},
				ResourceItem{
					Type:     "resource",
					MimeType: "text/plain",
					Blob:     base64.StdEncoding.EncodeToString([]byte(resultText)),
				},
			},
		}
	}

	// Return final response
	return Response{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  toolCallResult,
	}
}

// ProcessSingleRequest handles a single JSON-RPC request
func ProcessSingleRequest(ctx context.Context, req Request, providerConfigPath string, logger *slog.Logger) Response {
	// Set default JSON-RPC version
	if req.JSONRPC == "" {
		req.JSONRPC = "2.0"
	}

	// Route to appropriate handler based on method
	switch req.Method {
	case "initialize":
		return HandleInitialize(&req)
	case "tools.list":
		return HandleToolsList(&req)
	case "tools.call":
		return HandleToolsCall(ctx, &req, providerConfigPath, logger)
	default:
		// Check if it's a notification (no ID)
		if req.ID == nil {
			// Ignore notifications for unknown methods
			return Response{}
		}
		// Return method not found for requests with unknown methods
		logger.Warn("Method not found", "method", req.Method)
		return Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   ErrMethodNotFound,
		}
	}
}
