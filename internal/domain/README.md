# internal/domain/

비즈니스 도메인별로 독립적인 기능을 구현하는 **가장 핵심적인 디렉토리**입니다.

## 🎯 도메인이란?

도메인은 **비즈니스의 특정 영역**을 의미합니다.

**예시:**
- `user/` - 사용자 관리
- `blog/` - 블로그 포스트
- `order/` - 주문 관리
- `payment/` - 결제 처리
- `notification/` - 알림

## 📁 표준 도메인 구조

각 도메인은 **동일한 4개 파일**로 구성됩니다:

```
domain/
└── user/
    ├── model.go        # 1. 데이터 구조 정의
    ├── repository.go   # 2. 데이터 접근 계층
    ├── service.go      # 3. 비즈니스 로직
    └── handler.go      # 4. HTTP 핸들러
```

## 🔄 데이터 흐름

```
HTTP Request
    ↓
handler.go (요청 파싱, 검증)
    ↓
service.go (비즈니스 로직)
    ↓
repository.go (DB 접근)
    ↓
Database
```

## 📝 파일별 상세 가이드

### 1. model.go - 데이터 구조 정의

**역할:**
- 도메인 엔티티 정의
- 요청/응답 DTO 정의
- 데이터 변환 메서드

**작성 예시:**

```go
package blog

import "time"

// ========== 도메인 엔티티 ==========
// 데이터베이스와 1:1 매핑되는 구조체
type Blog struct {
    ID        int64     `json:"id" db:"b_id"`
    Title     string    `json:"title" db:"b_title"`
    Content   string    `json:"content" db:"b_content"`
    AuthorID  string    `json:"author_id" db:"b_author_id"`
    Views     int       `json:"views" db:"b_views"`
    CreatedAt time.Time `json:"created_at" db:"b_created_at"`
    UpdatedAt time.Time `json:"updated_at" db:"b_updated_at"`
}

// ========== 요청 DTO (Data Transfer Object) ==========
// 클라이언트로부터 받는 데이터
type CreateBlogRequest struct {
    Title   string `json:"title" binding:"required"`
    Content string `json:"content" binding:"required"`
}

type UpdateBlogRequest struct {
    Title   string `json:"title,omitempty"`
    Content string `json:"content,omitempty"`
}

type ListBlogsQuery struct {
    Page    int    `json:"page"`
    PerPage int    `json:"per_page"`
    Keyword string `json:"keyword"`
}

// ========== 응답 DTO ==========
// 클라이언트에게 보낼 데이터
type BlogResponse struct {
    ID        int64     `json:"id"`
    Title     string    `json:"title"`
    Content   string    `json:"content"`
    Author    string    `json:"author"`
    Views     int       `json:"views"`
    CreatedAt time.Time `json:"created_at"`
}

type BlogListResponse struct {
    Blogs      []BlogResponse `json:"blogs"`
    TotalCount int            `json:"total_count"`
    Page       int            `json:"page"`
    PerPage    int            `json:"per_page"`
}

// ========== 변환 메서드 ==========
// 도메인 엔티티 → 응답 DTO
func (b *Blog) ToResponse() *BlogResponse {
    return &BlogResponse{
        ID:        b.ID,
        Title:     b.Title,
        Content:   b.Content,
        Views:     b.Views,
        CreatedAt: b.CreatedAt,
    }
}
```

**작성 규칙:**
- 구조체는 명확한 이름 (Blog, not B or BlogModel)
- JSON 태그는 snake_case
- DB 태그는 실제 컬럼명
- 비밀번호 등 민감정보는 `json:"-"` 처리
- 요청/응답 DTO는 별도 구조체

---

### 2. repository.go - 데이터 접근 계층

**역할:**
- 데이터베이스 CRUD
- 쿼리 실행
- 데이터 매핑

**작성 예시:**

