package api

import (
	"fmt"
	"log"
	"net/http"

	"gin_starter/model" // 실제 모듈 경로에 맞게 수정하세요.
	// DB 및 CRUD 관련 함수가 포함된 패키지
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

		// 사용자 목록 조회
		userGroup.GET("/make", func(c *gin.Context) {

			// core 패키지의 SetDB 함수를 통해 전역 DB 변수 설정
			// core.SetDB(dbConn)

			// User 객체 생성 (빈 User 모델 생성)
			user := model.NewUser()
			// 삽입할 데이터를 map 형태로 준비 (예: "name", "email" 컬럼)
			data := map[string]interface{}{
				"name":  "Alice",
				"email": "alice@example.com",
				// 추가 컬럼이 있다면 데이터 추가
			}

			// Insert 메서드를 호출하여 데이터를 삽입하고, 삽입된 행의 ID를 받아옴
			insertedID, err := user.Insert(data)
			if err != nil {
				log.Fatalf("User Insert 에러: %v", err)
			} else {
				fmt.Printf("User가 성공적으로 추가 되었습니다. Inserted ID: %d\n", insertedID)
			}

			c.JSON(http.StatusOK, gin.H{"message": "User make"})
		})

		// 특정 사용자 조회 (예: id로)
		userGroup.GET("/:id", func(c *gin.Context) {
			id := c.Param("id")
			c.JSON(http.StatusOK, gin.H{"message": "User detail", "id": id})
		})

	}
}
