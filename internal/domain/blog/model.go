package blog

import "time"

// Blog 블로그 엔티티
type Blog struct {
	ID        int64     `json:"id"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	AuthorID  string    `json:"author_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

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