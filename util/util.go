package util

import (
	"log"
	"reflect"
	"sync"
)

// Util 구조체는 유틸리티 함수들을 포함합니다.
type Util struct{}

//
//
//
//
//

// AssignStringFields는 data map에서 값이 문자열인 경우, fieldMap에 지정된 포인터 변수에 대입합니다.
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

//
//
//
//
//

var (
	instance *Util     // 싱글톤 인스턴스
	once     sync.Once // instance 생성에 대한 동기화를 담당
)

// GetInstance는 Util의 싱글톤 인스턴스를 반환합니다.
// 여러 고루틴에서 동시에 호출해도 once.Do에 의해 단 한 번만 생성됩니다.
func GetInstance() *Util {
	once.Do(func() {
		instance = &Util{}
	})
	return instance
}
