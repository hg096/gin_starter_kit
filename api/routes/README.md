# api/routes/

**HTTP 라우트를 정의하고 등록**하는 패키지입니다. 모든 API 엔드포인트가 여기서 구성됩니다.

## 🎯 역할

- API 엔드포인트 정의
- 미들웨어 적용
- 도메인별 라우트 그룹화
- 의존성 주입 (DI) 수행

## 📁 파일 구조

```
routes/
└── routes.go    # 라우트 정의 및 등록
```

---

## 🗺️ 현재 라우트 구조

```
/
├── /health              # Health Check
│
├── /api/                # API 그룹
│   └── /user/           # User 도메인
│       ├── POST   /register         # 회원가입
│       ├── POST   /login            # 로그인
│       ├── POST   /refresh          # 토큰 갱신
│       ├── GET    /profile          # 프로필 조회 (인증 필요)
│       ├── PUT    /profile          # 프로필 수정 (인증 필요)
│       └── POST   /logout           # 로그아웃 (인증 필요)
│
└── /swagger/*any        # Swagger 문서
```

---

## 📖 라우트 등록 패턴

### 기본 구조

```go
func SetupRoutes(r *gin.Engine, db *database.DB, cfg *config.Config) {
    // 1. 전역 미들웨어
    r.Use(middleware.LoggerMiddleware())
    r.Use(middleware.CORSMiddleware())

    // 2. Health Check
    r.GET("/health", healthCheck)

    // 3. API 그룹
    api := r.Group("/api")
    {
        setupUserRoutes(api, db, cfg)
        setupBlogRoutes(api, db, cfg)
        // 다른 도메인 추가...
    }

    // 4. Swagger
    r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
}
```

### 도메인별 라우트 함수

```go
func setupUserRoutes(rg *gin.RouterGroup, db *database.DB, cfg *config.Config) {
    // 의존성 주입
    repo := user.NewRepository(db)
    service := user.NewService(repo, cfg)
    handler := user.NewHandler(service)

    // 라우트 그룹
    userGroup := rg.Group("/user")
    {
        // 인증 불필요
        userGroup.POST("/register", handler.Register)
        userGroup.POST("/login", handler.Login)

        // 인증 필요
        authGroup := userGroup.Group("")
        authGroup.Use(middleware.AuthMiddleware(cfg))
        {
            authGroup.GET("/profile", handler.GetProfile)
            authGroup.PUT("/profile", handler.UpdateProfile)
            authGroup.POST("/logout", handler.Logout)
        }
    }
}
```

---

## 🚀 새 라우트 추가하기

### 예시 1: Blog 도메인 추가

#### Step 1: 라우트 함수 작성

```go
// api/routes/routes.go에 추가

func setupBlogRoutes(rg *gin.RouterGroup, db *database.DB, cfg *config.Config) {
    // 의존성 주입
    repo := blog.NewRepository(db)
    service := blog.NewService(repo)
    handler := blog.NewHandler(service)

    // 라우트 그룹
    blogGroup := rg.Group("/blog")
    {
        // 공개 API
        blogGroup.GET("", handler.List)           // 목록
        blogGroup.GET("/:id", handler.Get)        // 상세

        // 인증 필요 API
        authGroup := blogGroup.Group("")
        authGroup.Use(middleware.AuthMiddleware(cfg))
        {
            authGroup.POST("", handler.Create)          // 생성
            authGroup.PUT("/:id", handler.Update)       // 수정
            authGroup.DELETE("/:id", handler.Delete)    // 삭제
        }
    }
}
```

#### Step 2: SetupRoutes에서 호출

```go
func SetupRoutes(r *gin.Engine, db *database.DB, cfg *config.Config) {
    // ...

    api := r.Group("/api")
    {
        setupUserRoutes(api, db, cfg)
        setupBlogRoutes(api, db, cfg)  // 추가!
    }
}
```

---

### 예시 2: Admin 전용 라우트

```go
func setupAdminRoutes(rg *gin.RouterGroup, db *database.DB, cfg *config.Config) {
    // 의존성 주입
    userRepo := user.NewRepository(db)
    adminService := admin.NewService(userRepo)
    adminHandler := admin.NewHandler(adminService)

    // Admin 그룹 (인증 + 권한 체크)
    adminGroup := rg.Group("/admin")
    adminGroup.Use(middleware.AuthMiddleware(cfg))
    adminGroup.Use(middleware.RequireUserType("A"))  // Admin만
    {
        adminGroup.GET("/users", adminHandler.ListUsers)
        adminGroup.DELETE("/users/:id", adminHandler.DeleteUser)
        adminGroup.PUT("/users/:id/auth", adminHandler.UpdateUserAuth)
    }
}
```

