package api

import (
	"fmt"
	"log"
	"net/http"

	"gin_starter/model"

	"github.com/gin-gonic/gin"
)

func SetupUserRoutes(rg *gin.RouterGroup) {

	// Util := util.GetInstance()

	userGroup := rg.Group("/user")
	{

		userGroup.GET("/", func(c *gin.Context) {
			fmt.Println(" Hello, Alice")
			c.JSON(http.StatusOK, gin.H{"message": "User list"})
		})

		userGroup.GET("/make", func(c *gin.Context) {

			user := model.NewUser()

			data := map[string]string{
				"u_id":    "Alice",
				"u_pass":  "Ali",
				"u_name":  "Alice",
				"u_email": "alice@example.com",
			}

			insertedID, valErr, sqlErr := user.Insert(c, nil, data, "api/user/make")
			if valErr != nil || sqlErr != nil {
				log.Printf("User Insert 에러: %v", valErr)
				return
			}

			fmt.Printf("User가 성공적으로 추가 되었습니다. Inserted ID: %d\n", insertedID)

			c.JSON(http.StatusOK, gin.H{"message": "User make"})
		})

		userGroup.GET("/makeUp", func(c *gin.Context) {

			user := model.NewUpUser()

			data := map[string]string{
				"u_id":    "Alice",
				"u_pass":  "Alice11",
				"u_name":  "Alice",
				"u_email": "alice@example.com",
			}

			sqlResult, valErr, sqlErr := user.Update(c, nil, data, "u_id = ?", []string{"Alice"}, "api/user/makeUp")
			if valErr != nil || sqlErr != nil {
				log.Printf("User Insert 에러: %v", valErr)
				return
			} else {
				fmt.Printf("User가 성공적으로 수정 되었습니다. Inserted ID: %s\n", sqlResult)
			}

			c.JSON(http.StatusOK, gin.H{"message": "User update"})
		})

		userGroup.GET("/:id", func(c *gin.Context) {
			id := c.Param("id")
			c.JSON(http.StatusOK, gin.H{"message": "User detail", "id": id})
		})

	}
}
