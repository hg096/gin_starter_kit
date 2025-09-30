package blog

import (
	"fmt"
	"gin_starter/pkg/errors"
	"gin_starter/pkg/logger"
)

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

// CreateBlog 블로그 생성
func (s *service) CreateBlog(authorID string, req *CreateBlogRequest) (*Blog, error) {
	// 제목 검증
	if req.Title == "" {
		return nil, errors.New("TITLE_REQUIRED", "제목은 필수입니다")
	}
	if len(req.Title) < 2 || len(req.Title) > 200 {
		return nil, errors.New("TITLE_LENGTH", "제목은 2-200자 사이여야 합니다")
	}

	// 내용 검증
	if req.Content == "" {
		return nil, errors.New("CONTENT_REQUIRED", "내용은 필수입니다")
	}
	if len(req.Content) > 10000 {
		return nil, errors.New("CONTENT_LENGTH", "내용은 10000자를 초과할 수 없습니다")
	}

	// 블로그 생성
	blog := &Blog{
		Title:    req.Title,
		Content:  req.Content,
		AuthorID: authorID,
	}

	if err := s.repo.Create(blog); err != nil {
		logger.Error("블로그 생성 실패: %v", err)
		return nil, errors.Wrap(err, "BLOG_CREATE_FAILED", "블로그 생성에 실패했습니다")
	}

	logger.Info("블로그 생성 성공: %d (작성자: %s)", blog.ID, authorID)
	return blog, nil
}

// GetBlog 블로그 조회
func (s *service) GetBlog(id int64) (*Blog, error) {
	blog, err := s.repo.FindByID(id)
	if err != nil {
		return nil, errors.New("BLOG_NOT_FOUND", "블로그를 찾을 수 없습니다")
	}

	return blog, nil
}

// GetBlogs 블로그 목록 조회
func (s *service) GetBlogs(page, limit int) (*BlogListResponse, error) {
	// 페이지네이션 검증
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	blogs, total, err := s.repo.FindAll(page, limit)
	if err != nil {
		logger.Error("블로그 목록 조회 실패: %v", err)
		return nil, errors.Wrap(err, "BLOG_LIST_FAILED", "블로그 목록 조회에 실패했습니다")
	}

	return &BlogListResponse{
		Blogs: blogs,
		Total: total,
		Page:  page,
		Limit: limit,
	}, nil
}

// GetBlogsByAuthor 작성자별 블로그 목록 조회
func (s *service) GetBlogsByAuthor(authorID string, page, limit int) (*BlogListResponse, error) {
	// 페이지네이션 검증
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	blogs, total, err := s.repo.FindByAuthorID(authorID, page, limit)
	if err != nil {
		logger.Error("작성자별 블로그 목록 조회 실패: %v", err)
		return nil, errors.Wrap(err, "BLOG_LIST_FAILED", "블로그 목록 조회에 실패했습니다")
	}

	return &BlogListResponse{
		Blogs: blogs,
		Total: total,
		Page:  page,
		Limit: limit,
	}, nil
}

// UpdateBlog 블로그 수정
func (s *service) UpdateBlog(id int64, authorID string, req *UpdateBlogRequest) (*Blog, error) {
	// 블로그 존재 확인
	blog, err := s.repo.FindByID(id)
	if err != nil {
		return nil, errors.New("BLOG_NOT_FOUND", "블로그를 찾을 수 없습니다")
	}

	// 작성자 확인
	if blog.AuthorID != authorID {
		return nil, errors.New("FORBIDDEN", "본인의 블로그만 수정할 수 있습니다")
	}

	// 수정 데이터 준비
	updates := make(map[string]interface{})
	if req.Title != "" {
		if len(req.Title) < 2 || len(req.Title) > 200 {
			return nil, errors.New("TITLE_LENGTH", "제목은 2-200자 사이여야 합니다")
		}
		updates["title"] = req.Title
	}
	if req.Content != "" {
		if len(req.Content) > 10000 {
			return nil, errors.New("CONTENT_LENGTH", "내용은 10000자를 초과할 수 없습니다")
		}
		updates["content"] = req.Content
	}

	// 수정할 내용이 없으면 에러
	if len(updates) == 0 {
		return nil, errors.New("NO_UPDATE_DATA", "수정할 내용이 없습니다")
	}

	// 업데이트
	if err := s.repo.Update(id, updates); err != nil {
		logger.Error("블로그 수정 실패: %v", err)
		return nil, errors.Wrap(err, "BLOG_UPDATE_FAILED", "블로그 수정에 실패했습니다")
	}

	// 수정된 블로그 조회
	updatedBlog, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}

	logger.Info("블로그 수정 성공: %d (작성자: %s)", id, authorID)
	return updatedBlog, nil
}

// DeleteBlog 블로그 삭제
func (s *service) DeleteBlog(id int64, authorID string) error {
	// 블로그 존재 확인
	blog, err := s.repo.FindByID(id)
	if err != nil {
		return errors.New("BLOG_NOT_FOUND", "블로그를 찾을 수 없습니다")
	}

	// 작성자 확인
	if blog.AuthorID != authorID {
		return errors.New("FORBIDDEN", "본인의 블로그만 삭제할 수 있습니다")
	}

	// 삭제
	if err := s.repo.Delete(id); err != nil {
		logger.Error("블로그 삭제 실패: %v", err)
		return errors.Wrap(err, "BLOG_DELETE_FAILED", "블로그 삭제에 실패했습니다")
	}

	logger.Info("블로그 삭제 성공: %d (작성자: %s)", id, authorID)
	return nil
}

// ValidateBlogAccess 블로그 접근 권한 검증 (헬퍼 함수)
func (s *service) ValidateBlogAccess(id int64, authorID string) error {
	blog, err := s.repo.FindByID(id)
	if err != nil {
		return errors.New("BLOG_NOT_FOUND", "블로그를 찾을 수 없습니다")
	}

	if blog.AuthorID != authorID {
		return fmt.Errorf("권한이 없습니다")
	}

	return nil
}