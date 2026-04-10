package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func clearAllSessions(t *testing.T) {
	t.Helper()
	globalSessionStore.mu.Lock()
	globalSessionStore.sessions = make(map[string]*Session)
	globalSessionStore.mu.Unlock()
}

func TestWriteAccessMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		name       string
		role       string
		method     string
		wantStatus int
	}{
		{"viewer GET", "viewer", http.MethodGet, http.StatusOK},
		{"viewer HEAD", "viewer", http.MethodHead, http.StatusOK},
		{"viewer OPTIONS", "viewer", http.MethodOptions, http.StatusOK},
		{"viewer POST", "viewer", http.MethodPost, http.StatusForbidden},
		{"viewer DELETE", "viewer", http.MethodDelete, http.StatusForbidden},
		{"editor POST", "editor", http.MethodPost, http.StatusOK},
		{"admin PUT", "admin", http.MethodPut, http.StatusOK},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearAllSessions(t)
			r := gin.New()
			r.Use(func(c *gin.Context) {
				c.Set("role", tt.role)
				c.Next()
			})
			r.Use(WriteAccessMiddleware())
			r.Handle(tt.method, "/x", func(c *gin.Context) { c.Status(http.StatusOK) })

			w := httptest.NewRecorder()
			req := httptest.NewRequest(tt.method, "/x", nil)
			r.ServeHTTP(w, req)
			assert.Equal(t, tt.wantStatus, w.Code)
		})
	}
}

func TestAdminMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tc := range []struct {
		role string
		want int
	}{
		{"admin", http.StatusOK},
		{"editor", http.StatusForbidden},
		{"viewer", http.StatusForbidden},
		{"", http.StatusForbidden},
	} {
		t.Run(tc.role, func(t *testing.T) {
			clearAllSessions(t)
			r := gin.New()
			r.Use(func(c *gin.Context) {
				c.Set("role", tc.role)
				c.Next()
			})
			r.Use(AdminMiddleware())
			r.GET("/admin", func(c *gin.Context) { c.Status(http.StatusOK) })

			w := httptest.NewRecorder()
			r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/admin", nil))
			assert.Equal(t, tc.want, w.Code)
		})
	}
}

func TestAuthMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	clearAllSessions(t)

	r := gin.New()
	r.Use(AuthMiddleware(nil))
	r.GET("/api/accounts", func(c *gin.Context) {
		u, _ := GetCurrentUser(c)
		role, _ := GetCurrentRole(c)
		c.JSON(http.StatusOK, gin.H{"user": u, "role": role})
	})

	t.Run("no cookie", func(t *testing.T) {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/accounts", nil))
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("invalid session", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/accounts", nil)
		req.Header.Set("Cookie", "session_id=not-a-real-session")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("valid session", func(t *testing.T) {
		sid := GenerateSessionID()
		GetSessionStore().SetSession(sid, "tester", "editor")
		t.Cleanup(func() { clearAllSessions(t) })

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/accounts", nil)
		req.Header.Set("Cookie", "session_id="+sid)
		r.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("public check skips auth", func(t *testing.T) {
		r2 := gin.New()
		r2.Use(AuthMiddleware(nil))
		r2.GET("/api/auth/check", func(c *gin.Context) { c.Status(http.StatusOK) })
		w := httptest.NewRecorder()
		r2.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/auth/check", nil))
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestSessionStoreExpiry(t *testing.T) {
	t.Cleanup(func() {
		globalSessionStore.mu.Lock()
		globalSessionStore.sessions = make(map[string]*Session)
		globalSessionStore.mu.Unlock()
	})
	clearAllSessions(t)
	s := GetSessionStore()
	id := "fixed-id"
	s.mu.Lock()
	s.sessions[id] = &Session{
		Username:  "u",
		Role:      "viewer",
		ExpiresAt: time.Now().Add(-1 * time.Hour),
	}
	s.mu.Unlock()

	_, ok := s.GetSession(id)
	assert.False(t, ok)
}

func TestClearSessionsForTest(t *testing.T) {
	ClearSessionsForTest()
	sid := GenerateSessionID()
	GetSessionStore().SetSession(sid, "u", "admin")
	ClearSessionsForTest()
	_, ok := GetSessionStore().GetSession(sid)
	assert.False(t, ok)
}

func TestGetCurrentUserAndRoleHelpers(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	_, ok := GetCurrentUser(c)
	assert.False(t, ok)
	_, ok = GetCurrentRole(c)
	assert.False(t, ok)

	c.Set("username", "alice")
	c.Set("role", "viewer")
	c.Set("authenticated", true)
	u, ok := GetCurrentUser(c)
	assert.True(t, ok)
	assert.Equal(t, "alice", u)
	r, ok := GetCurrentRole(c)
	assert.True(t, ok)
	assert.Equal(t, "viewer", r)
	assert.True(t, IsAuthenticated(c))

	c2, _ := gin.CreateTestContext(httptest.NewRecorder())
	c2.Set("authenticated", true)
	assert.True(t, IsAuthenticated(c2))
}
