package middleware

import (
	"crypto/rand"
	"encoding/base64"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// Session represents an authenticated session
type Session struct {
	Username  string
	Role      string
	ExpiresAt time.Time
}

// SessionStore manages active sessions
type SessionStore struct {
	sessions map[string]*Session
	mu       sync.RWMutex
}

var globalSessionStore = &SessionStore{
	sessions: make(map[string]*Session),
}

// GetSessionStore returns the global session store
func GetSessionStore() *SessionStore {
	return globalSessionStore
}

// CleanupExpiredSessions removes expired sessions periodically
func init() {
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			globalSessionStore.Cleanup()
		}
	}()
}

// GenerateSessionID generates a random session ID
func GenerateSessionID() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

// SetSession creates a new session
func (s *SessionStore) SetSession(sessionID, username, role string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[sessionID] = &Session{
		Username:  username,
		Role:      role,
		ExpiresAt: time.Now().Add(24 * time.Hour), // 24 hour expiration
	}
}

// GetSession retrieves a session by ID
func (s *SessionStore) GetSession(sessionID string) (*Session, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	session, exists := s.sessions[sessionID]
	if !exists {
		return nil, false
	}
	if time.Now().After(session.ExpiresAt) {
		return nil, false
	}
	return session, true
}

// DeleteSession removes a session
func (s *SessionStore) DeleteSession(sessionID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, sessionID)
}

// Cleanup removes expired sessions
func (s *SessionStore) Cleanup() {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	for id, session := range s.sessions {
		if now.After(session.ExpiresAt) {
			delete(s.sessions, id)
		}
	}
}

// ClearSessionsForTest drops all sessions (used by integration tests in other packages).
func ClearSessionsForTest() {
	globalSessionStore.mu.Lock()
	defer globalSessionStore.mu.Unlock()
	globalSessionStore.sessions = make(map[string]*Session)
}

// AuthMiddleware handles session-based authentication
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip authentication for health check and auth endpoints
		path := c.Request.URL.Path
		if isPublicEndpoint(path) {
			c.Next()
			return
		}

		// Check for session cookie
		sessionID, err := c.Cookie("session_id")
		if err != nil || sessionID == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "Authentication required",
				"message": "Please log in to access this resource",
			})
			c.Abort()
			return
		}

		// Validate session
		session, exists := globalSessionStore.GetSession(sessionID)
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "Invalid session",
				"message": "Your session has expired. Please log in again",
			})
			c.Abort()
			return
		}

		// Store user information in context
		c.Set("username", session.Username)
		c.Set("role", session.Role)
		c.Set("authenticated", true)

		log.Printf("[INFO] Authenticated user: %s (%s) accessing: %s %s",
			session.Username, session.Role, c.Request.Method, path)

		c.Next()
	}
}

// isPublicEndpoint checks if the path is a public endpoint that should skip auth
func isPublicEndpoint(path string) bool {
	publicEndpoints := []string{
		"/ping",
		"/health",
		"/ready",
		"/healthz",
		"/livez",
		"/api/auth/login",
		"/api/auth/logout",
		"/api/auth/check",
	}

	for _, endpoint := range publicEndpoints {
		if path == endpoint {
			return true
		}
	}

	return false
}

// GetCurrentUser retrieves the current authenticated user from the Gin context
func GetCurrentUser(c *gin.Context) (string, bool) {
	username, exists := c.Get("username")
	if !exists {
		return "", false
	}
	usernameStr, ok := username.(string)
	return usernameStr, ok
}

// GetCurrentRole returns the role from context (admin, editor, viewer).
func GetCurrentRole(c *gin.Context) (string, bool) {
	role, exists := c.Get("role")
	if !exists {
		return "", false
	}
	s, ok := role.(string)
	return s, ok
}

func canWrite(role string) bool {
	return role == "admin" || role == "editor"
}

// WriteAccessMiddleware allows GET/HEAD/OPTIONS for all authenticated users; other methods require editor or admin.
func WriteAccessMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		m := c.Request.Method
		if m == http.MethodGet || m == http.MethodHead || m == http.MethodOptions {
			c.Next()
			return
		}
		role, _ := GetCurrentRole(c)
		if !canWrite(role) {
			c.JSON(http.StatusForbidden, gin.H{
				"error":   "Forbidden",
				"message": "Your role does not allow modifying resources",
			})
			c.Abort()
			return
		}
		c.Next()
	}
}

// AdminMiddleware requires role admin.
func AdminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, _ := GetCurrentRole(c)
		if role != "admin" {
			c.JSON(http.StatusForbidden, gin.H{
				"error":   "Forbidden",
				"message": "Administrator access required",
			})
			c.Abort()
			return
		}
		c.Next()
	}
}

// IsAuthenticated checks if the user is authenticated
func IsAuthenticated(c *gin.Context) bool {
	authenticated, exists := c.Get("authenticated")
	if !exists {
		return false
	}
	return authenticated.(bool)
}