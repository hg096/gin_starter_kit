# CLAUDE.md

이 파일은 Claude Code(claude.ai/code)가 이 저장소에서 작업할 때 참고하는 가이드입니다.

## 프로젝트 개요

Clean Architecture 기반 Gin 웹 API 스타터 킷입니다. 코드 재사용성과 유지보수성을 최우선으로 설계되었습니다.

**핵심 원칙:**
- 의존성 역전 (Interface 기반)
- 레이어 분리 (Handler → Service → Repository)
- 명확한 책임 분리

## 명령어

### 개발 환경
```bash
# 서버 실행
go run cmd/server/main.go

# 빌드
go build -o bin/server cmd/server/main.go

# Swagger 문서 생성
swag init -g cmd/server/main.go

# 의존성 정리
go mod tidy

# 테스트
go test ./...
```

### 환경 설정
- `.env` 파일 필수 (`exenv.txt` 참고)
- JWT 시크릿: 각각 32자 필수
- Go 버전: 1.25.1

## 아키텍처

### 프로젝트 구조

```
gin_starter/
├── cmd/server/              # 진입점
├── internal/                # 외부 import 불가
│   ├── config/             # 중앙 설정 관리
│   ├── domain/             # 비즈니스 도메인
│   │   └── user/
│   │       ├── model.go        # 도메인 모델, 요청/응답 DTO
│   │       ├── repository.go   # DB 접근 인터페이스 및 구현
│   │       ├── service.go      # 비즈니스 로직
│   │       └── handler.go      # HTTP 핸들러
│   ├── middleware/         # HTTP 미들웨어
│   └── infrastructure/     # 외부 시스템
├── pkg/                     # 외부 import 가능
│   ├── response/           # 표준 응답
│   ├── validator/          # 입력 검증
│   ├── errors/             # 에러 정의
│   └── logger/             # 로거
└── api/routes/             # 라우트 정의
```

### 레이어 간 흐름

```
HTTP Request
    ↓
Handler (api/routes, internal/domain/*/handler.go)
  - HTTP 요청/응답 처리
  - 입력 검증 (pkg/validator)
  - 응답 포맷팅 (pkg/response)
    ↓
Service (internal/domain/*/service.go)
  - 비즈니스 로직
  - 트랜잭션 관리
  - 도메인 규칙 적용
    ↓
Repository (internal/domain/*/repository.go)
  - DB 쿼리 실행
  - 데이터 매핑
  - infrastructure/database 사용
    ↓
Database
```

## 코딩 패턴

### 1. 새 기능 추가 순서

**Step 1: Model 정의**
```go
// internal/domain/blog/model.go
package blog

type Blog struct {
    ID      int64  `json:"id"`
    Title   string `json:"title"`
    Content string `json:"content"`
}

// 요청/응답 DTO도 함께 정의
type CreateBlogRequest struct {
    Title   string `json:"title" binding:"required"`
    Content string `json:"content" binding:"required"`
}
```

**Step 2: Repository Interface & 구현**
```go
// internal/domain/blog/repository.go
package blog

import "gin_starter/internal/infrastructure/database"

// Interface 정의 (테스트 Mock 가능)
type Repository interface {
    Create(blog *Blog) error
    FindByID(id int64) (*Blog, error)
}

// 구현
type repository struct {
    base *database.Repository // 공통 DB 함수 재사용
}

func NewRepository(db *database.DB) Repository {
    return &repository{base: database.NewRepository(db)}
}

func (r *repository) Create(blog *Blog) error {
    // base.Insert 사용
    data := map[string]interface{}{
        "title": blog.Title,
        "content": blog.Content,
    }
    id, err := r.base.Insert("_blog", data)
    if err != nil {
        return err
    }
    blog.ID = id
    return nil
}
```

**Step 3: Service Interface & 구현**
```go
// internal/domain/blog/service.go
package blog

import "gin_starter/pkg/errors"

type Service interface {
    CreateBlog(req *CreateBlogRequest) (*Blog, error)
}

type service struct {
    repo Repository
}

func NewService(repo Repository) Service {
    return &service{repo: repo}
}

func (s *service) CreateBlog(req *CreateBlogRequest) (*Blog, error) {
    // 비즈니스 로직
    blog := &Blog{
        Title: req.Title,
        Content: req.Content,
    }

    if err := s.repo.Create(blog); err != nil {
        return nil, errors.Wrap(err, "BLOG_CREATE_FAILED", "블로그 생성 실패")
    }

    return blog, nil
}
```

