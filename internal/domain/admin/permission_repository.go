package admin

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"gin_starter/pkg/db/database"
	"gin_starter/pkg/errors"
	"gin_starter/pkg/logger"
	"gin_starter/pkg/utils"
	"sort"
	"strings"
)

// AuditLogEntry records admin mutation actions.
type AuditLogEntry struct {
	ActorID    string
	TargetID   string
	Action     string
	Status     string
	Message    string
	IP         string
	BeforeData map[string]interface{}
	AfterData  map[string]interface{}
}

type PermissionRepository interface {
	ListPermissions() ([]Permission, error)
	ListAuditLogs(req *AdminAuditLogListRequest) (*AdminAuditLogListResponse, error)
	GetLevelPolicyEnabled() (bool, error)
	SetLevelPolicyEnabledTx(tx *sql.Tx, enabled bool, updatedBy string) error
	ListUserPermissions(userID string) ([]string, error)
	ReplaceUserPermissionsTx(tx *sql.Tx, userID string, permissionCodes []string) error
	ListDelegablePermissions() ([]string, error)
	DelegableSet() (map[string]struct{}, error)
	ReplaceDelegablePermissionsTx(tx *sql.Tx, permissionCodes []string) error
	PermissionCodesExist(codes []string) (bool, error)
	GrantPermissionFromExistingTx(tx *sql.Tx, targetCode, sourceCode string) error
	WriteAuditLogTx(tx *sql.Tx, entry *AuditLogEntry) error
	WriteAuditLog(entry *AuditLogEntry) error
	DeleteUserPermissionsTx(tx *sql.Tx, userID string) error
}

type AdminPageRepository interface {
	EnsureAdminPageSchema() error
	ListAdminPages(includeDisabled bool) ([]AdminPage, error)
	GetAdminPageByKey(pageKey string) (*AdminPage, error)
	CreateAdminPageTx(tx *sql.Tx, page *AdminPage) error
	UpdateAdminPageTx(tx *sql.Tx, page *AdminPage) error
	DeleteAdminPageTx(tx *sql.Tx, pageKey string) error
	EnsurePagePermissionCodesTx(tx *sql.Tx, page *AdminPage) error
	DeletePagePermissionCodesTx(tx *sql.Tx, pageKey string) error
}

type permissionRepository struct {
	db   *database.DB
	base *database.Repository
}

const createSystemSettingsTableSQL = `CREATE TABLE IF NOT EXISTS _a_system_settings (
	setting_key VARCHAR(80) NOT NULL,
	setting_value VARCHAR(255) NOT NULL,
	updated_by VARCHAR(50) NULL,
	updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
	PRIMARY KEY (setting_key)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci`

const createAdminPagesTableSQL = `CREATE TABLE IF NOT EXISTS _a_admin_pages (
	page_key VARCHAR(48) NOT NULL,
	title VARCHAR(120) NOT NULL,
	path VARCHAR(255) NOT NULL,
	description VARCHAR(255) NOT NULL DEFAULT '',
	group_key VARCHAR(48) NOT NULL DEFAULT 'general',
	group_label VARCHAR(100) NOT NULL DEFAULT 'General',
	group_order INT NOT NULL DEFAULT 100,
	visible_roles JSON NULL,
	icon VARCHAR(40) NOT NULL DEFAULT '',
	sort_order INT NOT NULL DEFAULT 100,
	is_enabled TINYINT(1) NOT NULL DEFAULT 1,
	is_builtin TINYINT(1) NOT NULL DEFAULT 0,
	created_by VARCHAR(50) NULL,
	created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
	PRIMARY KEY (page_key),
	UNIQUE KEY uq_a_admin_pages_path (path),
	KEY idx_a_admin_pages_enabled_sort (is_enabled, sort_order),
	KEY idx_a_admin_pages_enabled_group_sort (is_enabled, group_order, sort_order)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci`

const createPermissionsTableSQL = `CREATE TABLE IF NOT EXISTS _a_permissions (
	permission_code VARCHAR(120) NOT NULL,
	description VARCHAR(255) NOT NULL,
	created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
	PRIMARY KEY (permission_code)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci`

func NewPermissionRepository(db *database.DB) PermissionRepository {
	return &permissionRepository{
		db:   db,
		base: database.NewRepository(db),
	}
}

// EnsurePermissionCodes inserts or updates permission catalog entries.
func (r *permissionRepository) EnsurePermissionCodes(codes map[string]string) error {
	if len(codes) == 0 {
		return nil
	}

	return r.db.WithTx(func(tx *sql.Tx) error {
		return r.EnsurePermissionCodesTx(tx, codes)
	})
}