```go
package blog

import (
    "database/sql"
    "gin_starter/internal/infrastructure/database"
    "gin_starter/pkg/errors"
    "gin_starter/pkg/logger"
)

// ========== Interface 정의 (필수!) ==========
// 테스트 시 Mock 생성 가능
type Repository interface {
    // 생성
    Create(blog *Blog) error
    CreateTx(tx *sql.Tx, blog *Blog) error

    // 조회
    FindByID(id int64) (*Blog, error)
    FindByAuthor(authorID string, offset, limit int) ([]Blog, error)
    FindAll(offset, limit int) ([]Blog, error)
    Count() (int64, error)

    // 수정
    Update(id int64, updates map[string]interface{}) error
    UpdateTx(tx *sql.Tx, id int64, updates map[string]interface{}) error
    IncrementViews(id int64) error

    // 삭제
    Delete(id int64) error

    // 유틸
    Exists(id int64) (bool, error)
}

// ========== 구현체 ==========
type repository struct {
    base *database.Repository // 공통 DB 함수 재사용
}

// 생성자
func NewRepository(db *database.DB) Repository {
    return &repository{
        base: database.NewRepository(db),
    }
}

// ========== 메서드 구현 ==========

// Create 블로그 생성
func (r *repository) Create(blog *Blog) error {
    data := map[string]interface{}{
        "b_title":     blog.Title,
        "b_content":   blog.Content,
        "b_author_id": blog.AuthorID,
        "b_views":     0,
    }

    id, err := r.base.Insert("_blog", data)
    if err != nil {
        logger.Error("블로그 생성 실패: %v", err)
        return errors.Wrap(err, "BLOG_CREATE_FAILED", "블로그 생성에 실패했습니다")
    }

    blog.ID = id
    return nil
}

// CreateTx 트랜잭션 내에서 생성
func (r *repository) CreateTx(tx *sql.Tx, blog *Blog) error {
    data := map[string]interface{}{
        "b_title":     blog.Title,
        "b_content":   blog.Content,
        "b_author_id": blog.AuthorID,
    }

    id, err := r.base.InsertTx(tx, "_blog", data)
    if err != nil {
        return errors.Wrap(err, "BLOG_CREATE_FAILED", "블로그 생성에 실패했습니다")
    }

    blog.ID = id
    return nil
}

// FindByID ID로 조회
func (r *repository) FindByID(id int64) (*Blog, error) {
    query := `
        SELECT b_id, b_title, b_content, b_author_id, b_views,
               b_created_at, b_updated_at
        FROM _blog
        WHERE b_id = ?
    `

    blog := &Blog{}
    err := r.base.QueryRow(query, id).Scan(
        &blog.ID, &blog.Title, &blog.Content, &blog.AuthorID,
        &blog.Views, &blog.CreatedAt, &blog.UpdatedAt,
    )

    if err == sql.ErrNoRows {
        return nil, errors.ErrRecordNotFound
    }

    if err != nil {
        logger.Error("블로그 조회 실패 (ID: %d): %v", id, err)
        return nil, errors.Wrap(err, "BLOG_FIND_FAILED", "블로그 조회에 실패했습니다")
    }

    return blog, nil
}

// FindByAuthor 작성자별 조회
func (r *repository) FindByAuthor(authorID string, offset, limit int) ([]Blog, error) {
    query := `
        SELECT b_id, b_title, b_content, b_author_id, b_views,
               b_created_at, b_updated_at
        FROM _blog
        WHERE b_author_id = ?
        ORDER BY b_created_at DESC
        LIMIT ? OFFSET ?
    `

    rows, err := r.base.Query(query, authorID, limit, offset)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var blogs []Blog
    for rows.Next() {
        var blog Blog
        if err := rows.Scan(
            &blog.ID, &blog.Title, &blog.Content, &blog.AuthorID,
            &blog.Views, &blog.CreatedAt, &blog.UpdatedAt,
        ); err != nil {
            return nil, err
        }
        blogs = append(blogs, blog)
    }

    return blogs, nil
}

// Update 수정
func (r *repository) Update(id int64, updates map[string]interface{}) error {
    affected, err := r.base.Update("_blog", updates, "b_id = ?", id)
    if err != nil {
        return errors.Wrap(err, "BLOG_UPDATE_FAILED", "블로그 수정에 실패했습니다")
    }

    if affected == 0 {
        return errors.ErrRecordNotFound
    }

    return nil
}

// IncrementViews 조회수 증가
func (r *repository) IncrementViews(id int64) error {
    query := "UPDATE _blog SET b_views = b_views + 1 WHERE b_id = ?"
    _, err := r.base.Exec(query, id)
    return err
}

// Delete 삭제
func (r *repository) Delete(id int64) error {
    affected, err := r.base.Delete("_blog", "b_id = ?", id)
    if err != nil {
        return errors.Wrap(err, "BLOG_DELETE_FAILED", "블로그 삭제에 실패했습니다")
    }

    if affected == 0 {
        return errors.ErrRecordNotFound
    }

    return nil
}

// Exists 존재 확인
func (r *repository) Exists(id int64) (bool, error) {
    return r.base.Exists("_blog", "b_id = ?", id)
}

// Count 전체 개수
func (r *repository) Count() (int64, error) {
    return r.base.Count("_blog", "1=1")
}
```

