package adm

import (
	"fmt"
	"gin_starter/model"
	"gin_starter/model/core"
	"gin_starter/util"
	"gin_starter/util/auth"
	"gin_starter/util/pageUtil"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

func SetupAdminRoutes(rg *gin.RouterGroup) {

	rg.GET("/", func(c *gin.Context) {
		// 예시: 세션 또는 쿠키 기반 로그인 정보
		pageUtil.RenderPage(c, "home", gin.H{
			// "IsLoggedIn": true,
			"UserName": "홍길동",
		})
	})

	adminGroup := rg.Group("/manage")
	{
		adminGroup.GET("/", func(c *gin.Context) {
			// 예시: 세션 또는 쿠키 기반 로그인 정보
			pageUtil.RenderPage(c, "home", gin.H{
				// "IsLoggedIn": true,
				"UserName": "홍길동",
			})
		})

		adminGroup.GET("/login", func(c *gin.Context) {
			pageUtil.RenderPage(c, "login", gin.H{"ShowFooter": false})
		})

	}

	adminApiGroup := rg.Group("/api")
	{

		adminApiGroup.POST("/make", func(c *gin.Context) { apiUserMake(c) })

		adminApiGroup.POST("/makeUp", func(c *gin.Context) { apiUserMakeUp(c) })

		adminApiGroup.POST("/logIn", func(c *gin.Context) { apiUserLogIn(c) })

		adminApiGroup.GET("/logOut", func(c *gin.Context) { apiUserLogOut(c) })

		adminApiGroup.POST("/refresh", auth.RefreshHandler)
	}
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

	// 트랜젝션 예시 불필요할시 제거
	tx, err := core.BeginTransaction(c)
	if err != nil {
		return
	}

	insertedID, valErr, sqlErr := user.Insert(c, tx, data, "api/user/make")
	if valErr != nil || sqlErr != nil {
		log.Printf("User Insert 에러: %v", valErr)
		return
	}

	// 트랜젝션 예시 불필요할시 제거
	if cerr := core.EndTransactionCommit(tx); cerr != nil {
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
