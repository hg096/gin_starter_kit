package admin

import (
	"html/template"
	"net/http"

	"github.com/gin-gonic/gin"
)

// PageHandler 관리자 페이지 핸들러
type PageHandler struct {
	templates *template.Template
}

// NewPageHandler 관리자 페이지 핸들러 생성
func NewPageHandler() *PageHandler {
	// 템플릿 로드
	templates := template.Must(template.ParseGlob("web/admin/templates/*.html"))

	return &PageHandler{
		templates: templates,
	}
}

// LoginPage 로그인 페이지
func (h *PageHandler) LoginPage(c *gin.Context) {
	c.Header("Content-Type", "text/html; charset=utf-8")
	h.templates.ExecuteTemplate(c.Writer, "login.html", nil)
}

// DashboardPage 대시보드 페이지
func (h *PageHandler) DashboardPage(c *gin.Context) {
	// 토큰 확인은 프론트엔드에서 처리
	data := gin.H{
		"Title":    "대시보드",
		"Active":   "dashboard",
		"UserName": "관리자",
		"UserType": "A",
	}

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.Writer.WriteHeader(http.StatusOK)

	// layout과 dashboard를 함께 렌더링
	h.templates.ExecuteTemplate(c.Writer, "layout.html", data)
	h.templates.ExecuteTemplate(c.Writer, "dashboard.html", data)
}

// UsersPage 사용자 관리 페이지
func (h *PageHandler) UsersPage(c *gin.Context) {
	data := gin.H{
		"Title":    "사용자 관리",
		"Active":   "users",
		"UserName": "관리자",
		"UserType": "A",
	}

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.Writer.WriteHeader(http.StatusOK)

	h.templates.ExecuteTemplate(c.Writer, "layout.html", data)
	h.templates.ExecuteTemplate(c.Writer, "users.html", data)
}

// LogoutPage 로그아웃
func (h *PageHandler) LogoutPage(c *gin.Context) {
	// 프론트엔드에서 토큰을 삭제하도록 스크립트 반환
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, `
		<!DOCTYPE html>
		<html>
		<head>
			<meta charset="UTF-8">
			<title>로그아웃</title>
		</head>
		<body>
			<script>
				localStorage.removeItem('access_token');
				localStorage.removeItem('refresh_token');
				localStorage.removeItem('user_name');
				localStorage.removeItem('user_type');
				window.location.href = '/admin/login';
			</script>
		</body>
		</html>
	`)
}