package user

import (
	"gin_starter/pkg/response"
	"gin_starter/pkg/validator"

	"github.com/gin-gonic/gin"
)

// Handler 사용자 핸들러
type Handler struct {
	service Service
}

// NewHandler 핸들러 생성자
func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

// Register 회원가입
// @Summary 회원가입
// @Tags User
// @Accept json
// @Produce json
// @Param body body CreateUserRequest true "회원가입 정보"
// @Success 201 {object} response.Response
// @Router /api/user/register [post]
func (h *Handler) Register(c *gin.Context) {
	// 입력값 검증
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

	// 요청 데이터 매핑
	req := &CreateUserRequest{
		ID:       result.Values["user_id"],
		Password: result.Values["user_pass"],
		Name:     result.Values["user_name"],
		Email:    result.Values["user_email"],
	}

	// 서비스 호출
	user, err := h.service.Register(req)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Created(c, gin.H{"user": user})
}

// Login 로그인
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
		response.Unauthorized(c, err.Error())
		return
	}

	response.Success(c, loginResp)
}

// GetProfile 프로필 조회
// @Summary 프로필 조회
// @Tags User
// @Security Bearer
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
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, gin.H{"user": user})
}

// UpdateProfile 프로필 수정
// @Summary 프로필 수정
// @Tags User
// @Security Bearer
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
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, gin.H{"message": "프로필이 수정되었습니다"})
}

// RefreshToken 토큰 갱신
// @Summary 토큰 갱신
// @Tags User
// @Accept json
// @Produce json
// @Param body body RefreshTokenRequest true "리프레시 토큰"
// @Success 200 {object} response.Response
// @Router /api/user/refresh [post]
func (h *Handler) RefreshToken(c *gin.Context) {
	rules := []validator.Rule{
		{Field: "refresh_token", Label: "리프레시 토큰", Required: true},
	}

	result := validator.Validate(c, rules)
	if !result.Valid {
		response.ValidationError(c, result.GetErrorMap())
		return
	}

	req := &RefreshTokenRequest{
		RefreshToken: result.Values["refresh_token"],
	}

	tokens, err := h.service.RefreshToken(req)
	if err != nil {
		response.Unauthorized(c, err.Error())
		return
	}

	response.Success(c, tokens)
}

// Logout 로그아웃
// @Summary 로그아웃
// @Tags User
// @Security Bearer
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
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, gin.H{"message": "로그아웃되었습니다"})
}