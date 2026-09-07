package blog

import (
	"database/sql"
	"gin_starter/pkg/db/database"
	"gin_starter/pkg/utils"
	"time"
)

// Repository 블로그 저장소 인터페이스
type Repository interface {
	Tx(tx *sql.Tx) Repository
	Create(blog *Blog) error
	FindByID(id int64) (*Blog, error)
	FindAll(page, limit int) ([]Blog, int64, error)
	FindByAuthorID(authorID string, page, limit int) ([]Blog, int64, error)
	Update(id int64, updates map[string]interface{}) error
	Delete(id int64) error
	Exists(id int64) (bool, error)
}

type repository struct {
	db database.Store
}

// NewRepository 블로그 저장소 생성
func NewRepository(db *database.DB) Repository {
	return &repository{db: database.NewRepository(db)}
}

func (r *repository) Tx(tx *sql.Tx) Repository {
	return &repository{db: r.db.Tx(tx)}
}

// Create 블로그 생성
func (r *repository) Create(blog *Blog) error {
	data := map[string]interface{}{
		"title":      blog.Title,
		"content":    blog.Content,
		"author_id":  blog.AuthorID,
		"created_at": time.Now(),
		"updated_at": time.Now(),
	}

	id, err := r.db.Insert("_blog", data)
	if err != nil {
		return err
	}
	blog.ID = id
	return nil
}

// FindByID ID로 블로그 조회
func (r *repository) FindByID(id int64) (*Blog, error) {
	query := `
		SELECT id, title, content, author_id, created_at, updated_at
		FROM _blog
		WHERE id = ?
	`

	var blog Blog
	err := r.db.QueryRow(query, id).Scan(&blog.ID, &blog.Title, &blog.Content,
		&blog.AuthorID, &blog.CreatedAt, &blog.UpdatedAt)
	if err != nil {
		return nil, err
	}

	return &blog, nil
}

// FindAll 모든 블로그 조회 (페이지네이션)
func (r *repository) FindAll(page, limit int) ([]Blog, int64, error) {
	pagination := utils.NewPagination(page, limit, 20, 100)

	// 전체 개수 조회
	total, err := r.db.Count("_blog", "")
	if err != nil {
		return nil, 0, err
	}

	// 목록 조회
	query := `
		SELECT id, title, content, author_id, created_at, updated_at
		FROM _blog
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`

	rows, err := r.db.Query(query, pagination.Limit, pagination.Offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var blogs []Blog
	for rows.Next() {
		var blog Blog
		if err := rows.Scan(&blog.ID, &blog.Title, &blog.Content,
			&blog.AuthorID, &blog.CreatedAt, &blog.UpdatedAt); err != nil {
			return nil, 0, err
		}
		blogs = append(blogs, blog)
	}

	return blogs, total, nil
}

// FindByAuthorID 작성자 ID로 블로그 목록 조회
func (r *repository) FindByAuthorID(authorID string, page, limit int) ([]Blog, int64, error) {
	pagination := utils.NewPagination(page, limit, 20, 100)

	// 전체 개수 조회
	total, err := r.db.Count("_blog", "author_id = ?", authorID)
	if err != nil {
		return nil, 0, err
	}

	// 목록 조회
	query := `
		SELECT id, title, content, author_id, created_at, updated_at
		FROM _blog
		WHERE author_id = ?
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`

	rows, err := r.db.Query(query, authorID, pagination.Limit, pagination.Offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var blogs []Blog
	for rows.Next() {
		var blog Blog
		if err := rows.Scan(&blog.ID, &blog.Title, &blog.Content,
			&blog.AuthorID, &blog.CreatedAt, &blog.UpdatedAt); err != nil {
			return nil, 0, err
		}
		blogs = append(blogs, blog)
	}

	return blogs, total, nil
}

// Update 블로그 수정
func (r *repository) Update(id int64, updates map[string]interface{}) error {
	updates["updated_at"] = time.Now()
	_, err := r.db.Update("_blog", updates, "id = ?", id)
	return err
}

// Delete 블로그 삭제
func (r *repository) Delete(id int64) error {
	_, err := r.db.Delete("_blog", "id = ?", id)
	return err
}

// Exists 블로그 존재 여부 확인
func (r *repository) Exists(id int64) (bool, error) {
	return r.db.Exists("_blog", "id = ?", id)
}
