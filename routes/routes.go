// @title       Gin Starter API
// @version     1.0
// @description 예제용 Gin + Swagger 서버
// @BasePath    /
package routes

import (
	_ "gin_starter/docs" // docs 폴더 import (자동 생성된 문서)
	"net/http"

	"gin_starter/util"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"gin_starter/routes/api"
	"gin_starter/routes/out"
)

// SetupRoutes 함수는 전달받은 gin.Engine에 모든 라우트를 등록
func SetupRoutes(r *gin.Engine) {

	// /swagger/index.html
	r.GET("/swagger/*any", ginSwagger.WrapHandler(
		swaggerFiles.Handler,
		// ginSwagger.URL("/swagger/swagger.json"),
	))

	r.GET("/", func(c *gin.Context) {
		util.EndResponse(c, http.StatusOK, gin.H{"message": "home"}, "rest /")
	})

	apiGroup := r.Group("/api")
	{
		// /api/user 라우트 등록
		api.SetupUserRoutes(apiGroup)

		// /api/blog 라우트 등록
		api.SetupBlogRoutes(apiGroup)

		out.SetupOutRoutes(apiGroup)

		// apiGroup.Use(auth.JWTAuthMiddleware("U", 0))
		// {

		// }
	}

	outGroup := r.Group("/out")
	{
		out.SetupOutRoutes(outGroup)
	}

}
