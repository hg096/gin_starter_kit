package user

import (
	"gin_starter/pkg/errors"
	"gin_starter/pkg/logger"
	"gin_starter/pkg/middleware"
	"gin_starter/pkg/response"
	"gin_starter/pkg/utils"
	"gin_starter/pkg/validator"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// LoginRequest is login payload.
type LoginRequest struct {
	ID       string `json:"user_id" binding:"required"`
	Password string `json:"user_pass" binding:"required"`
}

// LoginResponse is login response payload.
type LoginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	User         *User  `json:"user"`
}

// Login user login endpoint.
// @Summary 로그인
// @Tags User
// @Accept json
// @Produce json
// @Param body body LoginRequest true "로그인 정보"
// @Success 200 {object} response.Response
// @Router /api/user/login [post]
func (h *Handler) Login(c *gin.Context) {
	req, ok := readLoginRequest(c)
	if !ok {
		return
	}

	loginResp, err := h.service.Login(req)
	if err != nil {
		response.FromError(c, err)
		return
	}

	h.setAuthCookies(c, loginResp.AccessToken, loginResp.RefreshToken)
	response.Success(c, loginResp)
}

func readLoginRequest(c *gin.Context) (*LoginRequest, bool) {
	result := validator.Validate(c, []validator.Rule{
		{Field: "user_id", Label: "아이디", Required: true},
		{Field: "user_pass", Label: "비밀번호", Required: true},
	})
	if !result.Valid {
		response.ValidationError(c, result.GetErrorMap())
		return nil, false
	}

	return &LoginRequest{
		ID:       result.Values["user_id"],
		Password: result.Values["user_pass"],
	}, true
}

func (s *service) Login(req *LoginRequest) (*LoginResponse, error) {
	user, err := s.repo.FindByID(req.ID)
	if err != nil {
		if errors.Is(err, errors.ErrUserNotFound) {
			return nil, errors.ErrInvalidCredentials
		}
		return nil, err
	}

	if utils.TrimLower(user.Status) == UserStatusLocked {
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
