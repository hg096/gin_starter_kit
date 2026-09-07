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
	result := validator.Validate(c, []validator.Rule{
		{Field: "user_id", Label: "아이디", Required: true},
		{Field: "user_pass", Label: "비밀번호", Required: true},
	})
	if !result.Valid {
		response.ValidationError(c, result.GetErrorMap())
		return
	}

	req := &LoginRequest{
		ID:       result.Values["user_id"],
		Password: result.Values["user_pass"],
	}

	user, err := h.deps.repo.FindByID(req.ID)
	if err != nil {
		if errors.Is(err, errors.ErrUserNotFound) {
			response.FromError(c, errors.ErrInvalidCredentials)
			return
		}
		response.FromError(c, err)
		return
	}

	if utils.TrimLower(user.Status) == UserStatusLocked {
		response.FromError(c, errors.ErrAccountLocked)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		logger.Warn("login failed (bad password): %s", req.ID)
		response.FromError(c, errors.ErrInvalidCredentials)
		return
	}

	accessToken, err := middleware.GenerateToken(
		user.ID,
		user.AuthType,
		user.AuthLevel,
		h.deps.config.JWT.AccessExpireMin,
		h.deps.config.JWT.AccessSecret,
		h.deps.config.JWT.TokenSecret,
		h.deps.config.JWT.Issuer,
		h.deps.config.JWT.Audience,
		h.deps.config.JWT.Subject,
	)
	if err != nil {
		logger.Error("failed to generate access token: %v", err)
		response.FromError(c, errors.Wrap(err, "TOKEN_GENERATION_FAILED", "failed to generate token"))
		return
	}

	refreshToken, err := middleware.GenerateToken(
		user.ID,
		user.AuthType,
		user.AuthLevel,
		h.deps.config.JWT.RefreshExpireDays*24*60,
		h.deps.config.JWT.RefreshSecret,
		h.deps.config.JWT.TokenSecret,
		h.deps.config.JWT.Issuer,
		h.deps.config.JWT.Audience,
		h.deps.config.JWT.Subject,
	)
	if err != nil {
		logger.Error("failed to generate refresh token: %v", err)
		response.FromError(c, errors.Wrap(err, "TOKEN_GENERATION_FAILED", "failed to generate token"))
		return
	}

	if err := h.deps.repo.UpdateRefreshToken(user.ID, hashRefreshToken(refreshToken)); err != nil {
		response.FromError(c, err)
		return
	}

	logger.Info("login success: %s", user.ID)

	loginResp := &LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         user.ToPublic(),
	}
	h.setAuthCookies(c, loginResp.AccessToken, loginResp.RefreshToken)
	response.Success(c, loginResp)
}
