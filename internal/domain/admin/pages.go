package admin

import (
	"database/sql"
	"fmt"
	"gin_starter/pkg/authz"
	"gin_starter/pkg/errors"
	"gin_starter/pkg/logger"
	"gin_starter/pkg/response"
	"gin_starter/pkg/utils"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	PermissionPageManage = "admin.page.manage"

	PageActionRead   = "read"
	PageActionCreate = "create"
	PageActionUpdate = "update"
	PageActionDelete = "delete"
)

var (
	pageKeyPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{1,47}$`)
	pageActions    = []string{PageActionRead, PageActionCreate, PageActionUpdate, PageActionDelete}
)

// AdminPagePermissionCodes contains generated permission codes for one page.
type AdminPagePermissionCodes struct {
	Read   string `json:"read"`
	Create string `json:"create"`
	Update string `json:"update"`
	Delete string `json:"delete"`
}

// AdminPage represents a dynamic admin page registry item.
type AdminPage struct {
	Key             string                   `json:"page_key"`
	Title           string                   `json:"title"`
	Path            string                   `json:"path"`
	Description     string                   `json:"description"`
	GroupKey        string                   `json:"group_key"`
	GroupLabel      string                   `json:"group_label"`
	GroupOrder      int                      `json:"group_order"`
	VisibleRoles    []string                 `json:"visible_roles,omitempty"`
	Icon            string                   `json:"icon"`
	SortOrder       int                      `json:"sort_order"`
	Enabled         bool                     `json:"enabled"`
	Builtin         bool                     `json:"builtin"`
	PermissionCodes AdminPagePermissionCodes `json:"permission_codes"`
	CreatedBy       string                   `json:"created_by,omitempty"`
	CreatedAt       time.Time                `json:"created_at"`
	UpdatedAt       time.Time                `json:"updated_at"`
}

// AdminPageRouteSpec defines a route-driven page registration spec.
type AdminPageRouteSpec struct {
	PageKey      string   `json:"page_key"`
	Path         string   `json:"path"`
	Title        string   `json:"title"`
	Description  string   `json:"description"`
	GroupKey     string   `json:"group_key"`
	GroupLabel   string   `json:"group_label"`
	GroupOrder   int      `json:"group_order"`
	VisibleRoles []string `json:"visible_roles"`
	Icon         string   `json:"icon"`
	SortOrder    int      `json:"sort_order"`
}

// AdminCreatePageRequest creates a dynamic admin page.
type AdminCreatePageRequest struct {
	PageKey      string   `json:"page_key"`
	Title        string   `json:"title"`
	Description  string   `json:"description"`
	GroupKey     string   `json:"group_key"`
	GroupLabel   string   `json:"group_label"`
	GroupOrder   int      `json:"group_order"`
	VisibleRoles []string `json:"visible_roles"`
	Icon         string   `json:"icon"`
	SortOrder    int      `json:"sort_order"`
	Enabled      *bool    `json:"enabled"`
}

// AdminUpdatePageRequest updates dynamic admin page metadata.
type AdminUpdatePageRequest struct {
	Title        *string   `json:"title"`
	Description  *string   `json:"description"`
	GroupKey     *string   `json:"group_key"`
	GroupLabel   *string   `json:"group_label"`
	GroupOrder   *int      `json:"group_order"`
	VisibleRoles *[]string `json:"visible_roles"`
	Icon         *string   `json:"icon"`
	SortOrder    *int      `json:"sort_order"`
	Enabled      *bool     `json:"enabled"`
}

func NormalizePageKey(raw string) (string, bool) {
	key := utils.TrimLower(raw)
	if !pageKeyPattern.MatchString(key) {
		return "", false
	}
	return key, true
}

func IsValidPageAction(action string) bool {
	switch utils.TrimLower(action) {
	case PageActionRead, PageActionCreate, PageActionUpdate, PageActionDelete:
		return true
	default:
		return false
	}
}

func BuildPagePermissionCode(pageKey, action string) string {
	return fmt.Sprintf("admin.page.%s.%s", pageKey, action)
}

func BuildPagePermissionCodes(pageKey string) []string {
	codes := make([]string, 0, len(pageActions))
	for _, action := range pageActions {
		codes = append(codes, BuildPagePermissionCode(pageKey, action))
	}
	return codes
}

// GetAdminPages returns dynamic admin page catalog.
// @Summary      List admin pages
// @Tags         admin
// @Produce      json
// @Param        include_disabled query bool false "include disabled pages"
// @Success      200 {object} response.Response
// @Failure      401 {object} response.Response
// @Failure      403 {object} response.Response
// @Security     BearerAuth
// @Router       /api/admin/pages [get]
func (h *Handler) GetAdminPages(c *gin.Context) {
	includeDisabled := utils.TrimLower(c.DefaultQuery("include_disabled", "false")) == "true"
	pages, err := h.service.GetAdminPages(includeDisabled)
	if err != nil {
		response.FromError(c, err)
		return
	}

	response.Success(c, gin.H{"pages": pages})
}

// CreateAdminPage creates a dynamic admin page and page-scoped permissions.
// @Summary      Create admin page
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        request body AdminCreatePageRequest true "admin page payload"
// @Success      201 {object} response.Response
// @Failure      400 {object} response.Response
// @Failure      401 {object} response.Response
// @Failure      403 {object} response.Response
// @Security     BearerAuth
// @Router       /api/admin/pages [post]
func (h *Handler) CreateAdminPage(c *gin.Context) {
	actor, ok := actorFromContext(c)
	if !ok {
		response.Unauthorized(c, "authentication context missing")
		return
	}

	var req AdminCreatePageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request payload")
		return
	}

	page, err := h.service.CreateAdminPage(actor, &req)
	if err != nil {
		response.FromError(c, err)
		return
	}

	response.Created(c, gin.H{
		"message": "admin page created",
		"page":    page,
	})
}

// UpdateAdminPage updates a dynamic admin page metadata.
// @Summary      Update admin page
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        page_key path string true "page key"
// @Param        request body AdminUpdatePageRequest true "admin page payload"
// @Success      200 {object} response.Response
// @Failure      400 {object} response.Response
// @Failure      401 {object} response.Response
// @Failure      403 {object} response.Response
// @Security     BearerAuth
// @Router       /api/admin/pages/{page_key} [put]
func (h *Handler) UpdateAdminPage(c *gin.Context) {
	actor, ok := actorFromContext(c)
	if !ok {
		response.Unauthorized(c, "authentication context missing")
		return
	}

	pageKey := c.Param("page_key")
	if pageKey == "" {
		response.BadRequest(c, "page_key is required")
		return
	}

	var req AdminUpdatePageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request payload")
		return
	}

	page, err := h.service.UpdateAdminPage(actor, pageKey, &req)
	if err != nil {
		response.FromError(c, err)
		return
	}

	response.Success(c, gin.H{
		"message": "admin page updated",
		"page":    page,
	})
}

// DeleteAdminPage deletes a dynamic admin page and related page-scoped permissions.
// @Summary      Delete admin page
// @Tags         admin
// @Produce      json
// @Param        page_key path string true "page key"
// @Success      200 {object} response.Response
// @Failure      401 {object} response.Response
// @Failure      403 {object} response.Response
// @Security     BearerAuth
// @Router       /api/admin/pages/{page_key} [delete]
func (h *Handler) DeleteAdminPage(c *gin.Context) {
	actor, ok := actorFromContext(c)
	if !ok {
		response.Unauthorized(c, "authentication context missing")
		return
	}

	pageKey := c.Param("page_key")
	if pageKey == "" {
		response.BadRequest(c, "page_key is required")
		return
	}

	if err := h.service.DeleteAdminPage(actor, pageKey); err != nil {
		response.FromError(c, err)
		return
	}

	response.Success(c, gin.H{"message": "admin page deleted"})
}

func (s *service) GetAdminPages(includeDisabled bool) ([]AdminPage, error) {
	if s.pageRepo == nil {
		return defaultAdminPages(includeDisabled), nil
	}

	pages, err := s.pageRepo.ListAdminPages(includeDisabled)
	if err != nil {
		if isMissingTableErrorForBootstrap(err) {
			return defaultAdminPages(includeDisabled), nil
		}
		return nil, err
	}

	if len(pages) == 0 {
		return defaultAdminPages(includeDisabled), nil
	}
	return pages, nil
}

// SyncAdminPagesFromRouteSpecs creates or updates route-managed pages from route specs.
func (s *service) SyncAdminPagesFromRouteSpecs(routeSpecs []AdminPageRouteSpec) error {
	if s.pageRepo == nil || s.db == nil {
		return nil
	}

	normalized := normalizeAdminPageRouteSpecs(routeSpecs)
	if len(normalized) == 0 {
		return nil
	}

	existingPages, err := s.pageRepo.ListAdminPages(true)
	if err != nil {
		if isMissingTableErrorForBootstrap(err) {
			if ensureErr := s.pageRepo.EnsureAdminPageSchema(); ensureErr != nil {
				return ensureErr
			}
			existingPages, err = s.pageRepo.ListAdminPages(true)
			if err != nil && !isMissingTableErrorForBootstrap(err) {
				return err
			}
			if err != nil {
				existingPages = []AdminPage{}
			}
		} else {
			return err
		}
	}

	existingByKey := make(map[string]AdminPage, len(existingPages))
	for _, page := range existingPages {
		existingByKey[page.Key] = page
	}

	return s.db.WithTx(func(tx *sql.Tx) error {
		if err := s.ensureCorePermissionCatalogTx(tx); err != nil {
			return err
		}

		managedKeys := make(map[string]struct{}, len(normalized))
		for _, spec := range normalized {
			managedKeys[spec.PageKey] = struct{}{}

			existing, found := existingByKey[spec.PageKey]
			if !found {
				page := &AdminPage{
					Key:          spec.PageKey,
					Title:        spec.Title,
					Path:         spec.Path,
					Description:  spec.Description,
					GroupKey:     spec.GroupKey,
					GroupLabel:   spec.GroupLabel,
					GroupOrder:   spec.GroupOrder,
					VisibleRoles: spec.VisibleRoles,
					Icon:         spec.Icon,
					SortOrder:    spec.SortOrder,
					Enabled:      true,
					Builtin:      true,
					CreatedBy:    "system",
				}
				if err := s.pageRepo.CreateAdminPageTx(tx, page); err != nil {
					return err
				}
				if err := s.pageRepo.EnsurePagePermissionCodesTx(tx, page); err != nil {
					return err
				}
				s.syncGrantPageReadPermissionTx(tx, spec.PageKey)
				continue
			}

			changed := false
			if existing.Path != spec.Path {
				existing.Path = spec.Path
				changed = true
			}
			if !utils.HasText(existing.Title) {
				existing.Title = spec.Title
				changed = true
			}
			if !utils.HasText(existing.Description) && utils.HasText(spec.Description) {
				existing.Description = spec.Description
				changed = true
			}
			if !utils.HasText(existing.GroupKey) {
				existing.GroupKey = spec.GroupKey
				changed = true
			}
			if !utils.HasText(existing.GroupLabel) {
				existing.GroupLabel = spec.GroupLabel
				changed = true
			}
			if existing.GroupOrder <= 0 {
				existing.GroupOrder = spec.GroupOrder
				changed = true
			}
			if len(existing.VisibleRoles) == 0 && len(spec.VisibleRoles) > 0 {
				existing.VisibleRoles = append([]string(nil), spec.VisibleRoles...)
				changed = true
			}
			if !utils.HasText(existing.Icon) && utils.HasText(spec.Icon) {
				existing.Icon = spec.Icon
				changed = true
			}
			if existing.SortOrder <= 0 {
				existing.SortOrder = spec.SortOrder
				changed = true
			}
			if !existing.Builtin {
				existing.Builtin = true
				changed = true
			}
			if !existing.Enabled {
				existing.Enabled = true
				changed = true
			}

			if changed {
				if err := s.pageRepo.UpdateAdminPageTx(tx, &existing); err != nil {
					return err
				}
			}
			if err := s.pageRepo.EnsurePagePermissionCodesTx(tx, &existing); err != nil {
				return err
			}
			s.syncGrantPageReadPermissionTx(tx, spec.PageKey)
		}

		for _, page := range existingPages {
			if !page.Builtin {
				continue
			}
			if _, exists := managedKeys[page.Key]; exists {
				continue
			}
			if !page.Enabled {
				continue
			}

			page.Enabled = false
			if err := s.pageRepo.UpdateAdminPageTx(tx, &page); err != nil {
				return err
			}
		}

		return nil
	})
}

func (s *service) GetAdminPageByKey(pageKey string) (*AdminPage, error) {
	key, ok := NormalizePageKey(pageKey)
	if !ok {
		return nil, errors.New("BAD_REQUEST", "invalid page_key format")
	}

	if builtIn := builtInAdminPageByKey(key); builtIn != nil {
		return builtIn, nil
	}

	if s.pageRepo == nil {
		return nil, nil
	}

	page, err := s.pageRepo.GetAdminPageByKey(key)
	if err != nil {
		if isMissingTableErrorForBootstrap(err) {
			return nil, nil
		}
		return nil, err
	}
	return page, nil
}

func (s *service) CreateAdminPage(actor Actor, req *AdminCreatePageRequest) (*AdminPage, error) {
	if req == nil {
		return nil, errors.New("BAD_REQUEST", "request body is required")
	}
	if s.pageRepo == nil {
		return nil, errors.New("DATABASE_ERROR", "admin page repository is not available")
	}

	key, ok := NormalizePageKey(req.PageKey)
	if !ok {
		return nil, errors.New("BAD_REQUEST", "page_key must match [a-z0-9][a-z0-9_-]{1,47}")
	}
	if isReservedAdminPageKey(key) {
		return nil, errors.New("BAD_REQUEST", "reserved page_key")
	}

	title := strings.TrimSpace(req.Title)
	if title == "" {
		return nil, errors.New("BAD_REQUEST", "title is required")
	}
	if len(title) > 120 {
		return nil, errors.New("BAD_REQUEST", "title must be <= 120 chars")
	}

	description := strings.TrimSpace(req.Description)
	if len(description) > 255 {
		return nil, errors.New("BAD_REQUEST", "description must be <= 255 chars")
	}

	groupKey := normalizePageGroupKey(req.GroupKey)
	if groupKey == "" {
		groupKey = "custom"
	}
	groupLabel := normalizePageGroupLabel(req.GroupLabel, groupKey)
	if groupLabel == "" {
		groupLabel = defaultMenuGroupLabel(groupKey)
	}
	groupOrder := req.GroupOrder
	if groupOrder <= 0 {
		groupOrder = defaultMenuGroupOrder(groupKey)
	}
	visibleRoles := normalizeVisibleRoles(req.VisibleRoles)

	icon := strings.TrimSpace(req.Icon)
	if len(icon) > 40 {
		return nil, errors.New("BAD_REQUEST", "icon must be <= 40 chars")
	}

	sortOrder := req.SortOrder
	if sortOrder <= 0 {
		sortOrder = 100
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	existing, err := s.pageRepo.GetAdminPageByKey(key)
	if err != nil {
		if !isMissingTableErrorForBootstrap(err) {
			return nil, err
		}
	} else if existing != nil {
		return nil, errors.New("CONFLICT", "page_key already exists")
	}

	page := &AdminPage{
		Key:          key,
		Title:        title,
		Path:         buildDynamicAdminPagePath(key),
		Description:  description,
		GroupKey:     groupKey,
		GroupLabel:   groupLabel,
		GroupOrder:   groupOrder,
		VisibleRoles: visibleRoles,
		Icon:         icon,
		SortOrder:    sortOrder,
		Enabled:      enabled,
		Builtin:      false,
		CreatedBy:    actor.ID,
		PermissionCodes: AdminPagePermissionCodes{
			Read:   BuildPagePermissionCode(key, PageActionRead),
			Create: BuildPagePermissionCode(key, PageActionCreate),
			Update: BuildPagePermissionCode(key, PageActionUpdate),
			Delete: BuildPagePermissionCode(key, PageActionDelete),
		},
	}

	if err := s.db.WithTx(func(tx *sql.Tx) error {
		if err := s.pageRepo.CreateAdminPageTx(tx, page); err != nil {
			return err
		}
		if err := s.pageRepo.EnsurePagePermissionCodesTx(tx, page); err != nil {
			return err
		}
		return s.permissionRepo.WriteAuditLogTx(tx, &AuditLogEntry{
			ActorID: actor.ID,
			Action:  PermissionPageManage,
			Status:  "success",
			Message: "create_page",
			IP:      actor.IP,
			AfterData: map[string]interface{}{
				"page_key":    page.Key,
				"title":       page.Title,
				"path":        page.Path,
				"group_key":   page.GroupKey,
				"group_label": page.GroupLabel,
				"enabled":     page.Enabled,
			},
		})
	}); err != nil {
		return nil, err
	}

	created, err := s.pageRepo.GetAdminPageByKey(key)
	if err != nil {
		return nil, err
	}
	if created == nil {
		return page, nil
	}
	return created, nil
}

func (s *service) UpdateAdminPage(actor Actor, pageKey string, req *AdminUpdatePageRequest) (*AdminPage, error) {
	if req == nil {
		return nil, errors.New("BAD_REQUEST", "request body is required")
	}
	if s.pageRepo == nil {
		return nil, errors.New("DATABASE_ERROR", "admin page repository is not available")
	}

	key, ok := NormalizePageKey(pageKey)
	if !ok {
		return nil, errors.New("BAD_REQUEST", "invalid page_key format")
	}

	page, err := s.pageRepo.GetAdminPageByKey(key)
	if err != nil {
		if isMissingTableErrorForBootstrap(err) {
			return nil, errors.New("NOT_FOUND", "admin page not found")
		}
		return nil, err
	}
	fromRepository := page != nil
	if page == nil {
		page = builtInAdminPageByKey(key)
	}
	if page == nil {
		return nil, errors.New("NOT_FOUND", "admin page not found")
	}

	before := map[string]interface{}{
		"title":         page.Title,
		"description":   page.Description,
		"group_key":     page.GroupKey,
		"group_label":   page.GroupLabel,
		"group_order":   page.GroupOrder,
		"visible_roles": page.VisibleRoles,
		"icon":          page.Icon,
		"sort_order":    page.SortOrder,
		"enabled":       page.Enabled,
	}

	changed := false
	if req.Title != nil {
		title := strings.TrimSpace(*req.Title)
		if title == "" {
			return nil, errors.New("BAD_REQUEST", "title cannot be empty")
		}
		if len(title) > 120 {
			return nil, errors.New("BAD_REQUEST", "title must be <= 120 chars")
		}
		page.Title = title
		changed = true
	}
	if req.Description != nil {
		description := strings.TrimSpace(*req.Description)
		if len(description) > 255 {
			return nil, errors.New("BAD_REQUEST", "description must be <= 255 chars")
		}
		page.Description = description
		changed = true
	}
	if req.GroupKey != nil {
		groupKey := normalizePageGroupKey(*req.GroupKey)
		if groupKey == "" {
			return nil, errors.New("BAD_REQUEST", "group_key must match [a-z0-9][a-z0-9_-]{1,47}")
		}
		page.GroupKey = groupKey
		if !utils.HasText(page.GroupLabel) {
			page.GroupLabel = defaultMenuGroupLabel(groupKey)
		}
		if page.GroupOrder <= 0 {
			page.GroupOrder = defaultMenuGroupOrder(groupKey)
		}
		changed = true
	}
	if req.GroupLabel != nil {
		groupLabel := normalizePageGroupLabel(*req.GroupLabel, page.GroupKey)
		if groupLabel == "" {
			groupLabel = defaultMenuGroupLabel(page.GroupKey)
		}
		page.GroupLabel = groupLabel
		changed = true
	}
	if req.GroupOrder != nil {
		if *req.GroupOrder < 1 {
			return nil, errors.New("BAD_REQUEST", "group_order must be >= 1")
		}
		page.GroupOrder = *req.GroupOrder
		changed = true
	}
	if req.VisibleRoles != nil {
		page.VisibleRoles = normalizeVisibleRoles(*req.VisibleRoles)
		changed = true
	}
	if req.Icon != nil {
		icon := strings.TrimSpace(*req.Icon)
		if len(icon) > 40 {
			return nil, errors.New("BAD_REQUEST", "icon must be <= 40 chars")
		}
		page.Icon = icon
		changed = true
	}
	if req.SortOrder != nil {
		if *req.SortOrder < 1 {
			return nil, errors.New("BAD_REQUEST", "sort_order must be >= 1")
		}
		page.SortOrder = *req.SortOrder
		changed = true
	}
	if req.Enabled != nil {
		page.Enabled = *req.Enabled
		changed = true
	}

	if !changed {
		return page, nil
	}

	if err := s.db.WithTx(func(tx *sql.Tx) error {
		if fromRepository {
			if err := s.pageRepo.UpdateAdminPageTx(tx, page); err != nil {
				return err
			}
		}
		if err := s.pageRepo.EnsurePagePermissionCodesTx(tx, page); err != nil {
			return err
		}
		return s.permissionRepo.WriteAuditLogTx(tx, &AuditLogEntry{
			ActorID: actor.ID,
			Action:  PermissionPageManage,
			Status:  "success",
			Message: "update_page",
			IP:      actor.IP,
			BeforeData: map[string]interface{}{
				"page_key": key,
				"title":    before["title"],
				"enabled":  before["enabled"],
			},
			AfterData: map[string]interface{}{
				"page_key":    key,
				"title":       page.Title,
				"group_key":   page.GroupKey,
				"group_label": page.GroupLabel,
				"group_order": page.GroupOrder,
				"enabled":     page.Enabled,
			},
		})
	}); err != nil {
		return nil, err
	}

	if !fromRepository {
		return page, nil
	}
	updated, err := s.pageRepo.GetAdminPageByKey(key)
	if err != nil {
		return nil, err
	}
	if updated == nil {
		return page, nil
	}
	return updated, nil
}

func (s *service) DeleteAdminPage(actor Actor, pageKey string) error {
	if s.pageRepo == nil {
		return errors.New("DATABASE_ERROR", "admin page repository is not available")
	}

	key, ok := NormalizePageKey(pageKey)
	if !ok {
		return errors.New("BAD_REQUEST", "invalid page_key format")
	}
	if builtInAdminPageByKey(key) != nil {
		return errors.New("FORBIDDEN", "built-in pages cannot be deleted")
	}

	page, err := s.pageRepo.GetAdminPageByKey(key)
	if err != nil {
		if isMissingTableErrorForBootstrap(err) {
			return errors.New("NOT_FOUND", "admin page not found")
		}
		return err
	}
	if page == nil {
		return errors.New("NOT_FOUND", "admin page not found")
	}

	return s.db.WithTx(func(tx *sql.Tx) error {
		if err := s.pageRepo.DeletePagePermissionCodesTx(tx, key); err != nil {
			return err
		}
		if err := s.pageRepo.DeleteAdminPageTx(tx, key); err != nil {
			return err
		}
		return s.permissionRepo.WriteAuditLogTx(tx, &AuditLogEntry{
			ActorID: actor.ID,
			Action:  PermissionPageManage,
			Status:  "success",
			Message: "delete_page",
			IP:      actor.IP,
			BeforeData: map[string]interface{}{
				"page_key": page.Key,
				"title":    page.Title,
				"path":     page.Path,
			},
		})
	})
}

func (s *service) syncGrantPageReadPermissionTx(tx *sql.Tx, pageKey string) {
	sourceCode := defaultPageReadGrantSource(pageKey)
	if sourceCode == "" {
		return
	}

	targetCode := BuildPagePermissionCode(pageKey, PageActionRead)
	if err := s.permissionRepo.GrantPermissionFromExistingTx(tx, targetCode, sourceCode); err != nil {
		// Permission auto-copy is best-effort; page registry sync should still succeed.
		logger.Warn("skip read permission auto-copy for page %s: %v", pageKey, err)
	}
}

func buildDynamicAdminPagePath(pageKey string) string {
	return "/admin/p/" + pageKey
}

func builtInAdminPages() []AdminPage {
	return []AdminPage{
		{
			Key:          "dashboard",
			Title:        "대시보드",
			Path:         "/admin/dashboard",
			Description:  "운영 지표 대시보드",
			GroupKey:     "core",
			GroupLabel:   "운영",
			GroupOrder:   10,
			VisibleRoles: []string{authz.AuthTypeTopAdmin, authz.AuthTypeAdmin, authz.AuthTypeManager, authz.AuthTypeGuest},
			Icon:         "D",
			SortOrder:    10,
			Enabled:      true,
			Builtin:      true,
			PermissionCodes: AdminPagePermissionCodes{
				Read:   BuildPagePermissionCode("dashboard", PageActionRead),
				Create: BuildPagePermissionCode("dashboard", PageActionCreate),
				Update: BuildPagePermissionCode("dashboard", PageActionUpdate),
				Delete: BuildPagePermissionCode("dashboard", PageActionDelete),
			},
		},
		{
			Key:          "users",
			Title:        "사용자 관리",
			Path:         "/admin/users",
			Description:  "사용자/권한 관리",
			GroupKey:     "core",
			GroupLabel:   "운영",
			GroupOrder:   10,
			VisibleRoles: []string{authz.AuthTypeTopAdmin, authz.AuthTypeAdmin},
			Icon:         "U",
			SortOrder:    20,
			Enabled:      true,
			Builtin:      true,
			PermissionCodes: AdminPagePermissionCodes{
				Read:   BuildPagePermissionCode("users", PageActionRead),
				Create: BuildPagePermissionCode("users", PageActionCreate),
				Update: BuildPagePermissionCode("users", PageActionUpdate),
				Delete: BuildPagePermissionCode("users", PageActionDelete),
			},
		},
		{
			Key:          "blogs",
			Title:        "블로그 관리",
			Path:         "/admin/blogs",
			Description:  "게시글 생성/수정/삭제 및 작성자 관리",
			GroupKey:     "content",
			GroupLabel:   "콘텐츠",
			GroupOrder:   20,
			VisibleRoles: []string{authz.AuthTypeTopAdmin, authz.AuthTypeAdmin, authz.AuthTypeManager},
			Icon:         "B",
			SortOrder:    30,
			Enabled:      true,
			Builtin:      true,
			PermissionCodes: AdminPagePermissionCodes{
				Read:   BuildPagePermissionCode("blogs", PageActionRead),
				Create: BuildPagePermissionCode("blogs", PageActionCreate),
				Update: BuildPagePermissionCode("blogs", PageActionUpdate),
				Delete: BuildPagePermissionCode("blogs", PageActionDelete),
			},
		},
		{
			Key:          "admin_chat",
			Title:        "관리자 채팅",
			Path:         "/admin/chat",
			Description:  "관리자 전용 실시간 채팅",
			GroupKey:     "communication",
			GroupLabel:   "커뮤니케이션",
			GroupOrder:   30,
			VisibleRoles: []string{authz.AuthTypeTopAdmin, authz.AuthTypeAdmin, authz.AuthTypeManager, authz.AuthTypeGuest},
			Icon:         "C",
			SortOrder:    40,
			Enabled:      true,
			Builtin:      true,
			PermissionCodes: AdminPagePermissionCodes{
				Read:   BuildPagePermissionCode("admin_chat", PageActionRead),
				Create: BuildPagePermissionCode("admin_chat", PageActionCreate),
				Update: BuildPagePermissionCode("admin_chat", PageActionUpdate),
				Delete: BuildPagePermissionCode("admin_chat", PageActionDelete),
			},
		},
	}
}

func builtInAdminPageByKey(pageKey string) *AdminPage {
	for _, page := range builtInAdminPages() {
		if page.Key == pageKey {
			cloned := page
			return &cloned
		}
	}
	return nil
}

func defaultAdminPages(includeDisabled bool) []AdminPage {
	pages := builtInAdminPages()
	if includeDisabled {
		return pages
	}

	filtered := make([]AdminPage, 0, len(pages))
	for _, page := range pages {
		if page.Enabled {
			filtered = append(filtered, page)
		}
	}
	return filtered
}

func normalizeAdminPageRouteSpecs(routeSpecs []AdminPageRouteSpec) []AdminPageRouteSpec {
	if len(routeSpecs) == 0 {
		return nil
	}

	normalized := make([]AdminPageRouteSpec, 0, len(routeSpecs))
	seen := make(map[string]struct{}, len(routeSpecs))

	for _, spec := range routeSpecs {
		key, ok := NormalizePageKey(spec.PageKey)
		if !ok {
			logger.Warn("skip admin route sync spec due to invalid page key: %q", spec.PageKey)
			continue
		}
		if _, exists := seen[key]; exists {
			continue
		}

		path := normalizeAdminPageRoutePath(spec.Path)
		if path == "" {
			logger.Warn("skip admin route sync spec due to invalid path (key=%s path=%q)", key, spec.Path)
			continue
		}

		title := strings.TrimSpace(spec.Title)
		if title == "" {
			title = key
		}

		description := strings.TrimSpace(spec.Description)
		groupKey := normalizePageGroupKey(spec.GroupKey)
		if groupKey == "" {
			groupKey = defaultPageGroupKey(key)
		}
		groupLabel := normalizePageGroupLabel(spec.GroupLabel, groupKey)
		if groupLabel == "" {
			groupLabel = defaultMenuGroupLabel(groupKey)
		}
		groupOrder := spec.GroupOrder
		if groupOrder <= 0 {
			groupOrder = defaultMenuGroupOrder(groupKey)
		}
		visibleRoles := normalizeVisibleRoles(spec.VisibleRoles)
		if len(visibleRoles) == 0 {
			visibleRoles = defaultPageVisibleRoles(key)
		}

		icon := strings.TrimSpace(spec.Icon)
		if icon == "" {
			icon = strings.ToUpper(string([]rune(key)[0]))
		}
		if len(icon) > 40 {
			icon = icon[:40]
		}

		sortOrder := spec.SortOrder
		if sortOrder <= 0 {
			sortOrder = 100
		}

		normalized = append(normalized, AdminPageRouteSpec{
			PageKey:      key,
			Path:         path,
			Title:        title,
			Description:  description,
			GroupKey:     groupKey,
			GroupLabel:   groupLabel,
			GroupOrder:   groupOrder,
			VisibleRoles: visibleRoles,
			Icon:         icon,
			SortOrder:    sortOrder,
		})
		seen[key] = struct{}{}
	}

	return normalized
}

func normalizeAdminPageRoutePath(path string) string {
	normalized := strings.TrimSpace(path)
	if normalized == "" {
		return ""
	}
	if !strings.HasPrefix(normalized, "/admin") {
		return ""
	}
	if normalized != "/admin" {
		normalized = strings.TrimSuffix(normalized, "/")
	}
	if normalized == "" {
		return ""
	}
	return normalized
}

func normalizePageGroupKey(raw string) string {
	key := utils.TrimLower(raw)
	if key == "" {
		return ""
	}
	if normalized, ok := NormalizePageKey(key); ok {
		return normalized
	}
	return ""
}

func normalizePageGroupLabel(raw string, groupKey string) string {
	label := strings.TrimSpace(raw)
	if label == "" {
		return defaultMenuGroupLabel(groupKey)
	}
	if len([]rune(label)) > 100 {
		runes := []rune(label)
		return string(runes[:100])
	}
	return label
}

func normalizeVisibleRoles(roles []string) []string {
	if len(roles) == 0 {
		return []string{}
	}
	set := make(map[string]struct{}, len(roles))
	for _, role := range roles {
		normalized := authz.NormalizeAuthType(role)
		if normalized == "" {
			continue
		}
		set[normalized] = struct{}{}
	}
	out := make([]string, 0, len(set))
	for role := range set {
		out = append(out, role)
	}
	sort.Strings(out)
	return out
}

func defaultPageVisibleRoles(pageKey string) []string {
	switch pageKey {
	case "users":
		return []string{authz.AuthTypeTopAdmin, authz.AuthTypeAdmin}
	case "blogs":
		return []string{authz.AuthTypeTopAdmin, authz.AuthTypeAdmin, authz.AuthTypeManager}
	case "dashboard", "admin_chat":
		return []string{authz.AuthTypeTopAdmin, authz.AuthTypeAdmin, authz.AuthTypeManager, authz.AuthTypeGuest}
	default:
		return []string{authz.AuthTypeTopAdmin, authz.AuthTypeAdmin}
	}
}

func defaultPageGroupKey(pageKey string) string {
	switch pageKey {
	case "dashboard", "users":
		return "core"
	case "blogs":
		return "content"
	case "admin_chat":
		return "communication"
	default:
		return "custom"
	}
}

func defaultMenuGroupLabel(groupKey string) string {
	switch groupKey {
	case "core":
		return "운영"
	case "content":
		return "콘텐츠"
	case "communication":
		return "커뮤니케이션"
	case "settings":
		return "설정"
	default:
		return "기타"
	}
}

func defaultMenuGroupOrder(groupKey string) int {
	switch groupKey {
	case "core":
		return 10
	case "content":
		return 20
	case "communication":
		return 30
	case "settings":
		return 40
	default:
		return 90
	}
}

func defaultPageReadGrantSource(pageKey string) string {
	switch pageKey {
	case "dashboard":
		return "admin.stats.read"
	case "users", "blogs", "admin_chat":
		return "admin.account.read"
	default:
		return ""
	}
}

func isReservedAdminPageKey(pageKey string) bool {
	switch pageKey {
	case "admin", "login", "logout", "users", "blogs", "admin_chat", "chat", "bootstrap", "p":
		return true
	default:
		return false
	}
}
