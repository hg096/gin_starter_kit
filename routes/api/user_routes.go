package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// SetupUserRoutes는 /api 내의 /user 경로를 설정합니다.
func SetupUserRoutes(rg *gin.RouterGroup) {
	// /api/user 그룹 생성
	userGroup := rg.Group("/user")
	{
		// 사용자 목록 조회
		userGroup.GET("/", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "User list"})
		})

		// 특정 사용자 조회 (예: id로)
		userGroup.GET("/:id", func(c *gin.Context) {
			id := c.Param("id")
			c.JSON(http.StatusOK, gin.H{"message": "User detail", "id": id})
		})
	}
}
