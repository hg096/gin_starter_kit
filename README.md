# Gin Starter Kit v2.0 🚀

**엔터프라이즈급** Go 언어 웹 API 스타터 킷입니다. Clean Architecture 기반으로 설계되어 **코드 재사용성**과 **유지보수성**을 최우선으로 합니다.

## ✨ 주요 특징

- ✅ **Clean Architecture** - Domain 중심 설계, 레이어 분리
- ✅ **높은 재사용성** - Interface 기반, 의존성 주입
- ✅ **타입 안정성** - 명확한 타입 정의, 에러 처리
- ✅ **보안 강화** - JWT AES-GCM 암호화, 입력 검증
- ✅ **테스트 용이** - Mock 가능한 구조
- ✅ **개발 편의성** - Swagger, Hot reload, 구조화된 로깅

## 🏗️ 아키텍처

```
┌─────────────────────────────────────────────────┐
│                   Handler                        │ ← HTTP 요청 처리
├─────────────────────────────────────────────────┤
│                   Service                        │ ← 비즈니스 로직
├─────────────────────────────────────────────────┤
│                  Repository                      │ ← 데이터 접근
├─────────────────────────────────────────────────┤
│              Infrastructure (DB)                 │ ← 외부 의존성
└─────────────────────────────────────────────────┘
```

### 프로젝트 구조

```
gin_starter/
├── cmd/
│   └── server/
│       └── main.go              # 애플리케이션 진입점
│
├── internal/                     # 외부 import 불가
│   ├── config/                  # 설정 중앙 관리
│   │   └── config.go
│   │
│   ├── domain/                  # 도메인 로직 (핵심)
│   │   └── user/
│   │       ├── model.go         # 도메인 모델
│   │       ├── repository.go    # 데이터 접근 인터페이스
│   │       ├── service.go       # 비즈니스 로직
│   │       └── handler.go       # HTTP 핸들러
│   │
│   ├── middleware/              # HTTP 미들웨어
│   │   ├── auth.go              # JWT 인증
│   │   ├── logger.go            # 로깅
│   │   └── cors.go              # CORS
│   │
│   └── infrastructure/          # 외부 시스템 연동
│       └── database/
│           ├── mysql.go         # DB 연결
│           └── repository.go    # 공통 쿼리 함수
│
├── pkg/                         # 외부 import 가능
│   ├── response/                # 표준 API 응답
│   ├── validator/               # 입력 검증
│   ├── errors/                  # 에러 정의
│   └── logger/                  # 로거
│
├── api/
│   └── routes/                  # 라우트 정의
│
├── docs/                        # Swagger 자동생성
├── bin/                         # 빌드된 바이너리
└── .env                         # 환경 변수
```

## 🚀 빠른 시작

### 1. 요구사항

- **Go 1.25.1+**
- **MySQL 5.7+** 또는 **MariaDB**
- **Git**

### 2. 설치

```bash
git clone <your-repository-url>
cd gin_starter

# 의존성 설치
go mod tidy
```

### 3. 데이터베이스 설정

```sql
CREATE DATABASE gin_starter CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
```

```bash
# 테이블 생성
mysql -u root -p gin_starter < table.sql
```

### 4. 환경 변수 설정

```bash
# .env 파일 생성
cp exenv.txt .env

# .env 파일 수정
nano .env
```

**.env 예시:**
```env
PORT=8080
GIN_MODE=debug

# Database
DB_HOST=localhost
DB_PORT=3306
DB_USER=root
DB_PASS=your_password
DB_NAME=gin_starter
DB_MAX_OPEN_CONNS=25
DB_MAX_IDLE_CONNS=5

# JWT (각 32자 필수!)
JWT_SECRET=your-32-character-access-key!!
JWT_REFRESH_SECRET=your-32-character-refresh-key!
JWT_TOKEN_SECRET=your-32-character-encrypt-key!
JWT_EXPIRES_IN=30        # 액세스 토큰 (분)
JWT_EXPIRES_RE=7         # 리프레시 토큰 (일)

# App
SERVICE_NAME=GinStarter
```

### 5. 실행

```bash
# Swagger 문서 생성
swag init -g cmd/server/main.go

# 개발 모드 실행
go run cmd/server/main.go

# 또는 빌드 후 실행
go build -o bin/server cmd/server/main.go
./bin/server
```

서버 시작 후:
- **API 서버**: http://localhost:8080
- **Swagger 문서**: http://localhost:8080/swagger/index.html
- **Health Check**: http://localhost:8080/health

## 📖 API 예제

### 1. 회원가입

```bash
curl -X POST http://localhost:8080/api/user/register \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "testuser",
    "user_pass": "password123",
    "user_name": "홍길동",
    "user_email": "test@example.com"
  }'
```

**응답:**
```json
{
  "success": true,
  "data": {
    "user": {
      "id": "testuser",
      "name": "홍길동",
      "email": "test@example.com",
      "auth_type": "U",
      "auth_level": 1
    }
  }
}
```

### 2. 로그인

```bash
curl -X POST http://localhost:8080/api/user/login \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "testuser",
    "user_pass": "password123"
  }'
```

**응답:**
```json
{
  "success": true,
  "data": {
    "access_token": "eyJhbGci...",
    "refresh_token": "eyJhbGci...",
    "user": {
      "id": "testuser",
      "name": "홍길동",
      "email": "test@example.com"
    }
  }
}
```

### 3. 프로필 조회 (인증 필요)

```bash
curl -X GET http://localhost:8080/api/user/profile \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN"
```

## 💡 새로운 기능 추가하기

### Step 1: Domain 생성

