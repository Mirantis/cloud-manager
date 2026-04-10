package middleware

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/rusik69/aws-iam-manager/internal/db"

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
	if _, err := rand.Read(b); err != nil {
		panic("crypto/rand: " + err.Error())
	}
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

// HashAPIToken returns hex-encoded SHA-256 of the raw token (matches stored api_tokens.token_hash).
func HashAPIToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

// AuthMiddleware handles session-based and Bearer API token authentication.
func AuthMiddleware(appDB *db.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip authentication for health check and auth endpoints
		path := c.Request.URL.Path
		if isPublicEndpoint(path) {
			c.Next()
			return
		}

		// Session cookie (web UI)
		if sessionID, err := c.Cookie("session_id"); err == nil && sessionID != "" {
			if session, ok := globalSessionStore.GetSession(sessionID); ok {
				c.Set("username", session.Username)
				c.Set("role", session.Role)
				c.Set("authenticated", true)
				log.Printf("[INFO] Authenticated user: %s (%s) accessing: %s %s",
					session.Username, session.Role, c.Request.Method, path)
				c.Next()
				return
			}
		}

		// Bearer API token
		auth := strings.TrimSpace(c.GetHeader("Authorization"))
		if appDB != nil && strings.HasPrefix(strings.ToLower(auth), "bearer ") {
			raw := strings.TrimSpace(auth[7:])
			if raw != "" {
				user, err := appDB.ValidateToken(HashAPIToken(raw))
				if err == nil {
					c.Set("username", user.Username)
					c.Set("role", user.Role)
					c.Set("authenticated", true)
					log.Printf("[INFO] Authenticated user (API token): %s (%s) accessing: %s %s",
						user.Username, user.Role, c.Request.Method, path)
					c.Next()
					return
				}
				if err != sql.ErrNoRows {
					log.Printf("[WARN] API token validation error: %v", err)
				}
			}
		}

		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "Authentication required",
			"message": "Please log in or provide a valid API token",
		})
		c.Abort()
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
// Per-user API token CRUD under /api/auth/tokens is allowed for any authenticated role (including viewer).
func WriteAccessMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		m := c.Request.Method
		if m == http.MethodGet || m == http.MethodHead || m == http.MethodOptions {
			c.Next()
			return
		}
		if strings.HasPrefix(c.Request.URL.Path, "/api/auth/tokens") {
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