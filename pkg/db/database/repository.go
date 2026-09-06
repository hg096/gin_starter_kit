package database

import (
	"database/sql"
	"fmt"
	"gin_starter/pkg/errors"
	"gin_starter/pkg/logger"
	"regexp"
	"sort"
	"strings"
)

// Repository 공통 데이터베이스 리포지토리
type Repository struct {
	db *DB
}

// Expr is a trusted SQL expression used by repository helpers.
//
// Use it only for static, internal SQL snippets such as CURRENT_TIMESTAMP,
// NULL, or column arithmetic. Never wrap raw user input in Expr.
type Expr string

var identifierPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// TxRepository는 하나의 트랜잭션에 묶인 Repository view다.
//
// 새 트랜잭션 코드는 InsertTx/UpdateTx/DeleteTx보다 repo.Tx(tx).Insert(...)
// 형태를 우선 사용한다. 트랜잭션 여부와 관계없이 같은 함수명으로 흐름을 읽기 위함이다.
type TxRepository struct {
	repo *Repository
	tx   *sql.Tx
}

func sortedMapKeys(data map[string]interface{}) []string {
	keys := make([]string, 0, len(data))
	for col := range data {
		keys = append(keys, col)
	}
	sort.Strings(keys)
	return keys
}

func quoteIdentifier(name string) (string, error) {
	name = strings.TrimSpace(name)
	if !identifierPattern.MatchString(name) {
		return "", errors.New("INVALID_IDENTIFIER", "허용되지 않는 식별자입니다")
	}
	return fmt.Sprintf("`%s`", name), nil
}

func buildInsertParts(data map[string]interface{}) ([]string, []string, []interface{}, error) {
	columns := make([]string, 0, len(data))
	placeholders := make([]string, 0, len(data))
	values := make([]interface{}, 0, len(data))

	for _, col := range sortedMapKeys(data) {
		quotedCol, err := quoteIdentifier(col)
		if err != nil {
			return nil, nil, nil, err
		}

		val := data[col]
		columns = append(columns, quotedCol)
		if expr, ok := val.(Expr); ok {
			if err := validateExpr(expr); err != nil {
				return nil, nil, nil, err
			}
			placeholders = append(placeholders, string(expr))
			continue
		}
		placeholders = append(placeholders, "?")
		values = append(values, val)
	}

	return columns, placeholders, values, nil
}

func buildSetClauses(data map[string]interface{}) ([]string, []interface{}, error) {
	setClauses := make([]string, 0, len(data))
	values := make([]interface{}, 0, len(data))

	for _, col := range sortedMapKeys(data) {
		quotedCol, err := quoteIdentifier(col)
		if err != nil {
			return nil, nil, err
		}

		val := data[col]
		if expr, ok := val.(Expr); ok {
			if err := validateExpr(expr); err != nil {
				return nil, nil, err
			}
			setClauses = append(setClauses, fmt.Sprintf("%s = %s", quotedCol, string(expr)))
			continue
		}
		setClauses = append(setClauses, fmt.Sprintf("%s = ?", quotedCol))
		values = append(values, val)
	}

	return setClauses, values, nil
}

func validateExpr(expr Expr) error {
	value := strings.TrimSpace(string(expr))
	if value == "" {
		return errors.New("INVALID_EXPR", "빈 SQL 표현식은 사용할 수 없습니다")
	}
	if strings.Contains(value, ";") || strings.Contains(value, "--") || strings.Contains(value, "/*") || strings.Contains(value, "*/") {
		return errors.New("INVALID_EXPR", "안전하지 않은 SQL 표현식입니다")
	}
	return nil
}

// NewRepository Repository 생성자
func NewRepository(db *DB) *Repository {
	return &Repository{db: db}
}

// Tx 트랜잭션 전용 Repository view를 만든다.
func (r *Repository) Tx(tx *sql.Tx) *TxRepository {
	return &TxRepository{repo: r, tx: tx}
}

