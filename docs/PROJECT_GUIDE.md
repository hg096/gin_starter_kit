# Project Guide

초보자와 AI가 저장소를 빠르게 파악하기 위한 짧은 원패스 가이드입니다.

자세한 코드 스타일 기준은 [internal/README.md](../internal/README.md)를 우선합니다. 이 문서는 코드를 바꾸기 위한 기준이 아니라, 현재 코드 구조를 빠르게 읽기 위한 지도입니다.

## 10분 파악

먼저 볼 파일:

```text
cmd/server/main.go
internal/api/routes/routes.go
pkg/middleware/auth.go
internal/README.md
```

처음 보는 사람이 할 순서:

1. `cmd/server/main.go`에서 부팅 순서를 봅니다.
2. `internal/api/routes/routes.go`에서 API/admin page route와 middleware 연결을 봅니다.
3. `pkg/middleware/auth.go`, `pkg/middleware/permission.go`에서 인증/권한 흐름을 봅니다.
4. `internal/domain/user/login.go`처럼 기능 파일 하나를 위에서 아래로 읽습니다.
5. DB 호출은 기능 파일 또는 필요한 경우 `repository.go`에서 `pkg/db/database` helper를 어떻게 쓰는지 확인합니다.
6. 변경 후 가장 작은 관련 테스트부터 실행하고 필요하면 `go test ./...`를 실행합니다.

## Repository Map

```text
gin_starter/
  cmd/server/              실행 진입점
  internal/api/routes/     HTTP route 연결
  internal/config/         환경변수 설정
  internal/domain/         user, blog, admin 기능
  internal/websocket/      websocket/chat
  pkg/authz/               auth type/권한 판별
  pkg/db/database/         DB 연결과 공통 repository
  pkg/middleware/          인증, 권한, CORS, 요청 로그
  pkg/response/            표준 API 응답
  pkg/utils/               중립 공통 helper
  pkg/validator/           HTTP 입력 검증
  web/                     admin/public template/static
  docs/                    Swagger 산출물과 문서
```

현재 `internal/middleware`와 `internal/infrastructure` 디렉터리는 사용하지 않습니다. middleware는 `pkg/middleware`, DB 공통 코드는 `pkg/db/database`에 있습니다.

## Boot Flow

```text
config.Load
-> database.Connect
-> websocket.NewHub + hub.Run
-> gin.New
-> routes.SetupRoutes
-> websocket.SetupWebSocketRoutes
-> http.Server
-> graceful shutdown
```

진입점은 `cmd/server/main.go`입니다.

## Request Flow

일반 API:

```text
route -> middleware -> domain handler -> 입력 파싱/검증 -> 내부 업무 로직 -> DB helper/SQL -> response
```

Admin page:

```text
route -> AdminPageAuthMiddleware -> page permission middleware -> PageHandler -> template
```

WebSocket:

```text
route -> AuthMiddleware -> websocket handler -> hub/chat logic/DB
```

## Domain Style

도메인은 기능 중심 파일 구조를 우선합니다.
신규/수정 코드는 단순한 handler 중심 흐름을 기본으로 합니다.

예:

```text
internal/domain/user/
  user.go
  register.go
  login.go
  profile.go
  token.go
  repository.go

internal/domain/blog/
  blog.go
  posts.go
  repository.go

internal/domain/admin/
  handler.go
  users.go
  permissions.go
  pages.go
  blogs.go
  audit.go
  bootstrap.go
  page_views.go
  permission_repository.go
```

새 기능을 만들 때 무조건 `handler.go`, `service.go`, `model.go`로 쪼개지 않습니다. 한 기능 흐름이 한 파일에서 위에서 아래로 읽히는 편이 우선입니다.
단순 래핑용 service/interface는 만들지 않고, handler 의존성 묶음은 `deps`로 둡니다. service/repository 분리는 여러 기능이 같은 순서를 반드시 공유해야 하거나, 분리하지 않으면 데이터 일관성 문제가 생길 때만 선택합니다.

기능 파일 권장 순서:

```text
DTO
handler
handler helper
업무 흐름 helper
DB helper
```

## Data Access

DB 호출은 `pkg/db/database.Repository`를 사용합니다.
단순 기능은 기능 파일 안에서 바로 사용해도 됩니다. 여러 기능이 공유하는 SQL/row mapping은 `repository.go`로 분리합니다.

기본 CRUD:

```go
repo := database.NewRepository(db)
id, err := repo.Insert("_table", data)
affected, err := repo.Update("_table", updates, "id = ?", id)
affected, err := repo.Delete("_table", "id = ?", id)
```

트랜잭션:

```go
err := db.WithTx(func(tx *sql.Tx) error {
	q := database.NewRepository(db).Tx(tx)
	if _, err := q.Update("_table", updates, "id = ?", id); err != nil {
		return err
	}
	_, err := q.Insert("_audit_logs", auditData)
	return err
})
```

규칙:

