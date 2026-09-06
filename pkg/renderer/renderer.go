package renderer

import (
	"encoding/json"
	"fmt"
	"gin_starter/pkg/logger"
	"gin_starter/pkg/utils"
	"log"
	"net/http"
	"strconv"
	"strings"
	"text/template"

	"github.com/gin-gonic/gin"
)

var publicReaderUnifiedPages = map[string]struct{}{}

// RenderAdmPage 관리자 페이지 렌더링
func RenderAdmPage(c *gin.Context, page string, customData gin.H, isMakeMenu bool) {
	data := gin.H{
		"IsLoggedIn": isMakeMenu,
		"UserName":   "",
		"ShowFooter": true,
		"Menus":      []map[string]interface{}{},
	}

	for k, v := range customData {
		data[k] = v
	}

	funcMap := template.FuncMap{
		"formatNumber": formatNumber,
	}

	tmpl, err := template.New("admin").Funcs(funcMap).ParseFiles(
		"web/admin/templates/layouts/layout.tmpl",
		fmt.Sprintf("web/admin/templates/pages/%s.tmpl", page),
		"web/admin/templates/components/navbar.tmpl",
		"web/admin/templates/components/sidebar.tmpl",
		"web/admin/templates/components/footer.tmpl",
	)
	if err != nil {
		log.Fatalf("[종료] 템플릿 로딩 실패: %v", err)
	}

	if isMakeMenu {
		userType, _ := utils.GetContextVal(c, "user_type")
		if !utils.EmptyString(userType) {
			// 메뉴 생성 로직은 별도 서비스로 분리 필요
			data["Menus"] = []map[string]interface{}{}
		}
	}

	c.Status(http.StatusOK)
	tmpl.ExecuteTemplate(c.Writer, "layout", data)
}

// RenderPage 일반 페이지 렌더링
func RenderPage(c *gin.Context, page string, customData gin.H, isMakeMenu bool) {
	data := gin.H{
		"IsLoggedIn":     isMakeMenu,
		"UserName":       "",
		"ShowFooter":     true,
		"Menus":          []map[string]interface{}{},
		"CurrentPath":    c.Request.URL.Path,
		"CurrentPageKey": page,
		"ReaderUnified":  isPublicReaderUnifiedPage(page),
	}

	for k, v := range customData {
		data[k] = v
	}

	funcMap := template.FuncMap{
		"formatNumber": formatNumber,
	}

	tmpl, err := template.New("public").Funcs(funcMap).ParseFiles(
		"web/public/layouts/layout.tmpl",
		fmt.Sprintf("web/public/pages/%s.tmpl", page),
		"web/public/components/navbar.tmpl",
		"web/public/components/sidebar.tmpl",
		"web/public/components/footer.tmpl",
	)
	if err != nil {
		log.Fatalf("[종료] 템플릿 로딩 실패: %v", err)
	}

	if isMakeMenu {
		userType, _ := utils.GetContextVal(c, "user_type")
		if !utils.EmptyString(userType) {
			// 메뉴 생성 로직은 별도 서비스로 분리 필요
			data["Menus"] = []map[string]interface{}{}
		} else {
			showArticleBriefings, _ := data["ShowArticleBriefingsMenu"].(bool)
			showETFActualTest, _ := data["ShowETFActualTestMenu"].(bool)
			data["Menus"] = MakeUnLoginMenu(c.Request.Host, showArticleBriefings, showETFActualTest)
		}
	}

	c.Status(http.StatusOK)
	tmpl.ExecuteTemplate(c.Writer, "layout", data)
}

func isPublicReaderUnifiedPage(page string) bool {
	_, ok := publicReaderUnifiedPages[page]
	return ok
}

// MakeUnLoginMenu 비로그인 메뉴
func MakeUnLoginMenu(requestHost string, showDebugEntries ...bool) []map[string]interface{} {
	items := []map[string]string{}

	return []map[string]interface{}{
		{
			"ID":    "",
			"Label": "",
			"Items": items,
		},
	}
}

// SetCookie 쿠키 설정
func SetCookie(c *gin.Context, key string, val string, maxAge int) {
	c.SetCookie(
		key,    // 쿠키 이름
		val,    // 값
		maxAge, // max-age(초)
		"/",    // path
		"",     // domain (빈 문자열이면 Host 도메인)
		false,  // secure (https 전용)
		false,  // httpOnly
	)
}