// EnsurePermissionCodesTx inserts or updates permission catalog entries in transaction.
func (r *permissionRepository) EnsurePermissionCodesTx(tx *sql.Tx, codes map[string]string) error {
	if len(codes) == 0 {
		return nil
	}

	keys := make([]string, 0, len(codes))
	for code := range codes {
		trimmed := strings.TrimSpace(code)
		if !utils.HasText(trimmed) {
			continue
		}
		keys = append(keys, trimmed)
	}
	sort.Strings(keys)

	for _, code := range keys {
		description := strings.TrimSpace(codes[code])
		if !utils.HasText(description) {
			description = code
		}
		if err := r.upsertPermissionCodeTx(tx, code, description); err != nil {
			return err
		}
	}
	return nil
}

func (r *permissionRepository) ListPermissions() ([]Permission, error) {
	query := `SELECT permission_code, description FROM _a_permissions ORDER BY permission_code ASC`
	rows, err := r.base.Query(query)
	if err != nil {
		return nil, errors.Wrap(err, "DATABASE_ERROR", "failed to list permissions")
	}
	defer rows.Close()

	permissions := make([]Permission, 0, 16)
	for rows.Next() {
		var p Permission
		if err := rows.Scan(&p.Code, &p.Description); err != nil {
			return nil, errors.Wrap(err, "DATABASE_ERROR", "failed to parse permissions")
		}
		permissions = append(permissions, p)
	}
	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "DATABASE_ERROR", "failed to iterate permissions")
	}

	return permissions, nil
}

func (r *permissionRepository) EnsureAdminPageSchema() error {
	if _, err := r.base.ExecSchema(createAdminPagesTableSQL); err != nil {
		return errors.Wrap(err, "DATABASE_ERROR", "failed to create admin page schema")
	}
	if err := r.ensureAdminPageColumns(); err != nil {
		return err
	}
	if _, err := r.base.ExecSchema(createPermissionsTableSQL); err != nil {
		return errors.Wrap(err, "DATABASE_ERROR", "failed to create permissions schema")
	}
	return nil
}

func (r *permissionRepository) ListAdminPages(includeDisabled bool) ([]AdminPage, error) {
	query := `SELECT page_key, title, path, description, group_key, group_label, group_order, visible_roles, icon, sort_order, is_enabled, is_builtin, COALESCE(created_by, ''), created_at, updated_at
		FROM _a_admin_pages`
	args := make([]interface{}, 0, 1)
	if !includeDisabled {
		query += " WHERE is_enabled = ?"
		args = append(args, 1)
	}
	query += " ORDER BY group_order ASC, sort_order ASC, page_key ASC"

	rows, err := r.base.Query(query, args...)
	if err != nil && isUnknownColumnError(err) {
		if ensureErr := r.EnsureAdminPageSchema(); ensureErr != nil {
			return nil, ensureErr
		}
		rows, err = r.base.Query(query, args...)
	}
	if err != nil {
		return nil, errors.Wrap(err, "DATABASE_ERROR", "failed to list admin pages")
	}
	defer rows.Close()

	pages := make([]AdminPage, 0, 8)
	for rows.Next() {
		page, scanErr := scanAdminPageRow(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		pages = append(pages, *page)
	}
	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "DATABASE_ERROR", "failed to iterate admin pages")
	}

	return pages, nil
}

func (r *permissionRepository) GetAdminPageByKey(pageKey string) (*AdminPage, error) {
	query := `SELECT page_key, title, path, description, group_key, group_label, group_order, visible_roles, icon, sort_order, is_enabled, is_builtin, COALESCE(created_by, ''), created_at, updated_at
		FROM _a_admin_pages WHERE page_key = ? LIMIT 1`
	row := r.base.QueryRow(query, pageKey)
	page, err := scanAdminPageRow(row)
	if err != nil && isUnknownColumnError(err) {
		if ensureErr := r.EnsureAdminPageSchema(); ensureErr != nil {
			return nil, ensureErr
		}
		row = r.base.QueryRow(query, pageKey)
		page, err = scanAdminPageRow(row)
	}
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return page, nil
}

