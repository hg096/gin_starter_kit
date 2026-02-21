package admin

import (
	"database/sql"
	"gin_starter/internal/domain/blog"
	"gin_starter/internal/domain/user"
	"gin_starter/internal/infrastructure/database"
	"gin_starter/pkg/authz"
	"gin_starter/pkg/errors"
	"gin_starter/pkg/logger"
	"net/mail"
	"sort"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// Service is the admin domain service interface.
type Service interface {
	GetAllUsers(page, limit int, userType string) (*AdminUserListResponse, error)
	GetUserByID(id string) (*user.User, error)
	UpdateUserAuth(actor Actor, id string, authType string, authLevel int) error
	UpdateUserProfile(actor Actor, id string, req *AdminUpdateUserProfileRequest) error
	UpdateUserStatus(actor Actor, id string, status string) error
	ResetUserPassword(actor Actor, id string, newPassword string) error
	GetUserPermissions(id string) ([]string, error)
	UpdateUserPermissions(actor Actor, id string, permissionCodes []string) error
	GetPermissions() ([]Permission, error)
	GetDelegablePermissions() ([]string, error)
	UpdateDelegablePermissions(actor Actor, permissionCodes []string) error
	GetLevelPolicy() (*AdminLevelPolicyResponse, error)
	UpdateLevelPolicy(actor Actor, enabled bool) error
	DeleteUser(actor Actor, id string) error
	GetStats() (*AdminStatsResponse, error)
	GetAuditLogs(page, limit int, action, actorID, targetID, dateFrom, dateTo string) (*AdminAuditLogListResponse, error)
	GetBlogs(page, limit int) (*AdminBlogListResponse, error)
	CreateBlog(actor Actor, req *AdminCreateBlogRequest) (*blog.Blog, error)
	UpdateBlog(actor Actor, id int64, req *AdminUpdateBlogRequest) (*blog.Blog, error)
	DeleteBlog(actor Actor, id int64) error
	SyncAdminPagesFromRouteSpecs(routeSpecs []AdminPageRouteSpec) error
	GetAdminPages(includeDisabled bool) ([]AdminPage, error)
	GetAdminPageByKey(pageKey string) (*AdminPage, error)
	CreateAdminPage(actor Actor, req *AdminCreatePageRequest) (*AdminPage, error)
	UpdateAdminPage(actor Actor, pageKey string, req *AdminUpdatePageRequest) (*AdminPage, error)
	DeleteAdminPage(actor Actor, pageKey string) error
	GetBootstrapStatus() (*AdminBootstrapStatusResponse, error)
	BootstrapSuperAdmin(req *AdminBootstrapSuperAdminRequest, ip string) (*user.User, error)
}

type service struct {
	userRepo       user.Repository
	blogRepo       blog.Repository
	permissionRepo PermissionRepository
	pageRepo       AdminPageRepository
	db             *database.DB
}

func NewService(userRepo user.Repository, db *database.DB) Service {
	var permissionRepo PermissionRepository
	var pageRepo AdminPageRepository
	if db != nil && db.DB != nil {
		permissionRepo = NewPermissionRepository(db)
		pageRepo, _ = permissionRepo.(AdminPageRepository)
	}
	blogRepo := blog.NewRepository(db)
	if pageRepo != nil {
		if err := pageRepo.EnsureAdminPageSchema(); err != nil {
			logger.Warn("failed to ensure admin page schema: %v", err)
		}
	}

	return &service{
		userRepo:       userRepo,
		blogRepo:       blogRepo,
		permissionRepo: permissionRepo,
		pageRepo:       pageRepo,
		db:             db,
	}
}

func (s *service) GetAllUsers(page, limit int, userType string) (*AdminUserListResponse, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	offset := (page - 1) * limit

	query, countQuery, args := buildUserListQueries(userType)
	query += " ORDER BY u_regi_date DESC LIMIT ? OFFSET ?"

	var total int64
	err := s.db.DB.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		logger.Error("failed to query user count: %v", err)
		return nil, errors.Wrap(err, "DATABASE_ERROR", "failed to query user count")
	}

	queryArgs := append(args, limit, offset)
	rows, err := s.db.DB.Query(query, queryArgs...)
	if err != nil {
		logger.Error("failed to query users: %v", err)
		return nil, errors.Wrap(err, "DATABASE_ERROR", "failed to query users")
	}
	defer rows.Close()

	users := make([]user.User, 0, limit)
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
		Page:  page,
		Limit: limit,
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

	tx, err := s.db.BeginTx()
	if err != nil {
		return errors.Wrap(err, "DATABASE_ERROR", "failed to begin transaction")
	}
	defer database.RollbackTx(tx)

	now := time.Now()
	updates := map[string]interface{}{
		"u_auth_type":         authType,
		"u_auth_level":        authLevel,
		"u_token_valid_after": now,
		"u_re_token":          "",
	}
	if err := s.userRepo.UpdateTx(tx, id, updates); err != nil {
		return err
	}

	if !authz.IsAssignableAdminType(authType) {
		if err := s.permissionRepo.DeleteUserPermissionsTx(tx, id); err != nil {
			return err
		}
	}

	if err := s.permissionRepo.WriteAuditLogTx(tx, &AuditLogEntry{
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
	}); err != nil {
		return err
	}

	if err := database.CommitTx(tx); err != nil {
		return errors.Wrap(err, "DATABASE_ERROR", "failed to commit transaction")
	}

	return nil
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
	if strings.TrimSpace(req.Name) != "" {
		updates["u_name"] = strings.TrimSpace(req.Name)
	}
	if strings.TrimSpace(req.Email) != "" {
		updates["u_email"] = strings.TrimSpace(req.Email)
	}
	if len(updates) == 0 {
		return errors.New("NO_UPDATES", "no profile fields to update")
	}

	tx, err := s.db.BeginTx()
	if err != nil {
		return errors.Wrap(err, "DATABASE_ERROR", "failed to begin transaction")
	}
	defer database.RollbackTx(tx)

	if err := s.userRepo.UpdateTx(tx, id, updates); err != nil {
		return err
	}

	if err := s.permissionRepo.WriteAuditLogTx(tx, &AuditLogEntry{
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
	}); err != nil {
		return err
	}

	if err := database.CommitTx(tx); err != nil {
		return errors.Wrap(err, "DATABASE_ERROR", "failed to commit transaction")
	}

	return nil
}

