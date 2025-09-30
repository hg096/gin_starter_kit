package routes

import (
	"gin_starter/internal/config"
	"gin_starter/internal/domain/admin"
	"gin_starter/internal/domain/blog"
	"gin_starter/internal/domain/user"
	"gin_starter/internal/infrastructure/database"
	"gin_starter/internal/middleware"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// SetupRoutes 모든 라우트 설정
func SetupRoutes(r *gin.Engine, db *database.DB, cfg *config.Config) {
	// 미들웨어 설정
	r.Use(middleware.CORSMiddleware())
	r.Use(middleware.LoggerMiddleware())
	r.Use(gin.Recovery())

	// Swagger
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Health check
	r.GET("/health", healthCheckHandler(db))

	// 관리자 페이지 라우트
	setupAdminPageRoutes(r)

	// API 라우트 그룹
	api := r.Group("/api")
	{
		// User 도메인
		setupUserRoutes(api, db, cfg)

		// Blog 도메인
		setupBlogRoutes(api, db, cfg)

		// Admin 도메인 (관리자 전용)
		setupAdminRoutes(api, db, cfg)
	}
}

// setupUserRoutes 사용자 관련 라우트
func setupUserRoutes(rg *gin.RouterGroup, db *database.DB, cfg *config.Config) {
	// 의존성 주입
	repo := user.NewRepository(db)
	service := user.NewService(repo, cfg)
	handler := user.NewHandler(service)

	userGroup := rg.Group("/user")
	{
		// 인증 불필요한 라우트
		userGroup.POST("/register", handler.Register)
		userGroup.POST("/login", handler.Login)
		userGroup.POST("/refresh", handler.RefreshToken)

		// 인증 필요한 라우트
		auth := userGroup.Group("")
		auth.Use(middleware.AuthMiddleware(cfg))
		{
			auth.GET("/profile", handler.GetProfile)
			auth.PUT("/profile", handler.UpdateProfile)
			auth.POST("/logout", handler.Logout)
		}
	}
}

// setupBlogRoutes 블로그 관련 라우트
func setupBlogRoutes(rg *gin.RouterGroup, db *database.DB, cfg *config.Config) {
	// 의존성 주입
	repo := blog.NewRepository(db)
	service := blog.NewService(repo)
	handler := blog.NewHandler(service)

	blogGroup := rg.Group("/blog")
	{
		// 공개 라우트
		blogGroup.GET("", handler.List)                           // 목록
		blogGroup.GET("/:id", handler.Get)                        // 상세
		blogGroup.GET("/author/:author_id", handler.ListByAuthor) // 작성자별 목록

		// 인증 필요한 라우트
		auth := blogGroup.Group("")
		auth.Use(middleware.AuthMiddleware(cfg))
		{
			auth.POST("", handler.Create)       // 생성
			auth.PUT("/:id", handler.Update)    // 수정
			auth.DELETE("/:id", handler.Delete) // 삭제
		}
	}
}

// setupAdminPageRoutes 관리자 페이지 라우트
func setupAdminPageRoutes(r *gin.Engine) {
	pageHandler := admin.NewPageHandler()

	adminPage := r.Group("/admin")
	{
		adminPage.GET("/login", pageHandler.LoginPage)
		adminPage.GET("", pageHandler.DashboardPage)
		adminPage.GET("/", pageHandler.DashboardPage)
		adminPage.GET("/users", pageHandler.UsersPage)
		adminPage.GET("/logout", pageHandler.LogoutPage)
	}
}

// setupAdminRoutes 관리자 API 라우트
func setupAdminRoutes(rg *gin.RouterGroup, db *database.DB, cfg *config.Config) {
	// 의존성 주입
	userRepo := user.NewRepository(db)
	service := admin.NewService(userRepo, db)
	handler := admin.NewHandler(service)

	// Admin 그룹 (인증 + 관리자 권한 필요)
	adminGroup := rg.Group("/admin")
	adminGroup.Use(middleware.AuthMiddleware(cfg))
	adminGroup.Use(middleware.RequireUserType("A")) // 관리자만
	{
		// 사용자 관리
		adminGroup.GET("/users", handler.GetUsers)                     // 목록
		adminGroup.GET("/users/:id", handler.GetUser)                  // 상세
		adminGroup.PUT("/users/:id/auth", handler.UpdateUserAuth)      // 권한 수정
		adminGroup.DELETE("/users/:id", handler.DeleteUser)            // 삭제

		// 통계
		adminGroup.GET("/stats", handler.GetStats)
	}
}

// healthCheckHandler 헬스 체크 핸들러
func healthCheckHandler(db *database.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		status := gin.H{
			"status":   "ok",
			"database": "disconnected",
		}

		if db != nil {
			if err := db.HealthCheck(); err == nil {
				status["database"] = "connected"
			}
		}

		c.JSON(200, status)
	}
}