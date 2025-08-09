package utilCore

import (
	"fmt"
	"strconv"
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

// c 에 저장한 값 가져오기
func GetContextVal(c *gin.Context, target string) (string, bool) {
	val, ok := c.Get(target)
	if !ok {
		return "", false
	}
	result, ok := val.(string)
	return result, ok
}

func EndResponse(c *gin.Context, status int, jsonObj gin.H, debug string) {
	if gin.Mode() != gin.ReleaseMode {
		jsonObj["messageDebug"] = debug
	}
	c.AbortWithStatusJSON(status, jsonObj)
}

// JSON Body 캐시 유틸
func cacheJSONBody(c *gin.Context) map[string]any {
	if v, ok := c.Get("_jsonBody"); ok {
		if m, ok := v.(map[string]any); ok {
			return m
		}
	}
	var m map[string]any
	if err := c.ShouldBindJSON(&m); err == nil && m != nil {
		c.Set("_jsonBody", m)
		return m
	}
	c.Set("_jsonBody", map[string]any{})
	return map[string]any{}
}

// BindField는 POST·PUT·DELETE 요청에서 key에 해당하는 값을 JSON 바디, 폼, 쿼리 순으로 찾아 반환
func PostBindField(c *gin.Context, key string, defaultValue string) string {
	// JSON 바디 캐시에서 조회
	for _, ct := range []string{c.GetHeader("Content-Type"), c.GetHeader("Accept")} {
		if strings.Contains(ct, "application/json") {
			if raw := cacheJSONBody(c)[key]; raw != nil {
				// raw는 interface{} 형태 (string, float64, bool 등)
				str := fmt.Sprint(raw)
				if str != "" {
					return str
				}
			}
			break
		}
	}
	// Form 데이터
	if v := c.PostForm(key); v != "" {
		return v
	}
	return defaultValue
}

// POST·PUT·DELETE 한번에 처리 - map[string][2]string{ "findKey":{"insertKey","defaultValue"}, }
func PostFields(c *gin.Context, defaults map[string][2]string) map[string]string {
	out := make(map[string]string, len(defaults))
	for key, def := range defaults {
		out[def[0]] = PostBindField(c, key, def[1])
	}
	return out
}

// GetField는 URL 쿼리 파라미터에서 key에 해당하는 값을 찾아 반환
func GetBindField(c *gin.Context, key, defaultValue string) string {
	if v := c.Query(key); v != "" {
		return v
	}

	if v := c.Param(key); v != "" {
		return v
	}
	return defaultValue
}

// GET 한번에 처리 - map[string][2]string{ "findKey":{"returnKey","defaultValue"}, }
func GetFields(c *gin.Context, defaults map[string][2]string) map[string]string {
	out := make(map[string]string, len(defaults))
	for queryKey, def := range defaults {
		outKey, defaultValue := def[0], def[1]
		out[outKey] = GetBindField(c, queryKey, defaultValue)
	}
	return out
}

// 숫자 -> 문자
func NumericToString[T Numeric](n T) string {
	// 내부적으로 타입에 따라 Format 함수 분기
	switch any(n).(type) {
	case float32:
		return strconv.FormatFloat(float64(any(n).(float32)), 'f', -1, 32)
	case float64:
		return strconv.FormatFloat(any(n).(float64), 'f', -1, 64)
	case int, int8, int16, int32, int64:
		return strconv.FormatInt(any(n).(int64), 10)
	case uint, uint8, uint16, uint32, uint64, uintptr:
		return strconv.FormatUint(any(n).(uint64), 10)
	default:
		return fmt.Sprint(n)
	}
}

// 문자 -> 숫자
func StringToNumeric[T Numeric](s string) (T, error) {
	var zero T

	// “제로값”을 one-shot으로 타입 단언할 때 쓰기 위해 any(zero)를 사용
	switch any(zero).(type) {
	// 정수 타입: ParseInt → int64 로 파싱한 뒤 T로 변환
	case int, int8, int16, int32, int64:
		// 64비트 정수 범위로 파싱 (비트 너비는 64)
		i, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return zero, err
		}
		return T(i), nil

	// 부호 없는 정수 타입: ParseUint → uint64 로 파싱한 뒤 T로 변환
	case uint, uint8, uint16, uint32, uint64, uintptr:
		u, err := strconv.ParseUint(s, 10, 64)
		if err != nil {
			return zero, err
		}
		return T(u), nil

	// 실수 타입: ParseFloat (64비트 부동소수) → T로 변환
	case float32, float64:
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return zero, err
		}
		return T(f), nil

	default:
		return zero, fmt.Errorf("unsupported numeric type: %T", zero)
	}
}