func (r *permissionRepository) CreateAdminPageTx(tx *sql.Tx, page *AdminPage) error {
	if page == nil {
		return errors.New("BAD_REQUEST", "page payload is required")
	}

	data := map[string]interface{}{
		"page_key":      page.Key,
		"title":         page.Title,
		"path":          page.Path,
		"description":   page.Description,
		"group_key":     page.GroupKey,
		"group_label":   page.GroupLabel,
		"group_order":   page.GroupOrder,
		"visible_roles": nullableJSONSlice(page.VisibleRoles),
		"icon":          page.Icon,
		"sort_order":    page.SortOrder,
		"is_enabled":    boolToInt(page.Enabled),
		"is_builtin":    boolToInt(page.Builtin),
		"created_by":    nullableTrimmed(page.CreatedBy),
	}
	if _, err := r.base.Tx(tx).Insert("_a_admin_pages", data); err != nil {
		return errors.Wrap(err, "DATABASE_ERROR", "failed to create admin page")
	}
	return nil
}

func (r *permissionRepository) UpdateAdminPageTx(tx *sql.Tx, page *AdminPage) error {
	if page == nil {
		return errors.New("BAD_REQUEST", "page payload is required")
	}

	data := map[string]interface{}{
		"title":         page.Title,
		"path":          page.Path,
		"description":   page.Description,
		"group_key":     page.GroupKey,
		"group_label":   page.GroupLabel,
		"group_order":   page.GroupOrder,
		"visible_roles": nullableJSONSlice(page.VisibleRoles),
		"icon":          page.Icon,
		"sort_order":    page.SortOrder,
		"is_enabled":    boolToInt(page.Enabled),
		"is_builtin":    boolToInt(page.Builtin),
	}
	affected, err := r.base.Tx(tx).Update("_a_admin_pages", data, "page_key = ?", page.Key)
	if err != nil {
		return errors.Wrap(err, "DATABASE_ERROR", "failed to update admin page")
	}
	if affected == 0 {
		return errors.New("NOT_FOUND", "admin page not found")
	}
	return nil
}

func (r *permissionRepository) DeleteAdminPageTx(tx *sql.Tx, pageKey string) error {
	affected, err := r.base.Tx(tx).Delete("_a_admin_pages", "page_key = ?", pageKey)
	if err != nil {
		return errors.Wrap(err, "DATABASE_ERROR", "failed to delete admin page")
	}
	if affected == 0 {
		return errors.New("NOT_FOUND", "admin page not found")
	}
	return nil
}

func (r *permissionRepository) EnsurePagePermissionCodesTx(tx *sql.Tx, page *AdminPage) error {
	if page == nil {
		return errors.New("BAD_REQUEST", "page payload is required")
	}

	for _, action := range pageActions {
		code := BuildPagePermissionCode(page.Key, action)
		description := pagePermissionDescription(page.Title, action)
		if err := r.upsertPermissionCodeTx(tx, code, description); err != nil {
			return err
		}
	}
	return nil
}

func (r *permissionRepository) DeletePagePermissionCodesTx(tx *sql.Tx, pageKey string) error {
	likeExpr := fmt.Sprintf("admin.page.%s.%%", pageKey)

	txRepo := r.base.Tx(tx)
	if _, err := txRepo.Delete("_a_user_permissions", "permission_code LIKE ?", likeExpr); err != nil {
		return errors.Wrap(err, "DATABASE_ERROR", "failed to delete page user permissions")
	}
	if _, err := txRepo.Delete("_a_delegable_permissions", "permission_code LIKE ?", likeExpr); err != nil {
		return errors.Wrap(err, "DATABASE_ERROR", "failed to delete page delegable permissions")
	}
	if _, err := txRepo.Delete("_a_permissions", "permission_code LIKE ?", likeExpr); err != nil {
		return errors.Wrap(err, "DATABASE_ERROR", "failed to delete page permissions")
	}

	return nil
}

