package mcp

import "context"

type McpTransportHandler interface {
	GetSessionID(ctx context.Context, request any) (string, error)

	// ProcessRequest transforms a transport-layer request into an MCP-formatted request.
	//
	// Note: All numeric values in the MCP request must use json.Number as their data type.
	ProcessRequest(ctx context.Context, request any) (*McpRequest, error)

	ProcessResponse(ctx context.Context, response *McpResponse) (any, error)
}