func (s *service) UpdateUserStatus(actor Actor, id string, status string) error {
	normalizedStatus := strings.ToLower(strings.TrimSpace(status))
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

	tx, err := s.db.BeginTx()
	if err != nil {
		return errors.Wrap(err, "DATABASE_ERROR", "failed to begin transaction")
	}
	defer database.RollbackTx(tx)

	updates := map[string]interface{}{
		"u_status":            normalizedStatus,
		"u_token_valid_after": time.Now(),
		"u_re_token":          "",
	}
	if err := s.userRepo.UpdateTx(tx, id, updates); err != nil {
		return err
	}

	if err := s.permissionRepo.WriteAuditLogTx(tx, &AuditLogEntry{
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
	}); err != nil {
		return err
	}

	if err := database.CommitTx(tx); err != nil {
		return errors.Wrap(err, "DATABASE_ERROR", "failed to commit transaction")
	}

	return nil
}

func (s *service) ResetUserPassword(actor Actor, id string, newPassword string) error {
	if len(strings.TrimSpace(newPassword)) < 6 || len(newPassword) > 50 {
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

	tx, err := s.db.BeginTx()
	if err != nil {
		return errors.Wrap(err, "DATABASE_ERROR", "failed to begin transaction")
	}
	defer database.RollbackTx(tx)

	updates := map[string]interface{}{
		"u_pass":              string(hashedPassword),
		"u_token_valid_after": time.Now(),
		"u_re_token":          "",
	}
	if err := s.userRepo.UpdateTx(tx, id, updates); err != nil {
		return err
	}

	if err := s.permissionRepo.WriteAuditLogTx(tx, &AuditLogEntry{
		ActorID:  actor.ID,
		TargetID: id,
		Action:   "admin.account.password.reset",
		Status:   "success",
		IP:       actor.IP,
		AfterData: map[string]interface{}{
			"password_reset": true,
		},
	}); err != nil {
		return err
	}

	if err := database.CommitTx(tx); err != nil {
		return errors.Wrap(err, "DATABASE_ERROR", "failed to commit transaction")
	}

	return nil
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

	tx, err := s.db.BeginTx()
	if err != nil {
		return errors.Wrap(err, "DATABASE_ERROR", "failed to begin transaction")
	}
	defer database.RollbackTx(tx)

	if err := s.permissionRepo.ReplaceUserPermissionsTx(tx, id, codes); err != nil {
		return err
	}

	if err := s.userRepo.UpdateTx(tx, id, map[string]interface{}{
		"u_token_valid_after": time.Now(),
		"u_re_token":          "",
	}); err != nil {
		return err
	}

	if err := s.permissionRepo.WriteAuditLogTx(tx, &AuditLogEntry{
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
	}); err != nil {
		return err
	}

	if err := database.CommitTx(tx); err != nil {
		return errors.Wrap(err, "DATABASE_ERROR", "failed to commit transaction")
	}

	return nil
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

	tx, err := s.db.BeginTx()
	if err != nil {
		return errors.Wrap(err, "DATABASE_ERROR", "failed to begin transaction")
	}
	defer database.RollbackTx(tx)

	if err := s.permissionRepo.ReplaceDelegablePermissionsTx(tx, codes); err != nil {
		return err
	}

	if err := s.permissionRepo.WriteAuditLogTx(tx, &AuditLogEntry{
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
	}); err != nil {
		return err
	}

	if err := database.CommitTx(tx); err != nil {
		return errors.Wrap(err, "DATABASE_ERROR", "failed to commit transaction")
	}

	return nil
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

	tx, err := s.db.BeginTx()
	if err != nil {
		logger.Error("failed to begin transaction for level policy update (actor=%s): %v", actor.ID, err)
		return errors.Wrap(err, "DATABASE_ERROR", "failed to begin transaction")
	}
	defer database.RollbackTx(tx)

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

	if err := database.CommitTx(tx); err != nil {
		logger.Error("failed to commit level policy update transaction (actor=%s enabled=%t): %v", actor.ID, enabled, err)
		return errors.Wrap(err, "DATABASE_ERROR", "failed to commit transaction")
	}
	return nil
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

	tx, err := s.db.BeginTx()
	if err != nil {
		return errors.Wrap(err, "DATABASE_ERROR", "failed to begin transaction")
	}
	defer database.RollbackTx(tx)

	if err := s.permissionRepo.DeleteUserPermissionsTx(tx, id); err != nil {
		return err
	}
	if err := s.userRepo.DeleteTx(tx, id); err != nil {
		return err
	}

	if err := s.permissionRepo.WriteAuditLogTx(tx, &AuditLogEntry{
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
	}); err != nil {
		return err
	}

	if err := database.CommitTx(tx); err != nil {
		return errors.Wrap(err, "DATABASE_ERROR", "failed to commit transaction")
	}

	return nil
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

func (s *service) GetAuditLogs(page, limit int, action, actorID, targetID, dateFrom, dateTo string) (*AdminAuditLogListResponse, error) {
	req := &AdminAuditLogListRequest{
		Page:         page,
		Limit:        limit,
		Action:       strings.TrimSpace(action),
		ActorID:      strings.TrimSpace(actorID),
		TargetUserID: strings.TrimSpace(targetID),
		DateFrom:     strings.TrimSpace(dateFrom),
		DateTo:       strings.TrimSpace(dateTo),
	}

	var parsedFrom time.Time
	var parsedTo time.Time
	var hasFrom bool
	var hasTo bool

	if req.DateFrom != "" {
		t, err := time.Parse("2006-01-02", req.DateFrom)
		if err != nil {
			return nil, errors.New("BAD_REQUEST", "date_from must be YYYY-MM-DD")
		}
		parsedFrom = t
		hasFrom = true
		req.DateFrom = t.Format("2006-01-02")
	}

	if req.DateTo != "" {
		t, err := time.Parse("2006-01-02", req.DateTo)
		if err != nil {
			return nil, errors.New("BAD_REQUEST", "date_to must be YYYY-MM-DD")
		}
		parsedTo = t
		hasTo = true
		// Repository uses an exclusive upper bound, so pass next day.
		req.DateTo = t.AddDate(0, 0, 1).Format("2006-01-02")
	}

	if hasFrom && hasTo && parsedFrom.After(parsedTo) {
		return nil, errors.New("BAD_REQUEST", "date_from must be before or equal to date_to")
	}

	return s.permissionRepo.ListAuditLogs(req)
}

func (s *service) GetBlogs(page, limit int) (*AdminBlogListResponse, error) {
	if s.blogRepo == nil {
		return nil, errors.New("DATABASE_ERROR", "blog repository is not available")
	}
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	blogs, total, err := s.blogRepo.FindAll(page, limit)
	if err != nil {
		return nil, errors.Wrap(err, "DATABASE_ERROR", "failed to list blogs")
	}

	return &AdminBlogListResponse{
		Blogs: blogs,
		Total: total,
		Page:  page,
		Limit: limit,
	}, nil
}

func (s *service) CreateBlog(actor Actor, req *AdminCreateBlogRequest) (*blog.Blog, error) {
	if s.blogRepo == nil {
		return nil, errors.New("DATABASE_ERROR", "blog repository is not available")
	}
	if req == nil {
		return nil, errors.New("BAD_REQUEST", "request body is required")
	}

	title := strings.TrimSpace(req.Title)
	if len(title) < 2 || len(title) > 200 {
		return nil, errors.New("BAD_REQUEST", "title length must be 2-200")
	}

	content := strings.TrimSpace(req.Content)
	if content == "" || len(content) > 10000 {
		return nil, errors.New("BAD_REQUEST", "content length must be 1-10000")
	}

	authorID := strings.TrimSpace(req.AuthorID)
	if authorID == "" {
		authorID = actor.ID
	}
	if _, err := s.userRepo.FindByID(authorID); err != nil {
		return nil, errors.New("USER_NOT_FOUND", "author not found")
	}

	newBlog := &blog.Blog{
		Title:    title,
		Content:  content,
		AuthorID: authorID,
	}

	tx, err := s.db.BeginTx()
	if err != nil {
		return nil, errors.Wrap(err, "DATABASE_ERROR", "failed to begin transaction")
	}
	defer database.RollbackTx(tx)

	if err := s.blogRepo.CreateTx(tx, newBlog); err != nil {
		return nil, errors.Wrap(err, "DATABASE_ERROR", "failed to create blog")
	}
	if err := s.permissionRepo.WriteAuditLogTx(tx, &AuditLogEntry{
		ActorID:  actor.ID,
		TargetID: authorID,
		Action:   BuildPagePermissionCode("blogs", PageActionCreate),
		Status:   "success",
		IP:       actor.IP,
		AfterData: map[string]interface{}{
			"id":        newBlog.ID,
			"title":     newBlog.Title,
			"author_id": newBlog.AuthorID,
		},
	}); err != nil {
		return nil, err
	}

	if err := database.CommitTx(tx); err != nil {
		return nil, errors.Wrap(err, "DATABASE_ERROR", "failed to commit transaction")
	}

	return newBlog, nil
}

func (s *service) UpdateBlog(actor Actor, id int64, req *AdminUpdateBlogRequest) (*blog.Blog, error) {
	if s.blogRepo == nil {
		return nil, errors.New("DATABASE_ERROR", "blog repository is not available")
	}
	if req == nil {
		return nil, errors.New("BAD_REQUEST", "request body is required")
	}

	current, err := s.blogRepo.FindByID(id)
	if err != nil || current == nil {
		return nil, errors.New("BLOG_NOT_FOUND", "blog not found")
	}

	updates := map[string]interface{}{}
	if trimmed := strings.TrimSpace(req.Title); trimmed != "" {
		if len(trimmed) < 2 || len(trimmed) > 200 {
			return nil, errors.New("BAD_REQUEST", "title length must be 2-200")
		}
		updates["title"] = trimmed
	}
	if trimmed := strings.TrimSpace(req.Content); trimmed != "" {
		if len(trimmed) > 10000 {
			return nil, errors.New("BAD_REQUEST", "content length must be <= 10000")
		}
		updates["content"] = trimmed
	}
	if len(updates) == 0 {
		return nil, errors.New("BAD_REQUEST", "no update data")
	}

	tx, err := s.db.BeginTx()
	if err != nil {
		return nil, errors.Wrap(err, "DATABASE_ERROR", "failed to begin transaction")
	}
	defer database.RollbackTx(tx)

	if err := s.blogRepo.UpdateTx(tx, id, updates); err != nil {
		return nil, errors.Wrap(err, "DATABASE_ERROR", "failed to update blog")
	}
	if err := s.permissionRepo.WriteAuditLogTx(tx, &AuditLogEntry{
		ActorID:  actor.ID,
		TargetID: current.AuthorID,
		Action:   BuildPagePermissionCode("blogs", PageActionUpdate),
		Status:   "success",
		IP:       actor.IP,
		BeforeData: map[string]interface{}{
			"id":      current.ID,
			"title":   current.Title,
			"content": current.Content,
		},
		AfterData: map[string]interface{}{
			"id":      current.ID,
			"title":   coalesceString(updates["title"], current.Title),
			"content": coalesceString(updates["content"], current.Content),
		},
	}); err != nil {
		return nil, err
	}

	if err := database.CommitTx(tx); err != nil {
		return nil, errors.Wrap(err, "DATABASE_ERROR", "failed to commit transaction")
	}

	updated, err := s.blogRepo.FindByID(id)
	if err != nil || updated == nil {
		return nil, errors.New("BLOG_NOT_FOUND", "blog not found")
	}
	return updated, nil
}

func (s *service) DeleteBlog(actor Actor, id int64) error {
	if s.blogRepo == nil {
		return errors.New("DATABASE_ERROR", "blog repository is not available")
	}

	current, err := s.blogRepo.FindByID(id)
	if err != nil || current == nil {
		return errors.New("BLOG_NOT_FOUND", "blog not found")
	}

	tx, err := s.db.BeginTx()
	if err != nil {
		return errors.Wrap(err, "DATABASE_ERROR", "failed to begin transaction")
	}
	defer database.RollbackTx(tx)

	if err := s.blogRepo.DeleteTx(tx, id); err != nil {
		return errors.Wrap(err, "DATABASE_ERROR", "failed to delete blog")
	}
	if err := s.permissionRepo.WriteAuditLogTx(tx, &AuditLogEntry{
		ActorID:  actor.ID,
		TargetID: current.AuthorID,
		Action:   BuildPagePermissionCode("blogs", PageActionDelete),
		Status:   "success",
		IP:       actor.IP,
		BeforeData: map[string]interface{}{
			"id":        current.ID,
			"title":     current.Title,
			"author_id": current.AuthorID,
		},
	}); err != nil {
		return err
	}

	if err := database.CommitTx(tx); err != nil {
		return errors.Wrap(err, "DATABASE_ERROR", "failed to commit transaction")
	}

	return nil
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

	tx, err := s.db.BeginTx()
	if err != nil {
		return errors.Wrap(err, "DATABASE_ERROR", "failed to begin transaction")
	}
	defer database.RollbackTx(tx)

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
		if strings.TrimSpace(existing.Title) == "" {
			existing.Title = spec.Title
			changed = true
		}
		if strings.TrimSpace(existing.Description) == "" && spec.Description != "" {
			existing.Description = spec.Description
			changed = true
		}
		if strings.TrimSpace(existing.GroupKey) == "" {
			existing.GroupKey = spec.GroupKey
			changed = true
		}
		if strings.TrimSpace(existing.GroupLabel) == "" {
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
		if strings.TrimSpace(existing.Icon) == "" && spec.Icon != "" {
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

	if err := database.CommitTx(tx); err != nil {
		return errors.Wrap(err, "DATABASE_ERROR", "failed to commit transaction")
	}

	return nil
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

	tx, err := s.db.BeginTx()
	if err != nil {
		return nil, errors.Wrap(err, "DATABASE_ERROR", "failed to begin transaction")
	}
	defer database.RollbackTx(tx)

	if err := s.pageRepo.CreateAdminPageTx(tx, page); err != nil {
		return nil, err
	}
	if err := s.pageRepo.EnsurePagePermissionCodesTx(tx, page); err != nil {
		return nil, err
	}
	if err := s.permissionRepo.WriteAuditLogTx(tx, &AuditLogEntry{
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
	}); err != nil {
		return nil, err
	}

	if err := database.CommitTx(tx); err != nil {
		return nil, errors.Wrap(err, "DATABASE_ERROR", "failed to commit transaction")
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
		if strings.TrimSpace(page.GroupLabel) == "" {
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

	tx, err := s.db.BeginTx()
	if err != nil {
		return nil, errors.Wrap(err, "DATABASE_ERROR", "failed to begin transaction")
	}
	defer database.RollbackTx(tx)

	if fromRepository {
		if err := s.pageRepo.UpdateAdminPageTx(tx, page); err != nil {
			return nil, err
		}
	}
	if err := s.pageRepo.EnsurePagePermissionCodesTx(tx, page); err != nil {
		return nil, err
	}
	if err := s.permissionRepo.WriteAuditLogTx(tx, &AuditLogEntry{
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
	}); err != nil {
		return nil, err
	}

	if err := database.CommitTx(tx); err != nil {
		return nil, errors.Wrap(err, "DATABASE_ERROR", "failed to commit transaction")
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

	tx, err := s.db.BeginTx()
	if err != nil {
		return errors.Wrap(err, "DATABASE_ERROR", "failed to begin transaction")
	}
	defer database.RollbackTx(tx)

	if err := s.pageRepo.DeletePagePermissionCodesTx(tx, key); err != nil {
		return err
	}
	if err := s.pageRepo.DeleteAdminPageTx(tx, key); err != nil {
		return err
	}
	if err := s.permissionRepo.WriteAuditLogTx(tx, &AuditLogEntry{
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
	}); err != nil {
		return err
	}

	if err := database.CommitTx(tx); err != nil {
		return errors.Wrap(err, "DATABASE_ERROR", "failed to commit transaction")
	}
	return nil
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

	tx, err := s.db.BeginTx()
	if err != nil {
		return nil, errors.Wrap(err, "DATABASE_ERROR", "failed to begin transaction")
	}
	defer database.RollbackTx(tx)

	state, err := s.readBootstrapState(tx)
	if err != nil {
		return nil, err
	}
	canBootstrap, reason := evaluateBootstrapState(state)
	if !canBootstrap {
		return nil, errors.New("FORBIDDEN", "bootstrap is disabled because top-admin already exists")
	}

	var existsCount int
	if err := tx.QueryRow("SELECT COUNT(*) FROM _user WHERE u_id = ?", req.ID).Scan(&existsCount); err != nil {
		return nil, errors.Wrap(err, "DATABASE_ERROR", "failed to check user existence")
	}
	if existsCount > 0 {
		return nil, errors.New("CONFLICT", "user id already exists")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, errors.Wrap(err, "PASSWORD_HASH_FAILED", "failed to process password")
	}

	now := time.Now()
	bootstrapUser := &user.User{
		ID:        req.ID,
		Password:  string(hashedPassword),
		Name:      req.Name,
		Email:     req.Email,
		AuthType:  authz.AuthTypeTopAdmin,
		AuthLevel: 0,
		Status:    user.UserStatusActive,
		CreatedAt: now,
	}
	if err := s.userRepo.CreateTx(tx, bootstrapUser); err != nil {
		return nil, err
	}

	if err := s.assignAllPermissionsTx(tx, bootstrapUser.ID); err != nil {
		return nil, err
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
		return nil, auditErr
	}

	if err := database.CommitTx(tx); err != nil {
		return nil, errors.Wrap(err, "DATABASE_ERROR", "failed to commit transaction")
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

type bootstrapState struct {
	AdminCount      int64
	SuperAdminCount int64
}

type bootstrapQueryer interface {
	QueryRow(query string, args ...interface{}) *sql.Row
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
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "doesn't exist") || strings.Contains(msg, "no such table")
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
	key := strings.ToLower(strings.TrimSpace(raw))
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

func isReservedAdminPageKey(pageKey string) bool {
	switch pageKey {
	case "admin", "login", "logout", "users", "blogs", "admin_chat", "chat", "bootstrap", "p":
		return true
	default:
		return false
	}
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

func (s *service) denyWithAudit(actor Actor, targetID, action, message string) error {
	err := s.permissionRepo.WriteAuditLog(&AuditLogEntry{
		ActorID:  actor.ID,
		TargetID: targetID,
		Action:   action,
		Status:   "denied",
		Message:  message,
		IP:       actor.IP,
	})
	if err != nil {
		return errors.Wrap(err, "AUDIT_LOG_FAILED", "failed to write audit log")
	}
	return errors.New("FORBIDDEN", message)
}

func coalesceString(value interface{}, fallback string) string {
	str, ok := value.(string)
	if !ok {
		return fallback
	}
	trimmed := strings.TrimSpace(str)
	if trimmed == "" {
		return fallback
	}
	return trimmed
}

func normalizePermissionCodes(permissionCodes []string) []string {
	set := make(map[string]struct{}, len(permissionCodes))
	for _, code := range permissionCodes {
		trimmed := strings.TrimSpace(code)
		if trimmed == "" {
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
