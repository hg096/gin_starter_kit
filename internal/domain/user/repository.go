package user

import (
	"database/sql"
	"gin_starter/internal/infrastructure/database"
	"gin_starter/pkg/errors"
	"gin_starter/pkg/logger"
)

// Repository 사용자 리포지토리 인터페이스
type Repository interface {
	Create(user *User) error
	CreateTx(tx *sql.Tx, user *User) error
	FindByID(id string) (*User, error)
	FindByEmail(email string) (*User, error)
	Update(id string, updates map[string]interface{}) error
	UpdateTx(tx *sql.Tx, id string, updates map[string]interface{}) error
	Delete(id string) error
	Exists(id string) (bool, error)
	UpdateRefreshToken(id string, refreshToken string) error
	UpdateRefreshTokenTx(tx *sql.Tx, id string, refreshToken string) error
}

type repository struct {
	base *database.Repository
}

// NewRepository 리포지토리 생성자
func NewRepository(db *database.DB) Repository {
	return &repository{
		base: database.NewRepository(db),
	}
}

// Create 사용자 생성
func (r *repository) Create(user *User) error {
	data := map[string]interface{}{
		"u_id":         user.ID,
		"u_pass":       user.Password,
		"u_name":       user.Name,
		"u_email":      user.Email,
		"u_auth_type":  user.AuthType,
		"u_auth_level": user.AuthLevel,
	}

	_, err := r.base.Insert("_user", data)
	if err != nil {
		logger.Error("사용자 생성 실패: %v", err)
		return errors.Wrap(err, "USER_CREATE_FAILED", "사용자 생성에 실패했습니다")
	}

	return nil
}

// CreateTx 트랜잭션 내에서 사용자 생성
func (r *repository) CreateTx(tx *sql.Tx, user *User) error {
	data := map[string]interface{}{
		"u_id":         user.ID,
		"u_pass":       user.Password,
		"u_name":       user.Name,
		"u_email":      user.Email,
		"u_auth_type":  user.AuthType,
		"u_auth_level": user.AuthLevel,
	}

	_, err := r.base.InsertTx(tx, "_user", data)
	if err != nil {
		logger.Error("사용자 생성 실패 (TX): %v", err)
		return errors.Wrap(err, "USER_CREATE_FAILED", "사용자 생성에 실패했습니다")
	}

	return nil
}

// FindByID ID로 사용자 조회
func (r *repository) FindByID(id string) (*User, error) {
	query := `SELECT u_id, u_pass, u_name, u_email, u_auth_type, u_auth_level, u_re_token, u_regi_date
	          FROM _user WHERE u_id = ?`

	user := &User{}
	err := r.base.QueryRow(query, id).Scan(
		&user.ID, &user.Password, &user.Name, &user.Email,
		&user.AuthType, &user.AuthLevel, &user.RefreshToken, &user.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, errors.ErrUserNotFound
	}

	if err != nil {
		logger.Error("사용자 조회 실패 (ID: %s): %v", id, err)
		return nil, errors.Wrap(err, "USER_FIND_FAILED", "사용자 조회에 실패했습니다")
	}

	return user, nil
}

// FindByEmail 이메일로 사용자 조회
func (r *repository) FindByEmail(email string) (*User, error) {
	query := `SELECT u_id, u_pass, u_name, u_email, u_auth_type, u_auth_level, u_re_token, u_regi_date
	          FROM _user WHERE u_email = ?`

	user := &User{}
	err := r.base.QueryRow(query, email).Scan(
		&user.ID, &user.Password, &user.Name, &user.Email,
		&user.AuthType, &user.AuthLevel, &user.RefreshToken, &user.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, errors.ErrUserNotFound
	}

	if err != nil {
		logger.Error("사용자 조회 실패 (Email: %s): %v", email, err)
		return nil, errors.Wrap(err, "USER_FIND_FAILED", "사용자 조회에 실패했습니다")
	}

	return user, nil
}

// Update 사용자 정보 수정
func (r *repository) Update(id string, updates map[string]interface{}) error {
	affected, err := r.base.Update("_user", updates, "u_id = ?", id)
	if err != nil {
		logger.Error("사용자 수정 실패 (ID: %s): %v", id, err)
		return errors.Wrap(err, "USER_UPDATE_FAILED", "사용자 수정에 실패했습니다")
	}

	if affected == 0 {
		return errors.ErrUserNotFound
	}

	return nil
}

// UpdateTx 트랜잭션 내에서 사용자 정보 수정
func (r *repository) UpdateTx(tx *sql.Tx, id string, updates map[string]interface{}) error {
	affected, err := r.base.UpdateTx(tx, "_user", updates, "u_id = ?", id)
	if err != nil {
		logger.Error("사용자 수정 실패 (TX, ID: %s): %v", id, err)
		return errors.Wrap(err, "USER_UPDATE_FAILED", "사용자 수정에 실패했습니다")
	}

	if affected == 0 {
		return errors.ErrUserNotFound
	}

	return nil
}

// Delete 사용자 삭제
func (r *repository) Delete(id string) error {
	affected, err := r.base.Delete("_user", "u_id = ?", id)
	if err != nil {
		logger.Error("사용자 삭제 실패 (ID: %s): %v", id, err)
		return errors.Wrap(err, "USER_DELETE_FAILED", "사용자 삭제에 실패했습니다")
	}

	if affected == 0 {
		return errors.ErrUserNotFound
	}

	return nil
}

// Exists 사용자 존재 여부 확인
func (r *repository) Exists(id string) (bool, error) {
	exists, err := r.base.Exists("_user", "u_id = ?", id)
	if err != nil {
		logger.Error("사용자 존재 확인 실패 (ID: %s): %v", id, err)
		return false, errors.Wrap(err, "USER_EXISTS_CHECK_FAILED", "사용자 존재 확인에 실패했습니다")
	}

	return exists, nil
}

// UpdateRefreshToken 리프레시 토큰 업데이트
func (r *repository) UpdateRefreshToken(id string, refreshToken string) error {
	updates := map[string]interface{}{
		"u_re_token": refreshToken,
	}

	return r.Update(id, updates)
}

// UpdateRefreshTokenTx 트랜잭션 내에서 리프레시 토큰 업데이트
func (r *repository) UpdateRefreshTokenTx(tx *sql.Tx, id string, refreshToken string) error {
	updates := map[string]interface{}{
		"u_re_token": refreshToken,
	}

	return r.UpdateTx(tx, id, updates)
}