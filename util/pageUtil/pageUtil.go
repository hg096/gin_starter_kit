package pageUtil

import (
	"fmt"
	"log"
	"net/http"
	"text/template"

	"github.com/gin-gonic/gin"
)

func RenderPage(c *gin.Context, page string, data gin.H) {
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

	if _, ok := data["ShowFooter"]; !ok {
		data["ShowFooter"] = true
	}

	c.Status(http.StatusOK)
	tmpl.ExecuteTemplate(c.Writer, "layout", data)
}
