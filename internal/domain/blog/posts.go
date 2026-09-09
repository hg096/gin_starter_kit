package blog

import (
	"database/sql"
	stderrors "errors"
	"gin_starter/pkg/errors"
	"gin_starter/pkg/logger"
	"gin_starter/pkg/response"
	"gin_starter/pkg/utils"
	"gin_starter/pkg/validator"
	"strings"

	"github.com/gin-gonic/gin"
)

// CreateBlogRequest 블로그 생성 요청
type CreateBlogRequest struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}

// UpdateBlogRequest 블로그 수정 요청
type UpdateBlogRequest struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}

// BlogListResponse 블로그 목록 응답
type BlogListResponse struct {
	Blogs []Blog `json:"blogs"`
	Total int64  `json:"total"`
	Page  int    `json:"page"`
	Limit int    `json:"limit"`
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
	userID, ok := currentBlogUserID(c)
	if !ok {
		response.Unauthorized(c, "인증이 필요합니다")
		return
	}

	result := validator.Validate(c, createBlogRules())
	if !result.Valid {
		response.ValidationError(c, result.GetErrorMap())
		return
	}

	req := &CreateBlogRequest{
		Title:   result.Values["title"],
		Content: result.Values["content"],
	}

	blog := &Blog{
		Title:    req.Title,
		Content:  req.Content,
		AuthorID: userID,
	}
	if err := h.deps.repo.Create(blog); err != nil {
		logger.Error("블로그 생성 실패: %v", err)
		response.FromError(c, errors.Wrap(err, "BLOG_CREATE_FAILED", "블로그 생성에 실패했습니다"))
		return
	}

	logger.Info("블로그 생성 성공: %d (작성자: %s)", blog.ID, userID)
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
	id, ok := blogIDFromParam(c)
	if !ok {
		response.BadRequest(c, "유효하지 않은 블로그 ID입니다")
		return
	}

	blog, err := h.deps.repo.FindByID(id)
	if err != nil {
		if stderrors.Is(err, sql.ErrNoRows) {
			response.NotFound(c, "블로그를 찾을 수 없습니다")
		} else {
			response.FromError(c, errors.Wrap(err, "DATABASE_ERROR", "블로그 조회에 실패했습니다"))
		}
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
	pagination := utils.PaginationFromQuery(c, 20, 100)

	blogs, total, err := h.deps.repo.FindAll(pagination.Page, pagination.Limit)
	if err != nil {
		logger.Error("블로그 목록 조회 실패: %v", err)
		response.FromError(c, errors.Wrap(err, "BLOG_LIST_FAILED", "블로그 목록 조회에 실패했습니다"))
		return
	}

	response.Success(c, &BlogListResponse{
		Blogs: blogs,
		Total: total,
		Page:  pagination.Page,
		Limit: pagination.Limit,
	})
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
	authorID := strings.TrimSpace(utils.GetBindField(c, "author_id", ""))
	if !utils.HasText(authorID) {
		response.BadRequest(c, "작성자 ID는 필수입니다")
		return
	}

	pagination := utils.PaginationFromQuery(c, 20, 100)

	blogs, total, err := h.deps.repo.FindByAuthorID(authorID, pagination.Page, pagination.Limit)
	if err != nil {
		logger.Error("작성자별 블로그 목록 조회 실패: %v", err)
		response.FromError(c, errors.Wrap(err, "BLOG_LIST_FAILED", "블로그 목록 조회에 실패했습니다"))
		return
	}

	response.Success(c, &BlogListResponse{
		Blogs: blogs,
		Total: total,
		Page:  pagination.Page,
		Limit: pagination.Limit,
	})
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
	userID, ok := currentBlogUserID(c)
	if !ok {
		response.Unauthorized(c, "인증이 필요합니다")
		return
	}

	id, ok := blogIDFromParam(c)
	if !ok {
		response.BadRequest(c, "유효하지 않은 블로그 ID입니다")
		return
	}

	result := validator.Validate(c, updateBlogRules())
	if !result.Valid {
		response.ValidationError(c, result.GetErrorMap())
		return
	}

	req := &UpdateBlogRequest{
		Title:   result.Values["title"],
		Content: result.Values["content"],
	}

	blog, err := h.deps.repo.FindByID(id)
	if err != nil {
		if stderrors.Is(err, sql.ErrNoRows) {
			response.NotFound(c, "블로그를 찾을 수 없습니다")
		} else {
			response.FromError(c, errors.Wrap(err, "DATABASE_ERROR", "블로그 조회에 실패했습니다"))
		}
		return
	}
	if blog.AuthorID != userID {
		response.FromError(c, errors.New("FORBIDDEN", "본인의 블로그만 수정할 수 있습니다"))
		return
	}

	updates := make(map[string]interface{})
	if utils.HasText(req.Title) {
		updates["title"] = req.Title
	}
	if utils.HasText(req.Content) {
		updates["content"] = req.Content
	}
	if len(updates) == 0 {
		response.FromError(c, errors.New("NO_UPDATE_DATA", "수정할 내용이 없습니다"))
		return
	}

	if err := h.deps.repo.Update(id, updates); err != nil {
		logger.Error("블로그 수정 실패: %v", err)
		response.FromError(c, errors.Wrap(err, "BLOG_UPDATE_FAILED", "블로그 수정에 실패했습니다"))
		return
	}

	updatedBlog, err := h.deps.repo.FindByID(id)
	if err != nil {
		response.FromError(c, err)
		return
	}

	logger.Info("블로그 수정 성공: %d (작성자: %s)", id, userID)
	response.Success(c, updatedBlog.ToResponse())
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
	userID, ok := currentBlogUserID(c)
	if !ok {
		response.Unauthorized(c, "인증이 필요합니다")
		return
	}

	id, ok := blogIDFromParam(c)
	if !ok {
		response.BadRequest(c, "유효하지 않은 블로그 ID입니다")
		return
	}

	blog, err := h.deps.repo.FindByID(id)
	if err != nil {
		if stderrors.Is(err, sql.ErrNoRows) {
			response.NotFound(c, "블로그를 찾을 수 없습니다")
		} else {
			response.FromError(c, errors.Wrap(err, "DATABASE_ERROR", "블로그 조회에 실패했습니다"))
		}
		return
	}
	if blog.AuthorID != userID {
		response.FromError(c, errors.New("FORBIDDEN", "본인의 블로그만 삭제할 수 있습니다"))
		return
	}

	if err := h.deps.repo.Delete(id); err != nil {
		logger.Error("블로그 삭제 실패: %v", err)
		response.FromError(c, errors.Wrap(err, "BLOG_DELETE_FAILED", "블로그 삭제에 실패했습니다"))
		return
	}

	logger.Info("블로그 삭제 성공: %d (작성자: %s)", id, userID)
	response.Success(c, gin.H{"message": "블로그가 삭제되었습니다"})
}

func currentBlogUserID(c *gin.Context) (string, bool) {
	return utils.GetContextVal(c, "user_id")
}

func blogIDFromParam(c *gin.Context) (int64, bool) {
	raw := strings.TrimSpace(utils.GetBindField(c, "id", ""))
	id, err := utils.StringToNumeric[int64](raw)
	if err != nil || id <= 0 {
		return 0, false
	}
	return id, true
}

func createBlogRules() []validator.Rule {
	return []validator.Rule{
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
}

func updateBlogRules() []validator.Rule {
	return []validator.Rule{
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
}
