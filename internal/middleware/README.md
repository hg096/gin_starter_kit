# internal/middleware/

HTTP 요청/응답 처리 전후에 실행되는 **횡단 관심사(Cross-Cutting Concerns)**를 담당합니다.

## 🎯 미들웨어란?

미들웨어는 HTTP 요청이 핸들러에 도달하기 전/후에 실행되는 함수입니다.

```
Request → Middleware 1 → Middleware 2 → Handler → Middleware 2 → Middleware 1 → Response
```

## 📁 현재 미들웨어

```
middleware/
├── auth.go      # JWT 인증
├── logger.go    # 요청/응답 로깅
└── cors.go      # CORS 설정
```

---

## 🔐 auth.go - JWT 인증

### 기능
- JWT 토큰 생성 및 검증
- 사용자 인증
- 권한 확인

### 사용 예시

```go
// 인증 필요한 라우트
userGroup := rg.Group("/user")
userGroup.Use(middleware.AuthMiddleware(cfg))
{
    userGroup.GET("/profile", handler.GetProfile)
}

// 특정 사용자 타입 요구
adminGroup := rg.Group("/admin")
adminGroup.Use(middleware.AuthMiddleware(cfg))
adminGroup.Use(middleware.RequireUserType("A")) // Admin only
{
    adminGroup.GET("/dashboard", handler.Dashboard)
}

// 최소 권한 레벨 요구
vipGroup := rg.Group("/vip")
vipGroup.Use(middleware.AuthMiddleware(cfg))
vipGroup.Use(middleware.RequireAuthLevel(5)) // Level 5 이상
{
    vipGroup.GET("/content", handler.VIPContent)
}
```

### Handler에서 사용자 정보 가져오기

```go
func (h *Handler) GetProfile(c *gin.Context) {
    // 미들웨어가 설정한 값 가져오기
    userID, exists := c.Get("user_id")
    if !exists {
        response.Unauthorized(c, "인증 정보가 없습니다")
        return
    }

    // 사용자 정보로 처리
    user, err := h.service.GetUser(userID.(string))
    // ...
}
```

---

## 📝 logger.go - 로깅

### 기능
- 모든 HTTP 요청/응답 로깅
- 응답 시간 측정
- 클라이언트 IP 기록

### 사용 예시

```go
// 전역 적용
r := gin.New()
r.Use(middleware.LoggerMiddleware())
```

### 로그 출력 예시

```
2025-09-30 15:04:05 [INFO] Request processed {
    "status": 200,
    "latency": "12.5ms",
    "ip": "127.0.0.1",
    "method": "GET",
    "path": "/api/user/profile"
}
```

---

## 🌐 cors.go - CORS

### 기능
- Cross-Origin Resource Sharing 설정
- 프론트엔드 통신 허용

### 사용 예시

```go
r := gin.New()
r.Use(middleware.CORSMiddleware())
```

### 커스터마이징

특정 도메인만 허용:

```go
// middleware/cors.go 수정

func CORSMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        origin := c.GetHeader("Origin")

        // 허용된 도메인 확인
        allowedOrigins := []string{
            "https://example.com",
            "https://app.example.com",
        }

        for _, allowed := range allowedOrigins {
            if origin == allowed {
                c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
                break
            }
        }

        // 나머지 코드...
    }
}
```

---

## 🚀 새 미들웨어 추가 가이드

### 1. 파일 생성

```bash
cd internal/middleware
touch ratelimit.go
```

### 2. 미들웨어 작성

```go
// internal/middleware/ratelimit.go
package middleware

import (
    "gin_starter/pkg/response"
    "sync"
    "time"

    "github.com/gin-gonic/gin"
)

// Rate Limiter 구조체
type rateLimiter struct {
    requests map[string][]time.Time
    mu       sync.Mutex
    limit    int           // 최대 요청 수
    window   time.Duration // 시간 윈도우
}

var limiter *rateLimiter

func init() {
    limiter = &rateLimiter{
        requests: make(map[string][]time.Time),
        limit:    100,                 // 100 requests
        window:   time.Minute,         // per minute
    }

    // 주기적으로 오래된 데이터 정리
    go limiter.cleanup()
}

// RateLimitMiddleware Rate Limiting 미들웨어
func RateLimitMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        ip := c.ClientIP()

        if !limiter.allow(ip) {
            response.Error(c, 429, "TOO_MANY_REQUESTS",
                "요청 횟수 제한을 초과했습니다")
            c.Abort()
            return
        }

        c.Next()
    }
}

// allow 요청 허용 여부 확인
func (rl *rateLimiter) allow(key string) bool {
    rl.mu.Lock()
    defer rl.mu.Unlock()

    now := time.Now()
    windowStart := now.Add(-rl.window)

    // 윈도우 내의 요청만 필터링
    var validRequests []time.Time
    for _, reqTime := range rl.requests[key] {
        if reqTime.After(windowStart) {
            validRequests = append(validRequests, reqTime)
        }
    }

    // 제한 확인
    if len(validRequests) >= rl.limit {
        return false
    }

    // 현재 요청 추가
    validRequests = append(validRequests, now)
    rl.requests[key] = validRequests

    return true
}

// cleanup 주기적으로 오래된 데이터 정리
func (rl *rateLimiter) cleanup() {
    ticker := time.NewTicker(time.Minute * 5)
    defer ticker.Stop()

    for range ticker.C {
        rl.mu.Lock()
        now := time.Now()
        windowStart := now.Add(-rl.window)

        for key, requests := range rl.requests {
            var validRequests []time.Time
            for _, reqTime := range requests {
                if reqTime.After(windowStart) {
                    validRequests = append(validRequests, reqTime)
                }
            }

            if len(validRequests) == 0 {
                delete(rl.requests, key)
            } else {
                rl.requests[key] = validRequests
            }
        }
        rl.mu.Unlock()
    }
}
```