**Step 4: Handler 작성**
```go
// internal/domain/blog/handler.go
package blog

import (
    "gin_starter/pkg/response"
    "gin_starter/pkg/validator"
    "github.com/gin-gonic/gin"
)

type Handler struct {
    service Service
}

func NewHandler(service Service) *Handler {
    return &Handler{service: service}
}

// @Summary 블로그 생성
// @Tags Blog
// @Accept json
// @Produce json
// @Param body body CreateBlogRequest true "블로그 정보"
// @Success 201 {object} response.Response
// @Router /api/blog [post]
func (h *Handler) CreateBlog(c *gin.Context) {
    // 입력 검증
    rules := []validator.Rule{
        {Field: "title", Label: "제목", Required: true, MaxLen: 200},
        {Field: "content", Label: "내용", Required: true},
    }

    result := validator.Validate(c, rules)
    if !result.Valid {
        response.ValidationError(c, result.GetErrorMap())
        return
    }

    req := &CreateBlogRequest{
        Title: result.Values["title"],
        Content: result.Values["content"],
    }

    blog, err := h.service.CreateBlog(req)
    if err != nil {
        response.InternalError(c, err.Error())
        return
    }

    response.Created(c, blog)
}
```

**Step 5: Routes 등록**
```go
// api/routes/routes.go에 추가

import "gin_starter/internal/domain/blog"

func setupBlogRoutes(rg *gin.RouterGroup, db *database.DB) {
    // 의존성 주입
    repo := blog.NewRepository(db)
    service := blog.NewService(repo)
    handler := blog.NewHandler(service)

    blogGroup := rg.Group("/blog")
    {
        blogGroup.POST("", handler.CreateBlog)
    }
}

// SetupRoutes에서 호출
func SetupRoutes(r *gin.Engine, db *database.DB, cfg *config.Config) {
    // ...
    api := r.Group("/api")
    {
        setupBlogRoutes(api, db)
    }
}
```

### 2. 데이터베이스 작업

**공통 Repository 함수 사용 (infrastructure/database/repository.go):**

```go
// INSERT
id, err := r.base.Insert("table_name", map[string]interface{}{
    "column": "value",
})

// UPDATE
affected, err := r.base.Update("table_name",
    map[string]interface{}{"column": "new_value"},
    "id = ?", id)

// SELECT 단일
row := r.base.QueryRow("SELECT * FROM table WHERE id = ?", id)

// SELECT 다중
rows, err := r.base.Query("SELECT * FROM table")

// DELETE
affected, err := r.base.Delete("table_name", "id = ?", id)

// Exists
exists, err := r.base.Exists("table_name", "id = ?", id)

// Count
count, err := r.base.Count("table_name", "status = ?", "active")
```

**트랜잭션:**
```go
// Service에서
func (s *service) CreateWithTransaction() error {
    tx, err := s.db.BeginTx()
    if err != nil {
        return err
    }
    defer database.RollbackTx(tx) // 에러 시 자동 롤백

    // Repository에 tx 전달
    if err := s.repo.CreateTx(tx, data); err != nil {
        return err
    }

    return database.CommitTx(tx)
}
```

### 3. 응답 패턴 (pkg/response)

```go
// 성공
response.Success(c, gin.H{"data": result})

// 생성 성공
response.Created(c, createdItem)

// 검증 에러
response.ValidationError(c, errorMap)

// 인증 에러
response.Unauthorized(c, "메시지")

// 권한 에러
response.Forbidden(c, "메시지")

// 서버 에러
response.InternalError(c, "메시지")

// 커스텀 에러
response.Error(c, statusCode, "ERROR_CODE", "메시지", details)
```

### 4. 에러 처리 (pkg/errors)

```go
// 에러 생성
err := errors.New("ERROR_CODE", "에러 메시지")

// 에러 래핑
err := errors.Wrap(originalErr, "ERROR_CODE", "컨텍스트 메시지")

// 미리 정의된 에러 사용
return errors.ErrUserNotFound
return errors.ErrInvalidToken

// 메타데이터 추가
err.WithMeta("user_id", userID)
```

### 5. 로깅 (pkg/logger)

