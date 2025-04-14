package api

import (
	"fmt"
	"net/http"

	"gin_starter/util" // 모듈 경로에 맞게 수정하세요.

	"github.com/gin-gonic/gin"
)

// SetupUserRoutes는 /api 내의 /user 경로를 설정합니다.
func SetupUserRoutes(rg *gin.RouterGroup) {

	// 싱글톤 인스턴스를 가져옵니다.
	Util := util.GetInstance()

	// /api/user 그룹 생성
	userGroup := rg.Group("/user")
	{
		// 사용자 목록 조회
		userGroup.GET("/", func(c *gin.Context) {

			// FormatGreeting 메서드 호출
			greeting := Util.FormatGreeting("Alice")

			fmt.Println(greeting) // 출력: Hello, Alice!

			c.JSON(http.StatusOK, gin.H{"message": "User list"})
		})

		// 특정 사용자 조회 (예: id로)
		userGroup.GET("/:id", func(c *gin.Context) {
			id := c.Param("id")
			c.JSON(http.StatusOK, gin.H{"message": "User detail", "id": id})
		})
	}
}
