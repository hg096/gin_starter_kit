package admin

import (
	"database/sql"
	"gin_starter/internal/domain/user"
	"gin_starter/pkg/authz"
	"gin_starter/pkg/errors"
	"gin_starter/pkg/logger"
	"gin_starter/pkg/response"
	"gin_starter/pkg/utils"
	"net/mail"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// AdminBootstrapStatusResponse is public bootstrap availability response.
type AdminBootstrapStatusResponse struct {
	CanBootstrap    bool   `json:"can_bootstrap"`
	Reason          string `json:"reason,omitempty"`
	AdminCount      int64  `json:"admin_count"`
	SuperAdminCount int64  `json:"super_admin_count"`
}

// AdminBootstrapSuperAdminRequest creates the first/recovery top-admin.
type AdminBootstrapSuperAdminRequest struct {
	ID       string `json:"user_id"`
	Password string `json:"user_pass"`
	Name     string `json:"user_name"`
	Email    string `json:"user_email"`
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

func (s *service) GetBootstrapStatus() (*AdminBootstrapStatusResponse, error) {
	state, err := s.readBootstrapState(s.db.DB)
	if err != nil {
		return nil, err
	}
	canBootstrap, reason := evaluateBootstrapState(state)

	return &AdminBootstrapStatusResponse{
		CanBootstrap:    canBootstrap,
		Reason:          reason,
		AdminCount:      state.AdminCount,
		SuperAdminCount: state.SuperAdminCount,
	}, nil
}

func (s *service) BootstrapSuperAdmin(req *AdminBootstrapSuperAdminRequest, ip string) (*user.User, error) {
	if req == nil {
		return nil, errors.New("BAD_REQUEST", "request body is required")
	}
	if err := normalizeBootstrapRequest(req); err != nil {
		return nil, err
	}

	var bootstrapUser *user.User
	var reason string
	if err := s.db.WithTx(func(tx *sql.Tx) error {
		state, err := s.readBootstrapState(tx)
		if err != nil {
			return err
		}
		canBootstrap, evaluatedReason := evaluateBootstrapState(state)
		reason = evaluatedReason
		if !canBootstrap {
			return errors.New("FORBIDDEN", "bootstrap is disabled because top-admin already exists")
		}

		var existsCount int
		if err := tx.QueryRow("SELECT COUNT(*) FROM _user WHERE u_id = ?", req.ID).Scan(&existsCount); err != nil {
			return errors.Wrap(err, "DATABASE_ERROR", "failed to check user existence")
		}
		if existsCount > 0 {
			return errors.New("CONFLICT", "user id already exists")
		}

		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			return errors.Wrap(err, "PASSWORD_HASH_FAILED", "failed to process password")
		}

		now := time.Now()
		bootstrapUser = &user.User{
			ID:        req.ID,
			Password:  string(hashedPassword),
			Name:      req.Name,
			Email:     req.Email,
			AuthType:  authz.AuthTypeTopAdmin,
			AuthLevel: 0,
			Status:    user.UserStatusActive,
			CreatedAt: now,
		}
		userRepo := s.userRepo.Tx(tx)
		if err := userRepo.Create(bootstrapUser); err != nil {
			return err
		}

		if err := s.assignAllPermissionsTx(tx, bootstrapUser.ID); err != nil {
			return err
		}

		auditErr := s.permissionRepo.WriteAuditLogTx(tx, &AuditLogEntry{
			ActorID:  bootstrapUser.ID,
			TargetID: bootstrapUser.ID,
			Action:   "system.bootstrap.super_admin.create",
			Status:   "success",
			Message:  reason,
			IP:       strings.TrimSpace(ip),
			AfterData: map[string]interface{}{
				"user_id":    bootstrapUser.ID,
				"auth_type":  authz.AuthTypeTopAdmin,
				"auth_level": 0,
				"reason":     reason,
			},
		})
		if auditErr != nil && !isMissingTableErrorForBootstrap(auditErr) {
			return auditErr
		}

		return nil
	}); err != nil {
		return nil, err
	}

	logger.Warn("top-admin bootstrapped: %s (%s)", bootstrapUser.ID, reason)
	return bootstrapUser.ToPublic(), nil
}

func (s *service) assignAllPermissionsTx(tx *sql.Tx, userID string) error {
	permissions, err := s.permissionRepo.ListPermissions()
	if err != nil {
		if isMissingTableErrorForBootstrap(err) {
			return nil
		}
		return err
	}

	codes := make([]string, 0, len(permissions))
	for _, permission := range permissions {
		codes = append(codes, permission.Code)
	}

	if err := s.permissionRepo.ReplaceUserPermissionsTx(tx, userID, codes); err != nil {
		if isMissingTableErrorForBootstrap(err) {
			return nil
		}
		return err
	}

	return nil
}

func (s *service) readBootstrapState(queryer bootstrapQueryer) (*bootstrapState, error) {
	state := &bootstrapState{}

	if err := queryer.QueryRow("SELECT COUNT(*) FROM _user WHERE u_auth_type IN ('TA', 'A', 'M', 'G', 'AG')").Scan(&state.AdminCount); err != nil {
		return nil, errors.Wrap(err, "DATABASE_ERROR", "failed to check admin count")
	}
	if err := queryer.QueryRow("SELECT COUNT(*) FROM _user WHERE u_auth_type = 'TA'").Scan(&state.SuperAdminCount); err != nil {
		return nil, errors.Wrap(err, "DATABASE_ERROR", "failed to check top-admin count")
	}

	return state, nil
}

type bootstrapState struct {
	AdminCount      int64
	SuperAdminCount int64
}

type bootstrapQueryer interface {
	QueryRow(query string, args ...interface{}) *sql.Row
}

func evaluateBootstrapState(state *bootstrapState) (canBootstrap bool, reason string) {
	if state == nil {
		return false, ""
	}
	if state.AdminCount == 0 {
		return true, "no_admin_account"
	}
	if state.SuperAdminCount == 0 {
		return true, "no_super_admin_account"
	}
	return false, ""
}

func normalizeBootstrapRequest(req *AdminBootstrapSuperAdminRequest) error {
	req.ID = strings.TrimSpace(req.ID)
	req.Password = strings.TrimSpace(req.Password)
	req.Name = strings.TrimSpace(req.Name)
	req.Email = strings.TrimSpace(req.Email)

	if len(req.ID) < 3 || len(req.ID) > 20 || !isAlphaNumeric(req.ID) {
		return errors.New("BAD_REQUEST", "user_id must be 3-20 alphanumeric characters")
	}
	if len(req.Password) < 6 || len(req.Password) > 50 {
		return errors.New("BAD_REQUEST", "user_pass length must be 6-50")
	}
	if len(req.Name) < 2 || len(req.Name) > 50 {
		return errors.New("BAD_REQUEST", "user_name length must be 2-50")
	}
	if _, err := mail.ParseAddress(req.Email); err != nil {
		return errors.New("BAD_REQUEST", "user_email is invalid")
	}

	return nil
}

func isAlphaNumeric(value string) bool {
	if value == "" {
		return false
	}
	for _, ch := range value {
		if ch >= 'a' && ch <= 'z' {
			continue
		}
		if ch >= 'A' && ch <= 'Z' {
			continue
		}
		if ch >= '0' && ch <= '9' {
			continue
		}
		return false
	}
	return true
}

func isMissingTableErrorForBootstrap(err error) bool {
	if err == nil {
		return false
	}
	msg := utils.TrimLower(err.Error())
	return strings.Contains(msg, "doesn't exist") || strings.Contains(msg, "no such table")
}
