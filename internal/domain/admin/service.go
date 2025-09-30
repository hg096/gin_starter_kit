package admin

import (
	"gin_starter/internal/domain/user"
	"gin_starter/internal/infrastructure/database"
	"gin_starter/pkg/errors"
	"gin_starter/pkg/logger"
)

// Service 관리자 비즈니스 로직 인터페이스
type Service interface {
	GetAllUsers(page, limit int, userType string) (*AdminUserListResponse, error)
	GetUserByID(id string) (*user.User, error)
	UpdateUserAuth(id string, authType string, authLevel int) error
	DeleteUser(id string) error
	GetStats() (*AdminStatsResponse, error)
}

type service struct {
	userRepo user.Repository
	db       *database.DB
}

// NewService 관리자 서비스 생성
func NewService(userRepo user.Repository, db *database.DB) Service {
	return &service{
		userRepo: userRepo,
		db:       db,
	}
}

// GetAllUsers 모든 사용자 조회 (페이지네이션)
func (s *service) GetAllUsers(page, limit int, userType string) (*AdminUserListResponse, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	offset := (page - 1) * limit

	// 쿼리 생성
	query := "SELECT u_id, u_name, u_email, u_auth_type, u_auth_level, u_reg_date FROM _user"
	countQuery := "SELECT COUNT(*) FROM _user"
	var args []interface{}

	if userType != "" {
		query += " WHERE u_auth_type = ?"
		countQuery += " WHERE u_auth_type = ?"
		args = append(args, userType)
	}

	query += " ORDER BY u_reg_date DESC LIMIT ? OFFSET ?"

	// 전체 개수 조회
	var total int64
	err := s.db.DB.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		logger.Error("사용자 개수 조회 실패: %v", err)
		return nil, errors.Wrap(err, "DATABASE_ERROR", "사용자 개수 조회 실패")
	}

	// 사용자 목록 조회
	queryArgs := append(args, limit, offset)
	rows, err := s.db.DB.Query(query, queryArgs...)
	if err != nil {
		logger.Error("사용자 목록 조회 실패: %v", err)
		return nil, errors.Wrap(err, "DATABASE_ERROR", "사용자 목록 조회 실패")
	}
	defer rows.Close()

	var users []user.User
	for rows.Next() {
		var u user.User
		if err := rows.Scan(&u.ID, &u.Name, &u.Email, &u.AuthType, &u.AuthLevel, &u.CreatedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}

	return &AdminUserListResponse{
		Users: users,
		Total: total,
		Page:  page,
		Limit: limit,
	}, nil
}

// GetUserByID 사용자 상세 조회
func (s *service) GetUserByID(id string) (*user.User, error) {
	return s.userRepo.FindByID(id)
}

// UpdateUserAuth 사용자 권한 수정
func (s *service) UpdateUserAuth(id string, authType string, authLevel int) error {
	// 사용자 존재 확인
	_, err := s.userRepo.FindByID(id)
	if err != nil {
		return errors.New("USER_NOT_FOUND", "사용자를 찾을 수 없습니다")
	}

	// 권한 타입 검증
	if authType != "U" && authType != "A" {
		return errors.New("INVALID_AUTH_TYPE", "권한 타입은 U 또는 A여야 합니다")
	}

	// 권한 레벨 검증
	if authLevel < 1 || authLevel > 10 {
		return errors.New("INVALID_AUTH_LEVEL", "권한 레벨은 1-10 사이여야 합니다")
	}

	// 업데이트
	updates := map[string]interface{}{
		"u_auth_type":  authType,
		"u_auth_level": authLevel,
	}

	if err := s.userRepo.Update(id, updates); err != nil {
		logger.Error("사용자 권한 수정 실패: %v", err)
		return errors.Wrap(err, "UPDATE_FAILED", "사용자 권한 수정 실패")
	}

	logger.Info("사용자 권한 수정: %s (타입: %s, 레벨: %d)", id, authType, authLevel)
	return nil
}

// DeleteUser 사용자 삭제
func (s *service) DeleteUser(id string) error {
	// 사용자 존재 확인
	_, err := s.userRepo.FindByID(id)
	if err != nil {
		return errors.New("USER_NOT_FOUND", "사용자를 찾을 수 없습니다")
	}

	// 삭제
	if err := s.userRepo.Delete(id); err != nil {
		logger.Error("사용자 삭제 실패: %v", err)
		return errors.Wrap(err, "DELETE_FAILED", "사용자 삭제 실패")
	}

	logger.Info("사용자 삭제: %s", id)
	return nil
}

// GetStats 통계 조회
func (s *service) GetStats() (*AdminStatsResponse, error) {
	stats := &AdminStatsResponse{}

	// 전체 사용자 수
	err := s.db.DB.QueryRow("SELECT COUNT(*) FROM _user").Scan(&stats.TotalUsers)
	if err != nil {
		return nil, err
	}

	// 관리자 수
	err = s.db.DB.QueryRow("SELECT COUNT(*) FROM _user WHERE u_auth_type = 'A'").Scan(&stats.AdminUsers)
	if err != nil {
		return nil, err
	}

	// 일반 사용자 수
	err = s.db.DB.QueryRow("SELECT COUNT(*) FROM _user WHERE u_auth_type = 'U'").Scan(&stats.NormalUsers)
	if err != nil {
		return nil, err
	}

	// 전체 블로그 수
	err = s.db.DB.QueryRow("SELECT COUNT(*) FROM _blog").Scan(&stats.TotalBlogs)
	if err != nil {
		// 블로그 테이블이 없을 수 있으므로 에러 무시
		stats.TotalBlogs = 0
	}

	return stats, nil
}