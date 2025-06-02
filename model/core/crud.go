package core

import (
	"database/sql"
	"errors"
	"fmt"
	"gin_starter/db"
	"gin_starter/util"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	v10 "github.com/go-playground/validator/v10"
)

// ValidateModel은 전달받은 model에 대해 유효성 검사를 실행
// 모델은 validate 태그가 지정된 Exported 필드들을 가져야함
func ValidateModel(model interface{}) error {
	return getValidator().Struct(model)
}

// sql 에러 작성시 ? 대입
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

// HandleError는 주어진 에러를 처리하여 로컬 및 DB 로그를 기록
func HandleSqlError(c *gin.Context, tx *sql.Tx,
	sql string, errCode int, errWhere string, err error) {

	if tx != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			// log.Printf("트랜잭션 롤백 실패: %v", rbErr)
		} else {
			// log.Println("트랜잭션 롤백 성공")
		}
	}

	// log.Printf("에러 발생 (%s): %v", errWhere, err)

	if db.Conn != nil {
		_, dbErr := db.Conn.Exec(
			"INSERT INTO _a_error_logs (el_where, el_message, el_sql, el_regi_date) VALUES (?, ?, ?, ?)",
			errWhere, err.Error(), sql, time.Now(),
		)
		if dbErr != nil {
			// log.Printf("DB 로그 저장 실패: %v", dbErr)
		}
	} else {
		// log.Println("DB 연결이 설정되어 있지 않아 에러 로그를 저장하지 못했습니다.")
	}

	util.EndResponse(c, http.StatusBadRequest, gin.H{"errCode": errCode}, "fn crud/HandleSqlError")
}

// 유효성 검사 실패시 에러 문구리턴
func FormatValidationErrors(ve v10.ValidationErrors, validateConverts map[string]string) map[string]map[string]string {
	msgs := make(map[string]map[string]string, len(ve))
	for _, fe := range ve {
		var msg string
		switch fe.Tag() {
		case "required":
			msg = fmt.Sprintf("%s은(는) 필수 항목입니다", validateConverts[fe.Field()])
		case "min":
			msg = fmt.Sprintf("%s은(는) 최소 %s글자 이상 입력해주세요", validateConverts[fe.Field()], fe.Param())
		case "max":
			msg = fmt.Sprintf("%s은(는) 최대 %s글자 이하로 입력해주세요", validateConverts[fe.Field()], fe.Param())
		case "email":
			msg = fmt.Sprintf("%s은(는) 이메일 형식이 올바르지 않습니다", validateConverts[fe.Field()])
		case "alphaunicode":
			msg = fmt.Sprintf("%s은(는) 영문 또는 한글만 입력해주세요", validateConverts[fe.Field()])
		case "alpha":
			msg = fmt.Sprintf("%s은(는) 영문만 입력해주세요", validateConverts[fe.Field()])
		case "numeric":
			msg = fmt.Sprintf("%s은(는) 숫자만 입력해주세요", validateConverts[fe.Field()])
		default:
			msg = fmt.Sprintln("유효하지 않은 입력입니다")
		}
		msgs[fe.Field()]["massage"] = msg
		msgs[fe.Field()]["field"] = fe.Field()
		msgs[fe.Field()]["tag"] = fe.Tag()
		msgs[fe.Field()]["param"] = fe.Param()
	}
	return msgs
}

// 유효성 검사 실패 통합 트랜잭션 롤백, 에러문구까지 출력
func HandleValidationError(c *gin.Context, tx *sql.Tx, err error, converts map[string]string) bool {
	if err == nil {
		return false
	}
	// 트랜잭션 롤백
	if tx != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			c.Error(rbErr) // 로깅 용으로 Gin에 에러 등록
		}
	}
	// validator 에러인지 검사
	var ve v10.ValidationErrors
	if errors.As(err, &ve) {
		msgs := FormatValidationErrors(ve, converts)
		util.EndResponse(c, http.StatusBadRequest, gin.H{"errors": msgs}, "fn crud/HandleValidationError")
	}
	return true
}

// sql where 취약점 개선
func SanitizeWhereClause(where string) string {
	// 금지 키워드 (대소문자 구분 없이)
	badPatterns := []string{
		";",  // 명령어 구분자
		"--", // 주석 → 이후 쿼리 무력화 가능
		// "drop",   // DROP TABLE 등
		// "delete", // DELETE FROM
		// "truncate",
		// "alter",
		// "update", // where 안에서 sub update 막기
	}

	lowered := strings.ToLower(where)
	for _, pattern := range badPatterns {
		if strings.Contains(lowered, pattern) {
			// 로그로 남기고 삭제 또는 치환
			where = strings.ReplaceAll(where, pattern, "")
		}
	}

	return where
}

// 트랜젝션 시작
func BeginTransaction(c *gin.Context) (*sql.Tx, error) {
	if db.Conn == nil {
		return nil, fmt.Errorf("database connection is not set")
	}
	tx, err := db.Conn.Begin()
	if err != nil {
		// c.JSON(500, gin.H{"error": "트랜잭션 시작 실패"})
		return nil, err
	}
	return tx, nil
}

// 트랜젝션 종료
func EndTransactionCommit(tx *sql.Tx) error {
	if tx == nil {
		return fmt.Errorf("nil transaction provided")
	}
	if err := tx.Commit(); err != nil {
		// return fmt.Errorf("commit failed: %v", err)
	}
	return nil
}

