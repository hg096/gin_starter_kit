package admin

import (
	"gin_starter/internal/config"
	"gin_starter/pkg/middleware"
	"gin_starter/pkg/response"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func adminTestConfig() *config.Config {
	return &config.Config{
		JWT: config.JWTConfig{
			AccessSecret:      []byte("01234567890123456789012345678901"),
			RefreshSecret:     []byte("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),
			TokenSecret:       []byte("bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"),
			AccessExpireMin:   30,
			RefreshExpireDays: 7,
			RefreshReuseHours: 24,
			Issuer:            "issuer-test",
			Audience:          "aud-test",
			Subject:           "subject-test",
		},
	}
}

func makeToken(t *testing.T, cfg *config.Config, level int) string {
	t.Helper()
	token, err := middleware.GenerateToken(
		"admin-test",
		"A",
		level,
		cfg.JWT.AccessExpireMin,
		cfg.JWT.AccessSecret,
		cfg.JWT.TokenSecret,
		cfg.JWT.Issuer,
		cfg.JWT.Audience,
		cfg.JWT.Subject,
	)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}
	return token
}

func TestAdminStatsPermissionMatrix(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := adminTestConfig()
	okHandler := func(c *gin.Context) {
		response.Success(c, gin.H{"ok": true})
	}

	r := gin.New()
	r.GET(
		"/api/admin/stats",
		middleware.AuthMiddleware(cfg),
		middleware.RequireUserType("A"),
		middleware.RequirePermission("admin.stats.read"),
		okHandler,
	)

	unauthReq := httptest.NewRequest(http.MethodGet, "/api/admin/stats", nil)
	unauthRec := httptest.NewRecorder()
	r.ServeHTTP(unauthRec, unauthReq)
	if unauthRec.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthenticated request to return 401, got %d", unauthRec.Code)
	}

	forbiddenReq := httptest.NewRequest(http.MethodGet, "/api/admin/stats", nil)
	forbiddenReq.Header.Set("Authorization", "Bearer "+makeToken(t, cfg, 5))
	forbiddenRec := httptest.NewRecorder()
	r.ServeHTTP(forbiddenRec, forbiddenReq)
	if forbiddenRec.Code != http.StatusForbidden {
		t.Fatalf("expected missing permission to return 403, got %d", forbiddenRec.Code)
	}

	okReq := httptest.NewRequest(http.MethodGet, "/api/admin/stats", nil)
	okReq.Header.Set("Authorization", "Bearer "+makeToken(t, cfg, 10))
	okRec := httptest.NewRecorder()
	r.ServeHTTP(okRec, okReq)
	if okRec.Code != http.StatusOK {
		t.Fatalf("expected super-admin to pass with 200, got %d", okRec.Code)
	}
}

func TestAdminAuditLogsPermissionMatrix(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := adminTestConfig()
	okHandler := func(c *gin.Context) {
		response.Success(c, gin.H{"ok": true})
	}

	r := gin.New()
	r.GET(
		"/api/admin/audit-logs",
		middleware.AuthMiddleware(cfg),
		middleware.RequireUserType("A"),
		middleware.RequirePermission("admin.audit.read"),
		okHandler,
	)

	unauthReq := httptest.NewRequest(http.MethodGet, "/api/admin/audit-logs", nil)
	unauthRec := httptest.NewRecorder()
	r.ServeHTTP(unauthRec, unauthReq)
	if unauthRec.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthenticated request to return 401, got %d", unauthRec.Code)
	}

	forbiddenReq := httptest.NewRequest(http.MethodGet, "/api/admin/audit-logs", nil)
	forbiddenReq.Header.Set("Authorization", "Bearer "+makeToken(t, cfg, 5))
	forbiddenRec := httptest.NewRecorder()
	r.ServeHTTP(forbiddenRec, forbiddenReq)
	if forbiddenRec.Code != http.StatusForbidden {
		t.Fatalf("expected missing permission to return 403, got %d", forbiddenRec.Code)
	}

	okReq := httptest.NewRequest(http.MethodGet, "/api/admin/audit-logs", nil)
	okReq.Header.Set("Authorization", "Bearer "+makeToken(t, cfg, 10))
	okRec := httptest.NewRecorder()
	r.ServeHTTP(okRec, okReq)
	if okRec.Code != http.StatusOK {
		t.Fatalf("expected super-admin to pass with 200, got %d", okRec.Code)
	}
}

func TestAdminBootstrapEndpointsArePublic(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.GET("/api/admin/bootstrap/status", func(c *gin.Context) {
		response.Success(c, AdminBootstrapStatusResponse{CanBootstrap: true, Reason: "no_admin_account"})
	})
	r.POST("/api/admin/bootstrap/super-admin", func(c *gin.Context) {
		response.Created(c, gin.H{"message": "top-admin bootstrap completed"})
	})

	statusReq := httptest.NewRequest(http.MethodGet, "/api/admin/bootstrap/status", nil)
	statusRec := httptest.NewRecorder()
	r.ServeHTTP(statusRec, statusReq)
	if statusRec.Code != http.StatusOK {
		t.Fatalf("expected bootstrap status to return 200, got %d", statusRec.Code)
	}

	body := `{"user_id":"rootadmin","user_pass":"secret123","user_name":"Root Admin","user_email":"root@example.com"}`
	createReq := httptest.NewRequest(http.MethodPost, "/api/admin/bootstrap/super-admin", strings.NewReader(body))
	createReq.Header.Set("Content-Type", "application/json")
	createRec := httptest.NewRecorder()
	r.ServeHTTP(createRec, createReq)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("expected bootstrap create to return 201, got %d", createRec.Code)
	}
}
