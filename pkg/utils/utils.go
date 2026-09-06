package utils

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// 빈값 체크 함수들

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

// Context 관련 함수들

// GetContextVal c에 저장한 값 가져오기
func GetContextVal(c *gin.Context, target string) (string, bool) {
	val, ok := c.Get(target)
	if !ok {
		return "", false
	}
	result, ok := val.(string)
	return result, ok
}

// EndResponse JSON 응답 (디버그 모드에서 디버그 메시지 포함)
func EndResponse(c *gin.Context, status int, jsonObj gin.H, debug string) {
	if gin.Mode() != gin.ReleaseMode {
		jsonObj["messageDebug"] = debug
	}
	c.AbortWithStatusJSON(status, jsonObj)
}

// JSON Body 관련 함수들

// cacheJSONBody JSON Body 캐시
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

// PostBindField POST·PUT·DELETE 요청에서 key에 해당하는 값을 JSON 바디, 폼, 쿼리 순으로 찾아 반환
func PostBindField(c *gin.Context, key string, defaultValue string) string {
	// JSON 바디 캐시에서 조회
	for _, ct := range []string{c.GetHeader("Content-Type"), c.GetHeader("Accept")} {
		if strings.Contains(ct, "application/json") {
			if raw := cacheJSONBody(c)[key]; raw != nil {
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

// PostFields POST·PUT·DELETE 한번에 처리 - map[string][2]string{ "findKey":{"insertKey","defaultValue"}, }
func PostFields(c *gin.Context, defaults map[string][2]string) map[string]string {
	out := make(map[string]string, len(defaults))
	for key, def := range defaults {
		out[def[0]] = PostBindField(c, key, def[1])
	}
	return out
}

// GetBindField URL 쿼리 파라미터에서 key에 해당하는 값을 찾아 반환
func GetBindField(c *gin.Context, key, defaultValue string) string {
	if v := c.Query(key); v != "" {
		return v
	}

	if v := c.Param(key); v != "" {
		return v
	}
	return defaultValue
}

// GetFields GET 한번에 처리 - map[string][2]string{ "findKey":{"returnKey","defaultValue"}, }
func GetFields(c *gin.Context, defaults map[string][2]string) map[string]string {
	out := make(map[string]string, len(defaults))
	for queryKey, def := range defaults {
		outKey, defaultValue := def[0], def[1]
		out[outKey] = GetBindField(c, queryKey, defaultValue)
	}
	return out
}

// 타입 변환 함수들

// AssignStringFields data map에서 값이 문자열인 경우, fieldMap에 지정된 포인터 변수에 대입
func AssignStringFields(data map[string]string, fieldMap map[string]*string) {
	for key, ptr := range fieldMap {
		if value, exists := data[key]; exists {
			*ptr = value
		}
	}
}

// ToInterfaceSlice []string -> []interface{}
func ToInterfaceSlice(strs []string) []interface{} {
	result := make([]interface{}, len(strs))
	for i, s := range strs {
		result[i] = s
	}
	return result
}

// NumericToString 숫자 -> 문자
func NumericToString[T Numeric](n T) string {
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

// StringToNumeric 문자 -> 숫자
func StringToNumeric[T Numeric](s string) (T, error) {
	var zero T

	switch any(zero).(type) {
	case int, int8, int16, int32, int64:
		i, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return zero, err
		}
		return T(i), nil

	case uint, uint8, uint16, uint32, uint64, uintptr:
		u, err := strconv.ParseUint(s, 10, 64)
		if err != nil {
			return zero, err
		}
		return T(u), nil

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

func intFromBindField(c *gin.Context, key string, defaultValue int) int {
	raw := GetBindField(c, key, fmt.Sprint(defaultValue))
	value, err := StringToNumeric[int](strings.TrimSpace(raw))
	if err != nil {
		return defaultValue
	}
	return value
}

// 날짜/주차 관련 함수들

// GetKoreanYearWeek 한국 시간 기준 년도-주차 반환
func GetKoreanYearWeek() string {
	now := NowSeoul()
	year, week := now.ISOWeek()
	return fmt.Sprintf("%d년 %d주차", year, week)
}

// GetWeekStartEnd 년도-주차로 시작일/종료일 계산
func GetWeekStartEnd(yearWeek string) (startDt, endDt string) {
	var year, week int
	_, err := fmt.Sscanf(yearWeek, "%d년 %d주차", &year, &week)
	if err != nil {
		now := NowSeoul()
		return now.Format("20060102"), now.Format("20060102")
	}

	jan4 := time.Date(year, time.January, 4, 0, 0, 0, 0, SeoulLocation())
	shift := (int(jan4.Weekday()) + 6) % 7
	mondayOfWeek1 := jan4.AddDate(0, 0, -shift)

	monday := mondayOfWeek1.AddDate(0, 0, (week-1)*7)
	friday := monday.AddDate(0, 0, 4)

	return monday.Format("20060102"), friday.Format("20060102")
}
