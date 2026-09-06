package user

import (
	"gin_starter/pkg/errors"
	"gin_starter/pkg/logger"
	"gin_starter/pkg/response"
	"gin_starter/pkg/utils"
	"gin_starter/pkg/validator"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// UpdateUserRequest is profile update payload.
type UpdateUserRequest struct {
	Password string `json:"user_pass,omitempty"`
	Name     string `json:"user_name,omitempty"`
	Email    string `json:"user_email,omitempty"`
}

// GetProfile profile endpoint.
// @Summary 프로필 조회
// @Tags User
// @Security BearerAuth
// @Produce json
// @Success 200 {object} response.Response
// @Router /api/user/profile [get]
func (h *Handler) GetProfile(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	user, err := h.service.GetProfile(userID)
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
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	req, ok := readUpdateProfileRequest(c)
	if !ok {
		return
	}

	if err := h.service.UpdateProfile(userID, req); err != nil {
		response.FromError(c, err)
		return
	}

	response.Success(c, gin.H{"message": "프로필이 수정되었습니다"})
}

func readUpdateProfileRequest(c *gin.Context) (*UpdateUserRequest, bool) {
	result := validator.Validate(c, []validator.Rule{
		{Field: "user_name", Label: "이름", MinLen: 2, MaxLen: 50, Pattern: validator.PatternKorEng},
		{Field: "user_email", Label: "이메일", Pattern: validator.PatternEmail},
		{Field: "user_pass", Label: "비밀번호", MinLen: 6, MaxLen: 50},
	})
	if !result.Valid {
		response.ValidationError(c, result.GetErrorMap())
		return nil, false
	}

	return &UpdateUserRequest{
		Name:     result.Values["user_name"],
		Email:    result.Values["user_email"],
		Password: result.Values["user_pass"],
	}, true
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

	if !utils.EmptyString(req.Name) {
		updates["u_name"] = req.Name
	}
	if !utils.EmptyString(req.Email) {
		updates["u_email"] = req.Email
	}
	if !utils.EmptyString(req.Password) {
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

func currentUserID(c *gin.Context) (string, bool) {
	id, ok := utils.GetContextVal(c, "user_id")
	if !ok || utils.EmptyString(id) {
		response.Unauthorized(c, "인증 정보가 없습니다")
		return "", false
	}

	return id, true
}
