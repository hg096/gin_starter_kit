package admin

import (
	"database/sql"
	"gin_starter/internal/domain/user"
	"gin_starter/pkg/authz"
	"gin_starter/pkg/errors"
	"gin_starter/pkg/logger"
	"gin_starter/pkg/response"
	"gin_starter/pkg/utils"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// AdminUserListRequest is user list request payload.
type AdminUserListRequest struct {
	Page     int    `json:"page"`
	Limit    int    `json:"limit"`
	UserType string `json:"user_type"`
}

// AdminUserListResponse is user list response payload.
type AdminUserListResponse struct {
	Users []user.User `json:"users"`
	Total int64       `json:"total"`
	Page  int         `json:"page"`
	Limit int         `json:"limit"`
}

// AdminUpdateUserAuthRequest updates user auth type/level.
type AdminUpdateUserAuthRequest struct {
	AuthType  string `json:"auth_type"`
	AuthLevel int    `json:"auth_level"`
}

// AdminUpdateUserProfileRequest updates basic profile fields.
type AdminUpdateUserProfileRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

// AdminUpdateUserStatusRequest updates account status.
type AdminUpdateUserStatusRequest struct {
	Status string `json:"status"`
}

// AdminResetUserPasswordRequest sets a new password.
type AdminResetUserPasswordRequest struct {
	NewPassword string `json:"new_password"`
}

// AdminUpdateUserPermissionsRequest replaces assigned permissions.
type AdminUpdateUserPermissionsRequest struct {
	Permissions []string `json:"permissions"`
}

// AdminStatsResponse is admin dashboard stats.
type AdminStatsResponse struct {
	TotalUsers  int64 `json:"total_users"`
	AdminUsers  int64 `json:"admin_users"`
	NormalUsers int64 `json:"normal_users"`
	TotalBlogs  int64 `json:"total_blogs"`
}

// IsSuperAdmin returns true for TA and legacy A + level 10.
func IsSuperAdmin(userType string, userLevel int) bool {
	return authz.IsSuperAdmin(userType, userLevel)
}

// CanManageTarget enforces hierarchy for mutable admin actions.
func CanManageTarget(actor Actor, target *user.User) bool {
	if target == nil {
		return false
	}
	if !authz.IsAdminType(actor.UserType) {
		return false
	}
	if actor.IsSuperAdmin || IsSuperAdmin(actor.UserType, actor.UserLevel) {
		return true
	}

	targetType := authz.NormalizeAuthType(target.AuthType)
	if authz.IsTopAdminType(targetType) {
		return false
	}

	if authz.IsAdminType(targetType) {
		if !actor.LevelPolicyEnabled {
			return true
		}
		return target.AuthLevel < actor.UserLevel
	}

	return true
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
	pagination := utils.PaginationFromQuery(c, 20, 100)
	userType := strings.TrimSpace(c.Query("user_type"))

	result, err := h.service.GetAllUsers(pagination.Page, pagination.Limit, userType)
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

func (s *service) GetAllUsers(page, limit int, userType string) (*AdminUserListResponse, error) {
	pagination := utils.NewPagination(page, limit, 20, 100)

	query, countQuery, args := buildUserListQueries(userType)
	query += " ORDER BY u_regi_date DESC LIMIT ? OFFSET ?"

	var total int64
	err := s.db.DB.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		logger.Error("failed to query user count: %v", err)
		return nil, errors.Wrap(err, "DATABASE_ERROR", "failed to query user count")
	}

	queryArgs := append(args, pagination.Limit, pagination.Offset)
	rows, err := s.db.DB.Query(query, queryArgs...)
	if err != nil {
		logger.Error("failed to query users: %v", err)
		return nil, errors.Wrap(err, "DATABASE_ERROR", "failed to query users")
	}
	defer rows.Close()

	users := make([]user.User, 0, pagination.Limit)
	for rows.Next() {
		var u user.User
		if err := rows.Scan(&u.ID, &u.Name, &u.Email, &u.AuthType, &u.AuthLevel, &u.Status, &u.CreatedAt); err != nil {
			return nil, errors.Wrap(err, "DATABASE_ERROR", "failed to map users")
		}
		u.AuthType = authz.NormalizeAuthType(u.AuthType)
		users = append(users, u)
	}
	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "DATABASE_ERROR", "failed to iterate users")
	}

	return &AdminUserListResponse{
		Users: users,
		Total: total,
		Page:  pagination.Page,
		Limit: pagination.Limit,
	}, nil
}

