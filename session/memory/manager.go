// Simple in-memory session manager. Use for testing only.
package memory

import (
	"log"

	"github.com/google/uuid"
	"github.com/puttsk/go-mcp"
)

type SessionManager struct {
	Debug    bool
	Sessions map[string]mcp.McpSession
}

func NewSessionManager() *SessionManager {
	return &SessionManager{
		Sessions: make(map[string]mcp.McpSession),
	}
}

func (s *SessionManager) GetSession(sessionID string) (mcp.McpSession, bool) {
	sess, ok := s.Sessions[sessionID]
	return sess, ok
}

func (s *SessionManager) CreateSession() (mcp.McpSession, error) {
	// Generate a new UUID for the session ID
	sessID, err := uuid.NewRandom()
	if err != nil {
		return mcp.McpSession{}, err
	}

	sess := mcp.McpSession{
		SessionID: sessID.String(),
	}

	// Store the session in the map
	s.Sessions[sess.SessionID] = sess

	if s.Debug {
		log.Printf("Session: %#v", s.Sessions)
	}
	return sess, nil
}

func (s *SessionManager) SetSessionInitialized(session mcp.McpSession, init bool) (mcp.McpSession, error) {
	// Update the session in the map
	if _, ok := s.Sessions[session.SessionID]; !ok {
		// Session not found
		return session, mcp.ErrSessionNotFound
	}

	newSession := session
	newSession.Initialized = init

	s.Sessions[session.SessionID] = newSession

	if s.Debug {
		log.Printf("Update Session: %#v", s.Sessions)
	}
	return newSession, nil
}
