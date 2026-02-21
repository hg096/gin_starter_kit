package blog

import (
	"gin_starter/pkg/response"
	"gin_starter/pkg/validator"
	"strconv"

	"github.com/gin-gonic/gin"
)

// Handler handles blog HTTP endpoints.
type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{
		service: service,
	}
}

// Create creates a new blog post.
// @Summary      블로그 생성
// @Description  새로운 블로그 글을 작성합니다.
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
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "인증이 필요합니다")
		return
	}

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

	req := &CreateBlogRequest{
		Title:   result.Values["title"],
		Content: result.Values["content"],
	}

	blog, err := h.service.CreateBlog(userID.(string), req)
	if err != nil {
		response.FromError(c, err)
		return
	}

	response.Created(c, blog.ToResponse())
}

// Get returns a blog by id.
// @Summary      블로그 조회
// @Description  ID로 블로그 글을 조회합니다.
// @Tags         blog
// @Accept       json
// @Produce      json
// @Param        id path int true "블로그 ID"
// @Success      200 {object} response.Response{data=Blog}
// @Failure      404 {object} response.Response
// @Router       /api/blog/{id} [get]
func (h *Handler) Get(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.BadRequest(c, "유효하지 않은 블로그 ID입니다")
		return
	}

	blog, err := h.service.GetBlog(id)
	if err != nil {
		response.FromError(c, err)
		return
	}

	response.Success(c, blog.ToResponse())
}

// List returns blog list.
// @Summary      블로그 목록
// @Description  블로그 글 목록을 페이지네이션으로 조회합니다.
// @Tags         blog
// @Accept       json
// @Produce      json
// @Param        page query int false "페이지 번호 (기본: 1)"
// @Param        limit query int false "페이지 크기 (기본: 20, 최대: 100)"
// @Success      200 {object} response.Response{data=BlogListResponse}
// @Failure      500 {object} response.Response
// @Router       /api/blog [get]
func (h *Handler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	result, err := h.service.GetBlogs(page, limit)
	if err != nil {
		response.FromError(c, err)
		return
	}

	response.Success(c, result)
}

// ListByAuthor returns blogs by author.
// @Summary      작성자별 블로그 목록
// @Description  특정 작성자의 블로그 글 목록을 조회합니다.
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

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	result, err := h.service.GetBlogsByAuthor(authorID, page, limit)
	if err != nil {
		response.FromError(c, err)
		return
	}

	response.Success(c, result)
}

// Update updates a blog.
// @Summary      블로그 수정
// @Description  자신의 블로그 글을 수정합니다.
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
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "인증이 필요합니다")
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.BadRequest(c, "유효하지 않은 블로그 ID입니다")
		return
	}

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

	req := &UpdateBlogRequest{
		Title:   result.Values["title"],
		Content: result.Values["content"],
	}

	blog, err := h.service.UpdateBlog(id, userID.(string), req)
	if err != nil {
		response.FromError(c, err)
		return
	}

	response.Success(c, blog.ToResponse())
}

// Delete removes a blog.
// @Summary      블로그 삭제
// @Description  자신의 블로그 글을 삭제합니다.
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
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "인증이 필요합니다")
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.BadRequest(c, "유효하지 않은 블로그 ID입니다")
		return
	}

	if err := h.service.DeleteBlog(id, userID.(string)); err != nil {
		response.FromError(c, err)
		return
	}

	response.Success(c, gin.H{"message": "블로그가 삭제되었습니다"})
}
