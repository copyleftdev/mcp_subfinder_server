// Package mcp implements Model Context Protocol (MCP) handling logic
package mcp

import (
	jsoniter "github.com/json-iterator/go"
)

// Protocol versions
const (
	// SupportedProtocolVersion is the MCP protocol version this server supports
	SupportedProtocolVersion = "0.3"
)

// Common JSON-RPC 2.0 structures
// =============================

// Request represents a JSON-RPC 2.0 request
type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      *jsoniter.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  jsoniter.RawMessage  `json:"params,omitempty"`
}

// Response represents a JSON-RPC 2.0 response
type Response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      *jsoniter.RawMessage `json:"id,omitempty"`
	Result  interface{}     `json:"result,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`
}

// RPCError represents a JSON-RPC 2.0 error
type RPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Standard JSON-RPC 2.0 error codes
const (
	// ParseErrorCode indicates the server received invalid JSON
	ParseErrorCode = -32700
	// InvalidRequestCode indicates the JSON sent is not a valid Request object
	InvalidRequestCode = -32600
	// MethodNotFoundCode indicates the method does not exist / is not available
	MethodNotFoundCode = -32601
	// InvalidParamsCode indicates invalid method parameter(s)
	InvalidParamsCode = -32602
	// InternalErrorCode indicates an internal JSON-RPC error
	InternalErrorCode = -32603
)

// Standard RPC error instances for reuse
var (
	// ErrParse is returned when invalid JSON was received by the server
	ErrParse = &RPCError{Code: ParseErrorCode, Message: "Parse error"}
	// ErrInvalidRequest is returned when the JSON sent is not a valid Request object
	ErrInvalidRequest = &RPCError{Code: InvalidRequestCode, Message: "Invalid Request"}
	// ErrMethodNotFound is returned when the method does not exist / is not available
	ErrMethodNotFound = &RPCError{Code: MethodNotFoundCode, Message: "Method not found"}
	// ErrInvalidParams is returned when invalid method parameter(s) were supplied
	ErrInvalidParams = &RPCError{Code: InvalidParamsCode, Message: "Invalid params"}
	// ErrInternal is returned when there was an internal JSON-RPC error
	ErrInternal = &RPCError{Code: InternalErrorCode, Message: "Internal error"}
)

// MCP-specific structures
// ======================

// InitializeParams represents parameters for initialize method
type InitializeParams struct {
	ProtocolVersion string `json:"protocolVersion"`
}

// InitializeResult represents the result of initialize method
type InitializeResult struct {
	Name            string `json:"name"`
	Version         string `json:"version"`
	ProtocolVersion string `json:"protocolVersion"`
}

// Tool represents a tool available via the MCP protocol
type Tool struct {
	Name            string      `json:"name"`
	Title           string      `json:"title"`
	Description     string      `json:"description"`
	InputSchema     interface{} `json:"inputSchema"`
	SupportsBinary  bool        `json:"supportsBinary,omitempty"`
	RequiresAPIKeys bool        `json:"requiresAPIKeys,omitempty"`
}

// ToolsListResult represents the result of the tools.list method
type ToolsListResult struct {
	Tools []Tool `json:"tools"`
}

// ToolCallParams represents parameters for tools.call method
type ToolCallParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
	Binary    []byte                 `json:"binary,omitempty"`
}

// ContentItem represents a text content item
type ContentItem struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// ResourceItem represents a resource content item
type ResourceItem struct {
	Type     string `json:"type"`
	MimeType string `json:"mimeType"`
	URI      string `json:"uri,omitempty"`
	Blob     string `json:"blob,omitempty"`
}

// ToolCallResult represents the result of a tools.call method
type ToolCallResult struct {
	Content []interface{} `json:"content"`
	IsError bool          `json:"isError,omitempty"`
}
