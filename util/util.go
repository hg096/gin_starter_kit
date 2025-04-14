package util

import (
	"fmt"
	"strings"
	"sync"
)

// Util 구조체는 유틸리티 함수들을 포함합니다.
type Util struct{}

//
//
//
//
//

// FormatGreeting은 이름을 입력받아 인사말 문자열을 반환합니다.
func (u *Util) FormatGreeting(name string) string {
	return fmt.Sprintf("Hello, %s!", name)
}

// JoinStrings는 문자열 슬라이스를 주어진 구분자로 연결하여 반환합니다.
func (u *Util) JoinStrings(strs []string, sep string) string {
	return strings.Join(strs, sep)
}

// SumIntSlice는 정수 슬라이스의 합을 계산하여 반환합니다.
func (u *Util) SumIntSlice(numbers []int) int {
	sum := 0
	for _, v := range numbers {
		sum += v
	}
	return sum
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