// WithTx는 트랜잭션 안에서 같은 Repository 함수명을 사용할 수 있게 해준다.
func (r *Repository) WithTx(fn func(q *TxRepository) error) error {
	if fn == nil {
		return errors.New("INVALID_PARAM", "트랜잭션 함수가 필요합니다")
	}
	return r.db.WithTx(func(tx *sql.Tx) error {
		return fn(r.Tx(tx))
	})
}

// QueryRow SELECT 단일 행 조회
func (r *Repository) QueryRow(query string, args ...interface{}) *sql.Row {
	logger.Debug("SQL Query: %s, Args: %v", query, args)
	return r.db.QueryRow(query, args...)
}

// QueryRow SELECT 단일 행 조회.
func (q *TxRepository) QueryRow(query string, args ...interface{}) *sql.Row {
	return q.repo.QueryRowTx(q.tx, query, args...)
}

// Query SELECT 다중 행 조회
func (r *Repository) Query(query string, args ...interface{}) (*sql.Rows, error) {
	logger.Debug("SQL Query: %s, Args: %v", query, args)
	rows, err := r.db.Query(query, args...)
	if err != nil {
		logger.Error("Query 실행 실패: %v", err)
		r.LogError("Repository.Query", err.Error(), fmt.Sprintf("%s | Args: %v", query, args))
		return nil, errors.Wrap(err, "DATABASE_ERROR", "쿼리 실행에 실패했습니다")
	}
	return rows, nil
}

// Query SELECT 다중 행 조회.
func (q *TxRepository) Query(query string, args ...interface{}) (*sql.Rows, error) {
	logger.Debug("SQL Query (TX): %s, Args: %v", query, args)
	rows, err := q.tx.Query(query, args...)
	if err != nil {
		logger.Error("QueryTx 실행 실패: %v", err)
		q.repo.LogError("Repository.QueryTx", err.Error(), fmt.Sprintf("%s | Args: %v", query, args))
		return nil, errors.Wrap(err, "DATABASE_ERROR", "트랜잭션 쿼리 실행에 실패했습니다")
	}
	return rows, nil
}

// Exec 검증된 커스텀 DML 실행.
//
// 기본 INSERT/UPDATE/DELETE는 Insert, Update, Delete를 우선 사용한다.
// 트랜잭션 안에서는 TxRepository의 같은 함수명을 사용한다.
// Exec는 ON DUPLICATE KEY UPDATE처럼 공통 helper로
// 표현하기 어려운 DML에만 사용한다. CREATE/ALTER 등 schema 작업은 ExecSchema를
// 사용하고, 정말 예외적인 SQL은 ExecUnsafe로 의도를 드러낸다.
func (r *Repository) Exec(query string, args ...interface{}) (sql.Result, error) {
	if err := validateDMLQuery(query); err != nil {
		return nil, err
	}
	return r.exec("Repository.Exec", query, args...)
}

// Exec 검증된 커스텀 DML을 트랜잭션 안에서 실행.
func (q *TxRepository) Exec(query string, args ...interface{}) (sql.Result, error) {
	if err := validateDMLQuery(query); err != nil {
		return nil, err
	}
	return q.repo.execTx(q.tx, "Repository.ExecTx", query, args...)
}

// ExecTx 검증된 커스텀 DML을 트랜잭션 안에서 실행.
//
// 기본 INSERT/UPDATE/DELETE는 r.Tx(tx).Insert/Update/Delete를 우선 사용한다.
// CREATE/ALTER 등 schema 작업은 r.Tx(tx).ExecSchema를 사용한다.
//
// Deprecated: 새 코드는 r.Tx(tx).Exec(...)를 사용한다.
func (r *Repository) ExecTx(tx *sql.Tx, query string, args ...interface{}) (sql.Result, error) {
	return r.Tx(tx).Exec(query, args...)
}

