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

type deps struct {
	repo Repository
}

// NewDeps 블로그 handler dependency 생성
func NewDeps(repo Repository) *deps {
	return &deps{
		repo: repo,
	}
}

// Handler handles blog HTTP endpoints.
type Handler struct {
	deps *deps
}

func NewHandler(deps *deps) *Handler {
	return &Handler{
		deps: deps,
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
