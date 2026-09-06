# internal/middleware

HTTP 요청이 handler에 도달하기 전/후에 실행되는 공통 관문입니다.

이 문서는 짧게 유지합니다. AI와 개발자가 파일 역할을 빠르게 파악하는 것이 목적입니다.

## 구조

```text
middleware/
  auth.go           HTTP 인증 흐름
  token.go          JWT 생성/검증, AES-GCM 암복호화
  auth_snapshot.go  DB 기준 인증 snapshot 로딩/검증
  auth_context.go   gin.Context 인증 key/helper
  permission.go     권한 middleware
  cors.go           CORS
  logger.go         요청 로그
```

## 요청 흐름

```text
AuthMiddleware
  -> access token 추출
  -> token 검증
  -> DB snapshot 로딩
  -> token stale/status/level 검증
  -> auth context 저장
  -> handler
```

Admin page 흐름:

```text
AdminPageAuthMiddleware
  -> access token 검증
  -> 실패 시 refresh cookie로 access 재발급
  -> admin type 확인
  -> auth context 저장
  -> page handler
```

## Context Key

문자열 key를 직접 쓰지 말고 `auth_context.go` 상수를 우선 사용합니다.

```go
ContextUserID
ContextUserType
ContextUserLevel
ContextIsSuperAdmin
ContextUserPermissions
ContextUserPermissionsList
ContextLevelPolicyEnabled
```

## 권한

API 권한:

```go
middleware.RequirePermission("admin.stats.read")
middleware.RequireSuperAdmin()
middleware.RequireAuthLevel(5)
```

Admin page 권한:

```go
middleware.AdminPageAuthMiddleware(cfg)
middleware.AdminPagePermissionMiddleware("admin.page.dashboard.read")
middleware.AdminPageDynamicPermissionMiddleware()
```

권한 set 조회는 `permission.go`의 `hasPermission` helper를 사용합니다. API는 JSON 응답, page는 plain text/redirect 응답이므로 abort 방식은 섞지 않습니다.

## 수정 기준

- `auth.go`: HTTP middleware 흐름만 둡니다.
- `token.go`: token 생성/검증/암복호화만 둡니다.
- `auth_snapshot.go`: DB에서 현재 계정 상태를 읽고 token과 비교합니다.
- `auth_context.go`: context key와 context helper만 둡니다.
- `permission.go`: 권한 확인 middleware와 권한 helper만 둡니다.

## 주의

- token claim 검증은 보수적으로 유지합니다. issuer/audience/subject 비교를 느슨하게 바꾸지 않습니다.
- middleware에는 비즈니스 처리 로직을 넣지 않습니다.
- DB 조회가 커지면 domain/service로 옮길지 먼저 검토합니다.
- `strings.TrimSpace(x) != ""` 반복은 `utils.HasText(x)`를 우선 검토합니다.
- `strings.ToLower(strings.TrimSpace(x))` 반복은 `utils.TrimLower(x)`를 우선 검토합니다.

## 테스트

미들웨어 변경 후 최소 실행:

```powershell
go test ./internal/middleware
```

인증/라우트 영향이 있으면 추가 실행:

```powershell
go test ./internal/api/routes ./internal/domain/admin ./internal/websocket
go test ./...
```