// ExecSchema CREATE/ALTER 같은 schema 보정 SQL 실행.
//
// 실행 전 존재 여부를 먼저 확인해서 예상 가능한 duplicate column/key 에러가
// _a_error_logs에 쌓이지 않게 한다.
func (r *Repository) ExecSchema(query string, args ...interface{}) (sql.Result, error) {
	if err := validateSchemaQuery(query); err != nil {
		return nil, err
	}
	return r.exec("Repository.ExecSchema", query, args...)
}

// ExecSchemaTx CREATE/ALTER 같은 schema 보정 SQL을 트랜잭션 안에서 실행.
//
// Deprecated: 새 코드는 r.Tx(tx).ExecSchema(...)를 사용한다.
func (r *Repository) ExecSchemaTx(tx *sql.Tx, query string, args ...interface{}) (sql.Result, error) {
	return r.Tx(tx).ExecSchema(query, args...)
}

// ExecSchema CREATE/ALTER 같은 schema 보정 SQL을 트랜잭션 안에서 실행.
func (q *TxRepository) ExecSchema(query string, args ...interface{}) (sql.Result, error) {
	if err := validateSchemaQuery(query); err != nil {
		return nil, err
	}
	return q.repo.execTx(q.tx, "Repository.ExecSchemaTx", query, args...)
}

// ExecUnsafe 검증 없이 SQL 실행. 사용 전 반드시 더 명시적인 helper를 먼저 검토한다.
func (r *Repository) ExecUnsafe(query string, args ...interface{}) (sql.Result, error) {
	if strings.TrimSpace(query) == "" {
		return nil, errors.New("INVALID_QUERY", "빈 쿼리는 실행할 수 없습니다")
	}
	return r.exec("Repository.ExecUnsafe", query, args...)
}

// ExecTxUnsafe 검증 없이 트랜잭션 SQL 실행. 사용 전 반드시 더 명시적인 helper를 먼저 검토한다.
//
// Deprecated: 새 코드는 r.Tx(tx).ExecUnsafe(...)를 사용한다.
func (r *Repository) ExecTxUnsafe(tx *sql.Tx, query string, args ...interface{}) (sql.Result, error) {
	return r.Tx(tx).ExecUnsafe(query, args...)
}

// ExecUnsafe 검증 없이 트랜잭션 SQL 실행. 사용 전 반드시 더 명시적인 helper를 먼저 검토한다.
func (q *TxRepository) ExecUnsafe(query string, args ...interface{}) (sql.Result, error) {
	if strings.TrimSpace(query) == "" {
		return nil, errors.New("INVALID_QUERY", "빈 쿼리는 실행할 수 없습니다")
	}
	return q.repo.execTx(q.tx, "Repository.ExecTxUnsafe", query, args...)
}

func (r *Repository) exec(location string, query string, args ...interface{}) (sql.Result, error) {
	logger.Debug("SQL Exec: %s, Args: %v", query, args)
	result, err := r.db.Exec(query, args...)
	if err != nil {
		logger.Error("Exec 실행 실패: %v", err)
		r.LogError(location, err.Error(), fmt.Sprintf("%s | Args: %v", query, args))
		return nil, errors.Wrap(err, "DATABASE_ERROR", "쿼리 실행에 실패했습니다")
	}
	return result, nil
}

func (r *Repository) execTx(tx *sql.Tx, location string, query string, args ...interface{}) (sql.Result, error) {
	logger.Debug("SQL Exec (TX): %s, Args: %v", query, args)
	result, err := tx.Exec(query, args...)
	if err != nil {
		logger.Error("ExecTx 실행 실패: %v", err)
		r.LogError(location, err.Error(), fmt.Sprintf("%s | Args: %v", query, args))
		return nil, errors.Wrap(err, "DATABASE_ERROR", "트랜잭션 쿼리 실행에 실패했습니다")
	}
	return result, nil
}

