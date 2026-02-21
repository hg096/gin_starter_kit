package user

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"gin_starter/internal/config"
	"gin_starter/internal/middleware"
	"gin_starter/pkg/authz"
	"gin_starter/pkg/errors"
	"gin_starter/pkg/logger"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

const refreshTokenHashPrefix = "h1:"

// Service is the user domain service interface.
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

func NewService(repo Repository, cfg *config.Config) Service {
	return &service{repo: repo, config: cfg}
}

func (s *service) Register(req *CreateUserRequest) (*User, error) {
	exists, err := s.repo.Exists(req.ID)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, errors.ErrUserExists
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		logger.Error("failed to hash password: %v", err)
		return nil, errors.Wrap(err, "PASSWORD_HASH_FAILED", "failed to process password")
	}

	user := &User{
		ID:        req.ID,
		Password:  string(hashedPassword),
		Name:      req.Name,
		Email:     req.Email,
		AuthType:  authz.AuthTypeUser,
		AuthLevel: 1,
		Status:    UserStatusActive,
		CreatedAt: time.Now(),
	}

	if err := s.repo.Create(user); err != nil {
		return nil, err
	}

	logger.Info("user registered: %s", user.ID)
	return user.ToPublic(), nil
}

func (s *service) Login(req *LoginRequest) (*LoginResponse, error) {
	user, err := s.repo.FindByID(req.ID)
	if err != nil {
		if errors.Is(err, errors.ErrUserNotFound) {
			return nil, errors.ErrInvalidCredentials
		}
		return nil, err
	}

	if strings.ToLower(strings.TrimSpace(user.Status)) == UserStatusLocked {
		return nil, errors.ErrAccountLocked
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		logger.Warn("login failed (bad password): %s", req.ID)
		return nil, errors.ErrInvalidCredentials
	}

	accessToken, err := middleware.GenerateToken(
		user.ID,
		user.AuthType,
		user.AuthLevel,
		s.config.JWT.AccessExpireMin,
		s.config.JWT.AccessSecret,
		s.config.JWT.TokenSecret,
		s.config.JWT.Issuer,
		s.config.JWT.Audience,
		s.config.JWT.Subject,
	)
	if err != nil {
		logger.Error("failed to generate access token: %v", err)
		return nil, errors.Wrap(err, "TOKEN_GENERATION_FAILED", "failed to generate token")
	}

	refreshToken, err := middleware.GenerateToken(
		user.ID,
		user.AuthType,
		user.AuthLevel,
		s.config.JWT.RefreshExpireDays*24*60,
		s.config.JWT.RefreshSecret,
		s.config.JWT.TokenSecret,
		s.config.JWT.Issuer,
		s.config.JWT.Audience,
		s.config.JWT.Subject,
	)
	if err != nil {
		logger.Error("failed to generate refresh token: %v", err)
		return nil, errors.Wrap(err, "TOKEN_GENERATION_FAILED", "failed to generate token")
	}

	if err := s.repo.UpdateRefreshToken(user.ID, hashRefreshToken(refreshToken)); err != nil {
		return nil, err
	}

	logger.Info("login success: %s", user.ID)

	return &LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         user.ToPublic(),
	}, nil
}

func (s *service) GetProfile(userID string) (*User, error) {
	user, err := s.repo.FindByID(userID)
	if err != nil {
		return nil, err
	}
	return user.ToPublic(), nil
}

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
			logger.Error("failed to hash password: %v", err)
			return errors.Wrap(err, "PASSWORD_HASH_FAILED", "failed to process password")
		}
		updates["u_pass"] = string(hashedPassword)
		updates["u_token_valid_after"] = time.Now()
		updates["u_re_token"] = ""
	}

	if len(updates) == 0 {
		return errors.New("NO_UPDATES", "no fields to update")
	}

	if err := s.repo.Update(userID, updates); err != nil {
		return err
	}

	logger.Info("profile updated: %s", userID)
	return nil
}

