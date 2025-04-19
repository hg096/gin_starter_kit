package util

import (
	"sync"
)

// Util 구조체는 유틸리티 함수들을 포함
type Util struct{}

// AssignStringFields는 data map에서 값이 문자열인 경우, fieldMap에 지정된 포인터 변수에 대입
func AssignStringFields(data map[string]string, fieldMap map[string]*string) {
	for key, ptr := range fieldMap {
		if value, exists := data[key]; exists {
			*ptr = value
		}
	}
}

// []string -> []interface{}
func ToInterfaceSlice(strs []string) []interface{} {
	result := make([]interface{}, len(strs))
	for i, s := range strs {
		result[i] = s
	}
	return result
}

// 빈값체크
type Numeric interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr |
		~float32 | ~float64
}

func EmptyNumber[T Numeric](v T) bool {
	return v == 0
}

func EmptyString(s string) bool {
	return s == ""
}

func EmptyBool(b bool) bool {
	return !b
}

func EmptySlice[T any](sl []T) bool {
	return len(sl) == 0
}

func EmptyMap[K comparable, V any](m map[K]V) bool {
	return len(m) == 0
}

func EmptyPtr[T any](p *T) bool {
	return p == nil
}

var (
	instance *Util     // 싱글톤 인스턴스
	once     sync.Once // instance 생성에 대한 동기화를 담당
)

// GetInstance는 Util의 싱글톤 인스턴스를 반환
// 여러 고루틴에서 동시에 호출해도 once.Do에 의해 단 한번만 생성
func GetInstance() *Util {
	once.Do(func() {
		instance = &Util{}
	})
	return instance
}
