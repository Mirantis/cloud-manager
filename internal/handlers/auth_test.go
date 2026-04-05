package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/rusik69/aws-iam-manager/internal/config"
	"github.com/rusik69/aws-iam-manager/internal/db"
	"github.com/rusik69/aws-iam-manager/internal/middleware"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

func setupAuthRouter(t *testing.T) *gin.Engine {
	t.Helper()
	middleware.ClearSessionsForTest()
	t.Cleanup(middleware.ClearSessionsForTest)

	gin.SetMode(gin.TestMode)
	r := gin.New()

	appDB, err := db.Open(filepath.Join(t.TempDir(), "auth.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = appDB.Close() })

	seed := []struct {
		name, pass, role string
	}{
		{"admin", "adminpass", "admin"},
		{"editor1", "edpass", "editor"},
		{"viewer1", "viewpass", "viewer"},
	}
	for _, u := range seed {
		h, err := bcrypt.GenerateFromPassword([]byte(u.pass), bcrypt.MinCost)
		require.NoError(t, err)
		require.NoError(t, appDB.UpsertUser(u.name, string(h), u.role))
	}

	h := NewHandler(&MockAWSService{}, config.Config{}, appDB)

	r.POST("/api/auth/login", h.Login)
	r.POST("/api/auth/logout", h.Logout)
	r.GET("/api/auth/check", h.CheckAuth)

	admin := r.Group("/api/admin")
	admin.Use(middleware.AuthMiddleware(), middleware.AdminMiddleware())
	admin.POST("/users", h.CreateAppUser)
	admin.GET("/users", h.ListAppUsers)
	admin.DELETE("/users/:username", h.DeleteAppUser)
	admin.PUT("/users/:username/password", h.UpdateAppUserPassword)

	api := r.Group("/api")
	api.Use(middleware.AuthMiddleware(), middleware.WriteAccessMiddleware())
	api.GET("/accounts", h.ListAccounts)
	api.DELETE("/accounts/:accountId/users/:username", h.DeleteUser)
	api.GET("/auth/user", h.GetCurrentUser)

	return r
}

func loginCookie(t *testing.T, r *gin.Engine, username, password string) string {
	t.Helper()
	body, err := json.Marshal(map[string]string{"username": username, "password": password})
	require.NoError(t, err)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	for _, c := range w.Result().Cookies() {
		if c.Name == "session_id" {
			return c.Value
		}
	}
	t.Fatal("no session_id cookie")
	return ""
}

func TestLoginFailures(t *testing.T) {
	r := setupAuthRouter(t)

	t.Run("wrong password", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{"username": "admin", "password": "nope"})
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("invalid json", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader([]byte(`{`)))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestLoginSuccessAndCheckAuth(t *testing.T) {
	r := setupAuthRouter(t)
	sid := loginCookie(t, r, "viewer1", "viewpass")

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/auth/check", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: sid})
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	var out map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &out))
	assert.Equal(t, true, out["authenticated"])
	assert.Equal(t, "viewer1", out["username"])
	assert.Equal(t, "viewer", out["role"])
}

func TestCheckAuthUnauthenticated(t *testing.T) {
	r := setupAuthRouter(t)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/auth/check", nil))
	assert.Equal(t, http.StatusOK, w.Code)
	var out map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &out))
	assert.Equal(t, false, out["authenticated"])
}

func TestLogoutClearsSession(t *testing.T) {
	r := setupAuthRouter(t)
	sid := loginCookie(t, r, "editor1", "edpass")

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: sid})
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/api/accounts", nil)
	req2.AddCookie(&http.Cookie{Name: "session_id", Value: sid})
	r.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusUnauthorized, w2.Code)
}

func TestWriteAccessViewerVsEditor(t *testing.T) {
	r := setupAuthRouter(t)

	vCookie := loginCookie(t, r, "viewer1", "viewpass")
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/accounts", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: vCookie})
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodDelete, "/api/accounts/123456789012/users/testuser1", nil)
	req2.AddCookie(&http.Cookie{Name: "session_id", Value: vCookie})
	r.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusForbidden, w2.Code)

	middleware.ClearSessionsForTest()
	eCookie := loginCookie(t, r, "editor1", "edpass")
	w3 := httptest.NewRecorder()
	req3 := httptest.NewRequest(http.MethodDelete, "/api/accounts/123456789012/users/testuser1", nil)
	req3.AddCookie(&http.Cookie{Name: "session_id", Value: eCookie})
	r.ServeHTTP(w3, req3)
	assert.Equal(t, http.StatusOK, w3.Code)
}

