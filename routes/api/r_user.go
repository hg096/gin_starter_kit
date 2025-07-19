package api

import (
	"fmt"
	"gin_starter/model"
	"gin_starter/model/dbCore"
	"gin_starter/util/utilCore"
	"gin_starter/util/utilCore/auth"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

func SetupUserRoutes(rg *gin.RouterGroup) {

	// Util := utilCore.GetInstance()

	userGroup := rg.Group("/user")
	{

		userGroup.GET("/", func(c *gin.Context) { apiUser(c) })

		userGroup.POST("/make", func(c *gin.Context) { apiUserMake(c) })

		userGroup.POST("/makeUp", func(c *gin.Context) { apiUserMakeUp(c) })

		userGroup.POST("/logIn", func(c *gin.Context) { apiUserLogIn(c) })

		userGroup.GET("/logOut", func(c *gin.Context) { apiUserLogOut(c) })

		userGroup.POST("/refresh", func(c *gin.Context) { refreshUserToken(c) })

		// userGroup.Use(auth.ApiCheckLogin("U", 0))
		// {
		userGroup.GET("/profile", auth.ApiCheckLogin("U", 0), func(c *gin.Context) { apiUserProfile(c) })
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
	utilCore.EndResponse(c, http.StatusOK, gin.H{"message": "User list"}, "rest /user")
}

// 가입
func apiUserMake(c *gin.Context) {

	user := model.NewUser()

	data := utilCore.PostFields(c, map[string][2]string{
		"user_id":    {"u_id", ""},
		"user_pass":  {"u_pass", ""},
		"user_name":  {"u_name", ""},
		"user_email": {"u_email", ""},
	})

	data["u_auth_type"] = "U"

	// 트랜젝션 예시 불필요할시 제거
	tx, err := dbCore.BeginTransaction(c)
	if err != nil {
		return
	}

	insertedID, valErr, sqlErr := user.Insert(c, tx, data, "api/user/make")
	if valErr != nil || sqlErr != nil {
		log.Printf("User Insert 에러: %v", valErr)
		return
	}

	// 트랜젝션 예시 불필요할시 제거
	if err := dbCore.EndTransactionCommit(tx); err != nil {
		return
	}

	fmt.Printf("User가 성공적으로 추가 되었습니다. Inserted ID: %s\n", insertedID)

	utilCore.EndResponse(c, http.StatusOK, gin.H{"message": "User make"}, "rest /user/make")
}

// 수정
func apiUserMakeUp(c *gin.Context) {

	user := model.NewUpUser()

	data := utilCore.PostFields(c, map[string][2]string{
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

	utilCore.EndResponse(c, http.StatusOK, gin.H{"message": "User update"}, "rest /user/makeUp")
}

// 로그인
func apiUserLogIn(c *gin.Context) {

	data := utilCore.PostFields(c, map[string][2]string{
		"user_id":   {"u_id", ""},
		"user_pass": {"u_pass", ""},
	})

	userRows, err := dbCore.BuildSelectQuery(c, nil,
		"SELECT u_pass FROM _user WHERE u_id = ? LIMIT 1", []string{data["u_id"]}, "apiUserLogIn-getPass")

	if err != nil || len(userRows) == 0 {
		utilCore.EndResponse(c, http.StatusUnauthorized, gin.H{}, "rest /user/login getUser")
		return
	}
	storedHash := userRows[0]["u_pass"]

	err = bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(data["u_pass"]))
	if err != nil {
		utilCore.EndResponse(c, http.StatusUnauthorized, gin.H{}, "rest /user/login pass")
		return
	}

	at, rt, err := auth.GenerateTokens(data["u_id"], "")
	if err != nil {
		utilCore.EndResponse(c, http.StatusBadRequest, gin.H{}, "rest /user/login-GenerateTokens")
		return
	}

	_, err = dbCore.BuildUpdateQuery(c, nil, "_user", map[string]string{"u_re_token": rt}, "u_id = ?", []string{data["u_id"]}, "fn apiUserLogIn-BuildUpdateQuery")
	if err != nil {
		utilCore.EndResponse(c, http.StatusBadRequest, gin.H{}, "fn apiUserLogIn-BuildUpdateQuery")
		return
	}

	utilCore.EndResponse(c, http.StatusOK, gin.H{"access_token": at, "refresh_token": rt}, "rest /user/login")
}

// 로그아웃
func apiUserLogOut(c *gin.Context) {

	// refToken, _ := c.Cookie("ref_token")
	userId, _ := utilCore.GetContextVal(c, "user_id")

	_, _ = dbCore.BuildUpdateQuery(c, nil, "_user", map[string]string{"u_re_token": ""}, "u_id = ?", []string{userId}, "apiUserLogOut-BuildUpdateQuery")

	utilCore.EndResponse(c, http.StatusOK, gin.H{"message": "User logout"}, "rest /user/logout")
}

// 토큰 재발급
func refreshUserToken(c *gin.Context) {

	postData := utilCore.PostFields(c, map[string][2]string{
		"refresh_token": {"refresh_token", ""},
	})
	newAT, newRT, errMsg := auth.RefreshHandler(c, postData)
	if !utilCore.EmptyString(errMsg) {
		utilCore.EndResponse(c, http.StatusBadRequest, gin.H{}, errMsg)
		return
	}
	utilCore.EndResponse(c, http.StatusOK, gin.H{
		"access_token":  newAT,
		"refresh_token": newRT,
	}, "fn auth/RefreshHandler-end")
}

// 프로필 조회
func apiUserProfile(c *gin.Context) {
	uid := utilCore.GetBindField(c, "user_id", "")
	userID := c.MustGet("user_id").(string)

	utilCore.EndResponse(c, 200, gin.H{"user": uid, "userID": userID}, "rest /user/profile")
}
