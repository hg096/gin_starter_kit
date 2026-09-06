package admin

import (
	"gin_starter/internal/domain/blog"
	"gin_starter/internal/domain/user"
	"gin_starter/pkg/authz"
	"gin_starter/pkg/db/database"
	"gin_starter/pkg/logger"

	"github.com/gin-gonic/gin"
)

// Actor contains authenticated admin context.
type Actor struct {
	ID                 string
	UserType           string
	UserLevel          int
	IsSuperAdmin       bool
	LevelPolicyEnabled bool
	IP                 string
}

// Service is the admin domain service interface.
type Service interface {
	GetAllUsers(page, limit int, userType string) (*AdminUserListResponse, error)
	GetUserByID(id string) (*user.User, error)
	UpdateUserAuth(actor Actor, id string, authType string, authLevel int) error
	UpdateUserProfile(actor Actor, id string, req *AdminUpdateUserProfileRequest) error
	UpdateUserStatus(actor Actor, id string, status string) error
	ResetUserPassword(actor Actor, id string, newPassword string) error
	GetUserPermissions(id string) ([]string, error)
	UpdateUserPermissions(actor Actor, id string, permissionCodes []string) error
	GetPermissions() ([]Permission, error)
	GetDelegablePermissions() ([]string, error)
	UpdateDelegablePermissions(actor Actor, permissionCodes []string) error
	GetLevelPolicy() (*AdminLevelPolicyResponse, error)
	UpdateLevelPolicy(actor Actor, enabled bool) error
	DeleteUser(actor Actor, id string) error
	GetStats() (*AdminStatsResponse, error)
	GetAuditLogs(page, limit int, action, actorID, targetID, dateFrom, dateTo string) (*AdminAuditLogListResponse, error)
	GetBlogs(page, limit int) (*AdminBlogListResponse, error)
	CreateBlog(actor Actor, req *AdminCreateBlogRequest) (*blog.Blog, error)
	UpdateBlog(actor Actor, id int64, req *AdminUpdateBlogRequest) (*blog.Blog, error)
	DeleteBlog(actor Actor, id int64) error
	SyncAdminPagesFromRouteSpecs(routeSpecs []AdminPageRouteSpec) error
	GetAdminPages(includeDisabled bool) ([]AdminPage, error)
	GetAdminPageByKey(pageKey string) (*AdminPage, error)
	CreateAdminPage(actor Actor, req *AdminCreatePageRequest) (*AdminPage, error)
	UpdateAdminPage(actor Actor, pageKey string, req *AdminUpdatePageRequest) (*AdminPage, error)
	DeleteAdminPage(actor Actor, pageKey string) error
	GetBootstrapStatus() (*AdminBootstrapStatusResponse, error)
	BootstrapSuperAdmin(req *AdminBootstrapSuperAdminRequest, ip string) (*user.User, error)
}

type service struct {
	userRepo       user.Repository
	blogRepo       blog.Repository
	permissionRepo PermissionRepository
	pageRepo       AdminPageRepository
	db             *database.DB
}

func NewService(userRepo user.Repository, db *database.DB) Service {
	var permissionRepo PermissionRepository
	var pageRepo AdminPageRepository
	if db != nil && db.DB != nil {
		permissionRepo = NewPermissionRepository(db)
		pageRepo, _ = permissionRepo.(AdminPageRepository)
	}
	blogRepo := blog.NewRepository(db)
	if pageRepo != nil {
		if err := pageRepo.EnsureAdminPageSchema(); err != nil {
			logger.Warn("failed to ensure admin page schema: %v", err)
		}
	}

	return &service{
		userRepo:       userRepo,
		blogRepo:       blogRepo,
		permissionRepo: permissionRepo,
		pageRepo:       pageRepo,
		db:             db,
	}
}

// Handler handles admin HTTP endpoints.
type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

func actorFromContext(c *gin.Context) (Actor, bool) {
	userID := c.GetString("user_id")
	userType := authz.NormalizeAuthType(c.GetString("user_type"))
	userLevel := c.GetInt("user_level")
	isSuperAdminVal, exists := c.Get("is_super_admin")
	if userID == "" || userType == "" || !exists {
		return Actor{}, false
	}
	isSuperAdmin, ok := isSuperAdminVal.(bool)
	if !ok {
		isSuperAdmin = authz.IsSuperAdmin(userType, userLevel)
	}

	levelPolicyEnabled := true
	if value, hasValue := c.Get("level_policy_enabled"); hasValue {
		if casted, castOK := value.(bool); castOK {
			levelPolicyEnabled = casted
		}
	}

	return Actor{
		ID:                 userID,
		UserType:           userType,
		UserLevel:          userLevel,
		IsSuperAdmin:       isSuperAdmin,
		LevelPolicyEnabled: levelPolicyEnabled,
		IP:                 c.ClientIP(),
	}, true
}
