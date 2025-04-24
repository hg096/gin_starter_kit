package api

import (
	"gin_starter/routes/handlers"
	"gin_starter/util/auth"

	"github.com/gin-gonic/gin"
)

func SetupUserRoutes(rg *gin.RouterGroup) {

	// Util := util.GetInstance()

	userGroup := rg.Group("/user")
	{

		userGroup.GET("/", func(c *gin.Context) { handlers.ApiUser(c) })

		userGroup.GET("/make", func(c *gin.Context) { handlers.ApiUserMake(c) })

		userGroup.GET("/makeUp", func(c *gin.Context) { handlers.ApiUserMakeUp(c) })

		userGroup.GET("/logIn", func(c *gin.Context) { handlers.ApiUserLogIn(c) })

		userGroup.GET("/logOut", func(c *gin.Context) { handlers.ApiUserLogOut(c) })

		userGroup.GET("/refresh", auth.RefreshHandler)

		// userGroup.Use(auth.JWTAuthMiddleware("U", 0))
		// {
		userGroup.GET("/profile", auth.JWTAuthMiddleware("U", 0), func(c *gin.Context) { handlers.ApiUserProfile(c) })
		// }

		// userGroup.GET("/:id", func(c *gin.Context) {
		// 	id := c.Param("id")
		// 	c.JSON(http.StatusOK, gin.H{"message": "User detail", "id": id})
		// })

	}
}
