package admin

import (
	"gin_starter/internal/config"
	"gin_starter/internal/domain/blog"
	"gin_starter/internal/domain/user"
	"gin_starter/pkg/middleware"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

type stubAdminService struct{}

func (s *stubAdminService) GetAllUsers(page, limit int, userType string) (*AdminUserListResponse, error) {
	return &AdminUserListResponse{}, nil
}
func (s *stubAdminService) GetUserByID(id string) (*user.User, error) { return nil, nil }
func (s *stubAdminService) UpdateUserAuth(actor Actor, id string, authType string, authLevel int) error {
	return nil
}
func (s *stubAdminService) UpdateUserProfile(actor Actor, id string, req *AdminUpdateUserProfileRequest) error {
	return nil
}
func (s *stubAdminService) UpdateUserStatus(actor Actor, id string, status string) error {
	return nil
}
func (s *stubAdminService) ResetUserPassword(actor Actor, id string, newPassword string) error {
	return nil
}
func (s *stubAdminService) GetUserPermissions(id string) ([]string, error) { return nil, nil }
func (s *stubAdminService) UpdateUserPermissions(actor Actor, id string, permissionCodes []string) error {
	return nil
}
func (s *stubAdminService) GetPermissions() ([]Permission, error)      { return nil, nil }
func (s *stubAdminService) GetDelegablePermissions() ([]string, error) { return nil, nil }
func (s *stubAdminService) UpdateDelegablePermissions(actor Actor, permissionCodes []string) error {
	return nil
}
func (s *stubAdminService) GetLevelPolicy() (*AdminLevelPolicyResponse, error) {
	return &AdminLevelPolicyResponse{Enabled: true}, nil
}
func (s *stubAdminService) UpdateLevelPolicy(actor Actor, enabled bool) error { return nil }
func (s *stubAdminService) DeleteUser(actor Actor, id string) error           { return nil }
func (s *stubAdminService) GetStats() (*AdminStatsResponse, error) {
	return &AdminStatsResponse{TotalUsers: 1}, nil
}
func (s *stubAdminService) GetAuditLogs(page, limit int, action, actorID, targetID, dateFrom, dateTo string) (*AdminAuditLogListResponse, error) {
	return &AdminAuditLogListResponse{}, nil
}
func (s *stubAdminService) GetBlogs(page, limit int) (*AdminBlogListResponse, error) {
	return &AdminBlogListResponse{}, nil
}
func (s *stubAdminService) CreateBlog(actor Actor, req *AdminCreateBlogRequest) (*blog.Blog, error) {
	return &blog.Blog{ID: 1, Title: req.Title, Content: req.Content, AuthorID: actor.ID}, nil
}
func (s *stubAdminService) UpdateBlog(actor Actor, id int64, req *AdminUpdateBlogRequest) (*blog.Blog, error) {
	return &blog.Blog{ID: id, Title: req.Title, Content: req.Content, AuthorID: actor.ID}, nil
}
func (s *stubAdminService) DeleteBlog(actor Actor, id int64) error { return nil }
func (s *stubAdminService) SyncAdminPagesFromRouteSpecs(routeSpecs []AdminPageRouteSpec) error {
	return nil
}
func (s *stubAdminService) GetAdminPages(includeDisabled bool) ([]AdminPage, error) {
	return []AdminPage{}, nil
}
func (s *stubAdminService) GetAdminPageByKey(pageKey string) (*AdminPage, error) {
	return nil, nil
}
func (s *stubAdminService) CreateAdminPage(actor Actor, req *AdminCreatePageRequest) (*AdminPage, error) {
	return &AdminPage{Key: req.PageKey, Title: req.Title}, nil
}
func (s *stubAdminService) UpdateAdminPage(actor Actor, pageKey string, req *AdminUpdatePageRequest) (*AdminPage, error) {
	return &AdminPage{Key: pageKey}, nil
}
func (s *stubAdminService) DeleteAdminPage(actor Actor, pageKey string) error { return nil }
func (s *stubAdminService) GetBootstrapStatus() (*AdminBootstrapStatusResponse, error) {
	return &AdminBootstrapStatusResponse{CanBootstrap: true, Reason: "no_admin_account"}, nil
}
func (s *stubAdminService) BootstrapSuperAdmin(req *AdminBootstrapSuperAdminRequest, ip string) (*user.User, error) {
	return (&user.User{
		ID:        req.ID,
		Name:      req.Name,
		Email:     req.Email,
		AuthType:  "TA",
		AuthLevel: 0,
		Status:    user.UserStatusActive,
	}).ToPublic(), nil
}

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
	h := NewHandler(&stubAdminService{})

	r := gin.New()
	r.GET(
		"/api/admin/stats",
		middleware.AuthMiddleware(cfg),
		middleware.RequireUserType("A"),
		middleware.RequirePermission("admin.stats.read"),
		h.GetStats,
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
	h := NewHandler(&stubAdminService{})

	r := gin.New()
	r.GET(
		"/api/admin/audit-logs",
		middleware.AuthMiddleware(cfg),
		middleware.RequireUserType("A"),
		middleware.RequirePermission("admin.audit.read"),
		h.GetAuditLogs,
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
	h := NewHandler(&stubAdminService{})

	r := gin.New()
	r.GET("/api/admin/bootstrap/status", h.GetBootstrapStatus)
	r.POST("/api/admin/bootstrap/super-admin", h.BootstrapSuperAdmin)

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
