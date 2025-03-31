// Package main is the entry point for the MCP Subfinder Server
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"mcp-subfinder-server/internal/mcp"
	jsoniter "github.com/json-iterator/go"
)

const (
	// Protocol constants
	mcpProtocolVersion = "0.3"
	defaultServerPort  = 8080
	providerConfigFile = "provider-config.yaml"
	serverTimeout      = 30 * time.Second
	shutdownTimeout    = 10 * time.Second
)

func main() {
	// Setup structured logging with JSON output
	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	slog.SetDefault(logger)

	logger.Info("Starting MCP Subfinder Server",
		"version", "1.0.0",
		"protocolVersion", mcpProtocolVersion)

	// Get working directory for provider config file path
	workDir, err := os.Getwd()
	if err != nil {
		logger.Error("Failed to get working directory", "error", err)
		os.Exit(1)
	}

	// Set provider config path and ensure it exists
	providerConfigPath := filepath.Join(workDir, providerConfigFile)
	if _, err := os.Stat(providerConfigPath); os.IsNotExist(err) {
		logger.Warn("Provider config file not found, creating empty file", "path", providerConfigPath)
		// Create an empty file if it doesn't exist
		if err := os.WriteFile(providerConfigPath, []byte{}, 0644); err != nil {
			logger.Error("Failed to create provider config file", "error", err)
			os.Exit(1)
		}
	}
	logger.Info("Using provider config file", "path", providerConfigPath)

	// Create root context that will be canceled on SIGINT/SIGTERM
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Setup HTTP server
	mux := http.NewServeMux()
	mux.HandleFunc("/mcp", mcpHandler(providerConfigPath, logger))

	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Create HTTP server with timeouts
	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", defaultServerPort),
		Handler:           mux,
		ReadTimeout:       serverTimeout,
		WriteTimeout:      serverTimeout,
		IdleTimeout:       120 * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
		BaseContext: func(_ net.Listener) context.Context {
			return ctx
		},
	}

	// Start HTTP server in a goroutine
	go func() {
		logger.Info("HTTP server starting", "port", defaultServerPort)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("HTTP server error", "error", err)
			stop() // Signal application to shutdown
		}
	}()

	// Wait for shutdown signal
	<-ctx.Done()
	logger.Info("Shutdown initiated")

	// Create shutdown context with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	// Attempt graceful shutdown
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("HTTP server shutdown error", "error", err)
	} else {
		logger.Info("HTTP server shutdown complete")
	}
}

// mcpHandler creates a handler function for MCP protocol requests
func mcpHandler(providerConfigPath string, logger *slog.Logger) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		// Ensure the request method is POST
		if r.Method != http.MethodPost {
			logger.Warn("Invalid HTTP method", "method", r.Method, "remoteAddr", r.RemoteAddr)
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Ensure the content type is application/json
		contentType := r.Header.Get("Content-Type")
		if contentType != "application/json" {
			logger.Warn("Invalid content type", "contentType", contentType, "remoteAddr", r.RemoteAddr)
			http.Error(w, "Content type must be application/json", http.StatusUnsupportedMediaType)
			return
		}

		// Log request start
		requestID := fmt.Sprintf("%d", time.Now().UnixNano())
		logger.Info("Received MCP request",
			"requestID", requestID,
			"remoteAddr", r.RemoteAddr,
			"method", r.Method)

		// Ensure body is closed after processing
		defer r.Body.Close()

		// Read request body
		body, err := io.ReadAll(r.Body)
		if err != nil {
			logger.Error("Failed to read request body", "error", err, "requestID", requestID)
			http.Error(w, "Failed to read request", http.StatusBadRequest)
			return
		}

		// Prepare context with timeout
		ctx, cancel := context.WithTimeout(r.Context(), serverTimeout)
		defer cancel()

		// Process the request (batch or single)
		var response interface{}

		// Check if the request is a batch (array)
		if len(body) > 0 && body[0] == '[' {
			// Parse batch request
			var batchRequest []mcp.Request
			if err := jsoniter.Unmarshal(body, &batchRequest); err != nil {
				logger.Error("Failed to parse batch request", "error", err, "requestID", requestID)
				response = []mcp.Response{{
					JSONRPC: "2.0",
					Error:   mcp.ErrParse,
				}}
			} else {
				// Process each request in the batch
				batchResponse := make([]mcp.Response, 0, len(batchRequest))
				for _, req := range batchRequest {
					resp := mcp.ProcessSingleRequest(ctx, req, providerConfigPath, logger)
					// Only include non-empty responses (important for notifications)
					if resp.ID != nil || resp.Error != nil {
						batchResponse = append(batchResponse, resp)
					}
				}
				response = batchResponse
			}
		} else {
			// Parse single request
			var singleRequest mcp.Request
			if err := jsoniter.Unmarshal(body, &singleRequest); err != nil {
				logger.Error("Failed to parse single request", "error", err, "requestID", requestID)
				response = mcp.Response{
					JSONRPC: "2.0",
					Error:   mcp.ErrParse,
				}
			} else {
				// Process single request
				response = mcp.ProcessSingleRequest(ctx, singleRequest, providerConfigPath, logger)
			}
		}

		// Write response
		writeResponse(w, response, http.StatusOK, logger, requestID)
		logger.Info("Completed MCP request", "requestID", requestID)
	}
}

// writeResponse writes a JSON response to the HTTP response writer
func writeResponse(w http.ResponseWriter, resp interface{}, httpStatusCode int, logger *slog.Logger, requestID string) {
	// Set response headers
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatusCode)

	// Encode response as JSON
	if err := jsoniter.NewEncoder(w).Encode(resp); err != nil {
		logger.Error("Failed to encode response", "error", err, "requestID", requestID)
		// At this point headers are already sent, so we can only log the error
	}
}