```go
// internal/domain/blog/model.go
package blog

type Blog struct {
    ID      int64  `json:"id"`
    Title   string `json:"title"`
    Content string `json:"content"`
}
```

### Step 2: Repository 작성

```go
// internal/domain/blog/repository.go
package blog

type Repository interface {
    Create(blog *Blog) error
    FindByID(id int64) (*Blog, error)
}

type repository struct {
    base *database.Repository
}

func NewRepository(db *database.DB) Repository {
    return &repository{base: database.NewRepository(db)}
}

func (r *repository) Create(blog *Blog) error {
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

### Step 3: Service 작성

```go
// internal/domain/blog/service.go
package blog

type Service interface {
    CreateBlog(title, content string) (*Blog, error)
}

type service struct {
    repo Repository
}

func NewService(repo Repository) Service {
    return &service{repo: repo}
}

func (s *service) CreateBlog(title, content string) (*Blog, error) {
    blog := &Blog{Title: title, Content: content}
    if err := s.repo.Create(blog); err != nil {
        return nil, err
    }
    return blog, nil
}
```

### Step 4: Handler 작성

```go
// internal/domain/blog/handler.go
package blog

import (
    "gin_starter/pkg/response"
    "github.com/gin-gonic/gin"
)

type Handler struct {
    service Service
}

func NewHandler(service Service) *Handler {
    return &Handler{service: service}
}

func (h *Handler) CreateBlog(c *gin.Context) {
    var req struct {
        Title   string `json:"title"`
        Content string `json:"content"`
    }

    if err := c.ShouldBindJSON(&req); err != nil {
        response.BadRequest(c, err.Error())
        return
    }

    blog, err := h.service.CreateBlog(req.Title, req.Content)
    if err != nil {
        response.InternalError(c, err.Error())
        return
    }

    response.Created(c, blog)
}
```

### Step 5: Routes 등록

```go
// api/routes/routes.go 에 추가

import "gin_starter/internal/domain/blog"

func setupBlogRoutes(rg *gin.RouterGroup, db *database.DB) {
    repo := blog.NewRepository(db)
    service := blog.NewService(repo)
    handler := blog.NewHandler(service)

    blogGroup := rg.Group("/blog")
    {
        blogGroup.POST("", handler.CreateBlog)
    }
}

// SetupRoutes 함수에서 호출
setupBlogRoutes(api, db)
```

## 🔐 보안 기능

### JWT 토큰 시스템

1. **이중 암호화**
   - 페이로드를 AES-GCM으로 암호화
   - JWT로 서명

2. **토큰 종류**
   - Access Token: 30분 (API 인증용)
   - Refresh Token: 7일 (토큰 갱신용)

3. **자동 재사용**
   - Refresh Token이 24시간 이상 남으면 재사용

### 입력 검증

```go
rules := []validator.Rule{
    {Field: "email", Label: "이메일", Required: true, Pattern: validator.PatternEmail},
    {Field: "age", Label: "나이", Min: 18, Max: 100},
}

result := validator.Validate(c, rules)
if !result.Valid {
    response.ValidationError(c, result.GetErrorMap())
    return
}
```

## 🛠️ 개발 도구

### Makefile 추가 (선택)

```makefile
.PHONY: run build test clean swagger

run:
	go run cmd/server/main.go

build:
	go build -o bin/server cmd/server/main.go

test:
	go test ./...

clean:
	rm -rf bin/

swagger:
	swag init -g cmd/server/main.go
```

### Docker 지원 (선택)

```dockerfile
FROM golang:1.25.1-alpine AS builder
WORKDIR /app
COPY . .
RUN go mod download
RUN go build -o server cmd/server/main.go

FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/server .
COPY .env .
CMD ["./server"]
```

## 📊 코드 품질 가이드

### 1. 레이어 책임

- **Handler**: HTTP 요청/응답만 처리
- **Service**: 비즈니스 로직만 포함
- **Repository**: DB 접근만 담당

### 2. 에러 처리

```go
// ❌ 나쁜 예
if err != nil {
    return err
}

// ✅ 좋은 예
if err != nil {
    logger.Error("사용자 생성 실패: %v", err)
    return errors.Wrap(err, "USER_CREATE_FAILED", "사용자 생성에 실패했습니다")
}
```

### 3. 의존성 주입

```go
// ✅ 항상 인터페이스 사용
type Service interface {
    CreateUser() error
}

// ✅ 생성자에서 의존성 주입
func NewService(repo Repository) Service {
    return &service{repo: repo}
}
```

## 🧪 테스트 작성

```go
// internal/domain/user/service_test.go
package user_test

import (
    "testing"
    "gin_starter/internal/domain/user"
)

type mockRepository struct{}

func (m *mockRepository) Create(u *user.User) error {
    return nil
}

func TestRegister(t *testing.T) {
    repo := &mockRepository{}
    cfg := &config.Config{}
    service := user.NewService(repo, cfg)

    req := &user.CreateUserRequest{
        ID: "test",
        Password: "password",
        Name: "Test",
        Email: "test@test.com",
    }

    result, err := service.Register(req)
    if err != nil {
        t.Fatalf("Expected no error, got %v", err)
    }
    if result.ID != "test" {
        t.Errorf("Expected ID 'test', got '%s'", result.ID)
    }
}
```

## 📚 더 알아보기

- [Clean Architecture](https://blog.cleancoder.com/uncle-bob/2012/08/13/the-clean-architecture.html)
- [Gin 문서](https://gin-gonic.com/docs/)
- [Go 프로젝트 레이아웃](https://github.com/golang-standards/project-layout)

## 📄 라이선스

MIT License - 자유롭게 사용 가능합니다.