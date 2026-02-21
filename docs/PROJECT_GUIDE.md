# Project Guide: One-Pass Learning for Beginners and AI

## Who This Guide Is For
- 이 문서는 Go/Gin 프로젝트가 처음인 초보자와, 저장소를 빠르게 파악해야 하는 AI 에이전트를 위한 빠른 학습 가이드입니다.
- 이 문서의 목표는 한 번 읽고 "어디를 봐야 하는지"와 "어떻게 안전하게 수정하는지"를 이해하게 만드는 것입니다.
- 코드 설명은 한국어 중심으로 작성하고, 명령어/식별자/경로는 영문 그대로 유지합니다.
- 깊은 구현 세부보다 구조, 흐름, 확장 포인트, 검증 방법에 집중합니다.

## 10-Minute Orientation
### 서비스가 무엇인지 (3줄)
- 이 프로젝트는 `gin` 기반 API 서버로, `user`, `blog`, `admin` 도메인을 제공합니다.
- 인증은 `Bearer token`과 `HttpOnly Cookie`를 병행하며, 관리자 페이지(`/admin`)도 서버측 인증을 거칩니다.
- REST API, Swagger 문서, WebSocket 채널, 기본 CI(`go vet`, `go test`)까지 포함한 실전형 스타터입니다.

### 첫 진입 파일 3개
- `cmd/server/main.go`
- `api/routes/routes.go`
- `internal/middleware/auth.go`

### 오늘 처음 보는 사람이 할 5단계
1. `cmd/server/main.go`에서 부팅 순서를 확인합니다.
2. `api/routes/routes.go`에서 엔드포인트와 미들웨어 연결을 확인합니다.
3. `internal/domain/user/handler.go -> service.go -> repository.go` 순서로 한 기능(login)을 추적합니다.
4. `internal/middleware/auth.go`에서 인증 우선순위(헤더/쿠키)와 권한 컨텍스트를 확인합니다.
5. `go test ./...`와 `go vet ./...`를 실행해 현재 상태를 검증합니다.

### 핵심 엔드포인트 스냅샷
- Health: `GET /health/live`, `GET /health/ready`, `GET /health`
- User: `POST /api/user/login`, `GET /api/user/profile`
- Admin: `GET /api/admin/stats`
- WebSocket: `GET /ws/chat`, `GET /api/ws/stats`

