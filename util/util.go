package util

import (
	"database/sql"
	"fmt"
	"gin_starter/db"
	"log"
	"reflect"
	"sync"
	"time"
)

// Util 구조체는 유틸리티 함수들을 포함
type Util struct{}

// HTTPError 는 HTTP 에러 상태, 메시지 및 원본 에러를 포함하는 커스텀 에러 타입입니다.
type HTTPError struct {
	Code     int    // HTTP 상태 코드 (예: 401)
	ErrCode  int    // 에러구분 코드 (예: 401)
	Message  string // 사용자에게 보여줄 에러 메시지
	Original error  // 원본 에러
}

func (he *HTTPError) Error() string {
	return fmt.Sprintf("%d: %s - %v", he.Code, he.Message, he.Original)
}

// HandleError는 주어진 에러를 처리하여 로컬 및 DB 로그를 기록
func HandleError(tx *sql.Tx, code int, errCode int, errMsg string, errWhere string, err error) error {

	if tx != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			log.Printf("트랜잭션 롤백 실패: %v", rbErr)
		} else {
			log.Println("트랜잭션 롤백 성공")
		}
	}

	log.Printf("에러 발생 (%s): %v", errWhere, err)

	if db.Conn != nil {
		_, dbErr := db.Conn.Exec(
			"INSERT INTO _a_error_logs (el_where, el_message, el_regi_date) VALUES (?, ?, ?)",
			errWhere, err.Error(), time.Now(),
		)
		if dbErr != nil {
			log.Printf("DB 로그 저장 실패: %v", dbErr)
		}
	} else {
		log.Println("DB 연결이 설정되어 있지 않아 에러 로그를 저장하지 못했습니다.")
	}

	return &HTTPError{
		Code:     code,
		ErrCode:  errCode,
		Message:  errMsg,
		Original: err,
	}
}

// AssignStringFields는 data map에서 값이 문자열인 경우, fieldMap에 지정된 포인터 변수에 대입
func AssignStringFields(data map[string]interface{}, fieldMap map[string]*string) {
	for key, ptr := range fieldMap {
		if value, exists := data[key]; exists {
			if s, ok := value.(string); ok {
				*ptr = s
			} else {
				log.Printf("Key %s exists but is not a string", key)
			}
		}
	}
}

func Empty(v interface{}) bool {
	if v == nil {
		return false
	}
	return reflect.ValueOf(v).IsZero()
}

var (
	instance *Util     // 싱글톤 인스턴스
	once     sync.Once // instance 생성에 대한 동기화를 담당
)

// GetInstance는 Util의 싱글톤 인스턴스를 반환
// 여러 고루틴에서 동시에 호출해도 once.Do에 의해 단 한 번만 생성
func GetInstance() *Util {
	once.Do(func() {
		instance = &Util{}
	})
	return instance
}