func validateDMLQuery(query string) error {
	normalized := normalizeSQLForValidation(query)
	if normalized == "" {
		return errors.New("INVALID_QUERY", "빈 쿼리는 실행할 수 없습니다")
	}
	if hasMultipleStatements(normalized) {
		return errors.New("INVALID_QUERY", "여러 SQL 문을 한 번에 실행할 수 없습니다")
	}

	first := firstSQLToken(normalized)
	switch first {
	case "insert":
		return nil
	case "update":
		if !containsSQLToken(normalized, "where") {
			return errors.New("INVALID_QUERY", "WHERE 없는 UPDATE는 실행할 수 없습니다")
		}
		return nil
	case "delete":
		if !containsSQLToken(normalized, "where") {
			return errors.New("INVALID_QUERY", "WHERE 없는 DELETE는 실행할 수 없습니다")
		}
		return nil
	default:
		return errors.New("INVALID_QUERY", "Exec는 INSERT/UPDATE/DELETE 계열 DML만 실행할 수 있습니다")
	}
}

func validateSchemaQuery(query string) error {
	normalized := normalizeSQLForValidation(query)
	if normalized == "" {
		return errors.New("INVALID_QUERY", "빈 쿼리는 실행할 수 없습니다")
	}
	if hasMultipleStatements(normalized) {
		return errors.New("INVALID_QUERY", "여러 SQL 문을 한 번에 실행할 수 없습니다")
	}

	switch firstSQLToken(normalized) {
	case "create", "alter":
		return nil
	default:
		return errors.New("INVALID_QUERY", "ExecSchema는 CREATE/ALTER 계열 SQL만 실행할 수 있습니다")
	}
}

func normalizeSQLForValidation(query string) string {
	return strings.ToLower(strings.Join(strings.Fields(strings.TrimSpace(query)), " "))
}

func firstSQLToken(query string) string {
	fields := strings.Fields(query)
	if len(fields) == 0 {
		return ""
	}
	return fields[0]
}

func containsSQLToken(query string, token string) bool {
	return strings.Contains(" "+strings.Join(strings.Fields(query), " ")+" ", " "+token+" ")
}

func hasMultipleStatements(query string) bool {
	trimmed := strings.TrimSpace(query)
	trimmed = strings.TrimSuffix(trimmed, ";")
	return strings.Contains(trimmed, ";")
}

// QueryRowTx 트랜잭션 내에서 단일 행 조회
func (r *Repository) QueryRowTx(tx *sql.Tx, query string, args ...interface{}) *sql.Row {
	logger.Debug("SQL Query (TX): %s, Args: %v", query, args)
	return tx.QueryRow(query, args...)
}

// Insert INSERT 쿼리 실행 및 ID 반환
func (r *Repository) Insert(table string, data map[string]interface{}) (int64, error) {
	return insertWithExec(r.Exec, table, data)
}

// Insert INSERT 쿼리 실행 및 ID 반환.
func (q *TxRepository) Insert(table string, data map[string]interface{}) (int64, error) {
	return insertWithExec(q.Exec, table, data)
}

func insertWithExec(exec func(string, ...interface{}) (sql.Result, error), table string, data map[string]interface{}) (int64, error) {
	if len(data) == 0 {
		return 0, errors.New("INVALID_PARAM", "INSERT 데이터가 없습니다")
	}

	columns, placeholders, values, err := buildInsertParts(data)
	if err != nil {
		return 0, err
	}

	query := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s)",
		table,
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "),
	)

	result, err := exec(query, values...)
	if err != nil {
		return 0, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, errors.Wrap(err, "DATABASE_ERROR", "INSERT ID 조회 실패")
	}

	return id, nil
}

// Update UPDATE 쿼리 실행
func (r *Repository) Update(table string, data map[string]interface{}, where string, whereArgs ...interface{}) (int64, error) {
	return updateWithExec(r.Exec, table, data, where, whereArgs...)
}

