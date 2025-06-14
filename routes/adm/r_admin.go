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
		// 권한체크
		pageUtil.RenderPageCheckLogin(c, "", 0)

		pageUtil.RenderPage(c, "home", gin.H{
			"UserName": "홍길동",
		}, true)
	})

	rg.GET("/menu", func(c *gin.Context) {
		pageUtil.RenderPageCheckLogin(c, "", 0)

		pageUtil.RenderPage(c, "menu", gin.H{
			"UserName": "홍길동",
		}, true)
	})

	rg.GET("/users", func(c *gin.Context) {
		pageUtil.RenderPageCheckLogin(c, "", 0)

		pageUtil.RenderPage(c, "user", gin.H{
			"UserName": "홍길동",
		}, true)
	})

	rg.GET("/chat", func(c *gin.Context) {
		pageUtil.RenderPageCheckLogin(c, "", 0)

		userId, _ := util.GetContextVal(c, "user_id")
		UserName, _ := util.GetContextVal(c, "user_name")

		pageUtil.RenderPage(c, "chat", gin.H{
			"UserName": UserName,
			"MyID":     userId,
		}, true)
	})

	adminGroup := rg.Group("/manage")
	{

		adminGroup.GET("/login", func(c *gin.Context) {
			// pageUtil.RenderPageCheckLogin(c, "", 0)
			pageUtil.RenderPage(c, "login", gin.H{"ShowFooter": false}, false)
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
