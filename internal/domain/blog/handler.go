package blog

import (
	"gin_starter/pkg/response"
	"gin_starter/pkg/validator"
	"strconv"

	"github.com/gin-gonic/gin"
)

// Handler 블로그 HTTP 핸들러
type Handler struct {
	service Service
}

// NewHandler 블로그 핸들러 생성
func NewHandler(service Service) *Handler {
	return &Handler{
		service: service,
	}
}

// Create 블로그 생성
// @Summary      블로그 생성
// @Description  새로운 블로그 글을 작성합니다
// @Tags         blog
// @Accept       json
// @Produce      json
// @Param        request body CreateBlogRequest true "블로그 생성 정보"
// @Success      201 {object} response.Response{data=Blog}
// @Failure      400 {object} response.Response
// @Failure      401 {object} response.Response
// @Security     BearerAuth
// @Router       /api/blog [post]
func (h *Handler) Create(c *gin.Context) {
	// 인증 확인
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "인증이 필요합니다")
		return
	}

	// 입력 검증
	rules := []validator.Rule{
		{
			Field:    "title",
			Label:    "제목",
			Required: true,
			MinLen:   2,
			MaxLen:   200,
		},
		{
			Field:    "content",
			Label:    "내용",
			Required: true,
			MinLen:   1,
			MaxLen:   10000,
		},
	}

	result := validator.Validate(c, rules)
	if !result.Valid {
		response.ValidationError(c, result.GetErrorMap())
		return
	}

	// 요청 생성
	req := &CreateBlogRequest{
		Title:   result.Values["title"],
		Content: result.Values["content"],
	}

	// 블로그 생성
	blog, err := h.service.CreateBlog(userID.(string), req)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	response.Created(c, blog.ToResponse())
}

// Get 블로그 상세 조회
// @Summary      블로그 조회
// @Description  ID로 블로그 글을 조회합니다
// @Tags         blog
// @Accept       json
// @Produce      json
// @Param        id path int true "블로그 ID"
// @Success      200 {object} response.Response{data=Blog}
// @Failure      404 {object} response.Response
// @Router       /api/blog/{id} [get]
func (h *Handler) Get(c *gin.Context) {
	// ID 파라미터 추출
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.BadRequest(c, "유효하지 않은 블로그 ID입니다")
		return
	}

	// 블로그 조회
	blog, err := h.service.GetBlog(id)
	if err != nil {
		response.NotFound(c, err.Error())
		return
	}

	response.Success(c, blog.ToResponse())
}

// List 블로그 목록 조회
// @Summary      블로그 목록
// @Description  블로그 글 목록을 페이지네이션으로 조회합니다
// @Tags         blog
// @Accept       json
// @Produce      json
// @Param        page query int false "페이지 번호 (기본: 1)"
// @Param        limit query int false "페이지 크기 (기본: 20, 최대: 100)"
// @Success      200 {object} response.Response{data=BlogListResponse}
// @Failure      500 {object} response.Response
// @Router       /api/blog [get]
func (h *Handler) List(c *gin.Context) {
	// 페이지네이션 파라미터
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	// 블로그 목록 조회
	result, err := h.service.GetBlogs(page, limit)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, result)
}

// ListByAuthor 작성자별 블로그 목록 조회
// @Summary      작성자별 블로그 목록
// @Description  특정 작성자의 블로그 글 목록을 조회합니다
// @Tags         blog
// @Accept       json
// @Produce      json
// @Param        author_id path string true "작성자 ID"
// @Param        page query int false "페이지 번호 (기본: 1)"
// @Param        limit query int false "페이지 크기 (기본: 20, 최대: 100)"
// @Success      200 {object} response.Response{data=BlogListResponse}
// @Failure      500 {object} response.Response
// @Router       /api/blog/author/{author_id} [get]
func (h *Handler) ListByAuthor(c *gin.Context) {
	authorID := c.Param("author_id")
	if authorID == "" {
		response.BadRequest(c, "작성자 ID는 필수입니다")
		return
	}

	// 페이지네이션 파라미터
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	// 블로그 목록 조회
	result, err := h.service.GetBlogsByAuthor(authorID, page, limit)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, result)
}

// Update 블로그 수정
// @Summary      블로그 수정
// @Description  자신의 블로그 글을 수정합니다
// @Tags         blog
// @Accept       json
// @Produce      json
// @Param        id path int true "블로그 ID"
// @Param        request body UpdateBlogRequest true "수정할 정보"
// @Success      200 {object} response.Response{data=Blog}
// @Failure      400 {object} response.Response
// @Failure      401 {object} response.Response
// @Failure      403 {object} response.Response
// @Failure      404 {object} response.Response
// @Security     BearerAuth
// @Router       /api/blog/{id} [put]
func (h *Handler) Update(c *gin.Context) {
	// 인증 확인
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "인증이 필요합니다")
		return
	}

	// ID 파라미터 추출
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.BadRequest(c, "유효하지 않은 블로그 ID입니다")
		return
	}

	// 입력 검증
	rules := []validator.Rule{
		{
			Field:  "title",
			Label:  "제목",
			MinLen: 2,
			MaxLen: 200,
		},
		{
			Field:  "content",
			Label:  "내용",
			MinLen: 1,
			MaxLen: 10000,
		},
	}

	result := validator.Validate(c, rules)
	if !result.Valid {
		response.ValidationError(c, result.GetErrorMap())
		return
	}

	// 요청 생성
	req := &UpdateBlogRequest{
		Title:   result.Values["title"],
		Content: result.Values["content"],
	}

	// 블로그 수정
	blog, err := h.service.UpdateBlog(id, userID.(string), req)
	if err != nil {
		if err.Error() == "본인의 블로그만 수정할 수 있습니다" {
			response.Forbidden(c, err.Error())
		} else if err.Error() == "블로그를 찾을 수 없습니다" {
			response.NotFound(c, err.Error())
		} else {
			response.BadRequest(c, err.Error())
		}
		return
	}

	response.Success(c, blog.ToResponse())
}

// Delete 블로그 삭제
// @Summary      블로그 삭제
// @Description  자신의 블로그 글을 삭제합니다
// @Tags         blog
// @Accept       json
// @Produce      json
// @Param        id path int true "블로그 ID"
// @Success      200 {object} response.Response
// @Failure      400 {object} response.Response
// @Failure      401 {object} response.Response
// @Failure      403 {object} response.Response
// @Failure      404 {object} response.Response
// @Security     BearerAuth
// @Router       /api/blog/{id} [delete]
func (h *Handler) Delete(c *gin.Context) {
	// 인증 확인
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "인증이 필요합니다")
		return
	}

	// ID 파라미터 추출
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.BadRequest(c, "유효하지 않은 블로그 ID입니다")
		return
	}

	// 블로그 삭제
	err = h.service.DeleteBlog(id, userID.(string))
	if err != nil {
		if err.Error() == "본인의 블로그만 삭제할 수 있습니다" {
			response.Forbidden(c, err.Error())
		} else if err.Error() == "블로그를 찾을 수 없습니다" {
			response.NotFound(c, err.Error())
		} else {
			response.BadRequest(c, err.Error())
		}
		return
	}

	response.Success(c, gin.H{"message": "블로그가 삭제되었습니다"})
}