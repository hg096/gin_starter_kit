package validator

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// Rule 입력 검증 규칙
type Rule struct {
	Field    string          // 필드명
	Label    string          // 사용자에게 표시할 이름
	Required bool            // 필수 여부
	MinLen   int             // 최소 길이
	MaxLen   int             // 최대 길이
	Min      float64         // 최소값 (숫자)
	Max      float64         // 최대값 (숫자)
	Pattern  *regexp.Regexp  // 허용 패턴 (화이트리스트)
	Custom   CustomValidator // 커스텀 검증 함수
}

// CustomValidator 커스텀 검증 함수 타입
type CustomValidator func(value string) error

// ValidationError 검증 에러 정보
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Code    string `json:"code"`
}

// Result 검증 결과
type Result struct {
	Valid  bool
	Errors map[string]ValidationError
	Values map[string]string
}

// 미리 정의된 패턴들
var (
	PatternNumber      = regexp.MustCompile(`^[0-9]+$`)
	PatternDecimal     = regexp.MustCompile(`^[0-9.]+$`)
	PatternEnglish     = regexp.MustCompile(`^[a-zA-Z\s]+$`)
	PatternKorean      = regexp.MustCompile(`^[가-힣ㄱ-ㅎㅏ-ㅣ\s]+$`)
	PatternKorEng      = regexp.MustCompile(`^[가-힣ㄱ-ㅎㅏ-ㅣa-zA-Z\s]+$`)
	PatternKorEngNum   = regexp.MustCompile(`^[가-힣ㄱ-ㅎㅏ-ㅣa-zA-Z0-9\s]+$`)
	PatternEmail       = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	PatternAlphaNum    = regexp.MustCompile(`^[a-zA-Z0-9]+$`)
	PatternSlug        = regexp.MustCompile(`^[a-z0-9\-]+$`)
	PatternURL         = regexp.MustCompile(`^https?://[^\s]+$`)
	PatternPhone       = regexp.MustCompile(`^[0-9\-+\s()]+$`)
)

// Validate 검증 실행
func Validate(c *gin.Context, rules []Rule) *Result {
	result := &Result{
		Valid:  true,
		Errors: make(map[string]ValidationError),
		Values: make(map[string]string),
	}

	for _, rule := range rules {
		value := extractValue(c, rule.Field)
		value = strings.TrimSpace(value)

		// 필수 체크
		if rule.Required && value == "" {
			result.addError(rule.Field, rule.Label, "REQUIRED", fmt.Sprintf("%s은(는) 필수 항목입니다", rule.Label))
			continue
		}

		// 값이 없으면 다음 규칙으로
		if value == "" {
			result.Values[rule.Field] = value
			continue
		}

		// 길이 검증
		if rule.MinLen > 0 && len(value) < rule.MinLen {
			result.addError(rule.Field, rule.Label, "MIN_LENGTH",
				fmt.Sprintf("%s은(는) 최소 %d자 이상이어야 합니다", rule.Label, rule.MinLen))
			continue
		}

		if rule.MaxLen > 0 && len(value) > rule.MaxLen {
			result.addError(rule.Field, rule.Label, "MAX_LENGTH",
				fmt.Sprintf("%s은(는) 최대 %d자 이하여야 합니다", rule.Label, rule.MaxLen))
			continue
		}

		// 숫자 범위 검증
		if rule.Min != 0 || rule.Max != 0 {
			if num, err := strconv.ParseFloat(value, 64); err == nil {
				if rule.Min != 0 && num < rule.Min {
					result.addError(rule.Field, rule.Label, "MIN_VALUE",
						fmt.Sprintf("%s은(는) 최소 %.0f 이상이어야 합니다", rule.Label, rule.Min))
					continue
				}
				if rule.Max != 0 && num > rule.Max {
					result.addError(rule.Field, rule.Label, "MAX_VALUE",
						fmt.Sprintf("%s은(는) 최대 %.0f 이하여야 합니다", rule.Label, rule.Max))
					continue
				}
			}
		}

		// 패턴 검증
		if rule.Pattern != nil && !rule.Pattern.MatchString(value) {
			result.addError(rule.Field, rule.Label, "INVALID_FORMAT",
				fmt.Sprintf("%s의 형식이 올바르지 않습니다", rule.Label))
			continue
		}

		// 커스텀 검증
		if rule.Custom != nil {
			if err := rule.Custom(value); err != nil {
				result.addError(rule.Field, rule.Label, "CUSTOM_ERROR", err.Error())
				continue
			}
		}

		result.Values[rule.Field] = value
	}

	return result
}

// addError 에러 추가
func (r *Result) addError(field, label, code, message string) {
	r.Valid = false
	r.Errors[field] = ValidationError{
		Field:   field,
		Message: message,
		Code:    code,
	}
}

// extractValue gin.Context에서 값 추출 (JSON, Form, Query 순서)
func extractValue(c *gin.Context, field string) string {
	// JSON 바디 확인
	if c.ContentType() == "application/json" {
		if val, exists := c.Get("_jsonBody"); exists {
			if body, ok := val.(map[string]interface{}); ok {
				if v, ok := body[field]; ok {
					return fmt.Sprint(v)
				}
			}
		}
		// JSON 바디가 캐시되지 않았다면 캐시
		var jsonBody map[string]interface{}
		if err := c.ShouldBindJSON(&jsonBody); err == nil {
			c.Set("_jsonBody", jsonBody)
			if v, ok := jsonBody[field]; ok {
				return fmt.Sprint(v)
			}
		}
	}

	// Form 데이터
	if v := c.PostForm(field); v != "" {
		return v
	}

	// Query 파라미터
	if v := c.Query(field); v != "" {
		return v
	}

	// URL 파라미터
	if v := c.Param(field); v != "" {
		return v
	}

	return ""
}

// GetErrorMap 에러를 map으로 변환
func (r *Result) GetErrorMap() map[string]interface{} {
	errMap := make(map[string]interface{})
	for field, err := range r.Errors {
		errMap[field] = map[string]string{
			"message": err.Message,
			"code":    err.Code,
		}
	}
	return errMap
}