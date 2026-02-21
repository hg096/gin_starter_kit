package user

import (
	"database/sql"
	"gin_starter/internal/infrastructure/database"
	"gin_starter/pkg/authz"
	"gin_starter/pkg/errors"
	"gin_starter/pkg/logger"
	"strings"
	"time"
)

// Repository defines user persistence APIs.
type Repository interface {
	Create(user *User) error
	CreateTx(tx *sql.Tx, user *User) error
	FindByID(id string) (*User, error)
	FindByEmail(email string) (*User, error)
	FindAuthSnapshotByID(id string) (*AuthSnapshot, error)
	Update(id string, updates map[string]interface{}) error
	UpdateTx(tx *sql.Tx, id string, updates map[string]interface{}) error
	Delete(id string) error
	DeleteTx(tx *sql.Tx, id string) error
	Exists(id string) (bool, error)
	UpdateRefreshToken(id string, refreshToken string) error
	UpdateRefreshTokenTx(tx *sql.Tx, id string, refreshToken string) error
	UpdateStatus(id string, status string) error
	BumpTokenValidAfter(id string, at time.Time) error
	GetLevelPolicyEnabled() (bool, error)
}

type repository struct {
	base *database.Repository
}

// NewRepository creates a user repository.
func NewRepository(db *database.DB) Repository {
	return &repository{base: database.NewRepository(db)}
}

func (r *repository) Create(user *User) error {
	data := map[string]interface{}{
		"u_id":         user.ID,
		"u_pass":       user.Password,
		"u_name":       user.Name,
		"u_email":      user.Email,
		"u_auth_type":  user.AuthType,
		"u_auth_level": user.AuthLevel,
		"u_status":     normalizeStatus(user.Status),
	}

	_, err := r.base.Insert("_user", data)
	if err != nil {
		logger.Error("failed to create user: %v", err)
		return errors.Wrap(err, "USER_CREATE_FAILED", "failed to create user")
	}

	return nil
}

func (r *repository) CreateTx(tx *sql.Tx, user *User) error {
	data := map[string]interface{}{
		"u_id":         user.ID,
		"u_pass":       user.Password,
		"u_name":       user.Name,
		"u_email":      user.Email,
		"u_auth_type":  user.AuthType,
		"u_auth_level": user.AuthLevel,
		"u_status":     normalizeStatus(user.Status),
	}

	_, err := r.base.InsertTx(tx, "_user", data)
	if err != nil {
		logger.Error("failed to create user (tx): %v", err)
		return errors.Wrap(err, "USER_CREATE_FAILED", "failed to create user")
	}

	return nil
}

func (r *repository) FindByID(id string) (*User, error) {
	query := `SELECT u_id, u_pass, u_name, u_email, u_auth_type, u_auth_level, COALESCE(u_status, 'active'), u_token_valid_after, u_re_token, u_regi_date
			  FROM _user WHERE u_id = ?`

	row := r.base.QueryRow(query, id)
	user, err := scanUser(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.ErrUserNotFound
		}
		logger.Error("failed to find user by id (%s): %v", id, err)
		return nil, errors.Wrap(err, "USER_FIND_FAILED", "failed to find user")
	}

	return user, nil
}

func (r *repository) FindByEmail(email string) (*User, error) {
	query := `SELECT u_id, u_pass, u_name, u_email, u_auth_type, u_auth_level, COALESCE(u_status, 'active'), u_token_valid_after, u_re_token, u_regi_date
			  FROM _user WHERE u_email = ?`

	row := r.base.QueryRow(query, email)
	user, err := scanUser(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.ErrUserNotFound
		}
		logger.Error("failed to find user by email (%s): %v", email, err)
		return nil, errors.Wrap(err, "USER_FIND_FAILED", "failed to find user")
	}

	return user, nil
}

func (r *repository) FindAuthSnapshotByID(id string) (*AuthSnapshot, error) {
	query := `SELECT u_id, u_auth_type, u_auth_level, COALESCE(u_status, 'active'), u_token_valid_after FROM _user WHERE u_id = ?`

	var snapshot AuthSnapshot
	var tokenValidAfter sql.NullTime
	err := r.base.QueryRow(query, id).Scan(&snapshot.ID, &snapshot.AuthType, &snapshot.AuthLevel, &snapshot.Status, &tokenValidAfter)
	if err == sql.ErrNoRows {
		return nil, errors.ErrUserNotFound
	}
	if err != nil {
		logger.Error("failed to find auth snapshot by id (%s): %v", id, err)
		return nil, errors.Wrap(err, "USER_FIND_FAILED", "failed to load auth snapshot")
	}

	if tokenValidAfter.Valid {
		t := tokenValidAfter.Time
		snapshot.TokenValidAfter = &t
	}
	snapshot.AuthType = authz.NormalizeAuthType(snapshot.AuthType)

	if snapshot.Status == "" {
		snapshot.Status = UserStatusActive
	}

	return &snapshot, nil
}

