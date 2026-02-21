package middleware

import (
	"encoding/json"
	"gin_starter/internal/config"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func testConfig() *config.Config {
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
		App: config.AppConfig{
			ServiceName: "test-service",
			Environment: "debug",
		},
	}
}

func mustToken(t *testing.T, cfg *config.Config, userID, userType string, level int) string {
	t.Helper()

	token, err := GenerateToken(
		userID,
		userType,
		level,
		cfg.JWT.AccessExpireMin,
		cfg.JWT.AccessSecret,
		cfg.JWT.TokenSecret,
		cfg.JWT.Issuer,
		cfg.JWT.Audience,
		cfg.JWT.Subject,
	)
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}
	return token
}

func TestGenerateToken_RandomNonceAndValidate(t *testing.T) {
	cfg := testConfig()

	token1 := mustToken(t, cfg, "admin01", "A", 9)
	token2 := mustToken(t, cfg, "admin01", "A", 9)

	if token1 == token2 {
		t.Fatalf("expected different tokens for same payload, got identical tokens")
	}

	claims1, err := ValidateToken(token1, cfg.JWT.AccessSecret, cfg.JWT.TokenSecret, cfg.JWT.Issuer, cfg.JWT.Audience, cfg.JWT.Subject)
	if err != nil {
		t.Fatalf("ValidateToken token1 failed: %v", err)
	}
	if claims1.UserID != "admin01" || claims1.UserType != "A" || claims1.UserLevel != 9 {
		t.Fatalf("unexpected token1 claims: %+v", claims1)
	}

	claims2, err := ValidateToken(token2, cfg.JWT.AccessSecret, cfg.JWT.TokenSecret, cfg.JWT.Issuer, cfg.JWT.Audience, cfg.JWT.Subject)
	if err != nil {
		t.Fatalf("ValidateToken token2 failed: %v", err)
	}
	if claims2.UserID != "admin01" || claims2.UserType != "A" || claims2.UserLevel != 9 {
		t.Fatalf("unexpected token2 claims: %+v", claims2)
	}
}

func TestAuthMiddleware_AcceptsBearerAndCookie(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := testConfig()

	headerToken := mustToken(t, cfg, "header-user", "A", 5)
	cookieToken := mustToken(t, cfg, "cookie-user", "U", 1)

	r := gin.New()
	r.GET("/protected", AuthMiddleware(cfg), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"user_id":    c.GetString("user_id"),
			"user_type":  c.GetString("user_type"),
			"user_level": c.GetInt("user_level"),
		})
	})

	// Authorization header must take precedence over cookie.
	headReq := httptest.NewRequest(http.MethodGet, "/protected", nil)
	headReq.Header.Set("Authorization", "Bearer "+headerToken)
	headReq.AddCookie(&http.Cookie{Name: AccessTokenCookieName, Value: cookieToken})
	headRec := httptest.NewRecorder()
	r.ServeHTTP(headRec, headReq)

	if headRec.Code != http.StatusOK {
		t.Fatalf("expected 200 for header token, got %d", headRec.Code)
	}

	var headBody struct {
		UserID   string `json:"user_id"`
		UserType string `json:"user_type"`
	}
	if err := json.Unmarshal(headRec.Body.Bytes(), &headBody); err != nil {
		t.Fatalf("failed to decode header response: %v", err)
	}
	if headBody.UserID != "header-user" || headBody.UserType != "A" {
		t.Fatalf("unexpected header auth payload: %+v", headBody)
	}

	// Cookie fallback must work when Authorization header is absent.
	cookieReq := httptest.NewRequest(http.MethodGet, "/protected", nil)
	cookieReq.AddCookie(&http.Cookie{Name: AccessTokenCookieName, Value: cookieToken})
	cookieRec := httptest.NewRecorder()
	r.ServeHTTP(cookieRec, cookieReq)

	if cookieRec.Code != http.StatusOK {
		t.Fatalf("expected 200 for cookie token, got %d", cookieRec.Code)
	}

	var cookieBody struct {
		UserID   string `json:"user_id"`
		UserType string `json:"user_type"`
	}
	if err := json.Unmarshal(cookieRec.Body.Bytes(), &cookieBody); err != nil {
		t.Fatalf("failed to decode cookie response: %v", err)
	}
	if cookieBody.UserID != "cookie-user" || cookieBody.UserType != "U" {
		t.Fatalf("unexpected cookie auth payload: %+v", cookieBody)
	}
}

