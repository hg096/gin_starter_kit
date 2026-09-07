# pkg/db

DB 연결과 공통 repository helper를 둡니다. 도메인 업무 규칙은 이 패키지에 넣지 않습니다.

## 구조

```text
pkg/db/
  dbConn.go
  database/
    mysql.go        DB 연결, pool, health check, WithTx
    repository.go   공통 Query/Insert/Update/Delete/Exec helper
```

## DB 연결

```go
db, err := database.Connect(cfg)
if err != nil {
	logger.Fatal("DB 연결 실패: %v", err)
}
defer db.Close()
```

연결 상태 확인:

```go
if err := db.HealthCheck(); err != nil {
	return err
}
```

## 트랜잭션

수동 `BeginTx`, `defer Rollback`, `Commit` 반복 대신 `WithTx`를 사용합니다.

```go
err := db.WithTx(func(tx *sql.Tx) error {
	repo := database.NewRepository(db).Tx(tx)
	if _, err := repo.Insert("_table", data); err != nil {
		return err
	}
	_, err := repo.Update("_table", updates, "id = ?", id)
	return err
})
```

공통 repository를 이미 갖고 있다면:

```go
repo := database.NewRepository(db)

err := repo.WithTx(func(q *database.TxRepository) error {
	if _, err := q.Delete("_table", "id = ?", id); err != nil {
		return err
	}
	return nil
})
```

새 코드에서 `InsertTx`, `UpdateTx`, `DeleteTx` 접미사 함수를 만들거나 호출하지 않습니다.
트랜잭션 안에서는 `Tx(tx)` view를 만든 뒤 `Insert`, `Update`, `Delete` 같은 이름을 그대로 사용합니다.

## CRUD

```go
repo := database.NewRepository(db)

id, err := repo.Insert("_table", map[string]interface{}{
	"name": "홍길동",
})

affected, err := repo.Update("_table", map[string]interface{}{
	"name": "김철수",
}, "id = ?", id)

affected, err = repo.Delete("_table", "id = ?", id)
exists, err := repo.Exists("_table", "id = ?", id)
count, err := repo.Count("_table", "status = ?", "active")
```

식별자는 내부에서 검증/quote 됩니다. 동적 table/column 이름은 사용자 입력을 그대로 넘기지 않습니다.

## Expr

정적 SQL 표현식이 필요할 때만 `database.Expr`를 사용합니다.

```go
_, err := repo.Update("_table", map[string]interface{}{
	"updated_at": database.Expr("CURRENT_TIMESTAMP"),
}, "id = ?", id)
```

사용자 입력을 `Expr`로 감싸지 않습니다.

## Exec 계열

기본 CRUD는 `Insert`, `Update`, `Delete`를 우선 사용합니다.

```go
_, err := repo.Exec(`
	INSERT INTO _table (id, value)
	VALUES (?, ?)
	ON DUPLICATE KEY UPDATE value = VALUES(value)
`, id, value)
```

규칙:

- `Exec`: helper로 표현하기 어려운 `INSERT`, `UPDATE`, `DELETE` DML 전용
- `ExecSchema`: `CREATE`, `ALTER` 같은 schema 보정 전용
- `ExecUnsafe`: 검증 우회가 필요한 예외 전용
- `UPDATE`, `DELETE`는 `WHERE`가 없으면 거부
- 여러 SQL 문을 한 번에 실행 금지

## Schema 보정

`ALTER TABLE ADD COLUMN/KEY`는 먼저 존재 여부를 확인합니다.

```text
information_schema 확인 -> 없을 때만 ALTER 실행 -> 동시성 중복 에러는 fallback 처리
```

이유: duplicate column/key가 예상 가능한 상황이어도 repository logger가 먼저 `_a_error_logs`에 실패 SQL을 기록하기 때문입니다.

## 테스트

```powershell
go test ./pkg/db/database
go test ./...
```