// Update UPDATE 쿼리 실행.
func (q *TxRepository) Update(table string, data map[string]interface{}, where string, whereArgs ...interface{}) (int64, error) {
	return updateWithExec(q.Exec, table, data, where, whereArgs...)
}

func updateWithExec(exec func(string, ...interface{}) (sql.Result, error), table string, data map[string]interface{}, where string, whereArgs ...interface{}) (int64, error) {
	if len(data) == 0 {
		return 0, errors.New("INVALID_PARAM", "업데이트할 데이터가 없습니다")
	}

	setClauses, values, err := buildSetClauses(data)
	if err != nil {
		return 0, err
	}

	values = append(values, whereArgs...)

	query := fmt.Sprintf(
		"UPDATE %s SET %s WHERE %s",
		table,
		strings.Join(setClauses, ", "),
		where,
	)

	result, err := exec(query, values...)
	if err != nil {
		return 0, err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return 0, errors.Wrap(err, "DATABASE_ERROR", "영향받은 행 조회 실패")
	}

	return affected, nil
}

// Upsert INSERT ... ON DUPLICATE KEY UPDATE 실행.
// updateData가 nil이면 insertData와 동일하게 업데이트한다.
func (r *Repository) Upsert(table string, insertData map[string]interface{}, updateData map[string]interface{}) (int64, error) {
	return upsertWithExec(r.Exec, table, insertData, updateData)
}

// Upsert INSERT ... ON DUPLICATE KEY UPDATE 실행.
func (q *TxRepository) Upsert(table string, insertData map[string]interface{}, updateData map[string]interface{}) (int64, error) {
	return upsertWithExec(q.Exec, table, insertData, updateData)
}

func upsertWithExec(exec func(string, ...interface{}) (sql.Result, error), table string, insertData map[string]interface{}, updateData map[string]interface{}) (int64, error) {
	if len(insertData) == 0 {
		return 0, errors.New("INVALID_PARAM", "INSERT 데이터가 없습니다")
	}
	if updateData == nil {
		updateData = insertData
	}
	if len(updateData) == 0 {
		return 0, errors.New("INVALID_PARAM", "UPDATE 데이터가 없습니다")
	}

	columns, placeholders, values, err := buildInsertParts(insertData)
	if err != nil {
		return 0, err
	}
	updateClauses, updateValues, err := buildSetClauses(updateData)
	if err != nil {
		return 0, err
	}
	values = append(values, updateValues...)

	query := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s) ON DUPLICATE KEY UPDATE %s",
		table,
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "),
		strings.Join(updateClauses, ", "),
	)

	logger.Debug("[UPSERT] Table: %s, Query: %s, Args: %v", table, query, values)
	result, err := exec(query, values...)
	if err != nil {
		logger.Error("[UPSERT FAILED] Table: %s, Error: %v", table, err)
		return 0, err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return 0, errors.Wrap(err, "DATABASE_ERROR", "영향받은 행 조회 실패")
	}

	logger.Debug("[UPSERT SUCCESS] Table: %s, Affected: %d", table, affected)
	return affected, nil
}

// Delete DELETE 쿼리 실행
func (r *Repository) Delete(table string, where string, whereArgs ...interface{}) (int64, error) {
	return deleteWithExec(r.Exec, table, where, whereArgs...)
}

// Delete DELETE 쿼리 실행.
func (q *TxRepository) Delete(table string, where string, whereArgs ...interface{}) (int64, error) {
	return deleteWithExec(q.Exec, table, where, whereArgs...)
}

func deleteWithExec(exec func(string, ...interface{}) (sql.Result, error), table string, where string, whereArgs ...interface{}) (int64, error) {
	query := fmt.Sprintf("DELETE FROM %s WHERE %s", table, where)

	result, err := exec(query, whereArgs...)
	if err != nil {
		return 0, err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return 0, errors.Wrap(err, "DATABASE_ERROR", "영향받은 행 조회 실패")
	}

	return affected, nil
}

