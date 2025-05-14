package util

import (
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
)

// Util 구조체는 유틸리티 함수들을 포함
type Util struct{}

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

func EndResponse(c *gin.Context, status int, jsonObj gin.H, debug string) {
	if gin.Mode() != gin.ReleaseMode {
		jsonObj["messageDebug"] = debug
	}
	c.AbortWithStatusJSON(status, jsonObj)
}

// 바디 캐싱
func cacheJSONBody(c *gin.Context) map[string]interface{} {
	if b, exists := c.Get("jsonBody"); exists {
		return b.(map[string]interface{})
	}
	body := make(map[string]interface{})
	_ = c.ShouldBindJSON(&body)
	c.Set("jsonBody", body)
	return body
}

// BindField는 POST/PUT 요청에서 key에 해당하는 값을 JSON 바디, 폼, 쿼리 순으로 찾아 반환
func BindField(c *gin.Context, key string, defaultValue string) string {

	// JSON 바디 캐시에서 조회
	for _, ct := range []string{c.GetHeader("Content-Type"), c.GetHeader("Accept")} {
		if strings.Contains(ct, "application/json") {
			if v, ok := cacheJSONBody(c)[key]; ok {
				if s, ok := v.(string); ok && s != "" {
					return s
				}
			}
			break
		}
	}

	// Form 데이터
	if v := c.PostForm(key); v != "" {
		return v
	}

	// URL 쿼리
	if v := c.Query(key); v != "" {
		return v
	}

	return defaultValue
}

// 한번에 처리 - map[string][]string{ "findKey":{"insertKey","defaultValue"}, }
func BindFields(c *gin.Context, defaults map[string][]string) map[string]string {
	out := make(map[string]string, len(defaults))
	for key, def := range defaults {
		out[def[0]] = BindField(c, key, def[1])
	}
	return out
}