func (s *service) RefreshToken(req *RefreshTokenRequest) (*RefreshTokenResponse, error) {
	claims, err := middleware.ValidateToken(
		req.RefreshToken,
		s.config.JWT.RefreshSecret,
		s.config.JWT.TokenSecret,
		s.config.JWT.Issuer,
		s.config.JWT.Audience,
		s.config.JWT.Subject,
	)
	if err != nil {
		return nil, err
	}

	user, err := s.repo.FindByID(claims.UserID)
	if err != nil {
		return nil, err
	}

	if strings.ToLower(strings.TrimSpace(user.Status)) == UserStatusLocked {
		return nil, errors.ErrAccountLocked
	}

	if user.TokenValidAfter != nil {
		if claims.IssuedAt == nil || !claims.IssuedAt.Time.After(*user.TokenValidAfter) {
			return nil, errors.ErrTokenStale
		}
	}

	levelPolicyEnabled, err := s.repo.GetLevelPolicyEnabled()
	if err != nil {
		return nil, err
	}

	if authz.NormalizeAuthType(user.AuthType) != authz.NormalizeAuthType(claims.UserType) {
		return nil, errors.ErrTokenStale
	}
	if levelPolicyEnabled && user.AuthLevel != claims.UserLevel {
		return nil, errors.ErrTokenStale
	}

	matched, legacyToken := isRefreshTokenMatch(user.RefreshToken, req.RefreshToken)
	if !matched {
		logger.Warn("invalid refresh token: %s", claims.UserID)
		return nil, errors.ErrInvalidToken
	}

	if legacyToken {
		if err := s.repo.UpdateRefreshToken(user.ID, hashRefreshToken(req.RefreshToken)); err != nil {
			return nil, err
		}
	}

	accessToken, err := middleware.GenerateToken(
		user.ID,
		user.AuthType,
		user.AuthLevel,
		s.config.JWT.AccessExpireMin,
		s.config.JWT.AccessSecret,
		s.config.JWT.TokenSecret,
		s.config.JWT.Issuer,
		s.config.JWT.Audience,
		s.config.JWT.Subject,
	)
	if err != nil {
		return nil, errors.Wrap(err, "TOKEN_GENERATION_FAILED", "failed to generate token")
	}

	newRefreshToken := req.RefreshToken
	if claims.ExpiresAt != nil && time.Until(claims.ExpiresAt.Time) < time.Duration(s.config.JWT.RefreshReuseHours)*time.Hour {
		newRefreshToken, err = middleware.GenerateToken(
			user.ID,
			user.AuthType,
			user.AuthLevel,
			s.config.JWT.RefreshExpireDays*24*60,
			s.config.JWT.RefreshSecret,
			s.config.JWT.TokenSecret,
			s.config.JWT.Issuer,
			s.config.JWT.Audience,
			s.config.JWT.Subject,
		)
		if err != nil {
			return nil, errors.Wrap(err, "TOKEN_GENERATION_FAILED", "failed to generate token")
		}

		if err := s.repo.UpdateRefreshToken(user.ID, hashRefreshToken(newRefreshToken)); err != nil {
			return nil, err
		}
	}

	logger.Info("token refreshed: %s", user.ID)

	return &RefreshTokenResponse{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
	}, nil
}

func (s *service) Logout(userID string) error {
	updates := map[string]interface{}{
		"u_re_token":          "",
		"u_token_valid_after": time.Now(),
	}
	if err := s.repo.Update(userID, updates); err != nil {
		return err
	}

	logger.Info("logout completed: %s", userID)
	return nil
}

func hashRefreshToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return refreshTokenHashPrefix + hex.EncodeToString(sum[:])
}

func isRefreshTokenMatch(storedToken, incomingToken string) (matched bool, legacyToken bool) {
	if strings.TrimSpace(storedToken) == "" || strings.TrimSpace(incomingToken) == "" {
		return false, false
	}

	if strings.HasPrefix(storedToken, refreshTokenHashPrefix) {
		hashedIncoming := hashRefreshToken(incomingToken)
		return subtle.ConstantTimeCompare([]byte(storedToken), []byte(hashedIncoming)) == 1, false
	}

	return subtle.ConstantTimeCompare([]byte(storedToken), []byte(incomingToken)) == 1, true
}
