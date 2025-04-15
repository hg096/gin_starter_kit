package api

import (
	"fmt"
	"log"
	"net/http"

	"gin_starter/model" // 실제 모듈 경로에 맞게 수정하세요.
	// DB 및 CRUD 관련 함수가 포함된 패키지
	// 모듈 경로에 맞게 수정하세요.

	"github.com/gin-gonic/gin"
)

// SetupUserRoutes는 /api 내의 /user 경로를 설정합니다.
func SetupUserRoutes(rg *gin.RouterGroup) {

	// 싱글톤 인스턴스를 가져옵니다.
	// Util := util.GetInstance()

	// /api/user 그룹 생성
	userGroup := rg.Group("/user")
	{
		// 사용자 목록 조회
		userGroup.GET("/", func(c *gin.Context) {

			fmt.Println(" Hello, Alice") // 출력: Hello, Alice!

			c.JSON(http.StatusOK, gin.H{"message": "User list"})
		})

		userGroup.GET("/make", func(c *gin.Context) {

			user := model.NewUser()

			data := map[string]interface{}{
				"u_id":    "Alice",
				"u_pass":  "Alice11",
				"u_name":  "Alice",
				"u_email": "alice@example.com",
			}

			insertedID, err := user.Insert(data)
			if err != nil {
				log.Printf("User Insert 에러: %v", err)
			} else {
				fmt.Printf("User가 성공적으로 추가 되었습니다. Inserted ID: %d\n", insertedID)
			}

			c.JSON(http.StatusOK, gin.H{"message": "User make"})
		})

		userGroup.GET("/makeUp", func(c *gin.Context) {

			user := model.NewUpUser()

			data := map[string]interface{}{
				"u_id":    "Alice",
				"u_pass":  "Alice11",
				"u_name":  "Alice",
				"u_email": "alice@example.com",
			}

			sqlResult, err := user.Update(data, "u_id = ?", []interface{}{"Alice"})
			if err != nil {
				log.Printf("User Insert 에러: %v", err)
			} else {
				fmt.Printf("User가 성공적으로 수정 되었습니다. Inserted ID: %s\n", sqlResult)
			}

			c.JSON(http.StatusOK, gin.H{"message": "User update"})
		})

		// 특정 사용자 조회 (예: id로)
		userGroup.GET("/:id", func(c *gin.Context) {
			id := c.Param("id")
			c.JSON(http.StatusOK, gin.H{"message": "User detail", "id": id})
		})

	}
}
