package adm

import (
	"gin_starter/model/core"
	"gin_starter/util"
	"gin_starter/util/pageUtil"
	"net/http"

	"github.com/gin-gonic/gin"
)

func SetupAdminRoutes(rg *gin.RouterGroup) {

	rg.GET("/", func(c *gin.Context) {
		// 예시: 세션 또는 쿠키 기반 로그인 정보
		pageUtil.RenderPageCheckLogin(c, "", 0)

		userType, _ := util.GetContextVal(c, "user_type")
		menuData := pageUtil.MakeMenuRole(c, userType, false)

		// fmt.Println("menuData >> ")
		// fmt.Println(menuData)

		pageUtil.RenderPage(c, "home", gin.H{
			"Menus":    menuData,
			"UserName": "홍길동",
		})
	})

	rg.GET("/menu", func(c *gin.Context) {
		// 예시: 세션 또는 쿠키 기반 로그인 정보
		pageUtil.RenderPageCheckLogin(c, "", 0)

		userType, _ := util.GetContextVal(c, "user_type")
		menuData := pageUtil.MakeMenuRole(c, userType, false)

		pageUtil.RenderPage(c, "menu", gin.H{
			"Menus":    menuData,
			"UserName": "홍길동",
		})
	})

	rg.GET("/users", func(c *gin.Context) {
		pageUtil.RenderPageCheckLogin(c, "", 0)

		userType, _ := util.GetContextVal(c, "user_type")
		menuData := pageUtil.MakeMenuRole(c, userType, false)

		pageUtil.RenderPage(c, "user", gin.H{
			"Menus":    menuData,
			"UserName": "홍길동",
		})
	})

	rg.GET("/chat", func(c *gin.Context) {
		pageUtil.RenderPageCheckLogin(c, "", 0)

		userId, _ := util.GetContextVal(c, "user_id")
		userType, _ := util.GetContextVal(c, "user_type")
		menuData := pageUtil.MakeMenuRole(c, userType, false)

		pageUtil.RenderPage(c, "chat", gin.H{
			"Menus":    menuData,
			"UserName": "홍길동",
			"MyID":     userId,
		})
	})

	adminGroup := rg.Group("/manage")
	{

		adminGroup.GET("/login", func(c *gin.Context) {
			// pageUtil.RenderPageCheckLogin(c, "", 0)
			pageUtil.RenderPage(c, "login", gin.H{"ShowFooter": false})
		})

		adminGroup.GET("/logout", func(c *gin.Context) {

			pageUtil.RenderPageCheckLogin(c, "", 0)
			userId, _ := util.GetContextVal(c, "user_id")

			_, _ = core.BuildUpdateQuery(c, nil, "_user", map[string]string{"u_re_token": ""}, "u_id = ?", []string{userId}, "page adm/logout")
			pageUtil.SetCookie(c, "acc_token", "", 0)
			pageUtil.SetCookie(c, "ref_token", "", 0)
			c.Redirect(http.StatusFound, "/adm/manage/login")
		})
	}

}
