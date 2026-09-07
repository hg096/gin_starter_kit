# internal

Go `internal/` 패키지입니다. 애플리케이션 내부 코드만 두며, 외부 모듈에서는 import할 수 없습니다.

이 문서는 일부러 짧게 유지합니다. 사람과 AI가 적은 토큰으로 구조와 규칙을 빠르게 이해하는 것이 목적입니다.
사용자가 따로 다른 방향을 요청하지 않으면 AI는 이 문서를 기본 규칙으로 따른다.

## 구조

```text
internal/
  api/routes/     HTTP 라우트 연결만 담당
  config/         환경변수/설정 로딩
  domain/         비즈니스 기능
  websocket/      웹소켓/채팅 런타임
```

HTTP 미들웨어는 `pkg/middleware`에 있습니다.

## 설계 기준

각 도메인은 기능 중심 파일 구조를 우선합니다.
신규/수정 코드는 기본적으로 하나의 기능 파일에서 handler, 내부 업무 흐름, DB 호출 흐름이 한 번에 읽히게 작성합니다.
단순 래핑용 service/interface는 만들지 않습니다.

```text
internal/domain/{feature}/
  {feature}.go       엔티티, 생성자, 공통 도메인 타입
  {action}.go        하나의 업무 흐름: 요청 -> 검증 -> 처리 -> 응답
  repository.go      여러 기능이 공유하거나 데이터 일관성상 분리가 필요한 SQL만 선택적으로 둠
  *_test.go          변경된 동작 주변의 집중 테스트
```

예시:

```text
user/
  user.go
  register.go
  login.go
  profile.go
  token.go
  repository.go

admin/
  handler.go
  users.go
  permissions.go
  pages.go
  blogs.go
  audit.go
  bootstrap.go
  page_views.go
  permission_repository.go

websocket/
  handler.go
  chat_repository.go
  chat_helpers.go
  hub.go
  client.go
```

기계적인 `handler.go / service.go / model.go` 3분할로 되돌리지 않습니다.
service/repository 파일은 코드가 겹치거나, 여러 기능이 같은 순서를 반드시 거쳐야 데이터 문제가 생기지 않는 경우에만 둡니다.
현재 일부 기존 코드는 service/repository 분리가 남아 있을 수 있지만, 새 코드는 단순한 handler 중심 흐름을 우선합니다.
이미 handler 중심으로 정리된 영역은 service/interface 구조로 되돌리지 않습니다.

## 흐름

사용자에게 노출되는 기능은 위에서 아래로 한 번에 읽히게 유지합니다.

```text
route -> handler -> 입력 파싱 -> 검증 -> 내부 업무 로직 -> DB helper/SQL -> response
```

기능 파일 안의 권장 순서:

```text
DTO
handler 메서드
작은 handler helper
업무 흐름 helper
DB helper
```

목표는 한 파일을 열었을 때 여러 파일을 오가지 않고 하나의 흐름을 이해하는 것입니다.

## 의존성 규칙

```text
routes + pkg/middleware + websocket -> domain -> pkg/db
```

- `routes`는 handler 연결만 담당합니다. 비즈니스 로직을 넣지 않습니다.
- `handler`는 HTTP 입력 파싱, 검증 호출, 업무 흐름, 응답 반환을 한눈에 보이게 둡니다.
- handler 의존성 묶음은 `deps`로 이름 짓습니다. `service`라는 이름으로 단순 래핑 계층을 만들지 않습니다.
- `service`는 기본으로 만들지 않습니다. 단순히 handler에서 repository를 호출하는 래핑용 service/interface는 제거합니다.
- 여러 handler가 같은 업무 순서를 반드시 공유해야 하거나, 감사 로그/권한/트랜잭션 순서를 강제해야 할 때만 service helper를 둡니다.
- `repository`는 기본으로 만들지 않습니다. 여러 기능이 공유하는 SQL, row mapping, 데이터 일관성상 분리가 필요한 DB 작업만 둡니다.
- `domain`은 `pkg/*` 공통 helper를 사용할 수 있습니다.
- 이미 필요한 구조가 아니라면 domain 간 직접 의존은 피합니다.
- middleware에는 비즈니스 로직을 넣지 않습니다.
- websocket handler도 같은 기준을 따른다. 얇은 ChatService 래핑을 만들지 않고 `Handler -> ChatRepository/helper` 흐름을 우선합니다.

## 공통 Helper

새 로컬 코드를 만들기 전에 기존 공통 helper를 먼저 확인합니다.

```text
pkg/utils
  PaginationFromQuery(c, defaultLimit, maxLimit)
  NewPagination(page, limit, defaultLimit, maxLimit)
  HasText(value)
  TrimLower(value)
  NonEmptyStrings(...)
  UniqueStrings(values)
  TruncateText(value, maxRunes)

pkg/db/database
  Store: 일반 DB와 트랜잭션 DB가 공유하는 Query/Insert/Update/Delete/Count/Exists 표면
  Repository: Query/Insert/Update/Delete helper
  TxRepository: 트랜잭션 안에서도 Query/Insert/Update/Delete 같은 이름 사용
  Exec: 검증된 커스텀 INSERT/UPDATE/DELETE 전용
  ExecSchema: CREATE/ALTER 같은 스키마 보정 전용
  ExecUnsafe: 검증 없는 예외 실행. 사용 전 재검토 필수
  DB.WithTx(func(tx *sql.Tx) error)
```

공통화 기준:

- 같은 로직이 3곳 이상 반복됩니다.
- helper 이름이 원래 코드보다 의도를 더 잘 드러냅니다.
- 특정 도메인에 묶이지 않는 중립 로직입니다.

공통화하지 않는 기준:

- 한 곳에서만 사용됩니다.
- helper가 비즈니스 의미를 숨깁니다.
- 특정 기능에만 속한 로직입니다.

