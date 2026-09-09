package admin

import (
	"database/sql"
	stderrors "errors"
	"gin_starter/internal/domain/blog"
	"gin_starter/pkg/errors"
	"gin_starter/pkg/response"
	"gin_starter/pkg/utils"
	"strings"

	"github.com/gin-gonic/gin"
)

// AdminBlogListResponse is paginated blog list for admin console.
type AdminBlogListResponse struct {
	Blogs []blog.Blog `json:"blogs"`
	Total int64       `json:"total"`
	Page  int         `json:"page"`
	Limit int         `json:"limit"`
}

// AdminCreateBlogRequest creates blog post from admin console.
type AdminCreateBlogRequest struct {
	Title    string `json:"title"`
	Content  string `json:"content"`
	AuthorID string `json:"author_id"`
}

// AdminUpdateBlogRequest updates blog post from admin console.
type AdminUpdateBlogRequest struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}

// GetBlogs returns blog list for admin console.
// @Summary      List blogs (admin)
// @Tags         admin
// @Produce      json
// @Param        page query int false "page" default(1)
// @Param        limit query int false "limit (max 100)" default(20)
// @Success      200 {object} response.Response{data=AdminBlogListResponse}
// @Failure      401 {object} response.Response
// @Failure      403 {object} response.Response
// @Security     BearerAuth
// @Router       /api/admin/blogs [get]
func (h *Handler) GetBlogs(c *gin.Context) {
	pagination := utils.PaginationFromQuery(c, 20, 100)

	if h.deps.blogRepo == nil {
		response.FromError(c, errors.New("DATABASE_ERROR", "blog repository is not available"))
		return
	}

	blogs, total, err := h.deps.blogRepo.FindAll(pagination.Page, pagination.Limit)
	if err != nil {
		response.FromError(c, errors.Wrap(err, "DATABASE_ERROR", "failed to list blogs"))
		return
	}
	response.Success(c, &AdminBlogListResponse{
		Blogs: blogs,
		Total: total,
		Page:  pagination.Page,
		Limit: pagination.Limit,
	})
}

// CreateBlog creates a blog from admin console.
// @Summary      Create blog (admin)
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        request body AdminCreateBlogRequest true "create blog payload"
// @Success      201 {object} response.Response
// @Failure      400 {object} response.Response
// @Failure      401 {object} response.Response
// @Failure      403 {object} response.Response
// @Security     BearerAuth
// @Router       /api/admin/blogs [post]
func (h *Handler) CreateBlog(c *gin.Context) {
	actor, ok := actorFromContext(c)
	if !ok {
		response.Unauthorized(c, "authentication context missing")
		return
	}

	var req AdminCreateBlogRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request payload")
		return
	}

	if h.deps.blogRepo == nil {
		response.FromError(c, errors.New("DATABASE_ERROR", "blog repository is not available"))
		return
	}

	title := strings.TrimSpace(req.Title)
	if len(title) < 2 || len(title) > 200 {
		response.FromError(c, errors.New("BAD_REQUEST", "title length must be 2-200"))
		return
	}

	content := strings.TrimSpace(req.Content)
	if !utils.HasText(content) || len(content) > 10000 {
		response.FromError(c, errors.New("BAD_REQUEST", "content length must be 1-10000"))
		return
	}

	authorID := strings.TrimSpace(req.AuthorID)
	if !utils.HasText(authorID) {
		authorID = actor.ID
	}
	if _, err := h.deps.userRepo.FindByID(authorID); err != nil {
		response.FromError(c, errors.New("USER_NOT_FOUND", "author not found"))
		return
	}

	created := &blog.Blog{
		Title:    title,
		Content:  content,
		AuthorID: authorID,
	}
	if err := h.deps.db.WithTx(func(tx *sql.Tx) error {
		blogRepo := h.deps.blogRepo.Tx(tx)
		if err := blogRepo.Create(created); err != nil {
			return errors.Wrap(err, "DATABASE_ERROR", "failed to create blog")
		}
		return h.deps.permissionRepo.WriteAuditLogTx(tx, &AuditLogEntry{
			ActorID:  actor.ID,
			TargetID: authorID,
			Action:   BuildPagePermissionCode("blogs", PageActionCreate),
			Status:   "success",
			IP:       actor.IP,
			AfterData: map[string]interface{}{
				"id":        created.ID,
				"title":     created.Title,
				"author_id": created.AuthorID,
			},
		})
	}); err != nil {
		response.FromError(c, err)
		return
	}

	response.Created(c, gin.H{
		"message": "blog created",
		"blog":    created.ToResponse(),
	})
}

