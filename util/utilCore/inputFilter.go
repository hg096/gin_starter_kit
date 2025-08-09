package utilCore

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

type InputRule struct {
	InputKey  string // 요청에서 찾을 키
	OutputKey string // 결과 맵에서 쓸 키 (빈 문자열이면 InputKey 그대로 사용)
	Label     string // 에러 메시지에 쓸 표시용 이름 (없으면 OutputKey)
	// 허용 패턴(화이트리스트). 이 패턴에 "포함되지 않는" 문자를 제거합니다.
	Allow *regexp.Regexp

	// 길이/필수 검증
	Required bool
	MinLen   int
	MaxLen   int

	Min float64
	Max float64

	// 전처리 옵션
	TrimSpace bool // 앞뒤 공백 제거
	ToLower   bool // 소문자 변환
	ToUpper   bool // 대문자 변환

	// 비어 있을 때 기본값
	Default string

	// 개별 필드 커스텀 에러 메시지 템플릿 (옵션)
	// 키: "required" | "min_len" | "max_len" | "invalid" 등
	// {label}, {n} 플레이스홀더 지원 (예: "{label}은(는) {n}자 이상이어야 합니다.")
	ErrMsg map[string]string

	// 에러 코드 덮어쓰기 (옵션) — "required"|"min_len"|"max_len"|"invalid" 키 사용
	// 예: {"required":"E_MY_REQUIRED"}
	ErrCode map[string]string

	// 에러 숫자 코드 덮어쓰기 (옵션)
	ErrCodeNum map[string]int
}

// 미리 준비된 패턴들(원하는 대로 추가 가능)
var (
	AllowNumber      = regexp.MustCompile(`[^0-9]+`)                  // 숫자만
	AllowDecimal     = regexp.MustCompile(`[^0-9\.]+`)                // 0-9와 점(.)
	AllowEnglish     = regexp.MustCompile(`[^a-zA-Z\s]+`)             // 영문/공백
	AllowKorean      = regexp.MustCompile(`[^가-힣ㄱ-ㅎㅏ-ㅣ\s]+`)          // 한글/공백
	AllowKorEng      = regexp.MustCompile(`[^가-힣ㄱ-ㅎㅏ-ㅣa-zA-Z\s]+`)    // 한글+영문/공백
	AllowKorEngNum   = regexp.MustCompile(`[^가-힣ㄱ-ㅎㅏ-ㅣa-zA-Z0-9\s]+`) // 한글+영문+숫자/공백
	AllowKorEngNumSp = regexp.MustCompile(`[^가-힣ㄱ-ㅎㅏ-ㅣa-zA-Z0-9 \t\r\n!"#$%&'()*+,\-./:;<=>?@[\\\]^_` + "`" + `{|}~]+`)
	AllowEmailLike   = regexp.MustCompile(`[^a-zA-Z0-9._%+\-@]+`)                                          // 이메일에 흔한 문자들
	AllowSlug        = regexp.MustCompile(`[^a-z0-9\-]+`)                                                  // 소문자 영숫자와 하이픈
	AllowUUIDLike    = regexp.MustCompile(`[^a-fA-F0-9\-]+`)                                               // UUID 형식 문자
	AllowPhoneLike   = regexp.MustCompile(`[^0-9\-\+\s]+`)                                                 // 전화번호에서 쓸 법한 것
	AllowDateLike    = regexp.MustCompile(`[^0-9\-:T\sZ/]+`)                                               // 날짜 문자열에 흔한 것
	AllowSafeText    = regexp.MustCompile(`[^a-zA-Z0-9가-힣ㄱ-ㅎㅏ-ㅣ\s\.\,\-\_\(\)\[\]\{\}!"'#%&\+\:;@/\?\\]+`) // 일반 텍스트 안전 범위
)

// sanitize는 Allow에 "걸리지 않는" 문자들을 제거하는 방식
func sanitize(s string, allow *regexp.Regexp) string {
	if allow == nil {
		// Allow 미지정 시, 그대로 통과
		return s
	}
	// allow는 "허용되지 않는 문자 집합"에 매칭하도록 정의했음 → 그 문자를 빈칸으로 제거
	return allow.ReplaceAllString(s, "")
}
func postBindRaw(c *gin.Context, key string) (string, bool) {
	// JSON Body 우선
	for _, ct := range []string{c.GetHeader("Content-Type"), c.GetHeader("Accept")} {
		if strings.Contains(ct, "application/json") {
			if raw, ok := cacheJSONBody(c)[key]; ok && raw != nil {
				return fmt.Sprint(raw), true
			}
			break
		}
	}
	// Form
	if v := c.PostForm(key); v != "" {
		return v, true
	}
	return "", false
}
func getBindRaw(c *gin.Context, key string) (string, bool) {
	if v := c.Query(key); v != "" {
		return v, true
	}
	if v := c.Param(key); v != "" {
		return v, true
	}
	return "", false
}