func (r *permissionRepository) ListAuditLogs(req *AdminAuditLogListRequest) (*AdminAuditLogListResponse, error) {
	pagination := utils.NewPagination(req.Page, req.Limit, 20, 100)

	whereClauses := make([]string, 0, 6)
	args := make([]interface{}, 0, 8)

	if utils.HasText(req.Action) {
		actions := splitFilterCSV(req.Action)
		if len(actions) == 1 {
			whereClauses = append(whereClauses, "action = ?")
			args = append(args, actions[0])
		} else if len(actions) > 1 {
			whereClauses = append(whereClauses, "action IN ("+strings.Join(repeat("?", len(actions)), ",")+")")
			for _, action := range actions {
				args = append(args, action)
			}
		}
	}
	appendAuditFilter := func(column string, value string) {
		if !utils.HasText(value) {
			return
		}
		whereClauses = append(whereClauses, column)
		args = append(args, strings.TrimSpace(value))
	}
	appendAuditFilter("actor_id = ?", req.ActorID)
	appendAuditFilter("target_user_id = ?", req.TargetUserID)
	appendAuditFilter("created_at >= ?", req.DateFrom)
	appendAuditFilter("created_at < ?", req.DateTo)

	whereSQL := ""
	if len(whereClauses) > 0 {
		whereSQL = " WHERE " + strings.Join(whereClauses, " AND ")
	}

	countQuery := "SELECT COUNT(*) FROM _a_admin_audit_logs" + whereSQL
	var total int64
	if err := r.db.DB.QueryRow(countQuery, args...).Scan(&total); err != nil {
		logger.Error("failed to count audit logs: %v", err)
		r.base.LogError("PermissionRepository.ListAuditLogs.Count", err.Error(), fmt.Sprintf("%s | Args: %v", strings.TrimSpace(countQuery), args))
		return nil, errors.Wrap(err, "DATABASE_ERROR", "failed to count audit logs")
	}

	listQuery := `SELECT aal_idx, actor_id, target_user_id, action, status, message, ip_addr, before_data, after_data, created_at
		FROM _a_admin_audit_logs` + whereSQL + ` ORDER BY created_at DESC, aal_idx DESC LIMIT ? OFFSET ?`
	listArgs := append(append(make([]interface{}, 0, len(args)+2), args...), pagination.Limit, pagination.Offset)

	rows, err := r.base.Query(listQuery, listArgs...)
	if err != nil {
		return nil, errors.Wrap(err, "DATABASE_ERROR", "failed to list audit logs")
	}
	defer rows.Close()

	logs := make([]AdminAuditLogItem, 0, pagination.Limit)
	for rows.Next() {
		var item AdminAuditLogItem
		var actorID sql.NullString
		var targetID sql.NullString
		var message sql.NullString
		var ipAddr sql.NullString
		var beforeRaw sql.RawBytes
		var afterRaw sql.RawBytes

		if err := rows.Scan(
			&item.ID,
			&actorID,
			&targetID,
			&item.Action,
			&item.Status,
			&message,
			&ipAddr,
			&beforeRaw,
			&afterRaw,
			&item.CreatedAt,
		); err != nil {
			return nil, errors.Wrap(err, "DATABASE_ERROR", "failed to parse audit logs")
		}

		item.ActorID = actorID.String
		item.TargetUserID = targetID.String
		item.Message = message.String
		item.IPAddress = ipAddr.String
		item.BeforeData = parseJSONMap(beforeRaw)
		item.AfterData = parseJSONMap(afterRaw)
		logs = append(logs, item)
	}
	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "DATABASE_ERROR", "failed to iterate audit logs")
	}

	return &AdminAuditLogListResponse{
		Logs:  logs,
		Total: total,
		Page:  pagination.Page,
		Limit: pagination.Limit,
	}, nil
}

func (r *permissionRepository) GetLevelPolicyEnabled() (bool, error) {
	var raw sql.NullString
	query := `SELECT setting_value FROM _a_system_settings WHERE setting_key = 'level_policy_enabled' LIMIT 1`
	logger.Debug("SQL Query: %s", query)
	err := r.db.DB.QueryRow(query).Scan(&raw)
	if err != nil {
		if err == sql.ErrNoRows || isMissingTableErrorForSettings(err) {
			return true, nil
		}
		logger.Error("failed to read level policy setting: %v", err)
		r.base.LogError("PermissionRepository.GetLevelPolicyEnabled", err.Error(), fmt.Sprintf("%s | Args: %v", strings.TrimSpace(query), nil))
		return false, errors.Wrap(err, "DATABASE_ERROR", "failed to read level policy setting")
	}

	value := utils.TrimLower(raw.String)
	switch value {
	case "", "1", "true", "on", "yes", "y":
		return true, nil
	case "0", "false", "off", "no", "n":
		return false, nil
	default:
		return true, nil
	}
}

func (r *permissionRepository) SetLevelPolicyEnabledTx(tx *sql.Tx, enabled bool, updatedBy string) error {
	value := "0"
	if enabled {
		value = "1"
	}
	trimmedUpdatedBy := strings.TrimSpace(updatedBy)
	var updatedByValue interface{}
	if utils.HasText(trimmedUpdatedBy) {
		updatedByValue = trimmedUpdatedBy
	}

	updateData := map[string]interface{}{
		"setting_value": value,
		"updated_by":    updatedByValue,
	}

	txRepo := r.base.Tx(tx)
	affected, err := txRepo.Update("_a_system_settings", updateData, "setting_key = ?", "level_policy_enabled")
	if err != nil && isMissingTableErrorForSettings(err) {
		logger.Warn("_a_system_settings table missing; creating it on-demand")
		if ensureErr := r.ensureSystemSettingsTableTx(tx); ensureErr != nil {
			return ensureErr
		}
		affected, err = txRepo.Update("_a_system_settings", updateData, "setting_key = ?", "level_policy_enabled")
	}
	if err != nil {
		return errors.Wrap(err, "DATABASE_ERROR", "failed to update level policy setting")
	}

	if affected == 0 {
		insertData := map[string]interface{}{
			"setting_key":   "level_policy_enabled",
			"setting_value": value,
			"updated_by":    updatedByValue,
		}
		if _, err := txRepo.Insert("_a_system_settings", insertData); err != nil {
			return errors.Wrap(err, "DATABASE_ERROR", "failed to insert level policy setting")
		}
	}

	return nil
}

