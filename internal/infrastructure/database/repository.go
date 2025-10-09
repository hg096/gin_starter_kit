package database

import (
	"database/sql"
	"fmt"
	"gin_starter/pkg/errors"
	"gin_starter/pkg/logger"
	"strings"
)

// Repository 공통 데이터베이스 리포지토리
type Repository struct {
	db *DB
}

// NewRepository Repository 생성자
func NewRepository(db *DB) *Repository {
	return &Repository{db: db}
}

// QueryRow SELECT 단일 행 조회
func (r *Repository) QueryRow(query string, args ...interface{}) *sql.Row {
	logger.Debug("SQL Query: %s, Args: %v", query, args)
	return r.db.QueryRow(query, args...)
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

// Exec INSERT, UPDATE, DELETE 실행
func (r *Repository) Exec(query string, args ...interface{}) (sql.Result, error) {
	logger.Debug("SQL Exec: %s, Args: %v", query, args)
	result, err := r.db.Exec(query, args...)
	if err != nil {
		logger.Error("Exec 실행 실패: %v", err)
		r.LogError("Repository.Exec", err.Error(), fmt.Sprintf("%s | Args: %v", query, args))
		return nil, errors.Wrap(err, "DATABASE_ERROR", "쿼리 실행에 실패했습니다")
	}
	return result, nil
}

// ExecTx 트랜잭션 내에서 INSERT, UPDATE, DELETE 실행
func (r *Repository) ExecTx(tx *sql.Tx, query string, args ...interface{}) (sql.Result, error) {
	logger.Debug("SQL Exec (TX): %s, Args: %v", query, args)
	result, err := tx.Exec(query, args...)
	if err != nil {
		logger.Error("ExecTx 실행 실패: %v", err)
		r.LogError("Repository.ExecTx", err.Error(), fmt.Sprintf("%s | Args: %v", query, args))
		return nil, errors.Wrap(err, "DATABASE_ERROR", "트랜잭션 쿼리 실행에 실패했습니다")
	}
	return result, nil
}

// QueryRowTx 트랜잭션 내에서 단일 행 조회
func (r *Repository) QueryRowTx(tx *sql.Tx, query string, args ...interface{}) *sql.Row {
	logger.Debug("SQL Query (TX): %s, Args: %v", query, args)
	return tx.QueryRow(query, args...)
}

// Insert INSERT 쿼리 실행 및 ID 반환
func (r *Repository) Insert(table string, data map[string]interface{}) (int64, error) {
	columns := make([]string, 0, len(data))
	placeholders := make([]string, 0, len(data))
	values := make([]interface{}, 0, len(data))

	for col, val := range data {
		columns = append(columns, col)
		placeholders = append(placeholders, "?")
		values = append(values, val)
	}

	query := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s)",
		table,
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "),
	)

	result, err := r.Exec(query, values...)
	if err != nil {
		return 0, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, errors.Wrap(err, "DATABASE_ERROR", "INSERT ID 조회 실패")
	}

	return id, nil
}

// InsertTx 트랜잭션 내에서 INSERT 실행
func (r *Repository) InsertTx(tx *sql.Tx, table string, data map[string]interface{}) (int64, error) {
	columns := make([]string, 0, len(data))
	placeholders := make([]string, 0, len(data))
	values := make([]interface{}, 0, len(data))

	for col, val := range data {
		columns = append(columns, col)
		placeholders = append(placeholders, "?")
		values = append(values, val)
	}

	query := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s)",
		table,
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "),
	)

	result, err := r.ExecTx(tx, query, values...)
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
	setClauses := make([]string, 0, len(data))
	values := make([]interface{}, 0, len(data)+len(whereArgs))

	for col, val := range data {
		setClauses = append(setClauses, fmt.Sprintf("%s = ?", col))
		values = append(values, val)
	}

	values = append(values, whereArgs...)

	query := fmt.Sprintf(
		"UPDATE %s SET %s WHERE %s",
		table,
		strings.Join(setClauses, ", "),
		where,
	)

	result, err := r.Exec(query, values...)
	if err != nil {
		return 0, err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return 0, errors.Wrap(err, "DATABASE_ERROR", "영향받은 행 조회 실패")
	}

	return affected, nil
}

// UpdateTx 트랜잭션 내에서 UPDATE 실행
func (r *Repository) UpdateTx(tx *sql.Tx, table string, data map[string]interface{}, where string, whereArgs ...interface{}) (int64, error) {
	setClauses := make([]string, 0, len(data))
	values := make([]interface{}, 0, len(data)+len(whereArgs))

	for col, val := range data {
		setClauses = append(setClauses, fmt.Sprintf("%s = ?", col))
		values = append(values, val)
	}

	values = append(values, whereArgs...)

	query := fmt.Sprintf(
		"UPDATE %s SET %s WHERE %s",
		table,
		strings.Join(setClauses, ", "),
		where,
	)

	result, err := r.ExecTx(tx, query, values...)
	if err != nil {
		return 0, err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return 0, errors.Wrap(err, "DATABASE_ERROR", "영향받은 행 조회 실패")
	}

	return affected, nil
}

// Delete DELETE 쿼리 실행
func (r *Repository) Delete(table string, where string, whereArgs ...interface{}) (int64, error) {
	query := fmt.Sprintf("DELETE FROM %s WHERE %s", table, where)

	result, err := r.Exec(query, whereArgs...)
	if err != nil {
		return 0, err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return 0, errors.Wrap(err, "DATABASE_ERROR", "영향받은 행 조회 실패")
	}

	return affected, nil
}

// DeleteTx 트랜잭션 내에서 DELETE 실행
func (r *Repository) DeleteTx(tx *sql.Tx, table string, where string, whereArgs ...interface{}) (int64, error) {
	query := fmt.Sprintf("DELETE FROM %s WHERE %s", table, where)

	result, err := r.ExecTx(tx, query, whereArgs...)
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
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE %s", table, where)

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

	result, err := r.Exec(query, whereArgs...)
	if err != nil {
		return 0, err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return 0, errors.Wrap(err, "DATABASE_ERROR", "영향받은 행 조회 실패")
	}

	return affected, nil
}

// UpdateMathTx 트랜잭션 내에서 사칙연산 수행
func (r *Repository) UpdateMathTx(tx *sql.Tx, table string, operations map[string]string, where string, whereArgs ...interface{}) (int64, error) {
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

	result, err := r.ExecTx(tx, query, whereArgs...)
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