func TestGetCurrentUser(t *testing.T) {
	r := setupAuthRouter(t)
	sid := loginCookie(t, r, "admin", "adminpass")
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/auth/user", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: sid})
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	var out map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &out))
	assert.Equal(t, "admin", out["username"])
	assert.Equal(t, "admin", out["role"])
}

func TestAdminAPIRoleEnforcement(t *testing.T) {
	r := setupAuthRouter(t)

	v := loginCookie(t, r, "viewer1", "viewpass")
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/admin/users", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: v})
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusForbidden, w.Code)

	middleware.ClearSessionsForTest()
	e := loginCookie(t, r, "editor1", "edpass")
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/api/admin/users", nil)
	req2.AddCookie(&http.Cookie{Name: "session_id", Value: e})
	r.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusForbidden, w2.Code)

	middleware.ClearSessionsForTest()
	a := loginCookie(t, r, "admin", "adminpass")
	w3 := httptest.NewRecorder()
	req3 := httptest.NewRequest(http.MethodGet, "/api/admin/users", nil)
	req3.AddCookie(&http.Cookie{Name: "session_id", Value: a})
	r.ServeHTTP(w3, req3)
	assert.Equal(t, http.StatusOK, w3.Code)
}

func TestAdminUserCRUD(t *testing.T) {
	r := setupAuthRouter(t)
	a := loginCookie(t, r, "admin", "adminpass")

	body, _ := json.Marshal(map[string]string{
		"username": "newu", "password": "pw123", "role": "viewer",
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/admin/users", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "session_id", Value: a})
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)

	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodPost, "/api/admin/users", bytes.NewReader(body))
	req2.Header.Set("Content-Type", "application/json")
	req2.AddCookie(&http.Cookie{Name: "session_id", Value: a})
	r.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusConflict, w2.Code)

	badRole, _ := json.Marshal(map[string]string{
		"username": "bad", "password": "x", "role": "superuser",
	})
	w3 := httptest.NewRecorder()
	req3 := httptest.NewRequest(http.MethodPost, "/api/admin/users", bytes.NewReader(badRole))
	req3.Header.Set("Content-Type", "application/json")
	req3.AddCookie(&http.Cookie{Name: "session_id", Value: a})
	r.ServeHTTP(w3, req3)
	assert.Equal(t, http.StatusBadRequest, w3.Code)

	w4 := httptest.NewRecorder()
	req4 := httptest.NewRequest(http.MethodDelete, "/api/admin/users/admin", nil)
	req4.AddCookie(&http.Cookie{Name: "session_id", Value: a})
	r.ServeHTTP(w4, req4)
	assert.Equal(t, http.StatusBadRequest, w4.Code)

	w5 := httptest.NewRecorder()
	req5 := httptest.NewRequest(http.MethodDelete, "/api/admin/users/newu", nil)
	req5.AddCookie(&http.Cookie{Name: "session_id", Value: a})
	r.ServeHTTP(w5, req5)
	assert.Equal(t, http.StatusOK, w5.Code)

	w6 := httptest.NewRecorder()
	req6 := httptest.NewRequest(http.MethodDelete, "/api/admin/users/nobody", nil)
	req6.AddCookie(&http.Cookie{Name: "session_id", Value: a})
	r.ServeHTTP(w6, req6)
	assert.Equal(t, http.StatusNotFound, w6.Code)

	pwBody, _ := json.Marshal(map[string]string{"password": "reset99"})
	w7 := httptest.NewRecorder()
	req7 := httptest.NewRequest(http.MethodPut, "/api/admin/users/editor1/password", bytes.NewReader(pwBody))
	req7.Header.Set("Content-Type", "application/json")
	req7.AddCookie(&http.Cookie{Name: "session_id", Value: a})
	r.ServeHTTP(w7, req7)
	assert.Equal(t, http.StatusOK, w7.Code)

	_ = loginCookie(t, r, "editor1", "reset99")

	w8 := httptest.NewRecorder()
	req8 := httptest.NewRequest(http.MethodPut, "/api/admin/users/missing/password", bytes.NewReader(pwBody))
	req8.Header.Set("Content-Type", "application/json")
	req8.AddCookie(&http.Cookie{Name: "session_id", Value: a})
	r.ServeHTTP(w8, req8)
	assert.Equal(t, http.StatusNotFound, w8.Code)
}