func (r *permissionRepository) ensureSystemSettingsTableTx(tx *sql.Tx) error {
	logger.Debug("SQL Exec (TX): create _a_system_settings if not exists")
	if _, err := r.base.Tx(tx).ExecSchema(createSystemSettingsTableSQL); err != nil {
		return errors.Wrap(err, "DATABASE_ERROR", "failed to create system settings table")
	}
	return nil
}

func (r *permissionRepository) ListUserPermissions(userID string) ([]string, error) {
	query := `SELECT permission_code FROM _a_user_permissions WHERE u_id = ? ORDER BY permission_code ASC`
	rows, err := r.base.Query(query, userID)
	if err != nil {
		return nil, errors.Wrap(err, "DATABASE_ERROR", "failed to list user permissions")
	}
	defer rows.Close()

	codes := make([]string, 0, 16)
	for rows.Next() {
		var code string
		if err := rows.Scan(&code); err != nil {
			return nil, errors.Wrap(err, "DATABASE_ERROR", "failed to parse user permissions")
		}
		codes = append(codes, code)
	}
	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "DATABASE_ERROR", "failed to iterate user permissions")
	}

	return codes, nil
}

func (r *permissionRepository) ReplaceUserPermissionsTx(tx *sql.Tx, userID string, permissionCodes []string) error {
	txRepo := r.base.Tx(tx)
	if _, err := txRepo.Delete("_a_user_permissions", "u_id = ?", userID); err != nil {
		return errors.Wrap(err, "DATABASE_ERROR", "failed to clear user permissions")
	}

	for _, code := range permissionCodes {
		data := map[string]interface{}{
			"u_id":            userID,
			"permission_code": code,
		}
		if _, err := txRepo.Insert("_a_user_permissions", data); err != nil {
			return errors.Wrap(err, "DATABASE_ERROR", "failed to assign user permissions")
		}
	}

	return nil
}

func (r *permissionRepository) DeleteUserPermissionsTx(tx *sql.Tx, userID string) error {
	if _, err := r.base.Tx(tx).Delete("_a_user_permissions", "u_id = ?", userID); err != nil {
		return errors.Wrap(err, "DATABASE_ERROR", "failed to delete user permissions")
	}
	return nil
}

func (r *permissionRepository) ListDelegablePermissions() ([]string, error) {
	query := `SELECT permission_code FROM _a_delegable_permissions ORDER BY permission_code ASC`
	rows, err := r.base.Query(query)
	if err != nil {
		return nil, errors.Wrap(err, "DATABASE_ERROR", "failed to list delegable permissions")
	}
	defer rows.Close()

	codes := make([]string, 0, 16)
	for rows.Next() {
		var code string
		if err := rows.Scan(&code); err != nil {
			return nil, errors.Wrap(err, "DATABASE_ERROR", "failed to parse delegable permissions")
		}
		codes = append(codes, code)
	}
	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "DATABASE_ERROR", "failed to iterate delegable permissions")
	}

	return codes, nil
}

func (r *permissionRepository) DelegableSet() (map[string]struct{}, error) {
	codes, err := r.ListDelegablePermissions()
	if err != nil {
		return nil, err
	}

	set := make(map[string]struct{}, len(codes))
	for _, code := range codes {
		set[code] = struct{}{}
	}

	return set, nil
}

func (r *permissionRepository) ReplaceDelegablePermissionsTx(tx *sql.Tx, permissionCodes []string) error {
	txRepo := r.base.Tx(tx)
	if _, err := txRepo.Delete("_a_delegable_permissions", "1 = 1"); err != nil {
		return errors.Wrap(err, "DATABASE_ERROR", "failed to clear delegable permissions")
	}

	for _, code := range permissionCodes {
		data := map[string]interface{}{
			"permission_code": code,
		}
		if _, err := txRepo.Insert("_a_delegable_permissions", data); err != nil {
			return errors.Wrap(err, "DATABASE_ERROR", "failed to update delegable permissions")
		}
	}

	return nil
}