func (r *repository) Update(id string, updates map[string]interface{}) error {
	affected, err := r.base.Update("_user", updates, "u_id = ?", id)
	if err != nil {
		logger.Error("failed to update user (%s): %v", id, err)
		return errors.Wrap(err, "USER_UPDATE_FAILED", "failed to update user")
	}
	if affected == 0 {
		return errors.ErrUserNotFound
	}

	return nil
}

func (r *repository) UpdateTx(tx *sql.Tx, id string, updates map[string]interface{}) error {
	affected, err := r.base.UpdateTx(tx, "_user", updates, "u_id = ?", id)
	if err != nil {
		logger.Error("failed to update user (tx, %s): %v", id, err)
		return errors.Wrap(err, "USER_UPDATE_FAILED", "failed to update user")
	}
	if affected == 0 {
		return errors.ErrUserNotFound
	}

	return nil
}

func (r *repository) Delete(id string) error {
	affected, err := r.base.Delete("_user", "u_id = ?", id)
	if err != nil {
		logger.Error("failed to delete user (%s): %v", id, err)
		return errors.Wrap(err, "USER_DELETE_FAILED", "failed to delete user")
	}
	if affected == 0 {
		return errors.ErrUserNotFound
	}

	return nil
}

func (r *repository) DeleteTx(tx *sql.Tx, id string) error {
	affected, err := r.base.DeleteTx(tx, "_user", "u_id = ?", id)
	if err != nil {
		logger.Error("failed to delete user (tx, %s): %v", id, err)
		return errors.Wrap(err, "USER_DELETE_FAILED", "failed to delete user")
	}
	if affected == 0 {
		return errors.ErrUserNotFound
	}

	return nil
}

func (r *repository) Exists(id string) (bool, error) {
	exists, err := r.base.Exists("_user", "u_id = ?", id)
	if err != nil {
		logger.Error("failed to check user exists (%s): %v", id, err)
		return false, errors.Wrap(err, "USER_EXISTS_CHECK_FAILED", "failed to check user existence")
	}

	return exists, nil
}

func (r *repository) UpdateRefreshToken(id string, refreshToken string) error {
	updates := map[string]interface{}{"u_re_token": refreshToken}
	return r.Update(id, updates)
}

func (r *repository) UpdateRefreshTokenTx(tx *sql.Tx, id string, refreshToken string) error {
	updates := map[string]interface{}{"u_re_token": refreshToken}
	return r.UpdateTx(tx, id, updates)
}

func (r *repository) UpdateStatus(id string, status string) error {
	updates := map[string]interface{}{
		"u_status": normalizeStatus(status),
	}
	return r.Update(id, updates)
}

func (r *repository) BumpTokenValidAfter(id string, at time.Time) error {
	updates := map[string]interface{}{
		"u_token_valid_after": at,
	}
	return r.Update(id, updates)
}

func (r *repository) GetLevelPolicyEnabled() (bool, error) {
	var raw sql.NullString
	err := r.base.QueryRow(`SELECT setting_value FROM _a_system_settings WHERE setting_key = 'level_policy_enabled' LIMIT 1`).Scan(&raw)
	if err != nil {
		if err == sql.ErrNoRows || isMissingTableError(err) {
			return true, nil
		}
		return false, errors.Wrap(err, "DATABASE_ERROR", "failed to read level policy setting")
	}

	value := strings.ToLower(strings.TrimSpace(raw.String))
	switch value {
	case "", "1", "true", "on", "yes", "y":
		return true, nil
	case "0", "false", "off", "no", "n":
		return false, nil
	default:
		return true, nil
	}
}

func scanUser(scanner interface {
	Scan(dest ...interface{}) error
}) (*User, error) {
	var user User
	var tokenValidAfter sql.NullTime
	var refreshToken sql.NullString
	if err := scanner.Scan(
		&user.ID,
		&user.Password,
		&user.Name,
		&user.Email,
		&user.AuthType,
		&user.AuthLevel,
		&user.Status,
		&tokenValidAfter,
		&refreshToken,
		&user.CreatedAt,
	); err != nil {
		return nil, err
	}
	if tokenValidAfter.Valid {
		t := tokenValidAfter.Time
		user.TokenValidAfter = &t
	}
	if refreshToken.Valid {
		user.RefreshToken = refreshToken.String
	} else {
		user.RefreshToken = ""
	}
	user.AuthType = authz.NormalizeAuthType(user.AuthType)
	if user.Status == "" {
		user.Status = UserStatusActive
	}
	return &user, nil
}

func normalizeStatus(status string) string {
	s := status
	if s == "" {
		s = UserStatusActive
	}
	if s != UserStatusActive && s != UserStatusLocked {
		s = UserStatusActive
	}
	return s
}

func isMissingTableError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "doesn't exist") || strings.Contains(msg, "no such table")
}
