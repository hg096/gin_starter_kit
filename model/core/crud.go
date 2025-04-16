package core

import (
	"database/sql"
	"fmt"
	"gin_starter/db"
	"gin_starter/util"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// ValidateModel은 전달받은 model에 대해 유효성 검사를 실행
// 모델은 validate 태그가 지정된 Exported 필드들을 가져야함
func ValidateModel(model interface{}) error {
	return getValidator().Struct(model)
}

func SubstituteQuery(query string, args []string) string {
	parts := strings.Split(query, "?")
	if len(parts)-1 != len(args) {
		return query
	}
	var sb strings.Builder
	sb.WriteString(parts[0])
	for i, arg := range args {
		argStr := fmt.Sprintf("'%v'", arg)
		sb.WriteString(argStr)
		sb.WriteString(parts[i+1])
	}
	return sb.String()
}

// BuildInsertQueryAndExecute는 주어진 테이블 이름과 데이터 맵을 기반으로
// INSERT 쿼리를 생성하고, 전역 DB 객체를 사용하여 실행한 후 결과를 반환
func BuildInsertQuery(c *gin.Context, tx *sql.Tx,
	tableName string, data map[string]string, errWhere string) (int64, error) {

	if db.Conn == nil {
		return 0, fmt.Errorf("database connection is not set")
	}
	columns := []string{}
	placeholders := []string{}
	args := []string{}

	for col, value := range data {
		columns = append(columns, col)
		placeholders = append(placeholders, "?")
		args = append(args, value)
	}

	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		tableName,
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "),
	)
	result, err := db.Conn.Exec(query, util.ToInterfaceSlice(args)...)
	if err != nil {
		fullQuery := SubstituteQuery(query, args)
		util.HandleSqlError(c, tx, fullQuery, 0, "요청에 실패했습니다.", errWhere, err)
		return 0, err
	}

	insertedID, err := result.LastInsertId()
	if err != nil {
		fullQuery := SubstituteQuery(query, args)
		util.HandleSqlError(c, tx, fullQuery, 0, "요청에 실패했습니다.", errWhere, err)
		return 0, err
	}
	return insertedID, nil
}

// BuildUpdateQueryAndExecute는 테이블 이름, 업데이트할 데이터, WHERE 조건과 인자를 받아
// 동적으로 UPDATE 쿼리를 생성하고 전역 DB 객체(DB 변수)를 사용하여 안전하게 실행합
func BuildUpdateQuery(c *gin.Context, tx *sql.Tx,
	tableName string, updateData map[string]string, whereClause string, whereArgs []string, errWhere string) (sql.Result, error) {

	if db.Conn == nil {
		return nil, fmt.Errorf("database connection is not set")
	}

	var setClauses []string
	var args []string

	for col, value := range updateData {
		strVal := value
		// if strVal, ok := value.(string); ok {
		// 만약 문자열이 사칙연산 연산자(+=, -=, *=, /=)로 시작하면 처리
		if strings.HasPrefix(strVal, "+= ") || strings.HasPrefix(strVal, "-= ") ||
			strings.HasPrefix(strVal, "*= ") || strings.HasPrefix(strVal, "/= ") {

			// 연산자를 추출
			operator := strVal[:3]
			// 연산 값의 문자열 부분 (예: "100")
			operandStr := strings.TrimSpace(strVal[3:])
			// 문자열을 float64로 파싱 (숫자형으로 변환)
			operand, err := strconv.ParseFloat(operandStr, 64)
			operandStr2 := fmt.Sprintf("%f", operand)
			if err != nil {
				return nil, fmt.Errorf("invalid operand for column %s: %v", col, err)
			}

			// 각 연산자별로 SET 절을 구성
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
			args = append(args, operandStr2)
			continue
			// }
		}
		// 일반 값이라면 그냥 "컬럼 = ?" 형식으로 추가
		setClauses = append(setClauses, fmt.Sprintf("%s = ?", col))
		args = append(args, value)
	}

	// SET 절 구성: "col1 = ?, col2 = ?, ..."
	setPart := strings.Join(setClauses, ", ")

	// 최종 쿼리 구성: UPDATE 테이블 SET ... WHERE ...
	query := fmt.Sprintf("UPDATE %s SET %s WHERE %s", tableName, setPart, whereClause)

	// WHERE 인자도 arguments 배열에 추가
	args = append(args, whereArgs...)

	// 파라미터라이즈드 쿼리 실행 (SQL 인젝션 방지)
	result, err := db.Conn.Exec(query, util.ToInterfaceSlice(args)...)
	if err != nil {
		fullQuery := SubstituteQuery(query, args)
		util.HandleSqlError(c, tx, fullQuery, 0, "요청에 실패했습니다.", errWhere, err)
		return nil, err
	}

	return result, nil
}

// BuildDeleteQueryAndExecute는 주어진 테이블 이름과 WHERE 조건(절 및 인자)을 기반으로
// DELETE 쿼리를 생성하여 전역 DB 객체를 사용해 실행한 후 결과를 반환
func BuildDeleteQuery(c *gin.Context, tx *sql.Tx, tableName string, whereClause string, whereArgs []string, errWhere string) (sql.Result, error) {
	// DB가 설정되어 있는지 확인
	if db.Conn == nil {
		return nil, fmt.Errorf("database connection is not set")
	}

	// DELETE 쿼리 생성: 테이블 이름과 WHERE 조건을 사용하여 동적으로 쿼리 구성
	query := fmt.Sprintf("DELETE FROM %s WHERE %s", tableName, whereClause)

	// parameterized query 방식으로 쿼리 실행 (SQL 인젝션 방지)
	result, err := db.Conn.Exec(query, util.ToInterfaceSlice(whereArgs)...)
	if err != nil {
		util.HandleSqlError(c, tx, query, 0, "요청에 실패했습니다.", errWhere, err)
		return nil, err
	}

	return result, nil
}
