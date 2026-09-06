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
	base *database.Repository
	tx   *database.TxRepository
}

// NewRepository 블로그 저장소 생성
func NewRepository(db *database.DB) Repository {
	return &repository{
		base: database.NewRepository(db),
	}
}

func (r *repository) Tx(tx *sql.Tx) Repository {
	return &repository{
		base: r.base,
		tx:   r.base.Tx(tx),
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

	id, err := r.insert("_blog", data)
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
	pagination := utils.NewPagination(page, limit, 20, 100)

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

	rows, err := r.base.Query(query, pagination.Limit, pagination.Offset)
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

	rows, err := r.base.Query(query, authorID, pagination.Limit, pagination.Offset)
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
	_, err := r.update("_blog", updates, "id = ?", id)
	return err
}

// Delete 블로그 삭제
func (r *repository) Delete(id int64) error {
	_, err := r.delete("_blog", "id = ?", id)
	return err
}

// Exists 블로그 존재 여부 확인
func (r *repository) Exists(id int64) (bool, error) {
	return r.base.Exists("_blog", "id = ?", id)
}

func (r *repository) insert(table string, data map[string]interface{}) (int64, error) {
	if r.tx != nil {
		return r.tx.Insert(table, data)
	}
	return r.base.Insert(table, data)
}

func (r *repository) update(table string, data map[string]interface{}, where string, whereArgs ...interface{}) (int64, error) {
	if r.tx != nil {
		return r.tx.Update(table, data, where, whereArgs...)
	}
	return r.base.Update(table, data, where, whereArgs...)
}

func (r *repository) delete(table string, where string, whereArgs ...interface{}) (int64, error) {
	if r.tx != nil {
		return r.tx.Delete(table, where, whereArgs...)
	}
	return r.base.Delete(table, where, whereArgs...)
}
