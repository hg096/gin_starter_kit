package blog

import (
	"database/sql"
	"gin_starter/internal/infrastructure/database"
	"time"
)

// Repository 블로그 저장소 인터페이스
type Repository interface {
	Create(blog *Blog) error
	CreateTx(tx *sql.Tx, blog *Blog) error
	FindByID(id int64) (*Blog, error)
	FindAll(page, limit int) ([]Blog, int64, error)
	FindByAuthorID(authorID string, page, limit int) ([]Blog, int64, error)
	Update(id int64, updates map[string]interface{}) error
	UpdateTx(tx *sql.Tx, id int64, updates map[string]interface{}) error
	Delete(id int64) error
	DeleteTx(tx *sql.Tx, id int64) error
	Exists(id int64) (bool, error)
}

type repository struct {
	base *database.Repository
}

// NewRepository 블로그 저장소 생성
func NewRepository(db *database.DB) Repository {
	return &repository{
		base: database.NewRepository(db),
	}
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

	id, err := r.base.Insert("_blog", data)
	if err != nil {
		return err
	}
	blog.ID = id
	return nil
}

// CreateTx 트랜잭션으로 블로그 생성
func (r *repository) CreateTx(tx *sql.Tx, blog *Blog) error {
	data := map[string]interface{}{
		"title":      blog.Title,
		"content":    blog.Content,
		"author_id":  blog.AuthorID,
		"created_at": time.Now(),
		"updated_at": time.Now(),
	}

	id, err := r.base.InsertTx(tx, "_blog", data)
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
	err := r.base.QueryRow(query, id).Scan(&blog.ID, &blog.Title, &blog.Content,
		&blog.AuthorID, &blog.CreatedAt, &blog.UpdatedAt)
	if err != nil {
		return nil, err
	}

	return &blog, nil
}

// FindAll 모든 블로그 조회 (페이지네이션)
func (r *repository) FindAll(page, limit int) ([]Blog, int64, error) {
	offset := (page - 1) * limit

	// 전체 개수 조회
	total, err := r.base.Count("_blog", "")
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

	rows, err := r.base.Query(query, limit, offset)
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
	offset := (page - 1) * limit

	// 전체 개수 조회
	total, err := r.base.Count("_blog", "author_id = ?", authorID)
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

	rows, err := r.base.Query(query, authorID, limit, offset)
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
	_, err := r.base.Update("_blog", updates, "id = ?", id)
	return err
}

// UpdateTx 트랜잭션으로 블로그 수정
func (r *repository) UpdateTx(tx *sql.Tx, id int64, updates map[string]interface{}) error {
	updates["updated_at"] = time.Now()
	_, err := r.base.UpdateTx(tx, "_blog", updates, "id = ?", id)
	return err
}

// Delete 블로그 삭제
func (r *repository) Delete(id int64) error {
	_, err := r.base.Delete("_blog", "id = ?", id)
	return err
}

// DeleteTx 트랜잭션으로 블로그 삭제
func (r *repository) DeleteTx(tx *sql.Tx, id int64) error {
	_, err := r.base.DeleteTx(tx, "_blog", "id = ?", id)
	return err
}

// Exists 블로그 존재 여부 확인
func (r *repository) Exists(id int64) (bool, error) {
	return r.base.Exists("_blog", "id = ?", id)
}