// Contains 슬라이스에 값이 포함되어 있는지 확인
func Contains(roles []string, role string) bool {
	for _, r := range roles {
		if r == role {
			return true
		}
	}
	return false
}

func formatNumber(v interface{}) string {
	switch value := v.(type) {
	case int:
		return formatInt64WithComma(int64(value))
	case int8:
		return formatInt64WithComma(int64(value))
	case int16:
		return formatInt64WithComma(int64(value))
	case int32:
		return formatInt64WithComma(int64(value))
	case int64:
		return formatInt64WithComma(value)
	case uint:
		return formatUint64WithComma(uint64(value))
	case uint8:
		return formatUint64WithComma(uint64(value))
	case uint16:
		return formatUint64WithComma(uint64(value))
	case uint32:
		return formatUint64WithComma(uint64(value))
	case uint64:
		return formatUint64WithComma(value)
	case float32:
		return formatInt64WithComma(int64(value))
	case float64:
		return formatInt64WithComma(int64(value))
	case string:
		return formatNumericString(value)
	default:
		return fmt.Sprint(v)
	}
}

func formatNumericString(raw string) string {
	trimmed := strings.TrimSpace(strings.ReplaceAll(raw, ",", ""))
	if trimmed == "" {
		return ""
	}

	sign := ""
	if strings.HasPrefix(trimmed, "-") {
		sign = "-"
		trimmed = strings.TrimPrefix(trimmed, "-")
	}

	if parsed, err := utils.StringToNumeric[uint64](trimmed); err == nil {
		return sign + formatUint64WithComma(parsed)
	}

	return raw
}

func formatInt64WithComma(n int64) string {
	if n < 0 {
		return "-" + formatUint64WithComma(uint64(-n))
	}
	return formatUint64WithComma(uint64(n))
}

func formatUint64WithComma(n uint64) string {
	raw := strconv.FormatUint(n, 10)
	if len(raw) <= 3 {
		return raw
	}

	var b strings.Builder
	start := len(raw) % 3
	if start == 0 {
		start = 3
	}

	b.WriteString(raw[:start])
	for i := start; i < len(raw); i += 3 {
		b.WriteByte(',')
		b.WriteString(raw[i : i+3])
	}

	return b.String()
}

// MakeMenuRole 데이터베이스에서 역할 기반 메뉴 생성 (레거시 지원)
// 실제 구현은 admin 도메인에서 처리하는 것이 좋습니다
func MakeMenuRole(c *gin.Context, userRole string, isOutRole bool) []map[string]interface{} {
	// TODO: 이 함수는 admin 도메인의 서비스로 이동하는 것이 좋습니다
	logger.Warn("MakeMenuRole은 레거시 함수입니다. admin 도메인 서비스로 이동이 필요합니다.")
	return []map[string]interface{}{}
}

// RenderPageCheckLogin 로그인 체크 및 페이지 렌더링 (레거시 지원)
// 실제로는 middleware에서 처리하는 것이 좋습니다
func RenderPageCheckLogin(c *gin.Context, userType string, lv int8) []map[string]interface{} {
	// TODO: 이 함수는 middleware로 이동하는 것이 좋습니다
	logger.Warn("RenderPageCheckLogin은 레거시 함수입니다. middleware로 이동이 필요합니다.")

	// 쿠키에서 토큰 가져오기
	token, _ := c.Cookie("acc_token")
	refToken, err := c.Cookie("ref_token")

	if err != nil || refToken == "" {
		c.Redirect(http.StatusFound, "/")
		c.Abort()
		return nil
	}

	// 토큰 검증 로직은 middleware.AuthMiddleware를 사용하세요
	_ = token

	return nil
}

// ParseJSONRoles JSON 문자열을 역할 슬라이스로 파싱
func ParseJSONRoles(jsonStr string) ([]string, error) {
	var roles []string
	if err := json.Unmarshal([]byte(jsonStr), &roles); err != nil {
		return nil, err
	}
	return roles, nil
}

// CheckUserAccess 사용자 접근 권한 확인
func CheckUserAccess(userType, requiredType string) bool {
	if requiredType == "" {
		return true
	}
	if userType == requiredType {
		return true
	}
	// 복수 타입 지원
	return strings.Contains(requiredType, userType)
}
