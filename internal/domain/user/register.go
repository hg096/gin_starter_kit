package user

import (
	"gin_starter/pkg/authz"
	"gin_starter/pkg/errors"
	"gin_starter/pkg/logger"
	"gin_starter/pkg/response"
	"gin_starter/pkg/validator"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// CreateUserRequest is register payload.
type CreateUserRequest struct {
	ID       string `json:"user_id" binding:"required"`
	Password string `json:"user_pass" binding:"required"`
	Name     string `json:"user_name" binding:"required"`
	Email    string `json:"user_email" binding:"required,email"`
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
	req, ok := readRegisterRequest(c)
	if !ok {
		return
	}

	user, err := h.service.Register(req)
	if err != nil {
		response.FromError(c, err)
		return
	}

	response.Created(c, gin.H{"user": user})
}

func readRegisterRequest(c *gin.Context) (*CreateUserRequest, bool) {
	result := validator.Validate(c, []validator.Rule{
		{Field: "user_id", Label: "아이디", Required: true, MinLen: 3, MaxLen: 20, Pattern: validator.PatternAlphaNum},
		{Field: "user_pass", Label: "비밀번호", Required: true, MinLen: 6, MaxLen: 50},
		{Field: "user_name", Label: "이름", Required: true, MinLen: 2, MaxLen: 50, Pattern: validator.PatternKorEng},
		{Field: "user_email", Label: "이메일", Required: true, Pattern: validator.PatternEmail},
	})
	if !result.Valid {
		response.ValidationError(c, result.GetErrorMap())
		return nil, false
	}

	return &CreateUserRequest{
		ID:       result.Values["user_id"],
		Password: result.Values["user_pass"],
		Name:     result.Values["user_name"],
		Email:    result.Values["user_email"],
	}, true
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