**작성 규칙:**
- **반드시 Interface 먼저 정의**
- 구조체는 소문자 (외부 노출 X)
- 에러는 항상 래핑
- 트랜잭션 메서드는 별도 (CreateTx, UpdateTx)
- 공통 기능은 `r.base.*` 사용

---

### 3. service.go - 비즈니스 로직

**역할:**
- 비즈니스 규칙 적용
- 여러 Repository 조합
- 트랜잭션 관리
- 검증 및 변환

**작성 예시:**

```go
package blog

import (
    "gin_starter/internal/infrastructure/database"
    "gin_starter/pkg/errors"
    "gin_starter/pkg/logger"
    "time"
)

// ========== Interface 정의 ==========
type Service interface {
    // 생성
    CreateBlog(authorID string, req *CreateBlogRequest) (*Blog, error)

    // 조회
    GetBlog(id int64) (*BlogResponse, error)
    GetBlogsByAuthor(authorID string, page, perPage int) (*BlogListResponse, error)

    // 수정
    UpdateBlog(id int64, authorID string, req *UpdateBlogRequest) error

    // 삭제
    DeleteBlog(id int64, authorID string) error
}

// ========== 구현체 ==========
type service struct {
    repo   Repository
    userRepo UserRepository // 다른 도메인 Repository 사용 가능
}

// 생성자
func NewService(repo Repository) Service {
    return &service{
        repo: repo,
    }
}

// ========== 메서드 구현 ==========

// CreateBlog 블로그 생성
func (s *service) CreateBlog(authorID string, req *CreateBlogRequest) (*Blog, error) {
    // 1. 비즈니스 검증
    if len(req.Title) > 200 {
        return nil, errors.New("TITLE_TOO_LONG", "제목이 너무 깁니다")
    }

    if len(req.Content) < 10 {
        return nil, errors.New("CONTENT_TOO_SHORT", "내용이 너무 짧습니다")
    }

    // 2. 도메인 엔티티 생성
    blog := &Blog{
        Title:     req.Title,
        Content:   req.Content,
        AuthorID:  authorID,
        Views:     0,
        CreatedAt: time.Now(),
        UpdatedAt: time.Now(),
    }

    // 3. Repository 호출
    if err := s.repo.Create(blog); err != nil {
        return nil, err
    }

    logger.Info("블로그 생성 완료: %d (작성자: %s)", blog.ID, authorID)

    return blog, nil
}

// GetBlog 블로그 상세 조회
func (s *service) GetBlog(id int64) (*BlogResponse, error) {
    // 1. 조회
    blog, err := s.repo.FindByID(id)
    if err != nil {
        if errors.Is(err, errors.ErrRecordNotFound) {
            return nil, errors.New("BLOG_NOT_FOUND", "블로그를 찾을 수 없습니다")
        }
        return nil, err
    }

    // 2. 조회수 증가 (비동기 또는 별도 처리 권장)
    go func() {
        if err := s.repo.IncrementViews(id); err != nil {
            logger.Error("조회수 증가 실패: %v", err)
        }
    }()

    // 3. DTO 변환
    return blog.ToResponse(), nil
}

// GetBlogsByAuthor 작성자별 블로그 목록
func (s *service) GetBlogsByAuthor(authorID string, page, perPage int) (*BlogListResponse, error) {
    // 1. 페이지네이션 검증
    if page < 1 {
        page = 1
    }
    if perPage < 1 || perPage > 100 {
        perPage = 20
    }

    offset := (page - 1) * perPage

    // 2. 데이터 조회
    blogs, err := s.repo.FindByAuthor(authorID, offset, perPage)
    if err != nil {
        return nil, err
    }

    // 3. 전체 개수 조회
    total, err := s.repo.Count()
    if err != nil {
        return nil, err
    }

    // 4. 응답 DTO 생성
    responses := make([]BlogResponse, len(blogs))
    for i, blog := range blogs {
        responses[i] = *blog.ToResponse()
    }

    return &BlogListResponse{
        Blogs:      responses,
        TotalCount: int(total),
        Page:       page,
        PerPage:    perPage,
    }, nil
}

// UpdateBlog 블로그 수정
func (s *service) UpdateBlog(id int64, authorID string, req *UpdateBlogRequest) error {
    // 1. 존재 및 권한 확인
    blog, err := s.repo.FindByID(id)
    if err != nil {
        return err
    }

    if blog.AuthorID != authorID {
        return errors.New("FORBIDDEN", "수정 권한이 없습니다")
    }

    // 2. 수정 데이터 구성
    updates := make(map[string]interface{})

    if req.Title != "" {
        if len(req.Title) > 200 {
            return errors.New("TITLE_TOO_LONG", "제목이 너무 깁니다")
        }
        updates["b_title"] = req.Title
    }

    if req.Content != "" {
        if len(req.Content) < 10 {
            return errors.New("CONTENT_TOO_SHORT", "내용이 너무 짧습니다")
        }
        updates["b_content"] = req.Content
    }

    updates["b_updated_at"] = time.Now()

    // 3. 업데이트 실행
    if err := s.repo.Update(id, updates); err != nil {
        return err
    }

    logger.Info("블로그 수정 완료: %d", id)

    return nil
}

// DeleteBlog 블로그 삭제
func (s *service) DeleteBlog(id int64, authorID string) error {
    // 1. 권한 확인
    blog, err := s.repo.FindByID(id)
    if err != nil {
        return err
    }

    if blog.AuthorID != authorID {
        return errors.New("FORBIDDEN", "삭제 권한이 없습니다")
    }

    // 2. 삭제 (연관 데이터 처리 필요 시 트랜잭션)
    if err := s.repo.Delete(id); err != nil {
        return err
    }

    logger.Info("블로그 삭제 완료: %d", id)

    return nil
}

// ========== 트랜잭션 예시 ==========
// CreateBlogWithCategory 카테고리와 함께 생성 (트랜잭션)
func (s *service) CreateBlogWithCategory(authorID string, req *CreateBlogRequest, categoryID int64) (*Blog, error) {
    // 트랜잭션 시작
    tx, err := s.repo.(*repository).base.db.BeginTx()
    if err != nil {
        return nil, err
    }
    defer database.RollbackTx(tx)

    // 1. 블로그 생성
    blog := &Blog{
        Title:    req.Title,
        Content:  req.Content,
        AuthorID: authorID,
    }

    if err := s.repo.CreateTx(tx, blog); err != nil {
        return nil, err
    }

    // 2. 카테고리 연결 (다른 테이블 작업)
    // ... category 관련 작업

    // 3. 커밋
    if err := database.CommitTx(tx); err != nil {
        return nil, err
    }

    return blog, nil
}
```

