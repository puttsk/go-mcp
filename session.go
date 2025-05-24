package mcp

import (
	"context"
	"fmt"
)

type McpSessionManager interface {
	// CreateSession creates a new session and returns it.
	CreateSession() (McpSession, error)

	// GetSession retrieves a session by its ID.
	GetSession(sessionID string) (McpSession, bool)

	// SetSessionInitialized sets the initialized state of a session.
	SetSessionInitialized(session McpSession, init bool) (McpSession, error)
}

// McpSession represents a session in the MCP protocol.
// It is immutable and should not return a pointer to itself.
// Any changes to the session should be done through the session manager.
type McpSession struct {
	SessionID   string // Session ID
	Initialized bool   // Session initialized
}

type MpcContextKey string

const McpRequestIDKey MpcContextKey = "mcp_request_id"
const McpSessionContextKey MpcContextKey = "mcp_session"

// GetSessionFromContext retrieves the session from the context.
func GetSessionFromContext(ctx context.Context) (McpSession, error) {
	sess, ok := ctx.Value(McpSessionContextKey).(McpSession)
	if !ok {
		return McpSession{}, fmt.Errorf("session not found")
	}
	return sess, nil
}

func SetSessionInContext(ctx context.Context, session McpSession) context.Context {
	return context.WithValue(ctx, McpSessionContextKey, session)
}

// GetRequestIDFromContext retrieves the request ID from the context.
func GetRequestIDFromContext(ctx context.Context) (string, error) {
	sess, ok := ctx.Value(McpRequestIDKey).(string)
	if !ok {
		return "", fmt.Errorf("request ID not found")
	}
	return sess, nil
}

func SetRequestIDInContext(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, McpRequestIDKey, requestID)
}