---

### 예시 3: 권한 레벨별 라우트

```go
func setupVIPRoutes(rg *gin.RouterGroup, db *database.DB, cfg *config.Config) {
    repo := content.NewRepository(db)
    service := content.NewService(repo)
    handler := content.NewHandler(service)

    // VIP 전용 콘텐츠 (Level 5 이상)
    vipGroup := rg.Group("/vip")
    vipGroup.Use(middleware.AuthMiddleware(cfg))
    vipGroup.Use(middleware.RequireAuthLevel(5))
    {
        vipGroup.GET("/exclusive", handler.GetExclusiveContent)
        vipGroup.GET("/premium", handler.GetPremiumContent)
    }
}
```

---

### 예시 4: 파일 업로드 라우트

```go
func setupFileRoutes(rg *gin.RouterGroup, db *database.DB, cfg *config.Config) {
    repo := file.NewRepository(db)
    service := file.NewService(repo)
    handler := file.NewHandler(service)

    fileGroup := rg.Group("/file")
    fileGroup.Use(middleware.AuthMiddleware(cfg))
    {
        // 단일 파일
        fileGroup.POST("/upload", handler.Upload)

        // 다중 파일
        fileGroup.POST("/upload-multiple", handler.UploadMultiple)

        // 다운로드
        fileGroup.GET("/download/:id", handler.Download)

        // 삭제
        fileGroup.DELETE("/:id", handler.Delete)
    }
}
```

---

## 🎨 고급 라우트 패턴

### 1. 버전별 API

```go
func SetupRoutes(r *gin.Engine, db *database.DB, cfg *config.Config) {
    // v1 API
    v1 := r.Group("/api/v1")
    {
        setupUserRoutesV1(v1, db, cfg)
    }

    // v2 API (새 버전)
    v2 := r.Group("/api/v2")
    {
        setupUserRoutesV2(v2, db, cfg)
    }
}
```

### 2. Rate Limiting 적용

```go
func setupPublicAPIRoutes(rg *gin.RouterGroup, db *database.DB, cfg *config.Config) {
    // Rate Limiter 적용
    apiGroup := rg.Group("/public")
    apiGroup.Use(middleware.RateLimitMiddleware(100, time.Minute))  // 분당 100회
    {
        apiGroup.GET("/posts", handler.GetPosts)
        apiGroup.GET("/posts/:id", handler.GetPost)
    }
}
```

### 3. WebSocket 라우트

```go
func setupWebSocketRoutes(r *gin.Engine, db *database.DB, cfg *config.Config) {
    wsHandler := websocket.NewHandler()

    // WebSocket 엔드포인트
    r.GET("/ws/chat", wsHandler.HandleChat)
    r.GET("/ws/notifications", wsHandler.HandleNotifications)
}
```

### 4. Static 파일 서빙

```go
func SetupRoutes(r *gin.Engine, db *database.DB, cfg *config.Config) {
    // ...

    // Static 파일
    r.Static("/static", "./public")
    r.StaticFile("/favicon.ico", "./public/favicon.ico")

    // 업로드 파일
    r.Static("/uploads", "./uploads")
}
```

### 5. 조건부 미들웨어

```go
func setupDebugRoutes(rg *gin.RouterGroup, cfg *config.Config) {
    // 개발 환경에서만 활성화
    if cfg.Server.GinMode == "debug" {
        debugGroup := rg.Group("/debug")
        {
            debugGroup.GET("/pprof", gin.WrapH(http.HandlerFunc(pprof.Index)))
            debugGroup.GET("/config", func(c *gin.Context) {
                c.JSON(200, cfg)
            })
        }
    }
}
```

---

## 🔒 미들웨어 적용 전략

### 전역 미들웨어

모든 요청에 적용:

```go
r.Use(middleware.RecoveryMiddleware())   // Panic 복구
r.Use(middleware.LoggerMiddleware())     // 로깅
r.Use(middleware.CORSMiddleware())       // CORS
r.Use(middleware.RequestIDMiddleware())  // Request ID
```

### 그룹별 미들웨어

특정 그룹에만 적용:

```go
api := r.Group("/api")
api.Use(middleware.RateLimitMiddleware())  // API 그룹에만 Rate Limit
{
    // ...
}
```

### 라우트별 미들웨어

개별 라우트에만 적용:

