package routes

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"gin_starter/routes/api"
	"gin_starter/routes/out"
)

// SetupRoutes 함수는 전달받은 gin.Engine에 모든 라우트를 등록
func SetupRoutes(r *gin.Engine) {

	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "home"})
	})

	apiGroup := r.Group("/api")
	{
		// /api/user 라우트 등록
		api.SetupUserRoutes(apiGroup)

		// /api/blog 라우트 등록
		api.SetupBlogRoutes(apiGroup)

		// apiGroup.Use(auth.JWTAuthMiddleware(0))
		// {

		// }
	}

	outGroup := r.Group("/out")
	{
		out.SetupOutRoutes(outGroup)
	}

}
