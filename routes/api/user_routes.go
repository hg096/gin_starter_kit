package api

import (
	"fmt"
	"log"
	"net/http"

	"gin_starter/model"
	"gin_starter/util"
	"gin_starter/util/auth"

	"github.com/gin-gonic/gin"
)

func SetupUserRoutes(rg *gin.RouterGroup) {

	// Util := util.GetInstance()

	userGroup := rg.Group("/user")
	{

		userGroup.GET("/", func(c *gin.Context) {
			fmt.Println(" Hello, Alice")
			util.EndResponse(c, http.StatusOK, gin.H{"message": "User list"}, "rest /user")
		})

		userGroup.GET("/make", func(c *gin.Context) {

			user := model.NewUser()

			data := map[string]string{
				"u_id":        "Alice",
				"u_pass":      "Ali",
				"u_name":      "Alice",
				"u_email":     "alice@example.com",
				"u_auth_type": "U",
			}

			insertedID, valErr, sqlErr := user.Insert(c, nil, data, "api/user/make")
			if valErr != nil || sqlErr != nil {
				log.Printf("User Insert 에러: %v", valErr)
				return
			}

			fmt.Printf("User가 성공적으로 추가 되었습니다. Inserted ID: %d\n", insertedID)

			util.EndResponse(c, http.StatusOK, gin.H{"message": "User make"}, "rest /user/make")
		})

		userGroup.GET("/makeUp", func(c *gin.Context) {

			user := model.NewUpUser()

			data := map[string]string{
				"u_id": "Alice",
				// "u_pass": "Alice11",
				// "u_name":  "Alice",
				"u_email": "alice1@example.com",
			}

			valErr, sqlErr := user.Update(c, nil, data, "u_id = ?", []string{"Alice"}, "api/user/makeUp")
			if valErr != nil || sqlErr != nil {
				log.Printf("User Insert 에러: %v", valErr)
				return
			} else {
				// fmt.Printf("User가 성공적으로 수정 되었습니다. Inserted ID: %s\n", sqlResult)
			}

			util.EndResponse(c, http.StatusOK, gin.H{"message": "User update"}, "rest /user/makeUp")
		})

		userGroup.GET("/logIn", func(c *gin.Context) {

			// user := model.NewUser()
			data := map[string]string{
				"u_id":   "Alice",
				"u_pass": "Alice11",
				// "u_name":  "Alice",
				// "u_email": "alice@example.com",
			}

			at, rt, err := auth.GenerateTokens(data["u_id"])
			if err != nil {
				util.EndResponse(c, http.StatusBadRequest, gin.H{}, "rest /user/login-GenerateTokens")
				return
			}

			util.EndResponse(c, http.StatusOK, gin.H{"access_token": at, "refresh_token": rt}, "rest /user/login")
		})

		userGroup.GET("/refresh", auth.RefreshHandler)

		// userGroup.Use(auth.JWTAuthMiddleware("U", 0))
		// {
		userGroup.GET("/profile", auth.JWTAuthMiddleware("U", 0), func(c *gin.Context) {
			uid := c.GetString("user_id")
			util.EndResponse(c, 200, gin.H{"user": uid}, "rest /user/profile")
		})
		// }

		// userGroup.GET("/:id", func(c *gin.Context) {
		// 	id := c.Param("id")
		// 	c.JSON(http.StatusOK, gin.H{"message": "User detail", "id": id})
		// })

	}
}
