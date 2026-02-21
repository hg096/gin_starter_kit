package admin

import (
	"fmt"
	"gin_starter/internal/config"
	"gin_starter/internal/middleware"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"
)

type pageViewService interface {
	GetAdminPages(includeDisabled bool) ([]AdminPage, error)
	GetAdminPageByKey(pageKey string) (*AdminPage, error)
}

type adminMenuItem struct {
	Key        string
	Title      string
	Path       string
	Icon       string
	Active     bool
	GroupKey   string
	GroupLabel string
	GroupOrder int
	SortOrder  int
}

type adminMenuGroup struct {
	Key   string
	Label string
	Order int
	Items []adminMenuItem
}

// PageHandler serves admin pages.
type PageHandler struct {
	loginTpl     *template.Template
	homeTpl      *template.Template
	dashboardTpl *template.Template
	usersTpl     *template.Template
	blogsTpl     *template.Template
	chatTpl      *template.Template
	dynamicTpl   *template.Template
	secureCookie bool
	pageService  pageViewService
}

// NewPageHandler creates an admin page handler.
func NewPageHandler(cfg *config.Config, pageService pageViewService) *PageHandler {
	return &PageHandler{
		loginTpl:     mustParseTemplateFiles("login.html"),
		homeTpl:      mustParseTemplateFiles("layout.html", "home.html"),
		dashboardTpl: mustParseTemplateFiles("layout.html", "dashboard.html"),
		usersTpl:     mustParseTemplateFiles("layout.html", "users.html"),
		blogsTpl:     mustParseTemplateFiles("layout.html", "blog_manage.html"),
		chatTpl:      mustParseTemplateFiles("layout.html", "admin_chat.html"),
		dynamicTpl:   mustParseTemplateFiles("layout.html", "dynamic_page.html"),
		secureCookie: cfg != nil && cfg.IsProduction(),
		pageService:  pageService,
	}
}

func (h *PageHandler) LoginPage(c *gin.Context) {
	c.Header("Content-Type", "text/html; charset=utf-8")
	_ = h.loginTpl.ExecuteTemplate(c.Writer, "login.html", nil)
}

func (h *PageHandler) HomePage(c *gin.Context) {
	data := h.buildViewData(c, "페이지 선택", "", nil)
	c.Header("Content-Type", "text/html; charset=utf-8")
	_ = h.homeTpl.ExecuteTemplate(c.Writer, "layout.html", data)
}

func (h *PageHandler) DashboardPage(c *gin.Context) {
	page := builtInAdminPageByKey("dashboard")
	data := h.buildViewData(c, "대시보드", "dashboard", page)
	c.Header("Content-Type", "text/html; charset=utf-8")
	_ = h.dashboardTpl.ExecuteTemplate(c.Writer, "layout.html", data)
}

func (h *PageHandler) UsersPage(c *gin.Context) {
	page := builtInAdminPageByKey("users")
	data := h.buildViewData(c, "사용자 관리", "users", page)
	c.Header("Content-Type", "text/html; charset=utf-8")
	_ = h.usersTpl.ExecuteTemplate(c.Writer, "layout.html", data)
}

func (h *PageHandler) BlogsPage(c *gin.Context) {
	page := builtInAdminPageByKey("blogs")
	data := h.buildViewData(c, "블로그 관리", "blogs", page)
	c.Header("Content-Type", "text/html; charset=utf-8")
	_ = h.blogsTpl.ExecuteTemplate(c.Writer, "layout.html", data)
}

func (h *PageHandler) ChatPage(c *gin.Context) {
	page := builtInAdminPageByKey("admin_chat")
	data := h.buildViewData(c, "채팅", "admin_chat", page)
	c.Header("Content-Type", "text/html; charset=utf-8")
	_ = h.chatTpl.ExecuteTemplate(c.Writer, "layout.html", data)
}

func (h *PageHandler) DynamicPage(c *gin.Context) {
	pageKey, ok := NormalizePageKey(c.Param("page_key"))
	if !ok {
		c.String(http.StatusBadRequest, "invalid page key")
		return
	}

	if h.pageService == nil {
		c.String(http.StatusNotFound, "page not found")
		return
	}

	page, err := h.pageService.GetAdminPageByKey(pageKey)
	if err != nil || page == nil || !page.Enabled {
		c.String(http.StatusNotFound, "page not found")
		return
	}

	data := h.buildViewData(c, page.Title, page.Key, page)
	c.Header("Content-Type", "text/html; charset=utf-8")
	_ = h.dynamicTpl.ExecuteTemplate(c.Writer, "layout.html", data)
}

