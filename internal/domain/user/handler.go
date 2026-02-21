package user

import (
	"errors"
	"gin_starter/internal/config"
	"gin_starter/internal/middleware"
	"gin_starter/pkg/response"
	"gin_starter/pkg/validator"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// Handler handles user HTTP endpoints.
type Handler struct {
	service Service
	cfg     *config.Config
}

func NewHandler(service Service, cfg *config.Config) *Handler {
	return &Handler{
		service: service,
		cfg:     cfg,
	}
}

// Register user register endpoint.
// @Summary 회원가입
// @Tags User
// @Accept json
// @Produce json
// @Param body body CreateUserRequest true "회원가입 정보"
// @Success 201 {object} response.Response
// @Router /api/user/register [post]
func (h *Handler) Register(c *gin.Context) {
	rules := []validator.Rule{
		{Field: "user_id", Label: "아이디", Required: true, MinLen: 3, MaxLen: 20, Pattern: validator.PatternAlphaNum},
		{Field: "user_pass", Label: "비밀번호", Required: true, MinLen: 6, MaxLen: 50},
		{Field: "user_name", Label: "이름", Required: true, MinLen: 2, MaxLen: 50, Pattern: validator.PatternKorEng},
		{Field: "user_email", Label: "이메일", Required: true, Pattern: validator.PatternEmail},
	}

	result := validator.Validate(c, rules)
	if !result.Valid {
		response.ValidationError(c, result.GetErrorMap())
		return
	}

	req := &CreateUserRequest{
		ID:       result.Values["user_id"],
		Password: result.Values["user_pass"],
		Name:     result.Values["user_name"],
		Email:    result.Values["user_email"],
	}

	user, err := h.service.Register(req)
	if err != nil {
		response.FromError(c, err)
		return
	}

	response.Created(c, gin.H{"user": user})
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
	rules := []validator.Rule{
		{Field: "user_id", Label: "아이디", Required: true},
		{Field: "user_pass", Label: "비밀번호", Required: true},
	}

	result := validator.Validate(c, rules)
	if !result.Valid {
		response.ValidationError(c, result.GetErrorMap())
		return
	}

	req := &LoginRequest{
		ID:       result.Values["user_id"],
		Password: result.Values["user_pass"],
	}

	loginResp, err := h.service.Login(req)
	if err != nil {
		response.FromError(c, err)
		return
	}

	h.setAuthCookies(c, loginResp.AccessToken, loginResp.RefreshToken)
	response.Success(c, loginResp)
}

// GetProfile profile endpoint.
// @Summary 프로필 조회
// @Tags User
// @Security BearerAuth
// @Produce json
// @Success 200 {object} response.Response
// @Router /api/user/profile [get]
func (h *Handler) GetProfile(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "인증 정보가 없습니다")
		return
	}

	user, err := h.service.GetProfile(userID.(string))
	if err != nil {
		response.FromError(c, err)
		return
	}

	response.Success(c, gin.H{"user": user})
}

// UpdateProfile profile update endpoint.
// @Summary 프로필 수정
// @Tags User
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param body body UpdateUserRequest true "수정 정보"
// @Success 200 {object} response.Response
// @Router /api/user/profile [put]
func (h *Handler) UpdateProfile(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "인증 정보가 없습니다")
		return
	}

	rules := []validator.Rule{
		{Field: "user_name", Label: "이름", MinLen: 2, MaxLen: 50, Pattern: validator.PatternKorEng},
		{Field: "user_email", Label: "이메일", Pattern: validator.PatternEmail},
		{Field: "user_pass", Label: "비밀번호", MinLen: 6, MaxLen: 50},
	}

	result := validator.Validate(c, rules)
	if !result.Valid {
		response.ValidationError(c, result.GetErrorMap())
		return
	}

	req := &UpdateUserRequest{
		Name:     result.Values["user_name"],
		Email:    result.Values["user_email"],
		Password: result.Values["user_pass"],
	}

	if err := h.service.UpdateProfile(userID.(string), req); err != nil {
		response.FromError(c, err)
		return
	}

	response.Success(c, gin.H{"message": "프로필이 수정되었습니다"})
}

// RefreshToken refresh endpoint.
// @Summary 토큰 갱신
// @Tags User
// @Accept json
// @Produce json
// @Param body body RefreshTokenRequest false "리프레시 토큰"
// @Success 200 {object} response.Response
// @Router /api/user/refresh [post]
func (h *Handler) RefreshToken(c *gin.Context) {
	var req RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		response.BadRequest(c, "잘못된 요청 형식입니다")
		return
	}

	if strings.TrimSpace(req.RefreshToken) == "" {
		if cookieToken, err := c.Cookie(middleware.RefreshTokenCookieName); err == nil {
			req.RefreshToken = cookieToken
		}
	}

	if strings.TrimSpace(req.RefreshToken) == "" {
		response.BadRequest(c, "리프레시 토큰이 필요합니다")
		return
	}

	tokens, err := h.service.RefreshToken(&req)
	if err != nil {
		response.FromError(c, err)
		return
	}

	h.setAuthCookies(c, tokens.AccessToken, tokens.RefreshToken)
	response.Success(c, tokens)
}

// Logout user logout endpoint.
// @Summary 로그아웃
// @Tags User
// @Security BearerAuth
// @Produce json
// @Success 200 {object} response.Response
// @Router /api/user/logout [post]
func (h *Handler) Logout(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "인증 정보가 없습니다")
		return
	}

	if err := h.service.Logout(userID.(string)); err != nil {
		response.FromError(c, err)
		return
	}

	h.clearAuthCookies(c)
	response.Success(c, gin.H{"message": "로그아웃되었습니다"})
}

func (h *Handler) setAuthCookies(c *gin.Context, accessToken, refreshToken string) {
	secure := h.cfg != nil && h.cfg.IsProduction()
	accessMaxAge := 30 * 60
	refreshMaxAge := 7 * 24 * 60 * 60

	if h.cfg != nil {
		if h.cfg.JWT.AccessExpireMin > 0 {
			accessMaxAge = h.cfg.JWT.AccessExpireMin * 60
		}
		if h.cfg.JWT.RefreshExpireDays > 0 {
			refreshMaxAge = h.cfg.JWT.RefreshExpireDays * 24 * 60 * 60
		}
	}

	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(middleware.AccessTokenCookieName, accessToken, accessMaxAge, "/", "", secure, true)
	c.SetCookie(middleware.RefreshTokenCookieName, refreshToken, refreshMaxAge, "/", "", secure, true)
}

func (h *Handler) clearAuthCookies(c *gin.Context) {
	secure := h.cfg != nil && h.cfg.IsProduction()

	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(middleware.AccessTokenCookieName, "", -1, "/", "", secure, true)
	c.SetCookie(middleware.RefreshTokenCookieName, "", -1, "/", "", secure, true)
}
