package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"mcp-subfinder-server/internal/server"
)

// Basic integration test structure
type TestRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      int         `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

type TestResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      int         `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   interface{} `json:"error,omitempty"`
}

// setupServer creates and configures a test HTTP server
func setupServer() http.Handler {
	// Create a new server mux
	mux := http.NewServeMux()
	
	// Register the MCP and health endpoints
	mux.HandleFunc("/mcp", server.MCPHandler)
	mux.HandleFunc("/health", server.HealthHandler)
	
	return mux
}

// TestIntegration is a simple integration test to ensure the server starts up and can handle requests
func TestIntegration(t *testing.T) {
	// Skip the integration test unless explicitly enabled
	if os.Getenv("ENABLE_INTEGRATION_TESTS") != "1" {
		t.Skip("Skipping integration test. Set ENABLE_INTEGRATION_TESTS=1 to enable")
	}

	// Start the server with a test router in a goroutine
	server := setupServer()
	ts := httptest.NewServer(server)
	defer ts.Close()

	// Test the health endpoint
	t.Run("HealthCheck", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/health")
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status OK, got %v", resp.Status)
		}

		var result map[string]string
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if status, ok := result["status"]; !ok || status != "ok" {
			t.Errorf("Expected status 'ok', got %v", status)
		}
	})

	// Test the MCP protocol initialization
	t.Run("MCPInitialize", func(t *testing.T) {
		req := TestRequest{
			JSONRPC: "2.0",
			ID:      1,
			Method:  "initialize",
			Params: map[string]string{
				"protocolVersion": "0.3",
			},
		}

		reqBody, err := json.Marshal(req)
		if err != nil {
			t.Fatalf("Failed to marshal request: %v", err)
		}

		resp, err := http.Post(ts.URL+"/mcp", "application/json", bytes.NewBuffer(reqBody))
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status OK, got %v", resp.Status)
		}

		var response TestResponse
		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if response.Error != nil {
			t.Errorf("Expected no error, got %v", response.Error)
		}

		result, ok := response.Result.(map[string]interface{})
		if !ok {
			t.Fatalf("Result is not a map: %T", response.Result)
		}

		if name, ok := result["name"].(string); !ok || name != "MCP Subfinder Server" {
			t.Errorf("Expected 'MCP Subfinder Server', got %v", name)
		}
	})

	// Test the tools list endpoint
	t.Run("ToolsList", func(t *testing.T) {
		req := TestRequest{
			JSONRPC: "2.0",
			ID:      2,
			Method:  "tools.list",
		}

		reqBody, err := json.Marshal(req)
		if err != nil {
			t.Fatalf("Failed to marshal request: %v", err)
		}

		resp, err := http.Post(ts.URL+"/mcp", "application/json", bytes.NewBuffer(reqBody))
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status OK, got %v", resp.Status)
		}

		var response TestResponse
		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if response.Error != nil {
			t.Errorf("Expected no error, got %v", response.Error)
		}

		result, ok := response.Result.(map[string]interface{})
		if !ok {
			t.Fatalf("Result is not a map: %T", response.Result)
		}

		tools, ok := result["tools"].([]interface{})
		if !ok {
			t.Fatalf("tools is not an array: %T", result["tools"])
		}

		// Check that at least one tool is returned
		if len(tools) == 0 {
			t.Errorf("Expected at least one tool, got none")
		}
	})
}

// MockRunner is a function to run tests with a timeout
func MockRunner(t *testing.T, testFunc func(*testing.T), timeout time.Duration) {
	done := make(chan bool)
	go func() {
		testFunc(t)
		done <- true
	}()

	select {
	case <-done:
		return
	case <-time.After(timeout):
		t.Fatal("Test timed out")
	}
}
