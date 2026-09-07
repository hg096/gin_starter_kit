# pkg

외부 import가 가능한 공통 패키지 영역입니다. 특정 도메인의 업무 규칙은 `pkg`로 옮기지 않습니다.

## 구조

```text
pkg/
  authz/       인증 타입/권한 관련 공통 판별
  db/          DB 연결, 공통 repository
  errors/      애플리케이션 에러 타입
  logger/      로깅
  middleware/  Gin middleware
  renderer/    template rendering helper
  response/    표준 API 응답
  utils/       중립 유틸
  validator/   HTTP 입력 검증
```

## 추가 기준

`pkg`에 넣어도 되는 코드:

- 다른 도메인에서 2-3회 이상 반복됩니다.
- 비즈니스 의미가 없습니다.
- helper 이름이 호출부 의도를 더 잘 드러냅니다.
- `internal/domain/*`를 import하지 않습니다.

`pkg`에 넣지 않는 코드:

- 특정 도메인 정책입니다.
- 한 기능에서만 사용됩니다.
- 호출부 흐름을 숨깁니다.
- 편의를 위해 내부 도메인 타입을 끌어옵니다.

예:

```text
OK: utils.TrimLower, utils.HasText, utils.NewPagination
NO: admin page group 정책, user 가입 정책, blog 권한 정책
```

## 자주 쓰는 패키지

### response

API 응답은 `pkg/response`를 우선 사용합니다.

```go
response.Success(c, data)
response.Created(c, data)
response.BadRequest(c, "invalid request")
response.ValidationError(c, result.GetErrorMap())
response.FromError(c, err)
```

### validator

HTTP 입력 검증은 handler에서 `validator.Validate`를 사용합니다.

```go
result := validator.Validate(c, []validator.Rule{
	{Field: "user_id", Label: "아이디", Required: true, MinLen: 3, MaxLen: 20, Pattern: validator.PatternAlphaNum},
})
if !result.Valid {
	response.ValidationError(c, result.GetErrorMap())
	return
}
```

### utils

반복 파싱/문자열 처리는 기존 helper를 먼저 확인합니다.

```go
pagination := utils.PaginationFromQuery(c, 20, 100)
ok := utils.HasText(value)
status := utils.TrimLower(rawStatus)
items := utils.UniqueStrings(values)
id, err := utils.StringToNumeric[int64](raw)
```

### db/database

기본 CRUD는 명시 helper를 사용합니다.

```go
repo := database.NewRepository(db)
id, err := repo.Insert("_table", data)
affected, err := repo.Update("_table", updates, "id = ?", id)
affected, err := repo.Delete("_table", "id = ?", id)
```

트랜잭션은 `WithTx`와 `TxRepository` view를 사용합니다.

```go
err := repo.WithTx(func(q *database.TxRepository) error {
	if _, err := q.Insert("_table", data); err != nil {
		return err
	}
	_, err := q.Update("_table", updates, "id = ?", id)
	return err
})
```

새 코드에서 `InsertTx`, `UpdateTx`, `DeleteTx` 접미사 함수는 만들거나 호출하지 않습니다.

## SQL 실행 규칙

- `Insert`, `Update`, `Delete`를 우선 사용합니다.
- `Exec`는 helper로 표현하기 어려운 DML 전용입니다.
- `ExecSchema`는 `CREATE`, `ALTER` 같은 schema 보정 전용입니다.
- `ExecUnsafe`는 검증을 우회해야 하는 예외 상황에서만 사용합니다.
- `UPDATE`, `DELETE` DML은 `WHERE` 없는 실행을 거부합니다.
- 여러 SQL 문을 한 번에 실행하지 않습니다.

## 테스트

공통 패키지를 수정하면 관련 패키지 테스트 후 전체 테스트를 실행합니다.

```powershell
go test ./pkg/utils
go test ./pkg/db/database
go test ./pkg/middleware
go test ./...
```