**작성 규칙:**
- **Interface 필수**
- 비즈니스 검증은 Service에서
- Repository는 여러 개 조합 가능
- 트랜잭션은 Service에서 관리
- 중요한 작업은 로깅
- DTO 변환은 Service에서

---

### 4. handler.go - HTTP 핸들러

**역할:**
- HTTP 요청 파싱
- 입력 검증
- Service 호출
- 응답 포맷팅

**작성 예시:**

```go
package blog

import (
    "gin_starter/pkg/response"
    "gin_starter/pkg/validator"
    "strconv"

    "github.com/gin-gonic/gin"
)

// ========== Handler 구조체 ==========
type Handler struct {
    service Service
}

// 생성자
func NewHandler(service Service) *Handler {
    return &Handler{
        service: service,
    }
}

// ========== 핸들러 메서드 ==========

// CreateBlog 블로그 생성
// @Summary 블로그 생성
// @Description 새로운 블로그 포스트를 생성합니다
// @Tags Blog
// @Accept json
// @Produce json
// @Param body body CreateBlogRequest true "블로그 정보"
// @Success 201 {object} response.Response{data=Blog}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Router /api/blog [post]
// @Security Bearer
func (h *Handler) CreateBlog(c *gin.Context) {
    // 1. 인증 확인
    authorID, exists := c.Get("user_id")
    if !exists {
        response.Unauthorized(c, "인증이 필요합니다")
        return
    }

    // 2. 입력 검증
    rules := []validator.Rule{
        {
            Field:    "title",
            Label:    "제목",
            Required: true,
            MinLen:   1,
            MaxLen:   200,
        },
        {
            Field:    "content",
            Label:    "내용",
            Required: true,
            MinLen:   10,
            MaxLen:   10000,
        },
    }

    result := validator.Validate(c, rules)
    if !result.Valid {
        response.ValidationError(c, result.GetErrorMap())
        return
    }

    // 3. 요청 DTO 생성
    req := &CreateBlogRequest{
        Title:   result.Values["title"],
        Content: result.Values["content"],
    }

    // 4. Service 호출
    blog, err := h.service.CreateBlog(authorID.(string), req)
    if err != nil {
        response.InternalError(c, err.Error())
        return
    }

    // 5. 성공 응답
    response.Created(c, blog)
}

// GetBlog 블로그 조회
// @Summary 블로그 조회
// @Tags Blog
// @Produce json
// @Param id path int true "블로그 ID"
// @Success 200 {object} response.Response{data=BlogResponse}
// @Failure 404 {object} response.Response
// @Router /api/blog/{id} [get]
func (h *Handler) GetBlog(c *gin.Context) {
    // 1. URL 파라미터 파싱
    idStr := c.Param("id")
    id, err := strconv.ParseInt(idStr, 10, 64)
    if err != nil {
        response.BadRequest(c, "잘못된 ID 형식입니다")
        return
    }

    // 2. Service 호출
    blog, err := h.service.GetBlog(id)
    if err != nil {
        response.NotFound(c, err.Error())
        return
    }

    // 3. 성공 응답
    response.Success(c, blog)
}

// ListBlogs 블로그 목록
// @Summary 블로그 목록
// @Tags Blog
// @Produce json
// @Param page query int false "페이지 번호" default(1)
// @Param per_page query int false "페이지당 개수" default(20)
// @Param author_id query string false "작성자 ID"
// @Success 200 {object} response.Response{data=BlogListResponse}
// @Router /api/blog [get]
func (h *Handler) ListBlogs(c *gin.Context) {
    // 1. 쿼리 파라미터 파싱
    rules := []validator.Rule{
        {Field: "page", Label: "페이지", Pattern: validator.PatternNumber, Min: 1},
        {Field: "per_page", Label: "페이지당 개수", Pattern: validator.PatternNumber, Min: 1, Max: 100},
        {Field: "author_id", Label: "작성자 ID"},
    }

    result := validator.Validate(c, rules)

    page, _ := strconv.Atoi(result.Values["page"])
    if page < 1 {
        page = 1
    }

    perPage, _ := strconv.Atoi(result.Values["per_page"])
    if perPage < 1 {
        perPage = 20
    }

    authorID := result.Values["author_id"]

    // 2. Service 호출
    var blogs *BlogListResponse
    var err error

    if authorID != "" {
        blogs, err = h.service.GetBlogsByAuthor(authorID, page, perPage)
    } else {
        // blogs, err = h.service.GetAllBlogs(page, perPage)
    }

    if err != nil {
        response.InternalError(c, err.Error())
        return
    }

    // 3. 성공 응답
    response.Success(c, blogs)
}

// UpdateBlog 블로그 수정
// @Summary 블로그 수정
// @Tags Blog
// @Accept json
// @Produce json
// @Param id path int true "블로그 ID"
// @Param body body UpdateBlogRequest true "수정 정보"
// @Success 200 {object} response.Response
// @Failure 403 {object} response.Response
// @Router /api/blog/{id} [put]
// @Security Bearer
func (h *Handler) UpdateBlog(c *gin.Context) {
    // 1. 인증 확인
    authorID, exists := c.Get("user_id")
    if !exists {
        response.Unauthorized(c, "인증이 필요합니다")
        return
    }

    // 2. ID 파싱
    idStr := c.Param("id")
    id, err := strconv.ParseInt(idStr, 10, 64)
    if err != nil {
        response.BadRequest(c, "잘못된 ID 형식입니다")
        return
    }

    // 3. 입력 검증
    rules := []validator.Rule{
        {Field: "title", Label: "제목", MaxLen: 200},
        {Field: "content", Label: "내용", MinLen: 10},
    }

    result := validator.Validate(c, rules)

    req := &UpdateBlogRequest{
        Title:   result.Values["title"],
        Content: result.Values["content"],
    }

    // 4. Service 호출
    if err := h.service.UpdateBlog(id, authorID.(string), req); err != nil {
        response.InternalError(c, err.Error())
        return
    }

    // 5. 성공 응답
    response.Success(c, gin.H{"message": "수정되었습니다"})
}

// DeleteBlog 블로그 삭제
// @Summary 블로그 삭제
// @Tags Blog
// @Produce json
// @Param id path int true "블로그 ID"
// @Success 204
// @Failure 403 {object} response.Response
// @Router /api/blog/{id} [delete]
// @Security Bearer
func (h *Handler) DeleteBlog(c *gin.Context) {
    // 1. 인증 확인
    authorID, exists := c.Get("user_id")
    if !exists {
        response.Unauthorized(c, "인증이 필요합니다")
        return
    }

    // 2. ID 파싱
    idStr := c.Param("id")
    id, err := strconv.ParseInt(idStr, 10, 64)
    if err != nil {
        response.BadRequest(c, "잘못된 ID 형식입니다")
        return
    }

    // 3. Service 호출
    if err := h.service.DeleteBlog(id, authorID.(string)); err != nil {
        response.InternalError(c, err.Error())
        return
    }

    // 4. 성공 응답 (204 No Content)
    response.NoContent(c)
}
```