func (s *service) GetUserByID(id string) (*user.User, error) {
	return s.userRepo.FindByID(id)
}

func (s *service) UpdateUserAuth(actor Actor, id string, authType string, authLevel int) error {
	target, err := s.userRepo.FindByID(id)
	if err != nil {
		return errors.New("USER_NOT_FOUND", "user not found")
	}
	target.AuthType = authz.NormalizeAuthType(target.AuthType)

	if actor.ID == id {
		return s.denyWithAudit(actor, id, "admin.account.level.manage", "self auth change is not allowed")
	}
	if !actor.IsSuperAdmin {
		return s.denyWithAudit(actor, id, "admin.account.level.manage", "only top-admin can change auth level")
	}

	authType = authz.NormalizeAuthType(authType)
	switch authType {
	case authz.AuthTypeUser, authz.AuthTypeAdmin, authz.AuthTypeManager, authz.AuthTypeGuest, authz.AuthTypeTopAdmin:
	default:
		return errors.New("INVALID_AUTH_TYPE", "auth type must be one of TA, A, M, G, U")
	}

	if authType == authz.AuthTypeTopAdmin {
		authLevel = 0
	} else if authLevel < 1 || authLevel > 10 {
		return errors.New("INVALID_AUTH_LEVEL", "auth level must be between 1 and 10")
	}

	now := time.Now()
	updates := map[string]interface{}{
		"u_auth_type":         authType,
		"u_auth_level":        authLevel,
		"u_token_valid_after": now,
		"u_re_token":          "",
	}

	return s.db.WithTx(func(tx *sql.Tx) error {
		userRepo := s.userRepo.Tx(tx)
		if err := userRepo.Update(id, updates); err != nil {
			return err
		}

		if !authz.IsAssignableAdminType(authType) {
			if err := s.permissionRepo.DeleteUserPermissionsTx(tx, id); err != nil {
				return err
			}
		}

		return s.permissionRepo.WriteAuditLogTx(tx, &AuditLogEntry{
			ActorID:  actor.ID,
			TargetID: id,
			Action:   "admin.account.level.manage",
			Status:   "success",
			IP:       actor.IP,
			BeforeData: map[string]interface{}{
				"auth_type":  target.AuthType,
				"auth_level": target.AuthLevel,
			},
			AfterData: map[string]interface{}{
				"auth_type":  authType,
				"auth_level": authLevel,
			},
		})
	})
}

func (s *service) UpdateUserProfile(actor Actor, id string, req *AdminUpdateUserProfileRequest) error {
	target, err := s.userRepo.FindByID(id)
	if err != nil {
		return errors.New("USER_NOT_FOUND", "user not found")
	}
	target.AuthType = authz.NormalizeAuthType(target.AuthType)

	if authz.IsAdminType(target.AuthType) && !CanManageTarget(actor, target) {
		return s.denyWithAudit(actor, id, "admin.account.profile.update", "cannot manage this admin account")
	}

	updates := map[string]interface{}{}
	if name := strings.TrimSpace(req.Name); utils.HasText(name) {
		updates["u_name"] = name
	}
	if email := strings.TrimSpace(req.Email); utils.HasText(email) {
		updates["u_email"] = email
	}
	if len(updates) == 0 {
		return errors.New("NO_UPDATES", "no profile fields to update")
	}

	return s.db.WithTx(func(tx *sql.Tx) error {
		userRepo := s.userRepo.Tx(tx)
		if err := userRepo.Update(id, updates); err != nil {
			return err
		}

		return s.permissionRepo.WriteAuditLogTx(tx, &AuditLogEntry{
			ActorID:  actor.ID,
			TargetID: id,
			Action:   "admin.account.profile.update",
			Status:   "success",
			IP:       actor.IP,
			BeforeData: map[string]interface{}{
				"name":  target.Name,
				"email": target.Email,
			},
			AfterData: map[string]interface{}{
				"name":  updates["u_name"],
				"email": updates["u_email"],
			},
		})
	})
}

