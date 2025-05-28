package pageUtil

import (
	"fmt"
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
		Key:   "posts",
		Label: "게시물 관리",
		Items: []MenuItem{
			{Label: "공지사항", Href: "/adm/posts/notice", Roles: []string{"admin", "editor"}},
			{Label: "자주 묻는 질문", Href: "/adm/posts/faq", Roles: []string{"admin", "editor", "viewer"}},
		},
	},
	{
		Key:   "ads",
		Label: "광고 관리",
		Items: []MenuItem{
			{Label: "배너 설정", Href: "/adm/ads/banner", Roles: []string{"admin"}},
			{Label: "광고 승인", Href: "/adm/ads/approval", Roles: []string{"admin"}},
		},
	},
}

func RenderPage(c *gin.Context, page string, customData gin.H) {

	data := gin.H{
		"IsLoggedIn": false,
		"UserName":   "",
		"ShowFooter": true,
		"Menus":      []map[string]interface{}{},
	}

	data["Menus"] = FilterMenusByRole("admin")

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