func (r *permissionRepository) PermissionCodesExist(codes []string) (bool, error) {
	if len(codes) == 0 {
		return true, nil
	}

	query := fmt.Sprintf(
		`SELECT COUNT(*) FROM _a_permissions WHERE permission_code IN (%s)`,
		strings.Join(repeat("?", len(codes)), ","),
	)

	args := make([]interface{}, 0, len(codes))
	for _, code := range codes {
		args = append(args, code)
	}

	var count int
	if err := r.db.DB.QueryRow(query, args...).Scan(&count); err != nil {
		logger.Error("failed to validate permission codes: %v", err)
		r.base.LogError("PermissionRepository.PermissionCodesExist", err.Error(), fmt.Sprintf("%s | Args: %v", strings.TrimSpace(query), args))
		return false, errors.Wrap(err, "DATABASE_ERROR", "failed to validate permission codes")
	}

	return count == len(codes), nil
}

func (r *permissionRepository) GrantPermissionFromExistingTx(tx *sql.Tx, targetCode, sourceCode string) error {
	targetCode = strings.TrimSpace(targetCode)
	sourceCode = strings.TrimSpace(sourceCode)
	if !utils.HasText(targetCode) || !utils.HasText(sourceCode) {
		return nil
	}

	query := `INSERT INTO _a_user_permissions (u_id, permission_code)
		SELECT up.u_id, ?
		FROM _a_user_permissions up
		WHERE up.permission_code = ?
		ON DUPLICATE KEY UPDATE permission_code = VALUES(permission_code)`

	if _, err := r.base.Tx(tx).Exec(query, targetCode, sourceCode); err != nil {
		return errors.Wrap(err, "DATABASE_ERROR", "failed to grant permission from source")
	}
	return nil
}

func (r *permissionRepository) WriteAuditLogTx(tx *sql.Tx, entry *AuditLogEntry) error {
	beforeJSON, err := marshalJSON(entry.BeforeData)
	if err != nil {
		logger.Error("failed to encode audit before state: %v", err)
		return errors.Wrap(err, "AUDIT_LOG_FAILED", "failed to encode audit before state")
	}
	afterJSON, err := marshalJSON(entry.AfterData)
	if err != nil {
		logger.Error("failed to encode audit after state: %v", err)
		return errors.Wrap(err, "AUDIT_LOG_FAILED", "failed to encode audit after state")
	}

	logger.Debug("SQL Exec (TX): insert _a_admin_audit_logs, Args: [%s %s %s %s]", entry.ActorID, entry.TargetID, entry.Action, entry.Status)
	data := map[string]interface{}{
		"actor_id":       entry.ActorID,
		"target_user_id": entry.TargetID,
		"action":         entry.Action,
		"status":         entry.Status,
		"message":        entry.Message,
		"ip_addr":        entry.IP,
		"before_data":    beforeJSON,
		"after_data":     afterJSON,
	}
	_, err = r.base.Tx(tx).Insert("_a_admin_audit_logs", data)
	if err != nil {
		return errors.Wrap(err, "AUDIT_LOG_FAILED", "failed to write audit log")
	}

	return nil
}

func (r *permissionRepository) WriteAuditLog(entry *AuditLogEntry) error {
	beforeJSON, err := marshalJSON(entry.BeforeData)
	if err != nil {
		logger.Error("failed to encode audit before state: %v", err)
		return errors.Wrap(err, "AUDIT_LOG_FAILED", "failed to encode audit before state")
	}
	afterJSON, err := marshalJSON(entry.AfterData)
	if err != nil {
		logger.Error("failed to encode audit after state: %v", err)
		return errors.Wrap(err, "AUDIT_LOG_FAILED", "failed to encode audit after state")
	}

	logger.Debug("SQL Exec: insert _a_admin_audit_logs, Args: [%s %s %s %s]", entry.ActorID, entry.TargetID, entry.Action, entry.Status)
	data := map[string]interface{}{
		"actor_id":       entry.ActorID,
		"target_user_id": entry.TargetID,
		"action":         entry.Action,
		"status":         entry.Status,
		"message":        entry.Message,
		"ip_addr":        entry.IP,
		"before_data":    beforeJSON,
		"after_data":     afterJSON,
	}
	_, err = r.base.Insert("_a_admin_audit_logs", data)
	if err != nil {
		return errors.Wrap(err, "AUDIT_LOG_FAILED", "failed to write audit log")
	}

	return nil
}

