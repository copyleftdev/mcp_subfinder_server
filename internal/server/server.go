// Package server implements HTTP handlers for the MCP Subfinder Server
package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	jsoniter "github.com/json-iterator/go"
	"mcp-subfinder-server/internal/mcp"
)

// Server represents an HTTP server for handling MCP requests
type Server struct {
	ProviderConfigPath string
	Logger             *slog.Logger
}

// MCPHandler handles JSON-RPC requests for the MCP protocol
func MCPHandler(w http.ResponseWriter, r *http.Request) {
	// Only allow POST requests
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Set content type for the response
	w.Header().Set("Content-Type", "application/json")

	// Read the request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		// Return a JSON-RPC error response
		errorResponse := mcp.Response{
			JSONRPC: "2.0",
			Error:   mcp.ErrParse,
		}
		responseJSON, _ := json.Marshal(errorResponse)
		w.WriteHeader(http.StatusOK) // Always return 200 OK for JSON-RPC
		w.Write(responseJSON)
		return
	}

	// Parse the JSON-RPC request
	var req mcp.Request
	if err := jsoniter.Unmarshal(body, &req); err != nil {
		// Return a JSON-RPC error response for invalid JSON
		errorResponse := mcp.Response{
			JSONRPC: "2.0",
			Error:   mcp.ErrParse,
		}
		responseJSON, _ := json.Marshal(errorResponse)
		w.WriteHeader(http.StatusOK) // Always return 200 OK for JSON-RPC
		w.Write(responseJSON)
		return
	}

	// Create a default Logger for this request
	logger := slog.Default()

	// Process the request and get a response
	var response mcp.Response

	switch req.Method {
	case "initialize":
		response = mcp.HandleInitialize(&req)
	case "tools.list":
		response = mcp.HandleToolsList(&req)
	case "tools.call":
		// Create a context with timeout for the operation
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		response = mcp.HandleToolsCall(ctx, &req, "", logger)
	default:
		// Method not found
		response = mcp.Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   mcp.ErrMethodNotFound,
		}
	}

	// Always set the JSONRPC version in the response
	response.JSONRPC = "2.0"

	// Write the response
	responseJSON, err := json.Marshal(response)
	if err != nil {
		// If we can't marshal the response, return a server error
		errorResponse := mcp.Response{
			JSONRPC: "2.0",
			Error:   mcp.ErrInternal,
		}
		responseJSON, _ = json.Marshal(errorResponse)
	}

	w.WriteHeader(http.StatusOK) // Always return 200 OK for JSON-RPC
	w.Write(responseJSON)
}

// HealthHandler responds to health check requests
func HealthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}

// New creates a new server instance with the given provider config path
func New(providerConfigPath string, logger *slog.Logger) *Server {
	return &Server{
		ProviderConfigPath: providerConfigPath,
		Logger:             logger,
	}
}

// Start starts the HTTP server on the given port
func (s *Server) Start(port int) error {
	// Set up the HTTP handlers
	mux := http.NewServeMux()
	
	// Register the MCP handler
	mux.HandleFunc("/mcp", func(w http.ResponseWriter, r *http.Request) {
		MCPHandler(w, r)
	})
	
	// Register the health check handler
	mux.HandleFunc("/health", HealthHandler)
	
	// Start the server
	addr := fmt.Sprintf(":%d", port)
	s.Logger.Info("Starting MCP Subfinder Server", "address", addr)
	return http.ListenAndServe(addr, mux)
}