// Exists 레코드 존재 여부 확인
func (r *Repository) Exists(table string, where string, whereArgs ...interface{}) (bool, error) {
	query := fmt.Sprintf("SELECT EXISTS(SELECT 1 FROM %s WHERE %s)", table, where)

	var exists bool
	err := r.QueryRow(query, whereArgs...).Scan(&exists)
	if err != nil {
		return false, errors.Wrap(err, "DATABASE_ERROR", "존재 여부 확인 실패")
	}

	return exists, nil
}

// Count 레코드 개수 조회
func (r *Repository) Count(table string, where string, whereArgs ...interface{}) (int64, error) {
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s", table)
	if strings.TrimSpace(where) != "" {
		query = fmt.Sprintf("%s WHERE %s", query, where)
	}

	var count int64
	err := r.QueryRow(query, whereArgs...).Scan(&count)
	if err != nil {
		return 0, errors.Wrap(err, "DATABASE_ERROR", "개수 조회 실패")
	}

	return count, nil
}

// UpdateMath 숫자 필드에 사칙연산 수행 (원자적 업데이트)
// operations: map[컬럼명]연산 (예: map[string]string{"count": "+1", "price": "*2", "stock": "-5"})
// 지원 연산자: + (덧셈), - (뺄셈), * (곱셈), / (나눗셈)
func (r *Repository) UpdateMath(table string, operations map[string]string, where string, whereArgs ...interface{}) (int64, error) {
	return updateMathWithExec(r.Exec, table, operations, where, whereArgs...)
}

// UpdateMath 숫자 필드에 사칙연산 수행.
func (q *TxRepository) UpdateMath(table string, operations map[string]string, where string, whereArgs ...interface{}) (int64, error) {
	return updateMathWithExec(q.Exec, table, operations, where, whereArgs...)
}

func updateMathWithExec(exec func(string, ...interface{}) (sql.Result, error), table string, operations map[string]string, where string, whereArgs ...interface{}) (int64, error) {
	if len(operations) == 0 {
		return 0, errors.New("INVALID_PARAM", "연산할 필드가 없습니다")
	}

	setClauses := make([]string, 0, len(operations))
	for col, op := range operations {
		if len(op) < 2 {
			return 0, errors.New("INVALID_PARAM", fmt.Sprintf("잘못된 연산 형식: %s", op))
		}

		operator := string(op[0])
		value := op[1:]

		switch operator {
		case "+":
			setClauses = append(setClauses, fmt.Sprintf("%s = %s + %s", col, col, value))
		case "-":
			setClauses = append(setClauses, fmt.Sprintf("%s = %s - %s", col, col, value))
		case "*":
			setClauses = append(setClauses, fmt.Sprintf("%s = %s * %s", col, col, value))
		case "/":
			setClauses = append(setClauses, fmt.Sprintf("%s = %s / %s", col, col, value))
		default:
			return 0, errors.New("INVALID_PARAM", fmt.Sprintf("지원하지 않는 연산자: %s", operator))
		}
	}

	query := fmt.Sprintf(
		"UPDATE %s SET %s WHERE %s",
		table,
		strings.Join(setClauses, ", "),
		where,
	)

	result, err := exec(query, whereArgs...)
	if err != nil {
		return 0, err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return 0, errors.Wrap(err, "DATABASE_ERROR", "영향받은 행 조회 실패")
	}

	return affected, nil
}

// LogError 에러 로그를 데이터베이스에 저장 (트랜잭션과 무관하게 별도 커넥션 사용)
func (r *Repository) LogError(location string, message string, sqlQuery string) {
	query := `INSERT INTO _a_error_logs (el_where, el_message, el_sql) VALUES (?, ?, ?)`

	// 트랜잭션과 무관하게 별도 커넥션으로 실행
	go func() {
		_, err := r.db.Exec(query, location, message, sqlQuery)
		if err != nil {
			logger.Error("에러 로그 저장 실패 [%s]: %v", location, err)
		}
	}()
}
