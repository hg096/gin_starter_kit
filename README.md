# Gin Starter

Go/Gin 기반 웹 API 스타터입니다. 현재 구조의 기준 문서는 [internal/README.md](internal/README.md)입니다.

이 README는 프로젝트 진입점만 짧게 설명합니다. 코드 스타일과 도메인 작성 규칙은 `internal/README.md`, 공통 패키지 규칙은 `pkg/README.md`를 우선 확인합니다.

## 구조

```text
cmd/server/          애플리케이션 실행 진입점
internal/api/routes/ HTTP/API/admin page route 연결
internal/config/     환경변수 기반 설정
internal/domain/     user, blog, admin 기능
internal/websocket/  채팅/websocket 런타임
pkg/                 공통 패키지
docs/                Swagger 산출물
web/                 admin/public template/static
```

## 실행

```powershell
go mod tidy
go run ./cmd/server
```

빌드:

```powershell
go build -o bin/server.exe ./cmd/server
```

테스트:

```powershell
go test ./...
```

## 환경변수

`.env`는 `exenv.txt`를 기준으로 만듭니다.

필수:

```env
DB_NAME=gin_starter
JWT_SECRET=32-character-access-secret!!
JWT_REFRESH_SECRET=32-character-refresh-secret!
JWT_TOKEN_SECRET=32-character-token-secret!!!
JWT_ISSUER=GinStarter
JWT_AUDIENCE=gin-starter-client
JWT_SUBJECT=access
```

주요 선택값:

```env
PORT=8080
GIN_MODE=debug
DB_HOST=localhost
DB_PORT=3306
DB_USER=root
DB_PASS=
DB_MAX_OPEN_CONNS=25
DB_MAX_IDLE_CONNS=5
DB_CONN_MAX_LIFETIME=5
CORS_ALLOWED_ORIGINS=http://localhost:8080,http://localhost:5173
```

## API

서버 실행 후:

```text
GET /health/live
GET /health/ready
GET /health
GET /swagger/*any
```

주요 API:

```text
/api/user      회원가입, 로그인, 토큰 갱신, 프로필
/api/blog      게시글 목록/상세/작성자별 목록/작성/수정/삭제
/api/admin     관리자 bootstrap, 사용자, 권한, 블로그, 페이지, 감사 로그
/admin         관리자 HTML page
/ws            websocket/chat
```

## 개발 기준

- 기능 흐름은 가능하면 한 파일에서 위에서 아래로 읽히게 둡니다.
- 새 도메인은 무조건 `handler.go/service.go/model.go`로 쪼개지 않습니다.
- 단순 기능은 handler 중심으로 입력, 검증, 업무 흐름, DB 호출을 한 파일에서 읽히게 둡니다.
- service/repository 분리는 반복 흐름을 공유해야 하거나 데이터 일관성상 분리가 필요한 경우에만 사용합니다.
- 공통 입력 파싱/검증은 `pkg/validator`, `pkg/utils`를 먼저 확인합니다.
- 페이지네이션은 `utils.PaginationFromQuery` 또는 `utils.NewPagination`을 사용합니다.
- 트랜잭션은 `db.WithTx`를 사용합니다.
- repository를 둘 경우 트랜잭션 내부에서는 `repo.Tx(tx).Create/Update/Delete`처럼 같은 함수명을 사용합니다.
- 새 코드에서 `CreateTx`, `UpdateTx`, `DeleteTx` 접미사 메서드를 만들지 않습니다.
- SQL 기본 CRUD는 `Insert`, `Update`, `Delete` helper를 우선 사용합니다.
- 커스텀 DML은 `Exec`, schema 보정은 `ExecSchema`, 예외는 `ExecUnsafe`로 의도를 드러냅니다.

## 참고 문서

- [internal/README.md](internal/README.md)
- [pkg/README.md](pkg/README.md)
- [pkg/db/README.md](pkg/db/README.md)
- [pkg/middleware/README.md](pkg/middleware/README.md)