func TestAdminAuthMiddleware_PassAndBlock(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := testConfig()

	adminToken := mustToken(t, cfg, "admin-1", "A", 10)
	userToken := mustToken(t, cfg, "user-1", "U", 1)

	r := gin.New()
	r.GET("/api/admin/stats", AuthMiddleware(cfg), RequireUserType("A"), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	adminReq := httptest.NewRequest(http.MethodGet, "/api/admin/stats", nil)
	adminReq.Header.Set("Authorization", "Bearer "+adminToken)
	adminRec := httptest.NewRecorder()
	r.ServeHTTP(adminRec, adminReq)

	if adminRec.Code != http.StatusNoContent {
		t.Fatalf("expected admin request to pass with 204, got %d", adminRec.Code)
	}

	userReq := httptest.NewRequest(http.MethodGet, "/api/admin/stats", nil)
	userReq.Header.Set("Authorization", "Bearer "+userToken)
	userRec := httptest.NewRecorder()
	r.ServeHTTP(userRec, userReq)

	if userRec.Code != http.StatusForbidden {
		t.Fatalf("expected user request to be blocked with 403, got %d", userRec.Code)
	}
}

func TestValidateToken_RejectsMismatchedRegisteredClaims(t *testing.T) {
	cfg := testConfig()
	token := mustToken(t, cfg, "admin01", "A", 10)

	if _, err := ValidateToken(token, cfg.JWT.AccessSecret, cfg.JWT.TokenSecret, "wrong-issuer", cfg.JWT.Audience, cfg.JWT.Subject); err == nil {
		t.Fatalf("expected issuer mismatch to be rejected")
	}
	if _, err := ValidateToken(token, cfg.JWT.AccessSecret, cfg.JWT.TokenSecret, cfg.JWT.Issuer, "wrong-audience", cfg.JWT.Subject); err == nil {
		t.Fatalf("expected audience mismatch to be rejected")
	}
	if _, err := ValidateToken(token, cfg.JWT.AccessSecret, cfg.JWT.TokenSecret, cfg.JWT.Issuer, cfg.JWT.Audience, "wrong-subject"); err == nil {
		t.Fatalf("expected subject mismatch to be rejected")
	}
}

func TestAuthMiddleware_BlocksStaleToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := testConfig()
	token := mustToken(t, cfg, "stale-admin", "A", 10)

	restore := setAuthSnapshotLoaderForTest(func(userID string) (*authSnapshot, error) {
		v := time.Now().Add(time.Minute)
		return &authSnapshot{
			UserType:        "A",
			UserLevel:       10,
			Status:          "active",
			TokenValidAfter: &v,
			Permissions:     []string{"admin.stats.read"},
		}, nil
	})
	defer restore()

	r := gin.New()
	r.GET("/protected", AuthMiddleware(cfg), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected stale token to be blocked with 401, got %d", rec.Code)
	}
}

func TestRequireAuthLevel_GlobalLevelPolicyToggle(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	r.GET(
		"/bypass",
		func(c *gin.Context) {
			c.Set("level_policy_enabled", false)
			c.Set("user_level", 1)
			c.Next()
		},
		RequireAuthLevel(5),
		func(c *gin.Context) {
			c.Status(http.StatusNoContent)
		},
	)

	r.GET(
		"/enforced",
		func(c *gin.Context) {
			c.Set("level_policy_enabled", true)
			c.Set("user_level", 1)
			c.Next()
		},
		RequireAuthLevel(5),
		func(c *gin.Context) {
			c.Status(http.StatusNoContent)
		},
	)

	bypassReq := httptest.NewRequest(http.MethodGet, "/bypass", nil)
	bypassRec := httptest.NewRecorder()
	r.ServeHTTP(bypassRec, bypassReq)
	if bypassRec.Code != http.StatusNoContent {
		t.Fatalf("expected bypass route to return 204, got %d", bypassRec.Code)
	}

	enforcedReq := httptest.NewRequest(http.MethodGet, "/enforced", nil)
	enforcedRec := httptest.NewRecorder()
	r.ServeHTTP(enforcedRec, enforcedReq)
	if enforcedRec.Code != http.StatusForbidden {
		t.Fatalf("expected enforced route to return 403, got %d", enforcedRec.Code)
	}
}
