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

type deps struct {
	userRepo       user.Repository
	blogRepo       blog.Repository
	permissionRepo PermissionRepository
	pageRepo       AdminPageRepository
	db             *database.DB
}

func NewDeps(userRepo user.Repository, db *database.DB) *deps {
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

	return &deps{
		userRepo:       userRepo,
		blogRepo:       blogRepo,
		permissionRepo: permissionRepo,
		pageRepo:       pageRepo,
		db:             db,
	}
}

// Handler handles admin HTTP endpoints.
type Handler struct {
	deps *deps
}

func NewHandler(deps *deps) *Handler {
	return &Handler{deps: deps}
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