## 트랜잭션

트랜잭션은 업무 흐름이 보이는 위치에서 `db.WithTx`를 사용합니다.

```go
err := h.db.WithTx(func(tx *sql.Tx) error {
	q := database.NewRepository(h.db).Tx(tx)
	if _, err := q.Update("_table", updates, "id = ?", id); err != nil {
		return err
	}
	_, err := q.Insert("_audit_logs", auditData)
	return err
})
```

`BeginTx`, deferred rollback, commit을 반복 작성하지 않습니다.

도메인 repository가 필요한 경우에도 트랜잭션 안에서는 `Tx(tx)` view를 만든 뒤 같은 함수명을 사용합니다.

```go
userRepo := h.userRepo.Tx(tx)
if err := userRepo.Update(id, updates); err != nil {
	return err
}
```

새 코드는 `CreateTx`, `UpdateTx`, `DeleteTx` 같은 접미사 메서드를 만들지 않습니다.
트랜잭션 여부는 `Tx(tx)`에서만 드러내고, 실제 작업 이름은 `Create`, `Update`, `Delete`로 유지합니다.

트랜잭션 재시도는 실제로 필요한 런타임 영역에서만 둡니다.
예: `internal/websocket` 채팅 생성/메시지 저장은 일시적인 DB 연결 오류를 흡수하기 위해 repository helper에 재시도를 둘 수 있습니다.

## 문자열

의도가 있는 문자열 체크는 공통 helper를 사용합니다.

```go
if !utils.HasText(req.Name) { ... }
status := utils.TrimLower(req.Status)
```

정규화된 값을 변수에 저장할 때는 `strings.TrimSpace`를 그대로 사용합니다.

```go
name := strings.TrimSpace(req.Name)
```

## 스키마 변경

`ALTER TABLE ADD COLUMN/KEY`를 존재 확인 없이 반복 실행하지 않습니다.

권장 패턴:

```text
information_schema 확인 -> 없을 때만 ALTER 실행 -> 동시성 중복 에러는 fallback으로 무시
```

이유: 호출부가 duplicate-column 에러를 무시해도, 공통 repository logging은 실패 SQL을 `_a_error_logs`에 먼저 기록합니다.

## SQL 실행

기본 CRUD는 명시적인 helper를 사용합니다.

```go
repo.Insert("_table", data)
repo.Update("_table", data, "id = ?", id)
repo.Delete("_table", "id = ?", id)
```

트랜잭션 안에서는 `Tx` view를 만들어 같은 함수명을 사용합니다.

```go
err := r.base.WithTx(func(q *database.TxRepository) error {
	if _, err := q.Insert("_table", data); err != nil {
		return err
	}
	if _, err := q.Update("_table", updates, "id = ?", id); err != nil {
		return err
	}
	return nil
})
```

새 코드는 `InsertTx`, `UpdateTx`, `DeleteTx` 접미사를 만들거나 호출하지 않습니다.
트랜잭션 안에서는 `q.Insert`, `q.Update`, `q.Delete`를 사용합니다.

`Exec`는 공통 helper로 표현하기 어려운 커스텀 DML에만 사용합니다.

```go
r.base.Exec(`
	INSERT INTO _table (id, value)
	VALUES (?, ?)
	ON DUPLICATE KEY UPDATE value = VALUES(value)
`, id, value)
```

규칙:

- `Exec`: `INSERT`, `UPDATE`, `DELETE` 계열만 허용합니다.
- `UPDATE`/`DELETE`는 `WHERE`가 없으면 거부합니다.
- `ExecSchema`: `CREATE`, `ALTER` 계열만 허용합니다.
- `ExecUnsafe`: 정말 필요한 경우만 사용하고 호출부에서 이유가 드러나야 합니다.
- 여러 SQL 문을 한 번에 실행하지 않습니다.

## 테스트

가장 작은 관련 테스트를 먼저 실행하고, 공통 코드가 바뀌면 전체 테스트를 실행합니다.

```powershell
go test ./internal/domain/admin
go test ./internal/websocket
go test ./pkg/utils
go test ./...
```

테스트를 추가하기 좋은 대상:

- 트랜잭션 동작
- SQL/schema 방어 로직
- 권한/인증 판단
- 파싱/검증 helper
- 사용자 응답 변경

## AI 체크리스트

수정 전:

- 이 문서를 읽습니다.
- 수정 대상 기능 파일을 읽습니다.
- `rg`로 기존 helper와 유사 흐름을 검색합니다.
- worktree에 있는 사용자 변경사항을 보존합니다.

수정 중:

- 요청된 기능 범위 안에서만 수정합니다.
- 새 추상화보다 기존 패턴을 우선합니다.
- 수동 수정은 `apply_patch`를 사용합니다.
- 동작이 의도적으로 제거된 경우가 아니라면 테스트를 삭제하지 않습니다.

마무리 전:

- `gofmt`를 실행합니다.
- 관련 `go test`를 실행합니다.
- 실행하지 못한 테스트가 있으면 명시합니다.

## 피할 것

- 하나의 API 흐름을 너무 많은 파일로 분산
- 파일을 줄이기 위한 과도한 분리
- pagination 파싱 중복
- 트랜잭션 boilerplate 중복
- 존재 확인 없는 schema ALTER
- repository/middleware 안에 숨은 비즈니스 규칙
- 단순 CRUD마다 service/repository 파일을 기계적으로 추가
- 단순 repository 호출만 감싸는 service/interface 추가
- 이미 handler에서 읽히는 흐름을 다시 service로 숨기기
- 작은 수정에 섞인 광범위한 리팩터링
- 예상 가능한 duplicate/missing schema 체크를 런타임 에러 로그로 기록
