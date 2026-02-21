package admin

import (
	"gin_starter/pkg/authz"
	"gin_starter/pkg/response"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// Handler handles admin HTTP endpoints.
type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

// GetUsers returns users list.
// @Summary      List users (admin)
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        page query int false "page"
// @Param        limit query int false "limit"
// @Param        user_type query string false "user type filter"
// @Success      200 {object} response.Response{data=AdminUserListResponse}
// @Failure      401 {object} response.Response
// @Failure      403 {object} response.Response
// @Security     BearerAuth
// @Router       /api/admin/users [get]
func (h *Handler) GetUsers(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	userType := strings.TrimSpace(c.Query("user_type"))

	result, err := h.service.GetAllUsers(page, limit, userType)
	if err != nil {
		response.FromError(c, err)
		return
	}

	response.Success(c, result)
}

// GetUser returns user detail.
// @Summary      Get user (admin)
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        id path string true "user id"
// @Success      200 {object} response.Response
// @Failure      404 {object} response.Response
// @Security     BearerAuth
// @Router       /api/admin/users/{id} [get]
func (h *Handler) GetUser(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.BadRequest(c, "user id is required")
		return
	}

	userModel, err := h.service.GetUserByID(id)
	if err != nil {
		response.FromError(c, err)
		return
	}

	response.Success(c, gin.H{
		"id":         userModel.ID,
		"name":       userModel.Name,
		"email":      userModel.Email,
		"auth_type":  userModel.AuthType,
		"auth_level": userModel.AuthLevel,
		"status":     userModel.Status,
		"created_at": userModel.CreatedAt,
	})
}

// UpdateUserAuth updates user role and level.
// @Summary      Update user auth (super-admin)
// @Description  Updates user auth type and auth level.
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        id path string true "user id"
// @Param        request body AdminUpdateUserAuthRequest true "auth payload"
// @Success      200 {object} response.Response
// @Failure      400 {object} response.Response
// @Failure      404 {object} response.Response
// @Security     BearerAuth
// @Router       /api/admin/users/{id}/auth [put]
func (h *Handler) UpdateUserAuth(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.BadRequest(c, "user id is required")
		return
	}

	actor, ok := actorFromContext(c)
	if !ok {
		response.Unauthorized(c, "authentication context missing")
		return
	}

	var req AdminUpdateUserAuthRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request payload")
		return
	}

	if err := h.service.UpdateUserAuth(actor, id, req.AuthType, req.AuthLevel); err != nil {
		response.FromError(c, err)
		return
	}

	response.Success(c, gin.H{"message": "user auth updated"})
}

// UpdateUserProfile updates user profile.
// @Summary      Update user profile (admin)
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        id path string true "user id"
// @Param        request body AdminUpdateUserProfileRequest true "profile payload"
// @Success      200 {object} response.Response
// @Security     BearerAuth
// @Router       /api/admin/users/{id}/profile [put]
func (h *Handler) UpdateUserProfile(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.BadRequest(c, "user id is required")
		return
	}

	actor, ok := actorFromContext(c)
	if !ok {
		response.Unauthorized(c, "authentication context missing")
		return
	}

	var req AdminUpdateUserProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request payload")
		return
	}

	if err := h.service.UpdateUserProfile(actor, id, &req); err != nil {
		response.FromError(c, err)
		return
	}

	response.Success(c, gin.H{"message": "user profile updated"})
}

// UpdateUserStatus updates account status.
// @Summary      Update user status (admin)
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        id path string true "user id"
// @Param        request body AdminUpdateUserStatusRequest true "status payload"
// @Success      200 {object} response.Response
// @Security     BearerAuth
// @Router       /api/admin/users/{id}/status [put]
func (h *Handler) UpdateUserStatus(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.BadRequest(c, "user id is required")
		return
	}

	actor, ok := actorFromContext(c)
	if !ok {
		response.Unauthorized(c, "authentication context missing")
		return
	}

	var req AdminUpdateUserStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request payload")
		return
	}

	if err := h.service.UpdateUserStatus(actor, id, req.Status); err != nil {
		response.FromError(c, err)
		return
	}

	response.Success(c, gin.H{"message": "user status updated"})
}

// ResetUserPassword resets user password.
// @Summary      Reset user password (admin)
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        id path string true "user id"
// @Param        request body AdminResetUserPasswordRequest true "password payload"
// @Success      200 {object} response.Response
// @Security     BearerAuth
// @Router       /api/admin/users/{id}/password-reset [post]
func (h *Handler) ResetUserPassword(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.BadRequest(c, "user id is required")
		return
	}

	actor, ok := actorFromContext(c)
	if !ok {
		response.Unauthorized(c, "authentication context missing")
		return
	}

	var req AdminResetUserPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request payload")
		return
	}

	if err := h.service.ResetUserPassword(actor, id, req.NewPassword); err != nil {
		response.FromError(c, err)
		return
	}

	response.Success(c, gin.H{"message": "user password reset completed"})
}