func (s *service) UpdateUserStatus(actor Actor, id string, status string) error {
	normalizedStatus := utils.TrimLower(status)
	if normalizedStatus != user.UserStatusActive && normalizedStatus != user.UserStatusLocked {
		return errors.New("BAD_REQUEST", "status must be active or locked")
	}

	target, err := s.userRepo.FindByID(id)
	if err != nil {
		return errors.New("USER_NOT_FOUND", "user not found")
	}
	target.AuthType = authz.NormalizeAuthType(target.AuthType)

	if actor.ID == id {
		return s.denyWithAudit(actor, id, "admin.account.status.update", "self status change is not allowed")
	}
	if authz.IsAdminType(target.AuthType) && !CanManageTarget(actor, target) {
		return s.denyWithAudit(actor, id, "admin.account.status.update", "cannot manage this admin account")
	}

	updates := map[string]interface{}{
		"u_status":            normalizedStatus,
		"u_token_valid_after": time.Now(),
		"u_re_token":          "",
	}

	return s.db.WithTx(func(tx *sql.Tx) error {
		userRepo := s.userRepo.Tx(tx)
		if err := userRepo.Update(id, updates); err != nil {
			return err
		}

		return s.permissionRepo.WriteAuditLogTx(tx, &AuditLogEntry{
			ActorID:  actor.ID,
			TargetID: id,
			Action:   "admin.account.status.update",
			Status:   "success",
			IP:       actor.IP,
			BeforeData: map[string]interface{}{
				"status": target.Status,
			},
			AfterData: map[string]interface{}{
				"status": normalizedStatus,
			},
		})
	})
}

func (s *service) ResetUserPassword(actor Actor, id string, newPassword string) error {
	newPassword = strings.TrimSpace(newPassword)
	if len(newPassword) < 6 || len(newPassword) > 50 {
		return errors.New("BAD_REQUEST", "password length must be 6-50")
	}

	target, err := s.userRepo.FindByID(id)
	if err != nil {
		return errors.New("USER_NOT_FOUND", "user not found")
	}
	target.AuthType = authz.NormalizeAuthType(target.AuthType)

	if authz.IsAdminType(target.AuthType) && actor.ID != id && !CanManageTarget(actor, target) {
		return s.denyWithAudit(actor, id, "admin.account.password.reset", "cannot manage this admin account")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return errors.Wrap(err, "PASSWORD_HASH_FAILED", "failed to process password")
	}

	updates := map[string]interface{}{
		"u_pass":              string(hashedPassword),
		"u_token_valid_after": time.Now(),
		"u_re_token":          "",
	}

	return s.db.WithTx(func(tx *sql.Tx) error {
		userRepo := s.userRepo.Tx(tx)
		if err := userRepo.Update(id, updates); err != nil {
			return err
		}

		return s.permissionRepo.WriteAuditLogTx(tx, &AuditLogEntry{
			ActorID:  actor.ID,
			TargetID: id,
			Action:   "admin.account.password.reset",
			Status:   "success",
			IP:       actor.IP,
			AfterData: map[string]interface{}{
				"password_reset": true,
			},
		})
	})
}

func (s *service) GetUserPermissions(id string) ([]string, error) {
	if _, err := s.userRepo.FindByID(id); err != nil {
		return nil, errors.New("USER_NOT_FOUND", "user not found")
	}
	return s.permissionRepo.ListUserPermissions(id)
}