func (h *PageHandler) LogoutPage(c *gin.Context) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(middleware.AccessTokenCookieName, "", -1, "/", "", h.secureCookie, true)
	c.SetCookie(middleware.RefreshTokenCookieName, "", -1, "/", "", h.secureCookie, true)
	c.Redirect(http.StatusFound, "/admin/login")
}

func (h *PageHandler) buildViewData(c *gin.Context, title, active string, currentPage *AdminPage) gin.H {
	userID := c.GetString("user_id")
	userType := c.GetString("user_type")
	userInitial := "A"
	if trimmed := strings.TrimSpace(userID); trimmed != "" {
		userInitial = strings.ToUpper(string([]rune(trimmed)[0]))
	}
	isSuperAdmin := false
	if v, ok := c.Get("is_super_admin"); ok {
		if converted, castOK := v.(bool); castOK {
			isSuperAdmin = converted
		}
	}
	levelPolicyEnabled := true
	if v, ok := c.Get("level_policy_enabled"); ok {
		if converted, castOK := v.(bool); castOK {
			levelPolicyEnabled = converted
		}
	}

	permissions := make(map[string]struct{})
	if v, ok := c.Get("user_permissions"); ok {
		if converted, castOK := v.(map[string]struct{}); castOK {
			permissions = converted
		}
	}

	list := make([]string, 0, len(permissions))
	for code := range permissions {
		list = append(list, code)
	}

	has := func(code string) bool {
		if isSuperAdmin {
			return true
		}
		_, ok := permissions[code]
		return ok
	}

	menuItems := h.resolveMenuItems(has, active)
	menuGroups := h.resolveMenuGroups(menuItems)
	pageTitle := ""
	pageDescription := ""
	pagePath := ""
	pageKey := ""
	pagePerms := AdminPagePermissionCodes{}
	if currentPage != nil {
		pageTitle = currentPage.Title
		pageDescription = currentPage.Description
		pagePath = currentPage.Path
		pageKey = currentPage.Key
		pagePerms = currentPage.PermissionCodes
	}

	return gin.H{
		"Title":                 title,
		"Active":                active,
		"UserName":              userID,
		"UserInitial":           userInitial,
		"UserType":              userType,
		"UserLevel":             c.GetInt("user_level"),
		"IsSuperAdmin":          isSuperAdmin,
		"LevelPolicyEnabled":    levelPolicyEnabled,
		"Permissions":           list,
		"CanStatsRead":          has("admin.stats.read"),
		"CanAccountRead":        has("admin.account.read"),
		"CanProfileUpdate":      has("admin.account.profile.update"),
		"CanStatusUpdate":       has("admin.account.status.update"),
		"CanPasswordReset":      has("admin.account.password.reset"),
		"CanPermissionManage":   has("admin.account.permission.manage"),
		"CanLevelManage":        has("admin.account.level.manage"),
		"CanDeleteUser":         has("admin.account.delete"),
		"CanAllowlistManage":    has("admin.allowlist.manage"),
		"CanLevelPolicyManage":  has("admin.system.level_policy.manage"),
		"CanAuditRead":          has("admin.audit.read"),
		"CanPageManage":         has(PermissionPageManage),
		"MenuItems":             menuItems,
		"MenuGroups":            menuGroups,
		"CurrentPageTitle":      pageTitle,
		"CurrentPageDesc":       pageDescription,
		"CurrentPagePath":       pagePath,
		"CurrentPageKey":        pageKey,
		"CurrentPagePermRead":   pagePerms.Read,
		"CurrentPagePermCreate": pagePerms.Create,
		"CurrentPagePermUpdate": pagePerms.Update,
		"CurrentPagePermDelete": pagePerms.Delete,
		"CurrentPageCanRead":    has(pagePerms.Read),
		"CurrentPageCanCreate":  has(pagePerms.Create),
		"CurrentPageCanUpdate":  has(pagePerms.Update),
		"CurrentPageCanDelete":  has(pagePerms.Delete),
	}
}