- handler/업무 흐름에서 수동 `BeginTx`, `Rollback`, `Commit`을 반복하지 않습니다.
- 새 도메인 코드는 `CreateTx`, `UpdateTx`, `DeleteTx` 접미사 메서드를 만들거나 호출하지 않습니다.
- 트랜잭션 여부는 `Tx(tx)`에서만 드러내고 작업 이름은 `Create`, `Update`, `Delete`로 유지합니다.
- 기본 CRUD는 `Insert`, `Update`, `Delete` helper를 우선 사용합니다.
- helper로 표현하기 어려운 DML은 `Exec`를 사용합니다.
- schema 보정은 `ExecSchema`를 사용합니다.
- 검증 우회가 필요한 예외만 `ExecUnsafe`를 사용합니다.

## Common Helpers

먼저 확인할 helper:

```text
pkg/validator
  Validate(c, rules)

pkg/utils
  PaginationFromQuery
  NewPagination
  HasText
  TrimLower
  UniqueStrings
  StringToNumeric
  GetBindField
  GetContextVal

pkg/response
  Success
  Created
  BadRequest
  ValidationError
  FromError
```

반복 로직이 있어도 도메인 정책이면 `pkg/utils`로 빼지 않습니다. 예를 들어 admin page group, user 가입 정책, blog 권한 정책은 domain 안에 둡니다.

## Auth & Permission

인증:

```text
Authorization: Bearer token
-> access_token cookie fallback
-> token 검증
-> DB snapshot 검증
-> gin.Context 저장
```

주요 context:

```text
user_id
user_type
user_level
is_super_admin
user_permissions
level_policy_enabled
```

권한:

```go
middleware.RequirePermission("admin.stats.read")
middleware.RequireSuperAdmin()
middleware.RequireUserTypes("TA", "A", "M", "G", "AG")
middleware.AdminPagePermissionMiddleware("admin.page.dashboard.read")
middleware.AdminPageDynamicPermissionMiddleware()
```

## Routes Snapshot

Health:

```text
GET /health/live
GET /health/ready
GET /health
```

User:

```text
POST /api/user/register
POST /api/user/login
POST /api/user/refresh
GET  /api/user/profile
PUT  /api/user/profile
POST /api/user/logout
```

Blog:

```text
GET    /api/blog
GET    /api/blog/:id
GET    /api/blog/author/:author_id
POST   /api/blog
PUT    /api/blog/:id
DELETE /api/blog/:id
```

Admin:

```text
GET  /api/admin/bootstrap/status
POST /api/admin/bootstrap/super-admin
GET  /api/admin/stats
GET  /api/admin/users
GET  /api/admin/permissions
GET  /api/admin/level-policy
GET  /api/admin/audit-logs
GET  /api/admin/pages
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

WebSocket/chat:

```text
GET  /ws/chat
GET  /ws/admin-chat
GET  /ws/admin-chat-notify
GET  /api/ws/room/:room_id
GET  /api/ws/stats
GET  /api/admin/chat/rooms
GET  /api/admin/chat/users
POST /api/admin/chat/rooms/group
POST /api/admin/chat/rooms/direct/:target_id
GET  /api/admin/chat/rooms/:room_key/messages
POST /api/admin/chat/rooms/:room_key/messages
POST /api/admin/chat/rooms/:room_key/read
```

## Add Feature Checklist

- 기능 파일에서 요청 -> 검증 -> 처리 -> 응답 흐름이 읽히는가?
- `pkg/validator`, `pkg/utils`, `pkg/response` 기존 helper를 먼저 확인했는가?
- route 파일에는 연결과 middleware만 두었는가?
- 단순 기능은 handler 중심으로 업무 흐름과 DB 호출까지 읽히는가?
- service/repository 분리는 반복 공유나 데이터 일관성 때문에 필요한 경우인가?
- 트랜잭션은 `WithTx`와 필요 시 `repo.Tx(tx)` 흐름을 쓰는가?
- 새 `*Tx` 접미사 메서드를 만들지 않았는가?
- 관련 테스트와 `go test ./...`를 실행했는가?

## Verification

자주 쓰는 명령:

```powershell
go test ./internal/domain/admin
go test ./internal/api/routes
go test ./pkg/db/database
go test ./pkg/middleware
go test ./...
```

Swagger를 수정했다면:

```powershell
swag init -g cmd/server/main.go
```

## Common Pitfalls

- README만 보고 현재 코드와 다른 구조를 새로 만드는 것
- 기능 흐름을 필요 이상으로 여러 파일에 분산하는 것
- route에 비즈니스 로직을 넣는 것
- middleware에 도메인 업무 처리를 넣는 것
- pagination 파싱을 직접 반복하는 것
- 수동 transaction boilerplate를 다시 쓰는 것
- 존재 확인 없는 `ALTER TABLE ADD COLUMN/KEY`로 `_a_error_logs`를 쌓는 것
- 내부 에러를 사용자 응답에 그대로 노출하는 것

## Reading Order

1. `internal/README.md`
2. `cmd/server/main.go`
3. `internal/api/routes/routes.go`
4. `pkg/middleware/auth.go`
5. `pkg/middleware/permission.go`
6. `internal/domain/user/login.go`
7. `internal/domain/blog/posts.go`
8. `internal/domain/admin/users.go`
9. `pkg/db/database/repository.go`
10. `internal/websocket/handler.go`
