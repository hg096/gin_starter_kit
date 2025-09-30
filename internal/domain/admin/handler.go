package admin

import (
	"gin_starter/pkg/response"
	"strconv"

	"github.com/gin-gonic/gin"
)

// Handler 관리자 HTTP 핸들러
type Handler struct {
	service Service
}

// NewHandler 관리자 핸들러 생성
func NewHandler(service Service) *Handler {
	return &Handler{
		service: service,
	}
}

// GetUsers 사용자 목록 조회
// @Summary      사용자 목록 조회 (관리자)
// @Description  모든 사용자 목록을 조회합니다 (관리자 전용)
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        page query int false "페이지 번호 (기본: 1)"
// @Param        limit query int false "페이지 크기 (기본: 20)"
// @Param        user_type query string false "사용자 타입 (U, A)"
// @Success      200 {object} response.Response{data=AdminUserListResponse}
// @Failure      401 {object} response.Response
// @Failure      403 {object} response.Response
// @Security     BearerAuth
// @Router       /api/admin/users [get]
func (h *Handler) GetUsers(c *gin.Context) {
	// 페이지네이션
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	userType := c.Query("user_type")

	// 사용자 목록 조회
	result, err := h.service.GetAllUsers(page, limit, userType)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, result)
}

// GetUser 사용자 상세 조회
// @Summary      사용자 상세 조회 (관리자)
// @Description  특정 사용자의 상세 정보를 조회합니다
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        id path string true "사용자 ID"
// @Success      200 {object} response.Response
// @Failure      404 {object} response.Response
// @Security     BearerAuth
// @Router       /api/admin/users/{id} [get]
func (h *Handler) GetUser(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.BadRequest(c, "사용자 ID는 필수입니다")
		return
	}

	user, err := h.service.GetUserByID(id)
	if err != nil {
		response.NotFound(c, "사용자를 찾을 수 없습니다")
		return
	}

	response.Success(c, gin.H{
		"id":         user.ID,
		"name":       user.Name,
		"email":      user.Email,
		"auth_type":  user.AuthType,
		"auth_level": user.AuthLevel,
		"created_at": user.CreatedAt,
	})
}

// UpdateUserAuth 사용자 권한 수정
// @Summary      사용자 권한 수정 (관리자)
// @Description  사용자의 권한 타입과 레벨을 수정합니다
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        id path string true "사용자 ID"
// @Param        request body AdminUpdateUserAuthRequest true "권한 정보"
// @Success      200 {object} response.Response
// @Failure      400 {object} response.Response
// @Failure      404 {object} response.Response
// @Security     BearerAuth
// @Router       /api/admin/users/{id}/auth [put]
func (h *Handler) UpdateUserAuth(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.BadRequest(c, "사용자 ID는 필수입니다")
		return
	}

	var req AdminUpdateUserAuthRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "잘못된 요청 형식입니다")
		return
	}

	// 권한 수정
	if err := h.service.UpdateUserAuth(id, req.AuthType, req.AuthLevel); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	response.Success(c, gin.H{"message": "사용자 권한이 수정되었습니다"})
}

// DeleteUser 사용자 삭제
// @Summary      사용자 삭제 (관리자)
// @Description  사용자를 삭제합니다
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        id path string true "사용자 ID"
// @Success      200 {object} response.Response
// @Failure      404 {object} response.Response
// @Security     BearerAuth
// @Router       /api/admin/users/{id} [delete]
func (h *Handler) DeleteUser(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.BadRequest(c, "사용자 ID는 필수입니다")
		return
	}

	// 사용자 삭제
	if err := h.service.DeleteUser(id); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	response.Success(c, gin.H{"message": "사용자가 삭제되었습니다"})
}

// GetStats 통계 조회
// @Summary      통계 조회 (관리자)
// @Description  전체 사용자, 블로그 등의 통계를 조회합니다
// @Tags         admin
// @Accept       json
// @Produce      json
// @Success      200 {object} response.Response{data=AdminStatsResponse}
// @Security     BearerAuth
// @Router       /api/admin/stats [get]
func (h *Handler) GetStats(c *gin.Context) {
	stats, err := h.service.GetStats()
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, stats)
}