func marshalJSON(v map[string]interface{}) (string, error) {
	if v == nil {
		return "{}", nil
	}
	raw, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func parseJSONStringSlice(raw []byte) []string {
	if len(raw) == 0 {
		return []string{}
	}
	values := make([]string, 0, 4)
	if err := json.Unmarshal(raw, &values); err != nil {
		return []string{}
	}
	out := utils.UniqueStrings(values)
	sort.Strings(out)
	return out
}

func nullableJSONSlice(values []string) interface{} {
	if len(values) == 0 {
		return nil
	}
	normalized := utils.UniqueStrings(values)
	if len(normalized) == 0 {
		return nil
	}
	sort.Strings(normalized)
	raw, err := json.Marshal(normalized)
	if err != nil {
		return nil
	}
	return string(raw)
}

func parseJSONMap(raw []byte) map[string]interface{} {
	if len(raw) == 0 {
		return map[string]interface{}{}
	}

	out := make(map[string]interface{})
	if err := json.Unmarshal(raw, &out); err != nil {
		return map[string]interface{}{
			"_raw": string(raw),
		}
	}
	return out
}

func repeat(value string, count int) []string {
	out := make([]string, count)
	for i := 0; i < count; i++ {
		out[i] = value
	}
	return out
}

func splitFilterCSV(raw string) []string {
	return utils.UniqueStrings(strings.Split(raw, ","))
}

func isMissingTableErrorForSettings(err error) bool {
	if err == nil {
		return false
	}
	msg := utils.TrimLower(err.Error())
	return strings.Contains(msg, "doesn't exist") || strings.Contains(msg, "no such table")
}

func scanAdminPageRow(scanner interface {
	Scan(dest ...interface{}) error
}) (*AdminPage, error) {
	var page AdminPage
	var enabledInt int
	var builtinInt int
	var rolesRaw []byte
	if err := scanner.Scan(
		&page.Key,
		&page.Title,
		&page.Path,
		&page.Description,
		&page.GroupKey,
		&page.GroupLabel,
		&page.GroupOrder,
		&rolesRaw,
		&page.Icon,
		&page.SortOrder,
		&enabledInt,
		&builtinInt,
		&page.CreatedBy,
		&page.CreatedAt,
		&page.UpdatedAt,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, err
		}
		return nil, errors.Wrap(err, "DATABASE_ERROR", "failed to parse admin page")
	}

	page.Enabled = enabledInt == 1
	page.Builtin = builtinInt == 1
	page.VisibleRoles = parseJSONStringSlice(rolesRaw)
	page.PermissionCodes = AdminPagePermissionCodes{
		Read:   BuildPagePermissionCode(page.Key, PageActionRead),
		Create: BuildPagePermissionCode(page.Key, PageActionCreate),
		Update: BuildPagePermissionCode(page.Key, PageActionUpdate),
		Delete: BuildPagePermissionCode(page.Key, PageActionDelete),
	}
	return &page, nil
}

func (r *permissionRepository) upsertPermissionCodeTx(tx *sql.Tx, code string, description string) error {
	txRepo := r.base.Tx(tx)
	affected, err := txRepo.Update("_a_permissions", map[string]interface{}{
		"description": description,
	}, "permission_code = ?", code)
	if err != nil && isMissingTableErrorForSettings(err) {
		if ensureErr := r.ensurePermissionsTableTx(tx); ensureErr != nil {
			return ensureErr
		}
		affected, err = txRepo.Update("_a_permissions", map[string]interface{}{
			"description": description,
		}, "permission_code = ?", code)
	}
	if err != nil {
		return errors.Wrap(err, "DATABASE_ERROR", "failed to update page permission")
	}
	if affected > 0 {
		return nil
	}

	var existingCount int
	if err := r.base.QueryRowTx(tx, "SELECT COUNT(*) FROM _a_permissions WHERE permission_code = ?", code).Scan(&existingCount); err != nil {
		return errors.Wrap(err, "DATABASE_ERROR", "failed to check page permission existence")
	}
	if existingCount > 0 {
		return nil
	}

	if _, err := txRepo.Insert("_a_permissions", map[string]interface{}{
		"permission_code": code,
		"description":     description,
	}); err != nil {
		if isDuplicateKeyError(err) {
			return nil
		}
		return errors.Wrap(err, "DATABASE_ERROR", "failed to create page permission")
	}
	return nil
}

func (r *permissionRepository) ensurePermissionsTableTx(tx *sql.Tx) error {
	if _, err := r.base.Tx(tx).ExecSchema(createPermissionsTableSQL); err != nil {
		return errors.Wrap(err, "DATABASE_ERROR", "failed to create permissions table")
	}
	return nil
}

