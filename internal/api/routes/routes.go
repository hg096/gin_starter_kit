package routes

import (
	"gin_starter/internal/config"
	"gin_starter/internal/domain/admin"
	"gin_starter/internal/domain/blog"
	"gin_starter/internal/domain/user"
	"gin_starter/pkg/db/database"
	"gin_starter/pkg/logger"
	"gin_starter/pkg/middleware"
	"net/http"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

type adminPageViewRoute struct {
	RouteSpec      *admin.AdminPageRouteSpec
	RelativePath   string
	PermissionCode string
	Handler        gin.HandlerFunc
}

// SetupRoutes registers all HTTP routes.
func SetupRoutes(r *gin.Engine, db *database.DB, cfg *config.Config) {
	r.Use(middleware.CORSMiddleware(cfg))
	r.Use(middleware.LoggerMiddleware())
	r.Use(gin.Recovery())

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	r.GET("/health/live", liveCheckHandler())
	r.GET("/health/ready", readyCheckHandler(db))
	r.GET("/health", readyCheckHandler(db))

	setupAdminPageRoutes(r, db, cfg)

	api := r.Group("/api")
	{
		setupUserRoutes(api, db, cfg)
		setupBlogRoutes(api, db, cfg)
		setupAdminRoutes(api, db, cfg)
	}
}

func setupUserRoutes(rg *gin.RouterGroup, db *database.DB, cfg *config.Config) {
	repo := user.NewRepository(db)
	service := user.NewService(repo, cfg)
	handler := user.NewHandler(service, cfg)

	userGroup := rg.Group("/user")
	{
		userGroup.POST("/register", handler.Register)
		userGroup.POST("/login", handler.Login)
		userGroup.POST("/refresh", handler.RefreshToken)

		auth := userGroup.Group("")
		auth.Use(middleware.AuthMiddleware(cfg))
		{
			auth.GET("/profile", handler.GetProfile)
			auth.PUT("/profile", handler.UpdateProfile)
			auth.POST("/logout", handler.Logout)
		}
	}
}

func setupBlogRoutes(rg *gin.RouterGroup, db *database.DB, cfg *config.Config) {
	repo := blog.NewRepository(db)
	service := blog.NewService(repo)
	handler := blog.NewHandler(service)

	blogGroup := rg.Group("/blog")
	{
		blogGroup.GET("", handler.List)
		blogGroup.GET("/:id", handler.Get)
		blogGroup.GET("/author/:author_id", handler.ListByAuthor)

		auth := blogGroup.Group("")
		auth.Use(middleware.AuthMiddleware(cfg))
		{
			auth.POST("", handler.Create)
			auth.PUT("/:id", handler.Update)
			auth.DELETE("/:id", handler.Delete)
		}
	}
}

func setupAdminPageRoutes(r *gin.Engine, db *database.DB, cfg *config.Config) {
	var pageService admin.Service
	if db != nil && db.DB != nil {
		userRepo := user.NewRepository(db)
		pageService = admin.NewService(userRepo, db)
	}
	pageHandler := admin.NewPageHandler(cfg, pageService)
	viewRoutes := adminPageViewRoutes(pageHandler)

	if pageService != nil {
		routeSpecs := make([]admin.AdminPageRouteSpec, 0, len(viewRoutes))
		for _, route := range viewRoutes {
			if route.RouteSpec == nil {
				continue
			}
			routeSpecs = append(routeSpecs, *route.RouteSpec)
		}
		if err := pageService.SyncAdminPagesFromRouteSpecs(routeSpecs); err != nil {
			logger.Warn("failed to sync admin pages from routes: %v", err)
		} else if len(routeSpecs) > 0 {
			logger.Info("admin route pages synced: %d", len(routeSpecs))
		}
	}

	adminPage := r.Group("/admin")
	adminPage.GET("/login", pageHandler.LoginPage)

	adminProtected := adminPage.Group("")
	adminProtected.Use(middleware.AdminPageAuthMiddleware(cfg))
	{
		adminProtected.GET("", pageHandler.HomePage)
		adminProtected.GET("/", pageHandler.HomePage)
		for _, route := range viewRoutes {
			adminProtected.GET(route.RelativePath, middleware.AdminPagePermissionMiddleware(route.PermissionCode), route.Handler)
		}
		adminProtected.GET("/p/:page_key", middleware.AdminPageDynamicPermissionMiddleware(), pageHandler.DynamicPage)
		adminProtected.GET("/logout", pageHandler.LogoutPage)
	}
}

// adminPageViewRoutes 항목만 추가하면, 라우트와 페이지 레지스트리가 함께 반영된다.
func adminPageViewRoutes(pageHandler *admin.PageHandler) []adminPageViewRoute {
	return []adminPageViewRoute{
		{
			RouteSpec: &admin.AdminPageRouteSpec{
				PageKey:      "dashboard",
				Path:         "/admin/dashboard",
				Title:        "대시보드",
				Description:  "운영 지표 대시보드",
				GroupKey:     "core",
				GroupLabel:   "운영",
				GroupOrder:   10,
				VisibleRoles: []string{"TA", "A", "M", "G"},
				Icon:         "D",
				SortOrder:    10,
			},
			RelativePath:   "/dashboard",
			PermissionCode: "admin.stats.read",
			Handler:        pageHandler.DashboardPage,
		},
		{
			RouteSpec: &admin.AdminPageRouteSpec{
				PageKey:      "users",
				Path:         "/admin/users",
				Title:        "사용자 관리",
				Description:  "사용자/권한 관리",
				GroupKey:     "core",
				GroupLabel:   "운영",
				GroupOrder:   10,
				VisibleRoles: []string{"TA", "A"},
				Icon:         "U",
				SortOrder:    20,
			},
			RelativePath:   "/users",
			PermissionCode: "admin.account.read",
			Handler:        pageHandler.UsersPage,
		},
		{
			RouteSpec: &admin.AdminPageRouteSpec{
				PageKey:      "blogs",
				Path:         "/admin/blogs",
				Title:        "블로그 관리",
				Description:  "게시글 생성/수정/삭제 및 작성자 관리",
				GroupKey:     "content",
				GroupLabel:   "콘텐츠",
				GroupOrder:   20,
				VisibleRoles: []string{"TA", "A", "M"},
				Icon:         "B",
				SortOrder:    30,
			},
			RelativePath:   "/blogs",
			PermissionCode: "admin.page.blogs.read",
			Handler:        pageHandler.BlogsPage,
		},
		{
			RouteSpec: &admin.AdminPageRouteSpec{
				PageKey:      "admin_chat",
				Path:         "/admin/chat",
				Title:        "관리자 채팅",
				Description:  "관리자 전용 실시간 채팅",
				GroupKey:     "communication",
				GroupLabel:   "커뮤니케이션",
				GroupOrder:   30,
				VisibleRoles: []string{"TA", "A", "M", "G"},
				Icon:         "C",
				SortOrder:    40,
			},
			RelativePath:   "/chat",
			PermissionCode: "admin.page.admin_chat.read",
			Handler:        pageHandler.ChatPage,
		},
	}
}

func setupAdminRoutes(rg *gin.RouterGroup, db *database.DB, cfg *config.Config) {
	if db == nil || db.DB == nil {
		logger.Warn("skip admin api routes: database unavailable")
		return
	}

	userRepo := user.NewRepository(db)
	service := admin.NewService(userRepo, db)
	handler := admin.NewHandler(service)

	rg.GET("/admin/bootstrap/status", handler.GetBootstrapStatus)
	rg.POST("/admin/bootstrap/super-admin", handler.BootstrapSuperAdmin)

	adminGroup := rg.Group("/admin")
	adminGroup.Use(middleware.AuthMiddleware(cfg))
	adminGroup.Use(middleware.RequireUserTypes("TA", "A", "M", "G", "AG"))
	{
		adminGroup.GET("/stats", middleware.RequirePermission("admin.stats.read"), handler.GetStats)
		adminGroup.GET("/blogs", middleware.RequirePermission("admin.page.blogs.read"), handler.GetBlogs)
		adminGroup.POST("/blogs", middleware.RequirePermission("admin.page.blogs.create"), handler.CreateBlog)
		adminGroup.PUT("/blogs/:id", middleware.RequirePermission("admin.page.blogs.update"), handler.UpdateBlog)
		adminGroup.DELETE("/blogs/:id", middleware.RequirePermission("admin.page.blogs.delete"), handler.DeleteBlog)

		adminGroup.GET("/users", middleware.RequirePermission("admin.account.read"), handler.GetUsers)
		adminGroup.GET("/users/:id", middleware.RequirePermission("admin.account.read"), handler.GetUser)
		adminGroup.PUT("/users/:id/profile", middleware.RequirePermission("admin.account.profile.update"), handler.UpdateUserProfile)
		adminGroup.PUT("/users/:id/status", middleware.RequirePermission("admin.account.status.update"), handler.UpdateUserStatus)
		adminGroup.POST("/users/:id/password-reset", middleware.RequirePermission("admin.account.password.reset"), handler.ResetUserPassword)
		adminGroup.GET("/users/:id/permissions", middleware.RequirePermission("admin.account.read"), handler.GetUserPermissions)
		adminGroup.PUT("/users/:id/permissions", middleware.RequirePermission("admin.account.permission.manage"), handler.UpdateUserPermissions)
		adminGroup.PUT("/users/:id/auth", middleware.RequirePermission("admin.account.level.manage"), middleware.RequireSuperAdmin(), handler.UpdateUserAuth)
		adminGroup.DELETE("/users/:id", middleware.RequirePermission("admin.account.delete"), middleware.RequireSuperAdmin(), handler.DeleteUser)

		adminGroup.GET("/permissions", middleware.RequirePermission("admin.account.read"), handler.GetPermissions)
		adminGroup.GET("/permissions/delegable", middleware.RequirePermission("admin.account.read"), handler.GetDelegablePermissions)
		adminGroup.PUT("/permissions/delegable", middleware.RequirePermission("admin.allowlist.manage"), middleware.RequireSuperAdmin(), handler.UpdateDelegablePermissions)
		adminGroup.GET("/level-policy", middleware.RequirePermission("admin.account.read"), handler.GetLevelPolicy)
		adminGroup.PUT("/level-policy", middleware.RequirePermission("admin.system.level_policy.manage"), middleware.RequireSuperAdmin(), handler.UpdateLevelPolicy)
		adminGroup.GET("/audit-logs", middleware.RequirePermission("admin.audit.read"), handler.GetAuditLogs)
		adminGroup.GET("/pages", middleware.RequirePermission("admin.page.manage"), handler.GetAdminPages)
		adminGroup.POST("/pages", middleware.RequirePermission("admin.page.manage"), handler.CreateAdminPage)
		adminGroup.PUT("/pages/:page_key", middleware.RequirePermission("admin.page.manage"), handler.UpdateAdminPage)
		adminGroup.DELETE("/pages/:page_key", middleware.RequirePermission("admin.page.manage"), handler.DeleteAdminPage)
	}
}

func readyCheckHandler(db *database.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		statusCode := http.StatusOK
		status := gin.H{
			"status":   "ok",
			"database": "disconnected",
		}

		if db != nil && db.HealthCheck() == nil {
			status["database"] = "connected"
		} else {
			status["status"] = "degraded"
			statusCode = http.StatusServiceUnavailable
		}

		c.JSON(statusCode, status)
	}
}

func liveCheckHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	}
}
