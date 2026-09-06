package admin

import (
	"database/sql"
	"gin_starter/pkg/errors"
	"gin_starter/pkg/logger"
	"gin_starter/pkg/response"
	"gin_starter/pkg/utils"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"
)

// AdminUpdateDelegablePermissionsRequest replaces delegable allow-list.
type AdminUpdateDelegablePermissionsRequest struct {
	Permissions []string `json:"permissions"`
}

// AdminUpdateLevelPolicyRequest toggles global auth-level policy.
type AdminUpdateLevelPolicyRequest struct {
	Enabled *bool `json:"enabled"`
}

// AdminLevelPolicyResponse is global auth-level policy state.
type AdminLevelPolicyResponse struct {
	Enabled bool `json:"enabled"`
}

// Permission describes a manageable permission code.
type Permission struct {
	Code        string `json:"code"`
	Description string `json:"description"`
}

// ValidateDelegation checks all permissions are in delegable allow-list.
func ValidateDelegation(permissionCodes []string, delegable map[string]struct{}) error {
	for _, code := range permissionCodes {
		if _, ok := delegable[code]; !ok {
			return errors.New("FORBIDDEN", "permission delegation is not allowed")
		}
	}
	return nil
}

// GetPermissions returns all available permission codes.
// @Summary      List available permissions
// @Tags         admin
// @Produce      json
// @Success      200 {object} response.Response
// @Security     BearerAuth
// @Router       /api/admin/permissions [get]
func (h *Handler) GetPermissions(c *gin.Context) {
	permissions, err := h.service.GetPermissions()
	if err != nil {
		response.FromError(c, err)
		return
	}

	response.Success(c, gin.H{"permissions": permissions})
}

// GetDelegablePermissions returns delegable permission allow-list.
// @Summary      Get delegable permissions
// @Tags         admin
// @Produce      json
// @Success      200 {object} response.Response
// @Security     BearerAuth
// @Router       /api/admin/permissions/delegable [get]
func (h *Handler) GetDelegablePermissions(c *gin.Context) {
	permissions, err := h.service.GetDelegablePermissions()
	if err != nil {
		response.FromError(c, err)
		return
	}

	response.Success(c, gin.H{"permissions": permissions})
}

// UpdateDelegablePermissions updates delegable permission allow-list.
// @Summary      Update delegable permissions (super-admin)
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        request body AdminUpdateDelegablePermissionsRequest true "delegable permissions payload"
// @Success      200 {object} response.Response
// @Security     BearerAuth
// @Router       /api/admin/permissions/delegable [put]
func (h *Handler) UpdateDelegablePermissions(c *gin.Context) {
	actor, ok := actorFromContext(c)
	if !ok {
		response.Unauthorized(c, "authentication context missing")
		return
	}

	var req AdminUpdateDelegablePermissionsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request payload")
		return
	}

	if err := h.service.UpdateDelegablePermissions(actor, req.Permissions); err != nil {
		response.FromError(c, err)
		return
	}

	response.Success(c, gin.H{"message": "delegable permissions updated"})
}

// GetLevelPolicy returns global auth-level policy state.
// @Summary      Get global level policy
// @Tags         admin
// @Produce      json
// @Success      200 {object} response.Response{data=AdminLevelPolicyResponse}
// @Security     BearerAuth
// @Router       /api/admin/level-policy [get]
func (h *Handler) GetLevelPolicy(c *gin.Context) {
	result, err := h.service.GetLevelPolicy()
	if err != nil {
		response.FromError(c, err)
		return
	}

	response.Success(c, result)
}

// UpdateLevelPolicy updates global auth-level policy.
// @Summary      Update global level policy (top-admin)
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        request body AdminUpdateLevelPolicyRequest true "level policy payload"
// @Success      200 {object} response.Response
// @Security     BearerAuth
// @Router       /api/admin/level-policy [put]
func (h *Handler) UpdateLevelPolicy(c *gin.Context) {
	actor, ok := actorFromContext(c)
	if !ok {
		response.Unauthorized(c, "authentication context missing")
		return
	}

	var req AdminUpdateLevelPolicyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request payload")
		return
	}
	if req.Enabled == nil {
		response.BadRequest(c, "enabled is required")
		return
	}

	if err := h.service.UpdateLevelPolicy(actor, *req.Enabled); err != nil {
		response.FromError(c, err)
		return
	}

	response.Success(c, gin.H{"message": "level policy updated"})
}

