package pageUtil

import (
	"fmt"
	"gin_starter/model/core"
	"gin_starter/util"
	"gin_starter/util/auth"
	"log"
	"net/http"
	"text/template"

	"github.com/gin-gonic/gin"
)

type MenuItem struct {
	Label string
	Href  string
	Roles []string // 접근 가능한 권한
}

type MenuGroup struct {
	Key   string
	Label string
	Items []MenuItem
}

var allMenus = []MenuGroup{
	{
		Key:   "dashboard",
		Label: "기본 메뉴",
		Items: []MenuItem{
			{Label: "대시보드", Href: "/adm/dashboard", Roles: []string{"A", "M", "AG"}},
		},
	},
	{
		Key:   "posts",
		Label: "게시물 관리",
		Items: []MenuItem{
			{Label: "공지사항", Href: "/adm/posts/notice", Roles: []string{"A", "M", "AG"}},
			{Label: "자주 묻는 질문", Href: "/adm/posts/faq", Roles: []string{"A", "M", "AG"}},
		},
	},
	{
		Key:   "ads",
		Label: "광고 관리",
		Items: []MenuItem{
			{Label: "배너 설정", Href: "/adm/ads/banner", Roles: []string{"A", "M"}},
			{Label: "광고 승인", Href: "/adm/ads/approval", Roles: []string{"A", "M"}},
		},
	},
	{
		Key:   "settings",
		Label: "설정",
		Items: []MenuItem{
			{Label: "설정", Href: "/adm/settings", Roles: []string{"A"}},
		},
	},
	{
		Key:   "logout",
		Label: "",
		Items: []MenuItem{
			{Label: "로그아웃", Href: "/adm/manage/logout", Roles: []string{"A", "M", "AG"}},
		},
	},
}

func RenderPage(c *gin.Context, page string, customData gin.H, isCheckLogin bool) {

	data := gin.H{
		"IsLoggedIn": false,
		"UserName":   "",
		"ShowFooter": true,
		"Menus":      []map[string]interface{}{},
	}

	if !util.EmptyBool(isCheckLogin) {

		token, err := c.Cookie("acc_token")
		if err != nil || token == "" {
			c.Redirect(http.StatusFound, "/adm/manage/login")
			c.Abort()
			return
		}

		refToken, err := c.Cookie("ref_token")
		if err != nil || refToken == "" {
			c.Redirect(http.StatusFound, "/adm/manage/login")
			c.Abort()
			return
		}

		claims, err := auth.ValidateToken(token, auth.AccessSecret, auth.TokenSecret)
		if err != nil {
			newAT, newRT, errMsg := auth.RefreshHandler(c, map[string]string{"refresh_token": refToken})
			if !util.EmptyString(errMsg) {
				util.EndResponse(c, http.StatusBadRequest, gin.H{}, errMsg)
				return
			}
			claims, _ = auth.ValidateToken(newAT, auth.AccessSecret, auth.TokenSecret)

			// data["NEW_AT"] = newAT
			// data["NEW_RT"] = newRT

			SetCookie(c, "acc_token", newAT, 60*15)
			SetCookie(c, "ref_token", newRT, 60*60*24*7)
		}

		result, err := core.BuildSelectQuery(c, nil, "select u_auth_type, u_auth_level from _user where u_id = ? AND u_auth_type != 'U' ", []string{claims.JWTUserID}, "JWTAuthMiddleware.err")
		if err != nil {
			c.Redirect(http.StatusFound, "/adm/manage/login")
			c.Abort()
			return
		}

		c.Set("user_id", claims.JWTUserID)
		c.Set("user_type", result[0]["u_auth_type"])
		c.Set("user_level", result[0]["u_auth_level"])

		data["Menus"] = FilterMenusByRole(result[0]["u_auth_type"])

	}

	for k, v := range customData {
		data[k] = v
	}

	tmpl, err := template.ParseFiles(
		"templates/layouts/layout.tmpl",
		fmt.Sprintf("templates/pages/%s.tmpl", page),
		"templates/components/navbar.tmpl",
		"templates/components/sidebar.tmpl",
		"templates/components/footer.tmpl",
	)
	if err != nil {
		log.Fatalf("[종료] 템플릿 로딩 실패: %v", err)
	}

	// log.Println("RenderPage ")
	// log.Println(data)

	c.Status(http.StatusOK)
	tmpl.ExecuteTemplate(c.Writer, "layout", data)
}

func FilterMenusByRole(userRole string) []map[string]interface{} {
	filtered := []map[string]interface{}{}

	for _, group := range allMenus {
		groupItems := []map[string]string{}
		for _, item := range group.Items {
			if contains(item.Roles, userRole) {
				groupItems = append(groupItems, map[string]string{
					"Label": item.Label,
					"Href":  item.Href,
				})
			}
		}
		if len(groupItems) > 0 {
			filtered = append(filtered, map[string]interface{}{
				"Key":   group.Key,
				"Label": group.Label,
				"Items": groupItems,
			})
		}
	}
	return filtered
}

func contains(roles []string, role string) bool {
	for _, r := range roles {
		if r == role {
			return true
		}
	}
	return false
}

func SetCookie(c *gin.Context, key string, val string, time int) {
	c.SetCookie(
		key,  // 쿠키 이름
		val,  // 값
		time, // max-age(초)
		"/",  // path
		"",   // domain (빈 문자열이면 Host 도메인)
		true, // secure (https 전용)
		true, // httpOnly
	)
}
