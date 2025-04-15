package core

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	// 실제 DB 실행 로직은 db 패키지 등에서 처리할 수 있습니다.
	// "gin_starter/db"
)

// 전역 변수 DB: 초기화 후 crud.go 내에서 모두 사용 가능
var DB *sql.DB

// SetDB는 외부에서 데이터베이스 연결 객체(*sql.DB)를 설정합니다.
func SetDB(db *sql.DB) {
	DB = db
}

// ValidateModel은 전달받은 model에 대해 유효성 검사를 실행합니다.
// 모델은 validate 태그가 지정된 Exported 필드들을 가지고 있어야 합니다.
func ValidateModel(model interface{}) error {
	return getValidator().Struct(model)
}

// BuildInsertQueryAndExecute는 주어진 테이블 이름과 데이터 맵을 기반으로
// INSERT 쿼리를 생성하고, 전역 DB 객체를 사용하여 실행한 후 결과를 반환합니다.
func BuildInsertQuery(tableName string, data map[string]interface{}) (int64, error) {
	// DB가 설정되어 있는지 확인
	if DB == nil {
		return 0, fmt.Errorf("database connection is not set")
	}
	columns := []string{}
	placeholders := []string{}
	args := []interface{}{}

	// map은 순서가 보장되지 않으므로 각 컬럼과 값을 순회하여 처리합니다.
	for col, value := range data {
		columns = append(columns, col)
		placeholders = append(placeholders, "?")
		args = append(args, value)
	}

	// 생성된 컬럼명과 플레이스홀더들을 join하여 쿼리 문자열을 생성
	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		tableName,
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "),
	)

	// parameterized query 방식으로 쿼리 실행 (SQL 인젝션 방지)
	result, err := DB.Exec(query, args...)
	if err != nil {
		return 0, err
	}

	// 삽입된 행의 인덱스(LastInsertId) 값을 리턴합니다.
	insertedID, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}
	return insertedID, nil
}

// BuildUpdateQueryAndExecute는 테이블 이름, 업데이트할 데이터, WHERE 조건과 인자를 받아
// 동적으로 UPDATE 쿼리를 생성하고 전역 DB 객체(DB 변수)를 사용하여 안전하게 실행합니다.
func BuildUpdateQuery(tableName string, updateData map[string]interface{}, whereClause string, whereArgs []interface{}) (sql.Result, error) {
	// DB가 설정되어 있는지 확인
	if DB == nil {
		return nil, fmt.Errorf("database connection is not set")
	}

	var setClauses []string
	var args []interface{}

	// updateData 맵을 순회하며 SET 절을 구성합니다.
	for col, value := range updateData {
		// 값이 문자열 타입인지 확인합니다.
		if strVal, ok := value.(string); ok {
			// 만약 문자열이 사칙연산 연산자(+=, -=, *=, /=)로 시작하면 처리
			if strings.HasPrefix(strVal, "+= ") || strings.HasPrefix(strVal, "-= ") ||
				strings.HasPrefix(strVal, "*= ") || strings.HasPrefix(strVal, "/= ") {

				// 연산자를 추출합니다. (예: "+=")
				operator := strVal[:3]
				// 연산 값의 문자열 부분 (예: "100")
				operandStr := strings.TrimSpace(strVal[3:])
				// 문자열을 float64로 파싱 (숫자형으로 변환)
				operand, err := strconv.ParseFloat(operandStr, 64)
				if err != nil {
					return nil, fmt.Errorf("invalid operand for column %s: %v", col, err)
				}

				// 각 연산자별로 SET 절을 구성합니다.
				switch operator {
				case "+= ":
					setClauses = append(setClauses, fmt.Sprintf("%s = %s + ?", col, col))
				case "-= ":
					setClauses = append(setClauses, fmt.Sprintf("%s = %s - ?", col, col))
				case "*= ":
					setClauses = append(setClauses, fmt.Sprintf("%s = %s * ?", col, col))
				case "/= ":
					setClauses = append(setClauses, fmt.Sprintf("%s = %s / ?", col, col))
				}
				args = append(args, operand)
				continue
			}
		}
		// 일반 값이라면 그냥 "컬럼 = ?" 형식으로 추가합니다.
		setClauses = append(setClauses, fmt.Sprintf("%s = ?", col))
		args = append(args, value)
	}

	// SET 절 구성: 예를 들어 "col1 = ?, col2 = ?, ..."
	setPart := strings.Join(setClauses, ", ")

	// 최종 쿼리 구성: UPDATE 테이블 SET ... WHERE ...
	query := fmt.Sprintf("UPDATE %s SET %s WHERE %s", tableName, setPart, whereClause)

	// WHERE 인자도 arguments 배열에 추가합니다.
	args = append(args, whereArgs...)

	// 파라미터라이즈드 쿼리 실행 (SQL 인젝션 방지를 위해 안전)
	result, err := DB.Exec(query, args...)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// BuildDeleteQueryAndExecute는 주어진 테이블 이름과 WHERE 조건(절 및 인자)을 기반으로
// DELETE 쿼리를 생성하여 전역 DB 객체를 사용해 실행한 후 결과를 반환합니다.
func BuildDeleteQuery(tableName string, whereClause string, whereArgs []interface{}) (sql.Result, error) {
	// DB가 설정되어 있는지 확인
	if DB == nil {
		return nil, fmt.Errorf("database connection is not set")
	}

	// DELETE 쿼리 생성: 테이블 이름과 WHERE 조건을 사용하여 동적으로 쿼리 구성
	query := fmt.Sprintf("DELETE FROM %s WHERE %s", tableName, whereClause)

	// parameterized query 방식으로 쿼리 실행 (SQL 인젝션 방지)
	result, err := DB.Exec(query, whereArgs...)
	if err != nil {
		return nil, err
	}

	return result, nil
}