## Repository Map
### Top-level tree (depth 2~3)
```text
gin_starter/
|- cmd/
|  `- server/main.go
|- api/
|  `- routes/routes.go
|- internal/
|  |- config/
|  |- middleware/
|  |- domain/
|  |  |- user/
|  |  |- blog/
|  |  `- admin/
|  |- infrastructure/database/
|  `- websocket/
|- pkg/
|  |- response/
|  |- errors/
|  |- validator/
|  `- logger/
|- web/admin/templates/
|- docs/
|  |- docs.go
|  |- swagger.json
|  `- swagger.yaml
|- migrations/
|- .github/workflows/ci.yml
|- exenv.txt
`- README.md
```

### 각 디렉터리 책임 (1문장)
- `cmd/`: 실행 가능한 앱 진입점과 프로세스 생명주기를 담당합니다.
- `api/routes/`: 공개 라우팅과 도메인 핸들러/미들웨어 조합을 담당합니다.
- `internal/config/`: 환경 변수 로딩, 검증, 런타임 설정 객체를 담당합니다.
- `internal/middleware/`: 인증, CORS, 로깅 등 공통 요청 전처리를 담당합니다.
- `internal/domain/*`: 비즈니스 규칙(Handler/Service/Repository)을 도메인별로 분리합니다.
- `internal/infrastructure/database/`: SQL 실행 공통 유틸과 DB 연결 관리를 담당합니다.
- `internal/websocket/`: WebSocket hub/client/handler와 실시간 라우트를 담당합니다.
- `pkg/*`: 응답 포맷, 공통 에러, 검증, 로깅 등 재사용 유틸을 담당합니다.
- `web/admin/templates/`: 관리자 페이지 UI 템플릿을 담당합니다.
- `docs/`: Swagger 생성물과 프로젝트 문서를 담당합니다.

### 수정 시 먼저 확인할 파일 경로
- 서버 부팅/종료: `cmd/server/main.go`
- API 라우트: `api/routes/routes.go`
- 인증/권한: `internal/middleware/auth.go`
- CORS/Origin: `internal/middleware/cors.go`, `internal/config/config.go`, `internal/websocket/handler.go`
- 사용자 로그인/리프레시/로그아웃: `internal/domain/user/handler.go`, `internal/domain/user/service.go`
- 공통 응답/에러 매핑: `pkg/response/response.go`, `pkg/errors/errors.go`

## Runtime Boot Flow
### 무엇
- 서버는 설정 로딩부터 DB 연결, 라우팅 등록, WebSocket 등록, HTTP 서버 실행 순서로 부팅됩니다.

### 왜
- 부팅 순서를 명확히 유지해야 런타임 의존성(`cfg`, `db`, `hub`)이 안정적으로 주입됩니다.

### 어디서
- 전체 흐름은 `cmd/server/main.go`에 있습니다.

### 어떻게
```text
config.Load
  -> database.Connect
  -> routes.SetupRoutes
  -> websocket.SetupWebSocketRoutes
  -> http.Server (ListenAndServe + Graceful Shutdown)
```

## HTTP Request Lifecycle
### 무엇
- 요청은 `route -> middleware -> handler -> service -> repository -> response` 순서로 흐릅니다.

### 왜
- 계층 분리를 통해 HTTP 처리, 비즈니스 규칙, DB 접근, 응답 포맷 책임을 분리합니다.

### 어디서
- 라우트: `api/routes/routes.go`
- 핸들러: `internal/domain/*/handler.go`
- 서비스: `internal/domain/*/service.go`
- 저장소: `internal/domain/*/repository.go`, `internal/infrastructure/database/repository.go`
- 응답: `pkg/response/response.go`

### 어떻게 (예: `/api/user/profile`)
1. `api/routes/routes.go`에서 `auth.Use(middleware.AuthMiddleware(cfg))`로 보호 라우트 등록.
2. `AuthMiddleware`가 토큰 검증 후 `user_id`, `user_type`, `user_level`을 context에 저장.
3. `internal/domain/user/handler.go:GetProfile`가 context의 `user_id`를 읽음.
4. `internal/domain/user/service.go:GetProfile`가 repository를 호출.
5. handler가 `response.Success` 또는 `response.FromError`로 표준 응답 반환.

## Auth & Authorization Model
### 무엇
- 인증은 `Authorization: Bearer <token>` 우선, 없으면 `access_token` 쿠키 fallback을 사용합니다.

### 왜
- API 클라이언트(Postman/SDK)는 Bearer를, 브라우저 기반 관리자 UI는 HttpOnly 쿠키를 자연스럽게 사용하기 위함입니다.

### 어디서
- 핵심 로직: `internal/middleware/auth.go`
- 로그인/리프레시 쿠키 발급: `internal/domain/user/handler.go`
- 관리자 페이지 보호: `api/routes/routes.go` + `AdminPageAuthMiddleware`

### 어떻게
- 토큰 추출 우선순위: Header -> Cookie
- 인증 성공 시 context 키 세팅:
  - `user_id`
  - `user_type`
  - `user_level`
- 권한 체크:
  - API 관리자 보호: `RequireUserType("A")`
  - 관리자 페이지 보호: 실패 시 `302 /admin/login` 리다이렉트
- 로그아웃:
  - `POST /api/user/logout`는 저장된 refresh token 제거 + 인증 쿠키 만료 처리

### 현재 주요 인증 엔드포인트
- `POST /api/user/login`
- `POST /api/user/refresh`
- `POST /api/user/logout`
- `GET /api/user/profile`
- `PUT /api/user/profile`

## Domain Layer Structure
### 무엇
- 각 도메인은 `model`, `repository`, `service`, `handler` 조합으로 구성됩니다.

### 왜
- 변경 포인트를 계층별로 분리해 테스트와 유지보수를 쉽게 만듭니다.

### 어디서
- `internal/domain/user/*`
- `internal/domain/blog/*`
- `internal/domain/admin/*`

### 어떻게
- `model`: 엔티티/DTO 정의
- `repository`: DB 쿼리 및 데이터 매핑
- `service`: 비즈니스 규칙/정책
- `handler`: HTTP 입력 검증, 서비스 호출, 응답 변환

## Data Access & DB
### 무엇
- DB 접근은 도메인 repository가 `internal/infrastructure/database.Repository`를 활용하는 구조입니다.

### 왜
- SQL 실행 공통 로직을 재사용하고, 도메인별 쿼리 책임을 분리하기 위함입니다.

### 어디서
- 연결/헬스체크: `internal/infrastructure/database/mysql.go`
- 공통 SQL 도우미: `internal/infrastructure/database/repository.go`
- 도메인 SQL: `internal/domain/*/repository.go`
- 스키마 기준: `table.sql`, `migrations/*.sql`

### 어떻게
- 앱 시작 시 `database.Connect(cfg)`로 연결
- readiness는 `/health/ready`에서 `db.HealthCheck()` 결과로 결정
- 현재 마이그레이션 정책은 SQL 파일 수동 관리입니다.

## WebSocket Flow
### 무엇
- 인증된 사용자가 `/ws/chat`으로 연결해 room 단위 실시간 통신을 수행합니다.

### 왜
- API와 동일한 인증 컨텍스트를 재사용하면서 실시간 기능을 제공하기 위함입니다.

### 어디서
- 라우트 등록: `internal/websocket/handler.go:SetupWebSocketRoutes`
- 커넥션/메시지 처리: `internal/websocket/client.go`, `internal/websocket/hub.go`

### 어떻게
- WebSocket 라우트:
  - `GET /ws/chat?room_id=...`
  - `GET /api/ws/room/:room_id`
  - `GET /api/ws/stats`
- 인증: `AuthMiddleware(cfg)` 적용
- Origin 검증: `CheckOrigin`에서 `cfg.IsAllowedOrigin(origin)` 확인

## How to Add a New Feature
### 표준 구현 순서
1. `model` 작성 (`internal/domain/<feature>/model.go`)
2. `repository` 작성 (`internal/domain/<feature>/repository.go`)
3. `service` 작성 (`internal/domain/<feature>/service.go`)
4. `handler` 작성 (`internal/domain/<feature>/handler.go`)
5. 라우트 등록 (`api/routes/routes.go`)

### 무엇
- 위 순서는 이 프로젝트의 계층 분리 원칙을 깨지 않고 기능을 추가하는 기본 루틴입니다.

### 왜
- 구현 누락(검증/에러응답/권한/테스트)을 줄이고, 코드 리뷰 기준을 통일하기 위함입니다.

### 어디서
- 도메인 파일: `internal/domain/<feature>/*`
- 라우트 파일: `api/routes/routes.go`
- 공통 응답/검증: `pkg/response/response.go`, `pkg/validator/validator.go`

### 어떻게 (8개 체크리스트)
- [ ] model에 request/response 필드를 명확히 정의했는가
- [ ] repository가 SQL/매핑 책임만 가지는가
- [ ] service가 비즈니스 규칙과 에러 코드를 결정하는가
- [ ] handler가 입력 검증과 응답 변환만 수행하는가
- [ ] 필요한 인증/권한 미들웨어를 라우트에 적용했는가
- [ ] `response.FromError` 또는 표준 응답 함수로 일관되게 반환하는가
- [ ] Swagger 주석과 `swag init -g cmd/server/main.go` 재생성을 수행했는가
- [ ] `go test ./...`와 관련 테스트 케이스를 추가/통과했는가

## Testing & CI
### 무엇
- 로컬 검증은 Go 기본 명령으로, 원격 검증은 GitHub Actions CI로 수행합니다.

### 왜
- 작은 변경도 정적 검증(`vet`)과 회귀 테스트(`test`)를 자동화해 품질을 유지하기 위함입니다.

### 어디서
- 로컬: CLI
- CI 설정: `.github/workflows/ci.yml`

### 어떻게
- 로컬 기본 명령어:
  - `go test ./...`
  - `go vet ./...`
  - `go run cmd/server/main.go`
  - `swag init -g cmd/server/main.go`
- CI 트리거:
  - `pull_request`
  - `push` on `main`, `develop`

## AI Quick Context
### System Snapshot
- Language: Go (module: `gin_starter`)
- Framework: Gin (`github.com/gin-gonic/gin`)
- Entry point: `cmd/server/main.go`
- Main routing file: `api/routes/routes.go`

### Key Entry Points
- `cmd/server/main.go`
- `api/routes/routes.go`
- `internal/config/config.go`

### Safe Change Zones
- `docs/*`
- `README.md` (link-only updates)

### Risky Change Zones
- `internal/middleware/auth.go`
- `api/routes/routes.go`
- `internal/domain/*/service.go`

### Task Prompt Template
- Goal:
- In-scope paths:
- Out-of-scope paths:
- Verification commands:

### Verification Commands
- `go test ./...`
- `go vet ./...`

## Common Pitfalls
- `handler`에서 비즈니스 규칙까지 처리해 계층 책임이 섞이는 문제.
- 권한 보호 라우트에 `AuthMiddleware`만 적용하고 `RequireUserType`를 누락하는 문제.
- 쿠키 기반 브라우저 호출에서 `credentials: include`를 빼먹는 문제.
- 내부 에러를 직접 `err.Error()`로 노출해 보안 정책과 충돌하는 문제.
- Swagger 주석 수정 후 `swag init -g cmd/server/main.go` 재생성을 누락하는 문제.
- 변경 후 `go test ./...`와 `go vet ./...`를 생략하는 문제.

## Reading Order
1. `cmd/server/main.go` (부팅/종료 전체 맥락)
2. `api/routes/routes.go` (엔드포인트 및 미들웨어 조합)
3. `internal/middleware/auth.go` (인증/권한 핵심)
4. `internal/domain/user/*` (대표 도메인 흐름)
5. `internal/domain/blog/*`, `internal/domain/admin/*` (확장 패턴)
6. `internal/infrastructure/database/*` + `pkg/response/response.go` (공통 기반)
7. `internal/websocket/*` (실시간 채널)
8. `.github/workflows/ci.yml` (검증 파이프라인)