```go
logger.Debug("디버그 메시지: %s", value)
logger.Info("정보 메시지: %s", value)
logger.Warn("경고 메시지: %s", value)
logger.Error("에러 발생: %v", err)
logger.Fatal("치명적 에러: %v", err) // 프로그램 종료

// 필드와 함께
logger.WithField("user_id", userID).Info("로그인 성공")
logger.WithFields(map[string]interface{}{
    "user_id": userID,
    "ip": ip,
}).Warn("의심스러운 활동")
```

### 6. 설정 사용 (internal/config)

```go
// 설정 가져오기
cfg := config.Get()

// 사용 예시
cfg.Server.Port
cfg.Database.MaxOpenConns
cfg.JWT.AccessSecret
cfg.App.IsDevelopment()
cfg.App.IsProduction()
cfg.GetDSN() // MySQL DSN 문자열
```

## 인증 & 권한

### 미들웨어 사용

```go
// 인증 필요
auth := userGroup.Group("")
auth.Use(middleware.AuthMiddleware(cfg))
{
    auth.GET("/profile", handler.GetProfile)
}

// 특정 사용자 타입 요구
auth.Use(middleware.RequireUserType("A")) // Admin
auth.Use(middleware.RequireAuthLevel(5))  // 레벨 5 이상
```

### 토큰 생성 & 검증

```go
// Service에서 토큰 생성
accessToken, err := middleware.GenerateToken(
    userID,
    cfg.JWT.AccessExpireMin,
    cfg.JWT.AccessSecret,
    cfg.JWT.TokenSecret,
    cfg.App.ServiceName,
)

// 토큰 검증 (미들웨어가 자동 처리)
// Handler에서 사용자 정보 가져오기
userID, _ := c.Get("user_id")
```

## 입력 검증

```go
// Validator 규칙 정의
rules := []validator.Rule{
    {
        Field: "email",
        Label: "이메일",
        Required: true,
        Pattern: validator.PatternEmail,
    },
    {
        Field: "age",
        Label: "나이",
        Required: true,
        Min: 18,
        Max: 120,
    },
    {
        Field: "username",
        Label: "사용자명",
        Required: true,
        MinLen: 3,
        MaxLen: 20,
        Pattern: validator.PatternAlphaNum,
    },
}

// 검증 실행
result := validator.Validate(c, rules)
if !result.Valid {
    response.ValidationError(c, result.GetErrorMap())
    return
}

// 검증된 값 사용
email := result.Values["email"]
```

**사용 가능한 패턴:**
- `PatternEmail` - 이메일
- `PatternNumber` - 숫자만
- `PatternAlphaNum` - 영숫자
- `PatternKorean` - 한글
- `PatternKorEng` - 한글+영문
- `PatternKorEngNum` - 한글+영문+숫자
- `PatternURL` - URL
- `PatternPhone` - 전화번호

## 테스트 작성

```go
// Repository Mock
type mockRepository struct{}

func (m *mockRepository) Create(user *User) error {
    return nil
}

// Service 테스트
func TestService(t *testing.T) {
    repo := &mockRepository{}
    service := NewService(repo, config.Get())

    // 테스트 로직
}
```

## 개발 노트

### 필수 규칙
1. **항상 Interface 사용** - Repository, Service는 인터페이스로 정의
2. **의존성 주입** - 생성자 함수에서 의존성 주입
3. **에러 래핑** - 모든 에러는 컨텍스트와 함께 래핑
4. **로깅** - 중요한 작업은 반드시 로깅
5. **검증** - 모든 외부 입력은 검증

### 레이어별 책임
- **Handler**: HTTP 요청/응답, 입력 검증, 응답 포맷
- **Service**: 비즈니스 로직, 트랜잭션 관리, 도메인 규칙
- **Repository**: DB 쿼리, 데이터 매핑만

### 주의사항
- `internal/` 패키지는 외부에서 import 불가
- `pkg/` 패키지만 다른 프로젝트에서 재사용 가능
- Handler에서 직접 Repository 호출 금지 (반드시 Service 경유)
- 비밀번호는 항상 bcrypt로 해싱
- 모든 API 응답은 `pkg/response` 사용

## Swagger 주석

```go
// @Summary 요약
// @Description 상세 설명
// @Tags 태그명
// @Accept json
// @Produce json
// @Param id path int true "ID"
// @Param body body RequestDTO true "요청 바디"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Router /api/path [get]
// @Security Bearer
func Handler(c *gin.Context) {}
```