// 공통 처리기: 원문 추출 → 전처리(Trim/대소문자) → 허용문자 필터 → 길이/필수 검사 → 기본값
func bindOne(raw string, r InputRule) (string, string) {
	s := raw
	if r.TrimSpace {
		s = strings.TrimSpace(s)
	}
	if r.ToLower {
		s = strings.ToLower(s)
	}
	if r.ToUpper {
		s = strings.ToUpper(s)
	}
	s = sanitize(s, r.Allow)

	// 길이 검사
	if r.Required && s == "" {
		return "", "required" // 필수값
	}
	if r.MinLen > 0 && len(s) < r.MinLen {
		return "", fmt.Sprintf("min_len:%d", r.MinLen) // 예: min_len:3
	}
	if r.MaxLen > 0 && len(s) > r.MaxLen {
		return "", fmt.Sprintf("max_len:%d", r.MaxLen) // 예: max_len:20
	}

	// 숫자 범위 검사 (빈 문자열 제외)
	if s != "" && (r.Min != 0 || r.Max != 0) {
		if num, err := strconv.ParseFloat(s, 64); err == nil {
			if r.Min != 0 && num < r.Min {
				return s, fmt.Sprintf("min_val:%g", r.Min)
			}
			if r.Max != 0 && num > r.Max {
				return s, fmt.Sprintf("max_val:%g", r.Max)
			}
		}
	}

	// 기본값 적용
	if s == "" && r.Default != "" {
		s = r.Default
	}
	return s, ""
}

type FieldResult struct {
	Value string
	Err   string // "", "required", "min_len", "max_len"
}

func indexRulesByOutput(rules []InputRule) map[string]InputRule {
	m := make(map[string]InputRule, len(rules))
	for _, r := range rules {
		out := r.OutputKey
		if out == "" {
			out = r.InputKey
		}
		m[out] = r
	}
	return m
}

//	rules := []utilCore.InputRule{
//		{InputKey: "user_id", OutputKey: "u_id", Label: "아이디",
//			Allow: utilCore.AllowKorEngNum, Required: true, MinLen: 3, MaxLen: 20, TrimSpace: true},
//		{InputKey: "name", OutputKey: "u_name", Label: "이름",
//			Allow: utilCore.AllowKorEng, Required: true, MinLen: 1, MaxLen: 50, TrimSpace: true},
//		{InputKey: "email", OutputKey: "u_email", Label: "이메일",
//				Allow: utilCore.AllowEmailLike, Required: true, MinLen: 5, MaxLen: 100, ToLower: true, TrimSpace: true,
//				ErrMsg: map[string]string{
//					"required": "{label}을(를) 입력해 주세요.",
//					"min_len":  "{label}은(는) {n}자 이상이어야 합니다.",
//					"max_len":  "{label}은(는) {n}자 이하여야 합니다.",
//				},
//				ErrCode: map[string]string{
//					"required": "E_EMAIL_REQUIRED",
//				},
//				ErrCodeNum: map[string]int{
//					"required": 2101,
//				},
//			},
//			{InputKey: "age", OutputKey: "u_age", Label: "나이",
//				Allow: utilCore.AllowNumber, Default: "0"},
//			{InputKey: "bio", OutputKey: "u_bio", Label: "소개",
//				Allow: utilCore.AllowSafeText, MaxLen: 300, TrimSpace: true},
//		}
//
// bound := utilCore.PostFieldsBind(c, rules)
//
//	if utilCore.RespondIfBindError(c, bound, rules) {
//		return
//	}
//
// 매개변수:
//   - c: *gin.Context (현재 요청 컨텍스트)
//   - rules: []InputRule
//     InputKey   : 요청에서 찾을 파라미터 키
//     OutputKey  : 반환 시 사용할 키 (비어 있으면 InputKey 그대로 사용)
//     Label      : 에러 메시지나 로그에 표시할 필드명
//     Allow      : 허용 문자 정규식 (nil이면 필터링 없음)
//     Required   : 필수값 여부
//     MinLen     : 최소 허용 길이 (0이면 제한 없음)
//     MaxLen     : 최대 허용 길이 (0이면 제한 없음)
//     TrimSpace  : 앞뒤 공백 제거 여부
//     ToLower    : 소문자로 변환 여부
//     ToUpper    : 대문자로 변환 여부
//     Default    : 값이 없을 때 적용할 기본값
//     ErrMsg     : 에러 메시지 템플릿 (키: "required", "min_len", "max_len", "invalid")
//     ErrCode    : 커스텀 에러 코드 문자열 매핑
//     ErrCodeNum : 커스텀 에러 코드 숫자 매핑
//
// 반환값:
//   - map[string]FieldResult
//     key  : OutputKey (또는 InputKey)
//     Value: 필터링 및 전처리된 최종 값
//     Err  : 검증 실패 시 에러 코드 문자열 (예: "required", "min_len:3")
func PostFieldsBind(c *gin.Context, rules []InputRule) map[string]FieldResult {
	out := make(map[string]FieldResult, len(rules))
	for _, rule := range rules {
		raw, ok := postBindRaw(c, rule.InputKey)
		if !ok {
			raw = ""
		}
		val, err := bindOne(raw, rule)
		outKey := rule.OutputKey
		if outKey == "" {
			outKey = rule.InputKey
		}
		out[outKey] = FieldResult{Value: val, Err: err}
	}
	return out
}