func pagePermissionDescription(pageTitle string, action string) string {
	trimmedTitle := strings.TrimSpace(pageTitle)
	if !utils.HasText(trimmedTitle) {
		trimmedTitle = "admin page"
	}

	switch action {
	case PageActionRead:
		return fmt.Sprintf("Read page: %s", trimmedTitle)
	case PageActionCreate:
		return fmt.Sprintf("Create resources on page: %s", trimmedTitle)
	case PageActionUpdate:
		return fmt.Sprintf("Update resources on page: %s", trimmedTitle)
	case PageActionDelete:
		return fmt.Sprintf("Delete resources on page: %s", trimmedTitle)
	default:
		return fmt.Sprintf("Manage page permission: %s", trimmedTitle)
	}
}

func nullableTrimmed(value string) interface{} {
	trimmed := strings.TrimSpace(value)
	if !utils.HasText(trimmed) {
		return nil
	}
	return trimmed
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func isDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}
	msg := utils.TrimLower(err.Error())
	return strings.Contains(msg, "duplicate entry") || strings.Contains(msg, "unique constraint")
}

func (r *permissionRepository) ensureAdminPageColumns() error {
	columnStatements := []struct {
		name      string
		statement string
	}{
		{name: "group_key", statement: "ALTER TABLE _a_admin_pages ADD COLUMN group_key VARCHAR(48) NOT NULL DEFAULT 'general' AFTER description"},
		{name: "group_label", statement: "ALTER TABLE _a_admin_pages ADD COLUMN group_label VARCHAR(100) NOT NULL DEFAULT 'General' AFTER group_key"},
		{name: "group_order", statement: "ALTER TABLE _a_admin_pages ADD COLUMN group_order INT NOT NULL DEFAULT 100 AFTER group_label"},
		{name: "visible_roles", statement: "ALTER TABLE _a_admin_pages ADD COLUMN visible_roles JSON NULL AFTER group_order"},
	}

	for _, column := range columnStatements {
		exists, err := r.adminPageColumnExists(column.name)
		if err != nil {
			return err
		}
		if exists {
			continue
		}
		if _, err := r.base.ExecSchema(column.statement); err != nil {
			if isDuplicateColumnError(err) {
				continue
			}
			return errors.Wrap(err, "DATABASE_ERROR", "failed to extend admin page schema")
		}
	}

	const indexName = "idx_a_admin_pages_enabled_group_sort"
	indexExists, err := r.adminPageIndexExists(indexName)
	if err != nil {
		return err
	}
	if indexExists {
		return nil
	}
	if _, err := r.base.ExecSchema("ALTER TABLE _a_admin_pages ADD KEY idx_a_admin_pages_enabled_group_sort (is_enabled, group_order, sort_order)"); err != nil {
		if isDuplicateIndexNameError(err) {
			return nil
		}
		return errors.Wrap(err, "DATABASE_ERROR", "failed to extend admin page schema")
	}
	return nil
}

func (r *permissionRepository) adminPageColumnExists(columnName string) (bool, error) {
	var count int
	err := r.base.QueryRow(`
		SELECT COUNT(*)
		FROM information_schema.COLUMNS
		WHERE TABLE_SCHEMA = DATABASE()
		  AND TABLE_NAME = '_a_admin_pages'
		  AND COLUMN_NAME = ?
	`, columnName).Scan(&count)
	if err != nil {
		return false, errors.Wrap(err, "DATABASE_ERROR", "failed to inspect admin page columns")
	}
	return count > 0, nil
}

func (r *permissionRepository) adminPageIndexExists(indexName string) (bool, error) {
	var count int
	err := r.base.QueryRow(`
		SELECT COUNT(*)
		FROM information_schema.STATISTICS
		WHERE TABLE_SCHEMA = DATABASE()
		  AND TABLE_NAME = '_a_admin_pages'
		  AND INDEX_NAME = ?
	`, indexName).Scan(&count)
	if err != nil {
		return false, errors.Wrap(err, "DATABASE_ERROR", "failed to inspect admin page indexes")
	}
	return count > 0, nil
}

func isDuplicateColumnError(err error) bool {
	if err == nil {
		return false
	}
	msg := utils.TrimLower(err.Error())
	return strings.Contains(msg, "duplicate column name")
}

func isDuplicateIndexNameError(err error) bool {
	if err == nil {
		return false
	}
	msg := utils.TrimLower(err.Error())
	return strings.Contains(msg, "duplicate key name")
}

func isUnknownColumnError(err error) bool {
	if err == nil {
		return false
	}
	msg := utils.TrimLower(err.Error())
	return strings.Contains(msg, "unknown column")
}
