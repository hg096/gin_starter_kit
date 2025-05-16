package api

import (
	"fmt"
	"gin_starter/model"
	"gin_starter/model/core"
	"gin_starter/util"
	"gin_starter/util/auth"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

func SetupUserRoutes(rg *gin.RouterGroup) {

	// Util := util.GetInstance()

	userGroup := rg.Group("/user")
	{

		userGroup.GET("/", func(c *gin.Context) { apiUser(c) })

		userGroup.POST("/make", func(c *gin.Context) { apiUserMake(c) })

		userGroup.POST("/makeUp", func(c *gin.Context) { apiUserMakeUp(c) })

		userGroup.POST("/logIn", func(c *gin.Context) { apiUserLogIn(c) })

		userGroup.GET("/logOut", func(c *gin.Context) { apiUserLogOut(c) })

		userGroup.GET("/refresh", auth.RefreshHandler)

		// userGroup.Use(auth.JWTAuthMiddleware("U", 0))
		// {
		userGroup.GET("/profile", auth.JWTAuthMiddleware("U", 0), func(c *gin.Context) { apiUserProfile(c) })
		// }

		// userGroup.GET("/:id", func(c *gin.Context) {
		// 	id := c.Param("id")
		// 	c.JSON(http.StatusOK, gin.H{"message": "User detail", "id": id})
		// })

	}
}

//
//

// 사용자
// apiUser godoc
// @Summary 사용자 테스트
// @Router /api/user/ [get]
func apiUser(c *gin.Context) {
	util.EndResponse(c, http.StatusOK, gin.H{"message": "User list"}, "rest /user")
}

// 가입
func apiUserMake(c *gin.Context) {

	user := model.NewUser()

	data := util.PostFields(c, map[string][2]string{
		"user_id":    {"u_id", ""},
		"user_pass":  {"u_pass", ""},
		"user_name":  {"u_name", ""},
		"user_email": {"u_email", ""},
	})

	data["u_auth_type"] = "U"

	insertedID, valErr, sqlErr := user.Insert(c, nil, data, "api/user/make")
	if valErr != nil || sqlErr != nil {
		log.Printf("User Insert 에러: %v", valErr)
		return
	}

	fmt.Printf("User가 성공적으로 추가 되었습니다. Inserted ID: %d\n", insertedID)

	util.EndResponse(c, http.StatusOK, gin.H{"message": "User make"}, "rest /user/make")
}

// 수정
func apiUserMakeUp(c *gin.Context) {

	user := model.NewUpUser()

	data := util.PostFields(c, map[string][2]string{
		"user_id": {"u_id", ""},
		// "user_pass":  {"u_pass", ""},
		// "user_name":  {"u_name", ""},
		"user_email": {"u_email", ""},
	})

	valErr, sqlErr := user.Update(c, nil, data, "u_id = ?", []string{data["u_id"]}, "api/user/makeUp")
	if valErr != nil || sqlErr != nil {
		log.Printf("User Insert 에러: %v", valErr)
		return
	} else {
		// fmt.Printf("User가 성공적으로 수정 되었습니다. Inserted ID: %s\n", sqlResult)
	}

	util.EndResponse(c, http.StatusOK, gin.H{"message": "User update"}, "rest /user/makeUp")
}

// 로그인
func apiUserLogIn(c *gin.Context) {

	data := util.PostFields(c, map[string][2]string{
		"user_id":   {"u_id", ""},
		"user_pass": {"u_pass", ""},
	})

	at, rt, err := auth.GenerateTokens(data["u_id"], "")
	if err != nil {
		util.EndResponse(c, http.StatusBadRequest, gin.H{}, "rest /user/login-GenerateTokens")
		return
	}

	_, err = core.BuildUpdateQuery(c, nil, "_user", map[string]string{"u_re_token": rt}, "u_id = ?", []string{data["u_id"]}, "fn apiUserLogIn-BuildUpdateQuery")
	if err != nil {
		util.EndResponse(c, http.StatusBadRequest, gin.H{}, "fn apiUserLogIn-BuildUpdateQuery")
		return
	}

	util.EndResponse(c, http.StatusOK, gin.H{"access_token": at, "refresh_token": rt}, "rest /user/login")
}

// 로그아웃
func apiUserLogOut(c *gin.Context) {
	util.EndResponse(c, http.StatusOK, gin.H{"message": "User logout"}, "rest /user/logout")
}

// 프로필 조회
func apiUserProfile(c *gin.Context) {
	uid := util.GetBindField(c, "user_id", "")

	util.EndResponse(c, 200, gin.H{"user": uid}, "rest /user/profile")
}
