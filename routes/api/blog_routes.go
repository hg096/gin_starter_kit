package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// SetupBlogRoutes는 /api 내의 /blog 경로를 설정합니다.
func SetupBlogRoutes(rg *gin.RouterGroup) {
	// /api/blog 그룹 생성
	blogGroup := rg.Group("/blog")
	{
		// 블로그 목록 조회
		blogGroup.GET("/", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "Blog list"})
		})

		// 특정 블로그 상세 조회
		blogGroup.GET("/:id", func(c *gin.Context) {
			id := c.Param("id")
			c.JSON(http.StatusOK, gin.H{"message": "Blog detail", "id": id})
		})
	}
}
