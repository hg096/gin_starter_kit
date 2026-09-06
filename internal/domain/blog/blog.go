package blog

import (
	"time"
)

// Blog 블로그 엔티티
type Blog struct {
	ID        int64     `json:"id"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	AuthorID  string    `json:"author_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Service 블로그 비즈니스 로직 인터페이스
type Service interface {
	CreateBlog(authorID string, req *CreateBlogRequest) (*Blog, error)
	GetBlog(id int64) (*Blog, error)
	GetBlogs(page, limit int) (*BlogListResponse, error)
	GetBlogsByAuthor(authorID string, page, limit int) (*BlogListResponse, error)
	UpdateBlog(id int64, authorID string, req *UpdateBlogRequest) (*Blog, error)
	DeleteBlog(id int64, authorID string) error
}

type service struct {
	repo Repository
}

// NewService 블로그 서비스 생성
func NewService(repo Repository) Service {
	return &service{
		repo: repo,
	}
}

// Handler handles blog HTTP endpoints.
type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{
		service: service,
	}
}

// ToResponse 민감 정보 제외하고 응답용으로 변환
func (b *Blog) ToResponse() map[string]interface{} {
	return map[string]interface{}{
		"id":         b.ID,
		"title":      b.Title,
		"content":    b.Content,
		"author_id":  b.AuthorID,
		"created_at": b.CreatedAt,
		"updated_at": b.UpdatedAt,
	}
}
