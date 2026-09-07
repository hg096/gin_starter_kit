# internal/api/routes

HTTP route 연결만 담당합니다. 비즈니스 로직은 `internal/domain/*`의 기능 파일에 둡니다.

## 구조

```text
routes.go
  SetupRoutes
  setupUserRoutes
  setupBlogRoutes
  setupAdminPageRoutes
  setupAdminRoutes
  health handlers
```

## 전역 흐름

```text
gin.Engine
  -> CORS
  -> request logger
  -> recovery
  -> swagger
  -> health
  -> admin page routes
  -> /api routes
```

## 주요 라우트

Health:

```text
GET /health/live
GET /health/ready
GET /health
```

User API:

```text
POST /api/user/register
POST /api/user/login
POST /api/user/refresh
GET  /api/user/profile
PUT  /api/user/profile
POST /api/user/logout
```

Blog API:

```text
GET    /api/blog
GET    /api/blog/:id
GET    /api/blog/author/:author_id
POST   /api/blog
PUT    /api/blog/:id
DELETE /api/blog/:id
```

Admin page:

```text
GET /admin/login
GET /admin
GET /admin/dashboard
GET /admin/users
GET /admin/blogs
GET /admin/chat
GET /admin/p/:page_key
GET /admin/logout
```

Admin API:

```text
/api/admin/bootstrap/*
/api/admin/stats
/api/admin/blogs
/api/admin/users
/api/admin/permissions
/api/admin/level-policy
/api/admin/audit-logs
/api/admin/pages
```

## 작성 기준

- route 파일은 dependency wiring과 middleware 적용만 담당합니다.
- 요청 파싱, 검증, 업무 흐름, 응답 생성은 domain handler 중심으로 처리합니다.
- service/repository가 있는 기존 기능은 그대로 연결만 합니다.
- service/repository는 반복 흐름 공유나 데이터 일관성 때문에 필요할 때만 둡니다.
- DB helper나 repository를 route에서 직접 호출하지 않습니다.
- admin page route는 `adminPageViewRoutes`에 항목을 추가하면 route spec sync와 view route가 함께 반영됩니다.

## 미들웨어

API 인증:

```go
auth := group.Group("")
auth.Use(middleware.AuthMiddleware(cfg))
```

권한:

```go
adminGroup.Use(middleware.RequireUserTypes("TA", "A", "M", "G", "AG"))
adminGroup.GET("/stats", middleware.RequirePermission("admin.stats.read"), handler.GetStats)
```

Admin page:

```go
adminProtected.Use(middleware.AdminPageAuthMiddleware(cfg))
adminProtected.GET("/dashboard", middleware.AdminPagePermissionMiddleware("admin.stats.read"), pageHandler.DashboardPage)
adminProtected.GET("/p/:page_key", middleware.AdminPageDynamicPermissionMiddleware(), pageHandler.DynamicPage)
```

## 테스트

route나 middleware 연결을 바꾸면:

```powershell
go test ./internal/api/routes
```

인증/권한/admin page 영향이 있으면:

```powershell
go test ./internal/domain/admin ./pkg/middleware ./internal/websocket
go test ./...
```
