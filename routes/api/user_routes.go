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

		userGroup.GET("/:id", func(c *gin.Context) {
			id := c.Param("id")
			c.JSON(http.StatusOK, gin.H{"message": "User detail", "id": id})
		})

	}
}
