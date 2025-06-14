package api

import (
	"gin_starter/util/utilCore"
	"net/http"

	"github.com/gin-gonic/gin"
)

func SetupBlogRoutes(rg *gin.RouterGroup) {

	blogGroup := rg.Group("/blog")
	{

		blogGroup.GET("/", func(c *gin.Context) {
			utilCore.EndResponse(c, http.StatusOK, gin.H{"message": "Blog list"}, "rest /blog")
		})

		blogGroup.GET("/:id", func(c *gin.Context) {
			id := c.Param("id")
			utilCore.EndResponse(c, http.StatusOK, gin.H{"message": "Blog detail", "id": id}, "rest /blog/:id")
		})
	}
}