//	rules := []utilCore.InputRule{
//		{InputKey: "email", OutputKey: "u_email", Label: "이메일",
//			Allow: utilCore.AllowEmailLike, Required: true, MinLen: 5, MaxLen: 100, ToLower: true, TrimSpace: true,
//			ErrMsg: map[string]string{
//				"required": "{label}을(를) 입력해 주세요.",
//				"min_len":  "{label}은(는) {n}자 이상이어야 합니다.",
//				"max_len":  "{label}은(는) {n}자 이하여야 합니다.",
//			},
//			ErrCode: map[string]string{
//				"required": "E_EMAIL_REQUIRED",
//			},
//			ErrCodeNum: map[string]int{
//				"required": 2101,
//			},
//		},
//		{InputKey: "q", OutputKey: "search", Label: "검색어",
//			Allow: utilCore.AllowSafeText, MaxLen: 100, TrimSpace: true},
//		{InputKey: "page", OutputKey: "page", Label: "페이지",
//			Allow: utilCore.AllowNumber, Default: "1"},
//		{InputKey: "limit", OutputKey: "limit", Label: "페이지크기",
//			Allow: utilCore.AllowNumber, Default: "20"},
//		{InputKey: "source", OutputKey: "source", Label: "요청출처",
//			Allow: utilCore.AllowSlug, ToLower: true}, // 예: web-app, admin
//	}
//
// bound := utilCore.GetFieldsBind(c, rules)
//
//	if utilCore.RespondIfBindError(c, bound, rules) {
//		return
//	}
//
// 매개변수:
//   - c: *gin.Context (현재 요청 컨텍스트)
//   - rules: []InputRule
//     InputKey   : 요청에서 찾을 파라미터 키
//     OutputKey  : 반환 시 사용할 키 (비어 있으면 InputKey 그대로 사용)
//     Label      : 에러 메시지나 로그에 표시할 필드명
//     Allow      : 허용 문자 정규식 (nil이면 필터링 없음)
//     Required   : 필수값 여부
//     MinLen     : 최소 허용 길이 (0이면 제한 없음)
//     MaxLen     : 최대 허용 길이 (0이면 제한 없음)
//     TrimSpace  : 앞뒤 공백 제거 여부
//     ToLower    : 소문자로 변환 여부
//     ToUpper    : 대문자로 변환 여부
//     Default    : 값이 없을 때 적용할 기본값
//     ErrMsg     : 에러 메시지 템플릿 (키: "required", "min_len", "max_len", "invalid")
//     ErrCode    : 커스텀 에러 코드 문자열 매핑
//     ErrCodeNum : 커스텀 에러 코드 숫자 매핑
//
// 반환값:
//   - map[string]FieldResult
//     key  : OutputKey (또는 InputKey)
//     Value: 필터링 및 전처리된 최종 값
//     Err  : 검증 실패 시 에러 코드 문자열 (예: "required", "min_len:3")
func GetFieldsBind(c *gin.Context, rules []InputRule) map[string]FieldResult {
	out := make(map[string]FieldResult, len(rules))
	for _, rule := range rules {
		raw, ok := getBindRaw(c, rule.InputKey)
		if !ok {
			raw = ""
		}
		val, err := bindOne(raw, rule)
		outKey := rule.OutputKey
		if outKey == "" {
			outKey = rule.InputKey
		}
		out[outKey] = FieldResult{Value: val, Err: err}
	}
	return out
}

// BoundValues
// map[string]FieldResult → map[string]string 변환
// - Err가 비어있는 값만 포함할지 여부를 includeInvalid로 지정 가능
func BoundValues(bound map[string]FieldResult, includeInvalid bool) map[string]string {
	out := make(map[string]string, len(bound))
	for k, fr := range bound {
		if includeInvalid || fr.Err == "" {
			out[k] = fr.Value
		}
	}
	return out
}

// POST/PUT/DELETE: 바인딩 -> 검증/에러응답 -> 키:값 반환 (성공 시 true)
func BindPostAndRespond(c *gin.Context, rules []InputRule, debug string) (map[string]string, bool) {
	bound := PostFieldsBind(c, rules)
	if RespondIfBindError(c, bound, rules, debug) {
		return nil, false
	}
	return BoundValues(bound, false), true
}

// GET: 바인딩 -> 검증/에러응답 -> 키:값 반환 (성공 시 true)
func BindGetAndRespond(c *gin.Context, rules []InputRule, debug string) (map[string]string, bool) {
	bound := GetFieldsBind(c, rules)
	if RespondIfBindError(c, bound, rules, debug) {
		return nil, false
	}
	return BoundValues(bound, false), true
}

// HTTP 메서드에 따라 자동 분기 (POST/PUT/DELETE vs 그 외=GET)
func BindAutoAndRespond(c *gin.Context, rules []InputRule, debug string) (map[string]string, bool) {
	switch c.Request.Method {
	case "POST", "PUT", "DELETE", "PATCH":
		return BindPostAndRespond(c, rules, debug)
	default:
		return BindGetAndRespond(c, rules, debug)
	}
}