func (h *PageHandler) resolveMenuItems(has func(code string) bool, active string) []adminMenuItem {
	pages := defaultAdminPages(false)
	if h.pageService != nil {
		if loadedPages, err := h.pageService.GetAdminPages(false); err == nil && len(loadedPages) > 0 {
			pages = loadedPages
		}
	}

	items := make([]adminMenuItem, 0, len(pages))
	for _, page := range pages {
		if !page.Enabled {
			continue
		}

		readAllowed := false
		switch page.Key {
		case "dashboard":
			readAllowed = has("admin.stats.read") || has(BuildPagePermissionCode("dashboard", PageActionRead))
		case "users":
			readAllowed = has("admin.account.read") || has(BuildPagePermissionCode("users", PageActionRead))
		case "blogs":
			readAllowed = has(BuildPagePermissionCode("blogs", PageActionRead))
		case "admin_chat":
			readAllowed = has(BuildPagePermissionCode("admin_chat", PageActionRead))
		default:
			readAllowed = has(page.PermissionCodes.Read)
		}
		if !readAllowed {
			continue
		}

		path := strings.TrimSpace(page.Path)
		if path == "" {
			path = buildDynamicAdminPagePath(page.Key)
		}
		title := strings.TrimSpace(page.Title)
		if title == "" {
			title = page.Key
		}
		icon := strings.TrimSpace(page.Icon)
		if icon == "" {
			icon = strings.ToUpper(string([]rune(page.Key)[0]))
		}
		groupKey := strings.TrimSpace(page.GroupKey)
		if groupKey == "" {
			groupKey = "general"
		}
		groupLabel := strings.TrimSpace(page.GroupLabel)
		if groupLabel == "" {
			groupLabel = defaultMenuGroupLabel(groupKey)
		}
		groupOrder := page.GroupOrder
		if groupOrder <= 0 {
			groupOrder = defaultMenuGroupOrder(groupKey)
		}

		items = append(items, adminMenuItem{
			Key:        page.Key,
			Title:      title,
			Path:       path,
			Icon:       icon,
			Active:     page.Key == active,
			GroupKey:   groupKey,
			GroupLabel: groupLabel,
			GroupOrder: groupOrder,
			SortOrder:  page.SortOrder,
		})
	}

	return items
}

func (h *PageHandler) resolveMenuGroups(items []adminMenuItem) []adminMenuGroup {
	if len(items) == 0 {
		return []adminMenuGroup{}
	}

	groupMap := make(map[string]*adminMenuGroup, 8)
	for _, item := range items {
		key := strings.TrimSpace(item.GroupKey)
		if key == "" {
			key = "general"
		}
		group, exists := groupMap[key]
		if !exists {
			label := strings.TrimSpace(item.GroupLabel)
			if label == "" {
				label = defaultMenuGroupLabel(key)
			}
			order := item.GroupOrder
			if order <= 0 {
				order = defaultMenuGroupOrder(key)
			}
			group = &adminMenuGroup{
				Key:   key,
				Label: label,
				Order: order,
				Items: make([]adminMenuItem, 0, 4),
			}
			groupMap[key] = group
		}
		group.Items = append(group.Items, item)
	}

	groups := make([]adminMenuGroup, 0, len(groupMap))
	for _, group := range groupMap {
		sort.Slice(group.Items, func(i, j int) bool {
			if group.Items[i].SortOrder == group.Items[j].SortOrder {
				return group.Items[i].Key < group.Items[j].Key
			}
			return group.Items[i].SortOrder < group.Items[j].SortOrder
		})
		groups = append(groups, *group)
	}

	sort.Slice(groups, func(i, j int) bool {
		if groups[i].Order == groups[j].Order {
			return groups[i].Label < groups[j].Label
		}
		return groups[i].Order < groups[j].Order
	})

	return groups
}

func mustParseTemplateFiles(names ...string) *template.Template {
	paths, err := resolveTemplatePaths(names...)
	if err != nil {
		panic(err)
	}

	tpl, err := template.New("admin").Delims("[[", "]]").ParseFiles(paths...)
	if err != nil {
		panic(err)
	}

	return tpl
}

func resolveTemplatePaths(names ...string) ([]string, error) {
	baseCandidates := []string{
		filepath.Join("web", "admin", "templates"),
		filepath.Join("..", "..", "web", "admin", "templates"),
		filepath.Join("..", "..", "..", "web", "admin", "templates"),
	}

	for _, base := range baseCandidates {
		paths := make([]string, len(names))
		valid := true

		for i, name := range names {
			path := filepath.Join(base, name)
			if _, err := os.Stat(path); err != nil {
				valid = false
				break
			}
			paths[i] = path
		}

		if valid {
			return paths, nil
		}
	}

	return nil, fmt.Errorf("admin template files not found: %v", names)
}
