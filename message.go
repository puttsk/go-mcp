package mcp

import "encoding/json"

type McpRequest struct {
	JsonRPC string      `json:"jsonrpc"` // JSON-RPC version
	ID      json.Number `json:"id"`      // Request ID
	Method  string      `json:"method"`  // Method name
	Params  any         `json:"params"`  // Parameters
}

type McpResponse struct {
	JsonRPC JsonRPCVersion `json:"jsonrpc"`          // JSON-RPC version
	ID      json.Number    `json:"id"`               // Request ID
	Results any            `json:"result,omitempty"` // Parameters
	Error   *McpError      `json:"error,omitempty"`  // Error
}

type McpError struct {
	Code    int    `json:"code"`           // Error code
	Message string `json:"message"`        // Error message
	Data    any    `json:"data,omitempty"` // Error data
}

func (e McpError) Error() string {
	return e.Message
}

func NewMcpError(code int, message string, data any) *McpError {
	return &McpError{
		Code:    code,
		Message: message,
		Data:    data,
	}
}

var ErrNoSessionHeader = NewMcpError(ErrInvalidRequestCode, "no session header", nil)
var ErrSessionAlreadyInitialized = NewMcpError(ErrInvalidRequestCode, "session already initialized", nil)
var ErrSessionNotFound = NewMcpError(ErrInvalidRequestCode, "session not found", nil)

var ErrSessionNotInitialized = NewMcpError(ErrInvalidRequestCode, "session not initialized", nil)

var ErrInvalidMcpRequestParameters = NewMcpError(ErrInvalidParametersCode, "invalid params", nil)
var ErrInvalidToolArguments = NewMcpError(ErrInvalidParametersCode, "invalid tool arguments", nil)

func NewErrUnknownMethod(method string) *McpError {
	return &McpError{
		Code:    ErrMethodNotFoundCode,
		Message: "unknown method",
		Data: map[string]any{
			"method": method,
		},
	}
}

func NewErrInvalidArgumentType(name string, expected McpToolDataType) *McpError {
	return &McpError{
		Code:    ErrInvalidParametersCode,
		Message: "invalid argument type",
		Data: map[string]any{
			"argument": name,
			"type":     expected,
		},
	}
}

func NewErrInternalError(message string, data any) *McpError {
	return &McpError{
		Code:    ErrInternalErrorCode,
		Message: message,
		Data:    data,
	}
}

// Standard JSON-RPC errors
const ErrInvalidRequestCode = -32600
const ErrMethodNotFoundCode = -32601
const ErrInvalidParametersCode = -32602
const ErrInternalErrorCode = -32603
const ErrParseErrorCode = -32700
