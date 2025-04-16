package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func SetupBlogRoutes(rg *gin.RouterGroup) {

	blogGroup := rg.Group("/blog")
	{

		blogGroup.GET("/", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "Blog list"})
		})

		blogGroup.GET("/:id", func(c *gin.Context) {
			id := c.Param("id")
			c.JSON(http.StatusOK, gin.H{"message": "Blog detail", "id": id})
		})
	}
}