func (s *service) GetPermissions() ([]Permission, error) {
	return s.permissionRepo.ListPermissions()
}

func (s *service) GetDelegablePermissions() ([]string, error) {
	return s.permissionRepo.ListDelegablePermissions()
}

func (s *service) UpdateDelegablePermissions(actor Actor, permissionCodes []string) error {
	if !actor.IsSuperAdmin {
		return s.denyWithAudit(actor, "", "admin.allowlist.manage", "only top-admin can manage delegable permissions")
	}

	codes := normalizePermissionCodes(permissionCodes)
	if err := s.ensureKnownPermissionCodes(codes); err != nil {
		return err
	}
	exists, err := s.permissionRepo.PermissionCodesExist(codes)
	if err != nil {
		return err
	}
	if !exists {
		unknown, unknownErr := s.findUnknownPermissionCodes(codes)
		if unknownErr == nil && len(unknown) > 0 {
			return errors.New("BAD_REQUEST", "unknown permission code in request: "+strings.Join(unknown, ", "))
		}
		return errors.New("BAD_REQUEST", "unknown permission code in request")
	}

	beforeCodes, err := s.permissionRepo.ListDelegablePermissions()
	if err != nil {
		return err
	}

	return s.db.WithTx(func(tx *sql.Tx) error {
		if err := s.permissionRepo.ReplaceDelegablePermissionsTx(tx, codes); err != nil {
			return err
		}

		return s.permissionRepo.WriteAuditLogTx(tx, &AuditLogEntry{
			ActorID: actor.ID,
			Action:  "admin.allowlist.manage",
			Status:  "success",
			IP:      actor.IP,
			BeforeData: map[string]interface{}{
				"permissions": beforeCodes,
			},
			AfterData: map[string]interface{}{
				"permissions": codes,
			},
		})
	})
}

func (s *service) GetLevelPolicy() (*AdminLevelPolicyResponse, error) {
	enabled, err := s.permissionRepo.GetLevelPolicyEnabled()
	if err != nil {
		logger.Error("failed to get level policy: %v", err)
		return nil, err
	}
	return &AdminLevelPolicyResponse{Enabled: enabled}, nil
}

func (s *service) UpdateLevelPolicy(actor Actor, enabled bool) error {
	if !actor.IsSuperAdmin {
		return s.denyWithAudit(actor, "", "admin.system.level_policy.manage", "only top-admin can change level policy")
	}

	beforeEnabled, err := s.permissionRepo.GetLevelPolicyEnabled()
	if err != nil {
		logger.Error("failed to get current level policy before update (actor=%s): %v", actor.ID, err)
		return err
	}

	if err := s.db.WithTx(func(tx *sql.Tx) error {
		if err := s.permissionRepo.SetLevelPolicyEnabledTx(tx, enabled, actor.ID); err != nil {
			logger.Error("failed to persist level policy update (actor=%s enabled=%t): %v", actor.ID, enabled, err)
			return err
		}

		if err := s.permissionRepo.WriteAuditLogTx(tx, &AuditLogEntry{
			ActorID: actor.ID,
			Action:  "admin.system.level_policy.manage",
			Status:  "success",
			IP:      actor.IP,
			BeforeData: map[string]interface{}{
				"enabled": beforeEnabled,
			},
			AfterData: map[string]interface{}{
				"enabled": enabled,
			},
		}); err != nil {
			logger.Error("failed to write level policy audit log (actor=%s enabled=%t): %v", actor.ID, enabled, err)
			return err
		}

		return nil
	}); err != nil {
		logger.Error("failed to update level policy transaction (actor=%s enabled=%t): %v", actor.ID, enabled, err)
		return err
	}
	return nil
}