### 3. 사용

```go
// api/routes/routes.go

r.Use(middleware.RateLimitMiddleware())

// 또는 특정 라우트에만
apiGroup := r.Group("/api")
apiGroup.Use(middleware.RateLimitMiddleware())
```

---

## 📚 다양한 미들웨어 예시

### Request ID 추가

```go
// middleware/requestid.go
package middleware

import (
    "github.com/gin-gonic/gin"
    "github.com/google/uuid"
)

func RequestIDMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        requestID := c.GetHeader("X-Request-ID")
        if requestID == "" {
            requestID = uuid.New().String()
        }

        c.Set("request_id", requestID)
        c.Writer.Header().Set("X-Request-ID", requestID)

        c.Next()
    }
}
```

### Timeout 설정

```go
// middleware/timeout.go
package middleware

import (
    "context"
    "gin_starter/pkg/response"
    "time"

    "github.com/gin-gonic/gin"
)

func TimeoutMiddleware(timeout time.Duration) gin.HandlerFunc {
    return func(c *gin.Context) {
        ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
        defer cancel()

        c.Request = c.Request.WithContext(ctx)

        done := make(chan struct{})
        go func() {
            c.Next()
            close(done)
        }()

        select {
        case <-done:
            // 정상 완료
        case <-ctx.Done():
            // 타임아웃
            response.Error(c, 408, "REQUEST_TIMEOUT",
                "요청 처리 시간이 초과되었습니다")
            c.Abort()
        }
    }
}

// 사용
r.Use(middleware.TimeoutMiddleware(30 * time.Second))
```

### IP 화이트리스트

```go
// middleware/whitelist.go
package middleware

import (
    "gin_starter/pkg/response"

    "github.com/gin-gonic/gin"
)

func IPWhitelistMiddleware(allowedIPs []string) gin.HandlerFunc {
    ipMap := make(map[string]bool)
    for _, ip := range allowedIPs {
        ipMap[ip] = true
    }

    return func(c *gin.Context) {
        clientIP := c.ClientIP()

        if !ipMap[clientIP] {
            response.Forbidden(c, "허용되지 않은 IP입니다")
            c.Abort()
            return
        }

        c.Next()
    }
}

// 사용
adminGroup := r.Group("/admin")
adminGroup.Use(middleware.IPWhitelistMiddleware([]string{
    "127.0.0.1",
    "192.168.1.100",
}))
```

### Recovery (Panic 복구)

```go
// middleware/recovery.go
package middleware

import (
    "gin_starter/pkg/logger"
    "gin_starter/pkg/response"

    "github.com/gin-gonic/gin"
)

func RecoveryMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        defer func() {
            if err := recover(); err != nil {
                logger.Error("Panic 발생: %v", err)

                response.InternalError(c, "서버 내부 오류가 발생했습니다")
                c.Abort()
            }
        }()

        c.Next()
    }
}
```

---

## 🔄 미들웨어 실행 순서

순서가 **매우 중요**합니다!

```go
r := gin.New()

// 1. Recovery (가장 먼저 - 모든 Panic 잡기)
r.Use(middleware.RecoveryMiddleware())

// 2. Logger (요청/응답 로깅)
r.Use(middleware.LoggerMiddleware())

// 3. CORS (프론트엔드 통신)
r.Use(middleware.CORSMiddleware())

// 4. Request ID (요청 추적)
r.Use(middleware.RequestIDMiddleware())

// 5. Rate Limit (요청 제한)
r.Use(middleware.RateLimitMiddleware())

// 이후 라우트별 미들웨어
// - AuthMiddleware
// - RequireUserType
// - RequireAuthLevel
```

**실행 흐름:**
```
Request
  ↓
Recovery (defer)
  ↓
Logger (시작)
  ↓
CORS
  ↓
Request ID
  ↓
Rate Limit
  ↓
Auth (라우트별)
  ↓
Handler
  ↓
Rate Limit (끝)
  ↓
Request ID (끝)
  ↓
CORS (끝)
  ↓
Logger (끝, 시간 측정)
  ↓
Recovery (끝)
  ↓
Response
```

---

## ✅ 체크리스트

새 미들웨어 작성 시:

- [ ] `gin.HandlerFunc` 타입 반환
- [ ] 필요 시 `c.Next()` 호출
- [ ] 에러 발생 시 `c.Abort()` 호출
- [ ] Context에 값 저장 시 명확한 키 사용
- [ ] 성능 영향 고려 (모든 요청에 실행됨)
- [ ] Thread-safe 고려 (동시 요청)
- [ ] 테스트 코드 작성
- [ ] README 업데이트

---

## 💡 팁

### 조건부 미들웨어

```go
func ConditionalMiddleware(condition bool, middleware gin.HandlerFunc) gin.HandlerFunc {
    return func(c *gin.Context) {
        if condition {
            middleware(c)
        } else {
            c.Next()
        }
    }
}

// 사용
r.Use(ConditionalMiddleware(
    config.Get().App.IsDevelopment(),
    debugMiddleware,
))
```

### 미들웨어 체이닝

```go
func ChainMiddlewares(middlewares ...gin.HandlerFunc) gin.HandlerFunc {
    return func(c *gin.Context) {
        for _, m := range middlewares {
            m(c)
            if c.IsAborted() {
                return
            }
        }
    }
}
```

---

## 📚 참고

- [Gin Middleware 문서](https://gin-gonic.com/docs/examples/custom-middleware/)
- [HTTP Middleware 패턴](https://en.wikipedia.org/wiki/Middleware)