package user

import (
	"gin_starter/internal/config"
	"gin_starter/internal/middleware"
	"gin_starter/pkg/errors"
	"gin_starter/pkg/logger"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// Service 사용자 서비스 인터페이스
type Service interface {
	Register(req *CreateUserRequest) (*User, error)
	Login(req *LoginRequest) (*LoginResponse, error)
	GetProfile(userID string) (*User, error)
	UpdateProfile(userID string, req *UpdateUserRequest) error
	RefreshToken(req *RefreshTokenRequest) (*RefreshTokenResponse, error)
	Logout(userID string) error
}

type service struct {
	repo   Repository
	config *config.Config
}

// NewService 서비스 생성자
func NewService(repo Repository, cfg *config.Config) Service {
	return &service{
		repo:   repo,
		config: cfg,
	}
}

// Register 회원가입
func (s *service) Register(req *CreateUserRequest) (*User, error) {
	// 중복 체크
	exists, err := s.repo.Exists(req.ID)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, errors.ErrUserExists
	}

	// 비밀번호 해싱
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		logger.Error("비밀번호 해싱 실패: %v", err)
		return nil, errors.Wrap(err, "PASSWORD_HASH_FAILED", "비밀번호 처리에 실패했습니다")
	}

	// 사용자 생성
	user := &User{
		ID:        req.ID,
		Password:  string(hashedPassword),
		Name:      req.Name,
		Email:     req.Email,
		AuthType:  "U", // 일반 사용자
		AuthLevel: 1,   // 기본 레벨
		CreatedAt: time.Now(),
	}

	if err := s.repo.Create(user); err != nil {
		return nil, err
	}

	logger.Info("새 사용자 등록 완료: %s", user.ID)
	return user.ToPublic(), nil
}

// Login 로그인
func (s *service) Login(req *LoginRequest) (*LoginResponse, error) {
	// 사용자 조회
	user, err := s.repo.FindByID(req.ID)
	if err != nil {
		if errors.Is(err, errors.ErrUserNotFound) {
			return nil, errors.ErrInvalidCredentials
		}
		return nil, err
	}

	// 비밀번호 확인
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		logger.Warn("로그인 실패 (잘못된 비밀번호): %s", req.ID)
		return nil, errors.ErrInvalidCredentials
	}

	// 토큰 생성
	accessToken, err := middleware.GenerateToken(
		user.ID,
		s.config.JWT.AccessExpireMin,
		s.config.JWT.AccessSecret,
		s.config.JWT.TokenSecret,
		s.config.App.ServiceName,
	)
	if err != nil {
		logger.Error("액세스 토큰 생성 실패: %v", err)
		return nil, errors.Wrap(err, "TOKEN_GENERATION_FAILED", "토큰 생성에 실패했습니다")
	}

	refreshToken, err := middleware.GenerateToken(
		user.ID,
		s.config.JWT.RefreshExpireDays*24*60, // 일 -> 분
		s.config.JWT.RefreshSecret,
		s.config.JWT.TokenSecret,
		s.config.App.ServiceName,
	)
	if err != nil {
		logger.Error("리프레시 토큰 생성 실패: %v", err)
		return nil, errors.Wrap(err, "TOKEN_GENERATION_FAILED", "토큰 생성에 실패했습니다")
	}

	// 리프레시 토큰 DB 저장
	if err := s.repo.UpdateRefreshToken(user.ID, refreshToken); err != nil {
		return nil, err
	}

	logger.Info("로그인 성공: %s", user.ID)

	return &LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         user.ToPublic(),
	}, nil
}

// GetProfile 프로필 조회
func (s *service) GetProfile(userID string) (*User, error) {
	user, err := s.repo.FindByID(userID)
	if err != nil {
		return nil, err
	}

	return user.ToPublic(), nil
}

// UpdateProfile 프로필 수정
func (s *service) UpdateProfile(userID string, req *UpdateUserRequest) error {
	updates := make(map[string]interface{})

	if req.Name != "" {
		updates["u_name"] = req.Name
	}

	if req.Email != "" {
		updates["u_email"] = req.Email
	}

	if req.Password != "" {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			logger.Error("비밀번호 해싱 실패: %v", err)
			return errors.Wrap(err, "PASSWORD_HASH_FAILED", "비밀번호 처리에 실패했습니다")
		}
		updates["u_pass"] = string(hashedPassword)
	}

	if len(updates) == 0 {
		return errors.New("NO_UPDATES", "수정할 내용이 없습니다")
	}

	if err := s.repo.Update(userID, updates); err != nil {
		return err
	}

	logger.Info("프로필 수정 완료: %s", userID)
	return nil
}

// RefreshToken 토큰 갱신
func (s *service) RefreshToken(req *RefreshTokenRequest) (*RefreshTokenResponse, error) {
	// 리프레시 토큰 검증
	claims, err := middleware.ValidateToken(
		req.RefreshToken,
		s.config.JWT.RefreshSecret,
		s.config.JWT.TokenSecret,
	)
	if err != nil {
		return nil, err
	}

	// DB에 저장된 토큰과 대조
	user, err := s.repo.FindByID(claims.UserID)
	if err != nil {
		return nil, err
	}

	if user.RefreshToken != req.RefreshToken {
		logger.Warn("유효하지 않은 리프레시 토큰: %s", claims.UserID)
		return nil, errors.ErrInvalidToken
	}

	// 새 액세스 토큰 생성
	accessToken, err := middleware.GenerateToken(
		user.ID,
		s.config.JWT.AccessExpireMin,
		s.config.JWT.AccessSecret,
		s.config.JWT.TokenSecret,
		s.config.App.ServiceName,
	)
	if err != nil {
		return nil, errors.Wrap(err, "TOKEN_GENERATION_FAILED", "토큰 생성에 실패했습니다")
	}

	// 리프레시 토큰 재사용 판단 (24시간 이상 남았으면 재사용)
	newRefreshToken := req.RefreshToken
	if time.Until(claims.ExpiresAt.Time) < time.Duration(s.config.JWT.RefreshReuseHours)*time.Hour {
		// 새 리프레시 토큰 생성
		newRefreshToken, err = middleware.GenerateToken(
			user.ID,
			s.config.JWT.RefreshExpireDays*24*60,
			s.config.JWT.RefreshSecret,
			s.config.JWT.TokenSecret,
			s.config.App.ServiceName,
		)
		if err != nil {
			return nil, errors.Wrap(err, "TOKEN_GENERATION_FAILED", "토큰 생성에 실패했습니다")
		}

		// DB 업데이트
		if err := s.repo.UpdateRefreshToken(user.ID, newRefreshToken); err != nil {
			return nil, err
		}
	}

	logger.Info("토큰 갱신 완료: %s", user.ID)

	return &RefreshTokenResponse{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
	}, nil
}

// Logout 로그아웃
func (s *service) Logout(userID string) error {
	// 리프레시 토큰 삭제
	if err := s.repo.UpdateRefreshToken(userID, ""); err != nil {
		return err
	}

	logger.Info("로그아웃 완료: %s", userID)
	return nil
}