```go
userGroup.POST("/sensitive",
    middleware.RequireAuthLevel(10),
    middleware.IPWhitelistMiddleware([]string{"127.0.0.1"}),
    handler.SensitiveAction,
)
```

---

## 📋 RESTful API 규칙

### HTTP 메서드 매핑

| 메서드 | 용도 | 예시 |
|--------|------|------|
| GET | 조회 | `GET /users` (목록), `GET /users/:id` (상세) |
| POST | 생성 | `POST /users` |
| PUT | 전체 수정 | `PUT /users/:id` |
| PATCH | 부분 수정 | `PATCH /users/:id` |
| DELETE | 삭제 | `DELETE /users/:id` |

### URL 구조

```
/api/{version}/{resource}/{id}/{sub-resource}
```

예시:
```
GET    /api/v1/users              # 사용자 목록
GET    /api/v1/users/123          # 사용자 상세
POST   /api/v1/users              # 사용자 생성
PUT    /api/v1/users/123          # 사용자 수정
DELETE /api/v1/users/123          # 사용자 삭제
GET    /api/v1/users/123/posts    # 사용자의 게시글 목록
```

---

## 🧪 라우트 테스트

```go
// routes_test.go
package routes_test

import (
    "net/http"
    "net/http/httptest"
    "testing"

    "gin_starter/api/routes"
    "gin_starter/internal/config"
    "gin_starter/internal/infrastructure/database"

    "github.com/gin-gonic/gin"
    "github.com/stretchr/testify/assert"
)

func setupTestRouter() *gin.Engine {
    gin.SetMode(gin.TestMode)
    cfg := config.Load()
    db, _ := database.Connect(cfg)

    r := gin.New()
    routes.SetupRoutes(r, db, cfg)
    return r
}

func TestHealthCheck(t *testing.T) {
    router := setupTestRouter()

    w := httptest.NewRecorder()
    req, _ := http.NewRequest("GET", "/health", nil)
    router.ServeHTTP(w, req)

    assert.Equal(t, 200, w.Code)
}

func TestUserRegister(t *testing.T) {
    router := setupTestRouter()

    w := httptest.NewRecorder()
    body := `{"user_id":"test","user_pass":"password","user_name":"Test","user_email":"test@test.com"}`
    req, _ := http.NewRequest("POST", "/api/user/register", bytes.NewBufferString(body))
    req.Header.Set("Content-Type", "application/json")
    router.ServeHTTP(w, req)

    assert.Equal(t, 201, w.Code)
}
```

---

## ✅ 체크리스트

새 라우트 추가 시:

- [ ] 도메인별 setup 함수 작성
- [ ] 의존성 주입 (Repository → Service → Handler)
- [ ] RESTful 규칙 준수
- [ ] 적절한 미들웨어 적용
- [ ] Swagger 주석 작성
- [ ] 테스트 코드 작성
- [ ] README 업데이트

---

## 💡 팁

### 1. 라우트 목록 출력

```go
func printRoutes(r *gin.Engine) {
    for _, route := range r.Routes() {
        logger.Info("%s %s", route.Method, route.Path)
    }
}

// main.go에서
printRoutes(r)
```

### 2. 라우트 그룹 재사용

```go
func applyAuthMiddleware(group *gin.RouterGroup, cfg *config.Config) *gin.RouterGroup {
    group.Use(middleware.AuthMiddleware(cfg))
    return group
}

// 사용
authGroup := applyAuthMiddleware(userGroup.Group(""), cfg)
```

### 3. 동적 라우트 등록

```go
type RouteConfig struct {
    Method  string
    Path    string
    Handler gin.HandlerFunc
}

func registerRoutes(group *gin.RouterGroup, routes []RouteConfig) {
    for _, route := range routes {
        group.Handle(route.Method, route.Path, route.Handler)
    }
}
```

---

## ⚠️ 주의사항

### DO ✅

- RESTful 규칙 준수
- 명확한 URL 네이밍
- 적절한 HTTP 메서드 사용
- 미들웨어는 순서대로 적용
- 인증/권한 체크는 미들웨어로

### DON'T ❌

- 라우트에 비즈니스 로직 작성 금지
- 중복 라우트 등록 금지
- 너무 깊은 URL 구조 피하기
- 동사형 URL 피하기 (예: `/api/getUser` ❌)

---

## 📚 참고

- [RESTful API 설계 가이드](https://restfulapi.net/)
- [Gin Routing](https://gin-gonic.com/docs/examples/routing/)
- [HTTP Status Codes](https://developer.mozilla.org/en-US/docs/Web/HTTP/Status)