// GetUserPermissions returns permissions assigned to a user.
// @Summary      Get user permissions (admin)
// @Tags         admin
// @Produce      json
// @Param        id path string true "user id"
// @Success      200 {object} response.Response
// @Security     BearerAuth
// @Router       /api/admin/users/{id}/permissions [get]
func (h *Handler) GetUserPermissions(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.BadRequest(c, "user id is required")
		return
	}

	permissions, err := h.service.GetUserPermissions(id)
	if err != nil {
		response.FromError(c, err)
		return
	}

	response.Success(c, gin.H{"permissions": permissions})
}

// UpdateUserPermissions replaces permissions assigned to a user.
// @Summary      Update user permissions (admin)
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        id path string true "user id"
// @Param        request body AdminUpdateUserPermissionsRequest true "permissions payload"
// @Success      200 {object} response.Response
// @Security     BearerAuth
// @Router       /api/admin/users/{id}/permissions [put]
func (h *Handler) UpdateUserPermissions(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.BadRequest(c, "user id is required")
		return
	}

	actor, ok := actorFromContext(c)
	if !ok {
		response.Unauthorized(c, "authentication context missing")
		return
	}

	var req AdminUpdateUserPermissionsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request payload")
		return
	}

	if err := h.service.UpdateUserPermissions(actor, id, req.Permissions); err != nil {
		response.FromError(c, err)
		return
	}

	response.Success(c, gin.H{"message": "user permissions updated"})
}

// DeleteUser deletes a user.
// @Summary      Delete user (super-admin)
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        id path string true "user id"
// @Success      200 {object} response.Response
// @Failure      404 {object} response.Response
// @Security     BearerAuth
// @Router       /api/admin/users/{id} [delete]
func (h *Handler) DeleteUser(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.BadRequest(c, "user id is required")
		return
	}

	actor, ok := actorFromContext(c)
	if !ok {
		response.Unauthorized(c, "authentication context missing")
		return
	}

	if err := h.service.DeleteUser(actor, id); err != nil {
		response.FromError(c, err)
		return
	}

	response.Success(c, gin.H{"message": "user deleted"})
}

// GetStats returns admin dashboard stats.
// @Summary      Get admin stats
// @Tags         admin
// @Accept       json
// @Produce      json
// @Success      200 {object} response.Response{data=AdminStatsResponse}
// @Security     BearerAuth
// @Router       /api/admin/stats [get]
func (h *Handler) GetStats(c *gin.Context) {
	stats, err := h.service.GetStats()
	if err != nil {
		response.FromError(c, err)
		return
	}

	response.Success(c, stats)
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

// GetAuditLogs returns admin audit logs.
// @Summary      List admin audit logs
// @Tags         admin
// @Produce      json
// @Param        page query int false "page" default(1)
// @Param        limit query int false "limit (max 100)" default(20)
// @Param        action query string false "action filter"
// @Param        actor_id query string false "actor id filter"
// @Param        target_user_id query string false "target user id filter"
// @Param        date_from query string false "start date (YYYY-MM-DD)"
// @Param        date_to query string false "end date (YYYY-MM-DD)"
// @Success      200 {object} response.Response{data=AdminAuditLogListResponse}
// @Failure      400 {object} response.Response
// @Failure      401 {object} response.Response
// @Failure      403 {object} response.Response
// @Security     BearerAuth
// @Router       /api/admin/audit-logs [get]
func (h *Handler) GetAuditLogs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	result, err := h.service.GetAuditLogs(
		page,
		limit,
		strings.TrimSpace(c.Query("action")),
		strings.TrimSpace(c.Query("actor_id")),
		strings.TrimSpace(c.Query("target_user_id")),
		strings.TrimSpace(c.Query("date_from")),
		strings.TrimSpace(c.Query("date_to")),
	)
	if err != nil {
		response.FromError(c, err)
		return
	}

	response.Success(c, result)
}

// GetBlogs returns blog list for admin console.
// @Summary      List blogs (admin)
// @Tags         admin
// @Produce      json
// @Param        page query int false "page" default(1)
// @Param        limit query int false "limit (max 100)" default(20)
// @Success      200 {object} response.Response{data=AdminBlogListResponse}
// @Failure      401 {object} response.Response
// @Failure      403 {object} response.Response
// @Security     BearerAuth
// @Router       /api/admin/blogs [get]
func (h *Handler) GetBlogs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	result, err := h.service.GetBlogs(page, limit)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.Success(c, result)
}