// UpdateBlog updates a blog from admin console.
// @Summary      Update blog (admin)
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        id path int true "blog id"
// @Param        request body AdminUpdateBlogRequest true "update blog payload"
// @Success      200 {object} response.Response
// @Failure      400 {object} response.Response
// @Failure      401 {object} response.Response
// @Failure      403 {object} response.Response
// @Security     BearerAuth
// @Router       /api/admin/blogs/{id} [put]
func (h *Handler) UpdateBlog(c *gin.Context) {
	actor, ok := actorFromContext(c)
	if !ok {
		response.Unauthorized(c, "authentication context missing")
		return
	}

	id, ok := adminBlogIDFromParam(c)
	if !ok {
		response.BadRequest(c, "invalid blog id")
		return
	}

	var req AdminUpdateBlogRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request payload")
		return
	}

	if h.deps.blogRepo == nil {
		response.FromError(c, errors.New("DATABASE_ERROR", "blog repository is not available"))
		return
	}

	current, err := h.deps.blogRepo.FindByID(id)
	if err != nil || current == nil {
		if err == nil || stderrors.Is(err, sql.ErrNoRows) {
			response.NotFound(c, "blog not found")
		} else {
			response.FromError(c, errors.Wrap(err, "DATABASE_ERROR", "failed to query blog"))
		}
		return
	}

	updates := map[string]interface{}{}
	if trimmed := strings.TrimSpace(req.Title); utils.HasText(trimmed) {
		if len(trimmed) < 2 || len(trimmed) > 200 {
			response.FromError(c, errors.New("BAD_REQUEST", "title length must be 2-200"))
			return
		}
		updates["title"] = trimmed
	}
	if trimmed := strings.TrimSpace(req.Content); utils.HasText(trimmed) {
		if len(trimmed) > 10000 {
			response.FromError(c, errors.New("BAD_REQUEST", "content length must be <= 10000"))
			return
		}
		updates["content"] = trimmed
	}
	if len(updates) == 0 {
		response.FromError(c, errors.New("BAD_REQUEST", "no update data"))
		return
	}

	if err := h.deps.db.WithTx(func(tx *sql.Tx) error {
		blogRepo := h.deps.blogRepo.Tx(tx)
		if err := blogRepo.Update(id, updates); err != nil {
			return errors.Wrap(err, "DATABASE_ERROR", "failed to update blog")
		}
		return h.deps.permissionRepo.WriteAuditLogTx(tx, &AuditLogEntry{
			ActorID:  actor.ID,
			TargetID: current.AuthorID,
			Action:   BuildPagePermissionCode("blogs", PageActionUpdate),
			Status:   "success",
			IP:       actor.IP,
			BeforeData: map[string]interface{}{
				"id":      current.ID,
				"title":   current.Title,
				"content": current.Content,
			},
			AfterData: map[string]interface{}{
				"id":      current.ID,
				"title":   coalesceString(updates["title"], current.Title),
				"content": coalesceString(updates["content"], current.Content),
			},
		})
	}); err != nil {
		response.FromError(c, err)
		return
	}

	updated, err := h.deps.blogRepo.FindByID(id)
	if err != nil || updated == nil {
		if err == nil || stderrors.Is(err, sql.ErrNoRows) {
			response.NotFound(c, "blog not found")
		} else {
			response.FromError(c, errors.Wrap(err, "DATABASE_ERROR", "failed to query blog"))
		}
		return
	}

	response.Success(c, gin.H{
		"message": "blog updated",
		"blog":    updated.ToResponse(),
	})
}

// DeleteBlog deletes a blog from admin console.
// @Summary      Delete blog (admin)
// @Tags         admin
// @Produce      json
// @Param        id path int true "blog id"
// @Success      200 {object} response.Response
// @Failure      400 {object} response.Response
// @Failure      401 {object} response.Response
// @Failure      403 {object} response.Response
// @Security     BearerAuth
// @Router       /api/admin/blogs/{id} [delete]
func (h *Handler) DeleteBlog(c *gin.Context) {
	actor, ok := actorFromContext(c)
	if !ok {
		response.Unauthorized(c, "authentication context missing")
		return
	}

	id, ok := adminBlogIDFromParam(c)
	if !ok {
		response.BadRequest(c, "invalid blog id")
		return
	}

	if h.deps.blogRepo == nil {
		response.FromError(c, errors.New("DATABASE_ERROR", "blog repository is not available"))
		return
	}

	current, err := h.deps.blogRepo.FindByID(id)
	if err != nil || current == nil {
		if err == nil || stderrors.Is(err, sql.ErrNoRows) {
			response.NotFound(c, "blog not found")
		} else {
			response.FromError(c, errors.Wrap(err, "DATABASE_ERROR", "failed to query blog"))
		}
		return
	}

	if err := h.deps.db.WithTx(func(tx *sql.Tx) error {
		blogRepo := h.deps.blogRepo.Tx(tx)
		if err := blogRepo.Delete(id); err != nil {
			return errors.Wrap(err, "DATABASE_ERROR", "failed to delete blog")
		}
		return h.deps.permissionRepo.WriteAuditLogTx(tx, &AuditLogEntry{
			ActorID:  actor.ID,
			TargetID: current.AuthorID,
			Action:   BuildPagePermissionCode("blogs", PageActionDelete),
			Status:   "success",
			IP:       actor.IP,
			BeforeData: map[string]interface{}{
				"id":        current.ID,
				"title":     current.Title,
				"author_id": current.AuthorID,
			},
		})
	}); err != nil {
		response.FromError(c, err)
		return
	}
	response.Success(c, gin.H{"message": "blog deleted"})
}

func adminBlogIDFromParam(c *gin.Context) (int64, bool) {
	id, err := utils.StringToNumeric[int64](strings.TrimSpace(utils.GetBindField(c, "id", "")))
	if err != nil || id <= 0 {
		return 0, false
	}
	return id, true
}