func (s *service) knownPermissionDescriptionsForCodes(codes []string) map[string]string {
	if len(codes) == 0 {
		return nil
	}

	catalog := make(map[string]string, len(codes))
	core := corePermissionDescriptions()

	for _, code := range codes {
		if desc, exists := core[code]; exists {
			catalog[code] = desc
			continue
		}

		pageKey, action, ok := splitPagePermissionCode(code)
		if !ok {
			continue
		}

		page := builtInAdminPageByKey(pageKey)
		if page == nil && s.pageRepo != nil {
			pageFromDB, err := s.pageRepo.GetAdminPageByKey(pageKey)
			if err != nil {
				logger.Warn("skip page permission self-heal lookup (code=%s): %v", code, err)
			} else if pageFromDB != nil {
				page = pageFromDB
			}
		}
		if page == nil {
			continue
		}

		catalog[code] = pagePermissionDescription(page.Title, action)
	}

	return catalog
}

func (s *service) ensureKnownPermissionCodes(codes []string) error {
	if len(codes) == 0 {
		return nil
	}

	known := s.knownPermissionDescriptionsForCodes(codes)
	if len(known) == 0 {
		return nil
	}

	type permissionCodeEnsurer interface {
		EnsurePermissionCodes(codes map[string]string) error
	}
	ensurer, ok := s.permissionRepo.(permissionCodeEnsurer)
	if !ok {
		return nil
	}
	return ensurer.EnsurePermissionCodes(known)
}

func (s *service) ensureCorePermissionCatalogTx(tx *sql.Tx) error {
	type permissionCodeTxEnsurer interface {
		EnsurePermissionCodesTx(tx *sql.Tx, codes map[string]string) error
	}
	ensurer, ok := s.permissionRepo.(permissionCodeTxEnsurer)
	if !ok {
		return nil
	}
	return ensurer.EnsurePermissionCodesTx(tx, corePermissionDescriptions())
}

func (s *service) findUnknownPermissionCodes(codes []string) ([]string, error) {
	if len(codes) == 0 {
		return nil, nil
	}

	permissions, err := s.permissionRepo.ListPermissions()
	if err != nil {
		return nil, err
	}

	known := make(map[string]struct{}, len(permissions))
	for _, permission := range permissions {
		known[strings.TrimSpace(permission.Code)] = struct{}{}
	}

	unknown := make([]string, 0, len(codes))
	for _, code := range codes {
		if _, exists := known[code]; exists {
			continue
		}
		unknown = append(unknown, code)
	}
	return unknown, nil
}

func corePermissionDescriptions() map[string]string {
	return map[string]string{
		"admin.stats.read":                 "Read admin dashboard stats",
		"admin.account.read":               "Read user/admin accounts",
		"admin.audit.read":                 "Read admin audit logs",
		"admin.account.profile.update":     "Update user profile fields",
		"admin.account.status.update":      "Update user account status",
		"admin.account.password.reset":     "Reset user password",
		"admin.account.permission.manage":  "Manage assigned permissions",
		"admin.account.level.manage":       "Manage auth type/level",
		"admin.account.delete":             "Delete user account",
		"admin.allowlist.manage":           "Manage delegable permission allowlist",
		"admin.page.manage":                "Manage admin page catalog",
		"admin.system.level_policy.manage": "Manage global auth level policy",
	}
}

func splitPagePermissionCode(code string) (pageKey string, action string, ok bool) {
	parts := strings.Split(strings.TrimSpace(code), ".")
	if len(parts) != 4 {
		return "", "", false
	}
	if parts[0] != "admin" || parts[1] != "page" {
		return "", "", false
	}
	key, keyOK := NormalizePageKey(parts[2])
	if !keyOK {
		return "", "", false
	}
	action = strings.TrimSpace(parts[3])
	if !IsValidPageAction(action) {
		return "", "", false
	}
	return key, action, true
}

func normalizePermissionCodes(permissionCodes []string) []string {
	set := make(map[string]struct{}, len(permissionCodes))
	for _, code := range permissionCodes {
		trimmed := strings.TrimSpace(code)
		if !utils.HasText(trimmed) {
			continue
		}
		set[trimmed] = struct{}{}
	}

	codes := make([]string, 0, len(set))
	for code := range set {
		codes = append(codes, code)
	}
	sort.Strings(codes)
	return codes
}