**작성 규칙:**
- Handler는 얇게 (비즈니스 로직 X)
- 입력 검증은 validator 사용
- 응답은 response 패키지 사용
- Swagger 주석 필수
- 에러 처리는 간단히 (Service에서 처리됨)

---

## 🚀 새 도메인 추가 완전 가이드

### Step 1: 디렉토리 생성

```bash
mkdir -p internal/domain/blog
cd internal/domain/blog
```

### Step 2: 파일 생성 순서

```bash
# 1. model.go 먼저 (데이터 구조)
touch model.go

# 2. repository.go (데이터 접근)
touch repository.go

# 3. service.go (비즈니스 로직)
touch service.go

# 4. handler.go (HTTP)
touch handler.go
```

### Step 3: 코드 작성 순서

1. **model.go** - 도메인 엔티티, DTO 정의
2. **repository.go** - Interface 정의 → 구현
3. **service.go** - Interface 정의 → 구현
4. **handler.go** - 구현 (Interface 불필요)

### Step 4: Routes 등록

```go
// api/routes/routes.go에 추가
import "gin_starter/internal/domain/blog"

func setupBlogRoutes(rg *gin.RouterGroup, db *database.DB) {
    repo := blog.NewRepository(db)
    service := blog.NewService(repo)
    handler := blog.NewHandler(service)

    blogGroup := rg.Group("/blog")
    {
        blogGroup.POST("", handler.CreateBlog)
        blogGroup.GET("/:id", handler.GetBlog)
        blogGroup.GET("", handler.ListBlogs)
        blogGroup.PUT("/:id", handler.UpdateBlog)
        blogGroup.DELETE("/:id", handler.DeleteBlog)
    }
}
```

### Step 5: Swagger 재생성

```bash
swag init -g cmd/server/main.go
```

---

## ✅ 체크리스트

새 도메인 추가 시 확인사항:

- [ ] model.go - 엔티티, DTO 정의
- [ ] repository.go - Interface와 구현 분리
- [ ] service.go - Interface와 구현 분리
- [ ] handler.go - Swagger 주석 작성
- [ ] Routes 등록
- [ ] Swagger 재생성
- [ ] 테스트 코드 작성
- [ ] 에러 처리 확인
- [ ] 로깅 추가

---

## 🎓 학습 예제

완전한 예제는 `internal/domain/user/` 참고