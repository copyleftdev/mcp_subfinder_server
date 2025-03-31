package server

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMCPHandler(t *testing.T) {
	// Create a new instance of our handler
	handler := http.HandlerFunc(MCPHandler)

	tests := []struct {
		name           string
		method         string
		rawBody        string
		expectedStatus int
		checkResponse  func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:   "Valid initialize request",
			method: "POST",
			rawBody: `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"0.3"}}`,
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				var response map[string]interface{}
				err := json.Unmarshal(rr.Body.Bytes(), &response)
				if err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}
				
				// Check for error
				if errVal, exists := response["error"]; exists && errVal != nil {
					t.Errorf("Expected no error, got %v", errVal)
				}
				
				// Check ID - it might be base64 encoded in the response
				idVal, exists := response["id"]
				if !exists {
					t.Errorf("Expected ID to exist, but it's missing")
					return
				}
				
				// ID might be a base64 encoded string
				if idString, ok := idVal.(string); ok {
					// Try to decode it as base64
					decoded, err := base64.StdEncoding.DecodeString(idString)
					if err == nil {
						// If it's a valid base64 string, check if it decodes to "1"
						if string(decoded) != "1" {
							t.Errorf("Expected decoded ID to be 1, got %s", string(decoded))
						}
					} else {
						t.Errorf("Expected ID to be base64 encoded 1, got %s", idString)
					}
				} else if idVal != float64(1) {
					// If it's not a string, check if it's directly the number 1
					t.Errorf("Expected ID 1, got %v of type %T", idVal, idVal)
				}
				
				// Check result
				resultVal, exists := response["result"]
				if !exists {
					t.Errorf("Result field missing from response")
					return
				}
				
				result, ok := resultVal.(map[string]interface{})
				if !ok {
					t.Fatalf("Result is not a map: %T", resultVal)
				}
				
				if _, exists := result["protocolVersion"]; !exists {
					t.Errorf("Response missing protocolVersion field")
				}
			},
		},
		{
			name:   "Method not allowed",
			method: "GET",
			rawBody: "",
			expectedStatus: http.StatusMethodNotAllowed,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				if rr.Body.String() != "Method not allowed\n" {
					t.Errorf("Expected 'Method not allowed', got %s", rr.Body.String())
				}
			},
		},
		{
			name:   "Invalid JSON",
			method: "POST",
			rawBody: "{invalid json",
			expectedStatus: http.StatusOK, // Changed from 400 to 200 since we now return JSON-RPC errors with 200 OK
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				// Check that we get a proper JSON-RPC error response
				var response map[string]interface{}
				err := json.Unmarshal(rr.Body.Bytes(), &response)
				if err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}
				
				errObj, ok := response["error"].(map[string]interface{})
				if !ok {
					t.Fatalf("Error is not a map: %T", response["error"])
				}
				
				code, ok := errObj["code"].(float64)
				if !ok {
					t.Fatalf("Error code is not a number: %T", errObj["code"])
				}
				
				if code != -32700 {
					t.Errorf("Expected error code -32700, got %v", code)
				}
			},
		},
		{
			name:   "Unknown method",
			method: "POST",
			rawBody: `{"jsonrpc":"2.0","id":1,"method":"unknownMethod"}`,
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				var response map[string]interface{}
				err := json.Unmarshal(rr.Body.Bytes(), &response)
				if err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}
				
				errObj, ok := response["error"].(map[string]interface{})
				if !ok {
					t.Fatalf("Error is not a map: %T", response["error"])
					return
				}
				
				code, ok := errObj["code"].(float64)
				if !ok {
					t.Fatalf("Error code is not a number: %T", errObj["code"])
				}
				
				if code != -32601 {
					t.Errorf("Expected error code -32601, got %v", code)
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create a request with the test body
			req, err := http.NewRequest(tc.method, "/mcp", bytes.NewBufferString(tc.rawBody))
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}
			req.Header.Set("Content-Type", "application/json")

			// Create a ResponseRecorder to record the response
			rr := httptest.NewRecorder()

			// Call the handler
			handler.ServeHTTP(rr, req)

			// Check the status code
			if rr.Code != tc.expectedStatus {
				t.Errorf("Expected status code %d, got %d", tc.expectedStatus, rr.Code)
			}

			// Additional response checks
			tc.checkResponse(t, rr)
		})
	}
}

func TestHealthHandler(t *testing.T) {
	// Create a new instance of our handler
	handler := http.HandlerFunc(HealthHandler)

	// Create a request
	req, err := http.NewRequest("GET", "/health", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Create a ResponseRecorder to record the response
	rr := httptest.NewRecorder()

	// Call the handler
	handler.ServeHTTP(rr, req)

	// Check the status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Check the response body
	expected := `{"status":"ok"}`
	if rr.Body.String() != expected {
		t.Errorf("Handler returned unexpected body: got %v want %v", rr.Body.String(), expected)
	}
}