// CreateBlog creates a blog from admin console.
// @Summary      Create blog (admin)
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        request body AdminCreateBlogRequest true "create blog payload"
// @Success      201 {object} response.Response
// @Failure      400 {object} response.Response
// @Failure      401 {object} response.Response
// @Failure      403 {object} response.Response
// @Security     BearerAuth
// @Router       /api/admin/blogs [post]
func (h *Handler) CreateBlog(c *gin.Context) {
	actor, ok := actorFromContext(c)
	if !ok {
		response.Unauthorized(c, "authentication context missing")
		return
	}

	var req AdminCreateBlogRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request payload")
		return
	}

	created, err := h.service.CreateBlog(actor, &req)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.Created(c, gin.H{
		"message": "blog created",
		"blog":    created.ToResponse(),
	})
}

// UpdateBlog updates a blog from admin console.
// @Summary      Update blog (admin)
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        id path int true "blog id"
// @Param        request body AdminUpdateBlogRequest true "update blog payload"
// @Success      200 {object} response.Response
// @Failure      400 {object} response.Response
// @Failure      401 {object} response.Response
// @Failure      403 {object} response.Response
// @Security     BearerAuth
// @Router       /api/admin/blogs/{id} [put]
func (h *Handler) UpdateBlog(c *gin.Context) {
	actor, ok := actorFromContext(c)
	if !ok {
		response.Unauthorized(c, "authentication context missing")
		return
	}

	id, err := strconv.ParseInt(strings.TrimSpace(c.Param("id")), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "invalid blog id")
		return
	}

	var req AdminUpdateBlogRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request payload")
		return
	}

	updated, err := h.service.UpdateBlog(actor, id, &req)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.Success(c, gin.H{
		"message": "blog updated",
		"blog":    updated.ToResponse(),
	})
}

// DeleteBlog deletes a blog from admin console.
// @Summary      Delete blog (admin)
// @Tags         admin
// @Produce      json
// @Param        id path int true "blog id"
// @Success      200 {object} response.Response
// @Failure      400 {object} response.Response
// @Failure      401 {object} response.Response
// @Failure      403 {object} response.Response
// @Security     BearerAuth
// @Router       /api/admin/blogs/{id} [delete]
func (h *Handler) DeleteBlog(c *gin.Context) {
	actor, ok := actorFromContext(c)
	if !ok {
		response.Unauthorized(c, "authentication context missing")
		return
	}

	id, err := strconv.ParseInt(strings.TrimSpace(c.Param("id")), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "invalid blog id")
		return
	}

	if err := h.service.DeleteBlog(actor, id); err != nil {
		response.FromError(c, err)
		return
	}
	response.Success(c, gin.H{"message": "blog deleted"})
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
	includeDisabled := strings.EqualFold(strings.TrimSpace(c.DefaultQuery("include_disabled", "false")), "true")
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

// GetBootstrapStatus returns public super-admin bootstrap availability.
// @Summary      Get super-admin bootstrap status
// @Tags         admin
// @Produce      json
// @Success      200 {object} response.Response{data=AdminBootstrapStatusResponse}
// @Router       /api/admin/bootstrap/status [get]
func (h *Handler) GetBootstrapStatus(c *gin.Context) {
	status, err := h.service.GetBootstrapStatus()
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.Success(c, status)
}

// BootstrapSuperAdmin creates a super-admin when bootstrap is allowed.
// @Summary      Bootstrap super-admin account
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        request body AdminBootstrapSuperAdminRequest true "bootstrap payload"
// @Success      201 {object} response.Response
// @Failure      400 {object} response.Response
// @Failure      403 {object} response.Response
// @Failure      409 {object} response.Response
// @Router       /api/admin/bootstrap/super-admin [post]
func (h *Handler) BootstrapSuperAdmin(c *gin.Context) {
	var req AdminBootstrapSuperAdminRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request payload")
		return
	}

	createdUser, err := h.service.BootstrapSuperAdmin(&req, c.ClientIP())
	if err != nil {
		response.FromError(c, err)
		return
	}

	response.Created(c, gin.H{
		"message": "top-admin bootstrap completed",
		"user":    createdUser,
	})
}

func actorFromContext(c *gin.Context) (Actor, bool) {
	userID := c.GetString("user_id")
	userType := authz.NormalizeAuthType(c.GetString("user_type"))
	userLevel := c.GetInt("user_level")
	isSuperAdminVal, exists := c.Get("is_super_admin")
	if userID == "" || userType == "" || !exists {
		return Actor{}, false
	}
	isSuperAdmin, ok := isSuperAdminVal.(bool)
	if !ok {
		isSuperAdmin = authz.IsSuperAdmin(userType, userLevel)
	}

	levelPolicyEnabled := true
	if value, hasValue := c.Get("level_policy_enabled"); hasValue {
		if casted, castOK := value.(bool); castOK {
			levelPolicyEnabled = casted
		}
	}

	return Actor{
		ID:                 userID,
		UserType:           userType,
		UserLevel:          userLevel,
		IsSuperAdmin:       isSuperAdmin,
		LevelPolicyEnabled: levelPolicyEnabled,
		IP:                 c.ClientIP(),
	}, true
}