func (s *service) UpdateUserPermissions(actor Actor, id string, permissionCodes []string) error {
	target, err := s.userRepo.FindByID(id)
	if err != nil {
		return errors.New("USER_NOT_FOUND", "user not found")
	}
	target.AuthType = authz.NormalizeAuthType(target.AuthType)
	if !authz.IsAssignableAdminType(target.AuthType) {
		return errors.New("BAD_REQUEST", "permissions can only be assigned to admin users")
	}

	if actor.ID == id {
		return s.denyWithAudit(actor, id, "admin.account.permission.manage", "self permission change is not allowed")
	}
	if !actor.IsSuperAdmin && !CanManageTarget(actor, target) {
		return s.denyWithAudit(actor, id, "admin.account.permission.manage", "cannot manage this admin account")
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

	if !actor.IsSuperAdmin {
		delegableSet, err := s.permissionRepo.DelegableSet()
		if err != nil {
			return err
		}
		if err := ValidateDelegation(codes, delegableSet); err != nil {
			return s.denyWithAudit(actor, id, "admin.account.permission.manage", "permission delegation is not allowed")
		}
	}

	beforePermissions, err := s.permissionRepo.ListUserPermissions(id)
	if err != nil {
		return err
	}

	return s.db.WithTx(func(tx *sql.Tx) error {
		if err := s.permissionRepo.ReplaceUserPermissionsTx(tx, id, codes); err != nil {
			return err
		}

		userRepo := s.userRepo.Tx(tx)
		if err := userRepo.Update(id, map[string]interface{}{
			"u_token_valid_after": time.Now(),
			"u_re_token":          "",
		}); err != nil {
			return err
		}

		return s.permissionRepo.WriteAuditLogTx(tx, &AuditLogEntry{
			ActorID:  actor.ID,
			TargetID: id,
			Action:   "admin.account.permission.manage",
			Status:   "success",
			IP:       actor.IP,
			BeforeData: map[string]interface{}{
				"permissions": beforePermissions,
			},
			AfterData: map[string]interface{}{
				"permissions": codes,
			},
		})
	})
}

func (s *service) DeleteUser(actor Actor, id string) error {
	target, err := s.userRepo.FindByID(id)
	if err != nil {
		return errors.New("USER_NOT_FOUND", "user not found")
	}

	if actor.ID == id {
		return s.denyWithAudit(actor, id, "admin.account.delete", "self deletion is not allowed")
	}
	if !actor.IsSuperAdmin {
		return s.denyWithAudit(actor, id, "admin.account.delete", "only top-admin can delete user")
	}

	return s.db.WithTx(func(tx *sql.Tx) error {
		if err := s.permissionRepo.DeleteUserPermissionsTx(tx, id); err != nil {
			return err
		}
		userRepo := s.userRepo.Tx(tx)
		if err := userRepo.Delete(id); err != nil {
			return err
		}

		return s.permissionRepo.WriteAuditLogTx(tx, &AuditLogEntry{
			ActorID:  actor.ID,
			TargetID: id,
			Action:   "admin.account.delete",
			Status:   "success",
			IP:       actor.IP,
			BeforeData: map[string]interface{}{
				"auth_type":  target.AuthType,
				"auth_level": target.AuthLevel,
				"status":     target.Status,
			},
		})
	})
}

func (s *service) GetStats() (*AdminStatsResponse, error) {
	stats := &AdminStatsResponse{}

	err := s.db.DB.QueryRow("SELECT COUNT(*) FROM _user").Scan(&stats.TotalUsers)
	if err != nil {
		return nil, errors.Wrap(err, "DATABASE_ERROR", "failed to query total users")
	}

	err = s.db.DB.QueryRow("SELECT COUNT(*) FROM _user WHERE u_auth_type IN ('TA', 'A', 'M', 'G', 'AG')").Scan(&stats.AdminUsers)
	if err != nil {
		return nil, errors.Wrap(err, "DATABASE_ERROR", "failed to query admin users")
	}

	err = s.db.DB.QueryRow("SELECT COUNT(*) FROM _user WHERE u_auth_type = 'U'").Scan(&stats.NormalUsers)
	if err != nil {
		return nil, errors.Wrap(err, "DATABASE_ERROR", "failed to query normal users")
	}

	err = s.db.DB.QueryRow("SELECT COUNT(*) FROM _blog").Scan(&stats.TotalBlogs)
	if err != nil {
		stats.TotalBlogs = 0
	}

	return stats, nil
}

func buildUserListQueries(userType string) (query string, countQuery string, args []interface{}) {
	query = "SELECT u_id, u_name, u_email, u_auth_type, u_auth_level, COALESCE(u_status, 'active') AS u_status, u_regi_date FROM _user"
	countQuery = "SELECT COUNT(*) FROM _user"
	args = make([]interface{}, 0, 2)

	normalizedUserType := authz.NormalizeAuthType(userType)
	if normalizedUserType != "" {
		if normalizedUserType == authz.AuthTypeGuest {
			query += " WHERE u_auth_type IN (?, ?)"
			countQuery += " WHERE u_auth_type IN (?, ?)"
			args = append(args, authz.AuthTypeGuest, authz.AuthTypeGuestLegacy)
		} else {
			query += " WHERE u_auth_type = ?"
			countQuery += " WHERE u_auth_type = ?"
			args = append(args, normalizedUserType)
		}
	}

	return query, countQuery, args
}