// select 문
func BuildSelectQuery(c *gin.Context, tx *sql.Tx,
	query string, args []string, errWhere string) ([]map[string]string, error) {

	if db.Conn == nil {
		return nil, fmt.Errorf("database connection is not set")
	}

	// 문자열 슬라이스 → interface{} 슬라이스로 변환
	interArgs := util.ToInterfaceSlice(args)

	rows, err := db.Conn.Query(query, interArgs...)
	if err != nil {
		fullQuery := SubstituteQuery(query, args)
		HandleSqlError(c, tx, fullQuery, 0, errWhere, err)
		return nil, err
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		fullQuery := SubstituteQuery(query, args)
		HandleSqlError(c, tx, fullQuery, 0, errWhere, err)
		return nil, err
	}

	var results []map[string]string
	// 스캔용 RawBytes 슬라이스 + interface{} 포인터 슬라이스
	rawVals := make([]sql.RawBytes, len(cols))
	scanArgs := make([]interface{}, len(cols))
	for i := range rawVals {
		scanArgs[i] = &rawVals[i]
	}

	for rows.Next() {
		if err := rows.Scan(scanArgs...); err != nil {
			fullQuery := SubstituteQuery(query, args)
			HandleSqlError(c, tx, fullQuery, 0, errWhere, err)
			return nil, err
		}
		rowMap := make(map[string]string, len(cols))
		for i, col := range cols {
			rowMap[col] = string(rawVals[i])
		}
		results = append(results, rowMap)
	}

	return results, nil
}

// insert 문
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
		HandleSqlError(c, tx, fullQuery, 0, errWhere, err)
		return 0, err
	}

	insertedID, err := result.LastInsertId()
	if err != nil {
		fullQuery := SubstituteQuery(query, args)
		HandleSqlError(c, tx, fullQuery, 0, errWhere, err)
		return 0, err
	}
	return insertedID, nil
}

// update 문
func BuildUpdateQuery(c *gin.Context, tx *sql.Tx,
	tableName string, updateData map[string]string, whereClause string, whereArgs []string, errWhere string) (sql.Result, error) {

	if db.Conn == nil {
		return nil, fmt.Errorf("database connection is not set")
	}

	var setClauses []string
	var args []string

	for col, value := range updateData {
		strVal := value
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
		}
		// 일반 값이라면 그냥 "컬럼 = ?" 형식으로 추가
		setClauses = append(setClauses, fmt.Sprintf("%s = ?", col))
		args = append(args, value)
	}

	// SET 절 구성: "col1 = ?, col2 = ?, ..."
	setPart := strings.Join(setClauses, ", ")

	safeWhere := SanitizeWhereClause(whereClause)
	query := fmt.Sprintf("UPDATE %s SET %s WHERE %s", tableName, setPart, safeWhere)
	args = append(args, whereArgs...)

	result, err := db.Conn.Exec(query, util.ToInterfaceSlice(args)...)
	if err != nil {
		fullQuery := SubstituteQuery(query, args)
		HandleSqlError(c, tx, fullQuery, 0, errWhere, err)
		return nil, err
	}

	return result, nil
}

// delete 문
func BuildDeleteQuery(c *gin.Context, tx *sql.Tx,
	tableName string, whereClause string, whereArgs []string, errWhere string) (sql.Result, error) {
	if db.Conn == nil {
		return nil, fmt.Errorf("database connection is not set")
	}
	safeWhere := SanitizeWhereClause(whereClause)
	query := fmt.Sprintf("DELETE FROM %s WHERE %s", tableName, safeWhere)

	// parameterized query 방식
	result, err := db.Conn.Exec(query, util.ToInterfaceSlice(whereArgs)...)
	if err != nil {
		HandleSqlError(c, tx, query, 0, errWhere, err)
		return nil, err
	}

	return result, nil
}

// insert 멀티
func BuildInsertQueryMulti(c *gin.Context, tx *sql.Tx,
	tableName string, dataList []map[string]string, errWhere string) (sql.Result, error) {

	if db.Conn == nil {
		return nil, fmt.Errorf("database connection is not set")
	}
	if len(dataList) == 0 {
		return nil, fmt.Errorf("no data to insert")
	}

	// 모든 컬럼 수집 (순서 고정)
	columnSet := map[string]struct{}{}
	var columns []string
	for _, data := range dataList {
		for col := range data {
			if _, exists := columnSet[col]; !exists {
				columnSet[col] = struct{}{}
				columns = append(columns, col)
			}
		}
	}

	var valueRows []string
	var args []string

	for _, data := range dataList {
		var rowParts []string
		for _, col := range columns {
			if val, ok := data[col]; ok {
				rowParts = append(rowParts, "?")
				args = append(args, val)
			} else {
				rowParts = append(rowParts, "DEFAULT")
			}
		}
		valueRows = append(valueRows, "("+strings.Join(rowParts, ", ")+")")
	}

	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES %s",
		tableName,
		strings.Join(columns, ", "),
		strings.Join(valueRows, ", "),
	)

	result, err := db.Conn.Exec(query, util.ToInterfaceSlice(args)...)
	if err != nil {
		fullQuery := SubstituteQuery(query, args)
		HandleSqlError(c, tx, fullQuery, 0, errWhere, err)
		return nil, err
	}

	return result, nil
}
