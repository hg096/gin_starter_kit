// util/inputbind/errors.go (기존 파일 확장)
package utilCore

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type FieldError struct {
	Code    string            `json:"code"`           // 예: "E_REQUIRED", "E_MIN_LEN"
	CodeNum int               `json:"codeNum"`        // 예: 1001, 1002 ...
	Message string            `json:"message"`        // 사람이 읽는 문구
	Meta    map[string]string `json:"meta,omitempty"` // 부가정보 (예: {"n":"3"})
}

// 기본 에러코드 매핑
var defaultCodeMap = map[string]struct {
	Code    string
	CodeNum int
}{
	"required": {Code: "E_REQUIRED", CodeNum: 1001},
	"min_len":  {Code: "E_MIN_LEN", CodeNum: 1002},
	"max_len":  {Code: "E_MAX_LEN", CodeNum: 1003},
	"min_val":  {Code: "E_MIN_VAL", CodeNum: 1004},
	"max_val":  {Code: "E_MAX_VAL", CodeNum: 1005},
	"invalid":  {Code: "E_INVALID", CodeNum: 1099},
}

// BuildFieldErrors: 코드/문구/메타를 모두 생성
func BuildFieldErrors(bound map[string]FieldResult, rules []InputRule) (map[string]FieldError, bool) {
	errs := map[string]FieldError{}
	idx := indexRulesByOutput(rules)

	for outKey, fr := range bound {
		if fr.Err == "" {
			continue
		}
		rule := idx[outKey]
		label := rule.Label
		if label == "" {
			label = rule.InputKey
		}

		codeKey, metaStr := splitErr(fr.Err) // ex) "min_len", "3"
		n := metaStr
		meta := map[string]string{}
		if n != "" {
			meta["n"] = n
		}

		// 메시지
		var msg string
		if rule.ErrMsg != nil {
			if tpl, ok := rule.ErrMsg[codeKey]; ok && tpl != "" {
				msg = fillTemplate(tpl, label, n)
			}
		}
		if msg == "" {
			msg = defaultMessage(codeKey, label, n)
		}

		// 코드 문자열
		code := defaultCodeMap[codeKey].Code
		if rule.ErrCode != nil {
			if v, ok := rule.ErrCode[codeKey]; ok && v != "" {
				code = v
			}
		}
		if code == "" { // 알 수 없는 코드키
			code = defaultCodeMap["invalid"].Code
		}

		// 숫자 코드
		codeNum := defaultCodeMap[codeKey].CodeNum
		if rule.ErrCodeNum != nil {
			if v, ok := rule.ErrCodeNum[codeKey]; ok && v > 0 {
				codeNum = v
			}
		}
		if codeNum == 0 {
			codeNum = defaultCodeMap["invalid"].CodeNum
		}

		errs[outKey] = FieldError{
			Code:    code,
			CodeNum: codeNum,
			Message: msg,
			Meta:    meta,
		}
	}
	return errs, len(errs) > 0
}

// 기존 헬퍼를 코드 동시 리턴 형태로 교체
func RespondIfBindError(c *gin.Context, bound map[string]FieldResult, rules []InputRule, debug string) bool {
	if fields, hasErr := BuildFieldErrors(bound, rules); hasErr {
		EndResponse(c, http.StatusBadRequest, gin.H{
			"error": gin.H{
				"type":    "VALIDATION_ERROR",
				"code":    "E_VALIDATION", // 전체 요청 수준 코드
				"codeNum": 1000,           // 전체 요청 수준 숫자 코드
				"fields":  fields,         // 각 필드별 상세
			},
		}, debug)
		return true
	}
	return false
}

// BuildErrorMessages:
// - bound: PostFieldsBind/GetFieldsBind 결과
// - rules: 같은 요청에서 사용한 규칙들 (Label/ErrMsg 찾기용)
// 반환: fieldKey->message 맵, 에러 존재 여부
func BuildErrorMessages(bound map[string]FieldResult, rules []InputRule) (map[string]string, bool) {
	msgs := map[string]string{}
	idx := indexRulesByOutput(rules)

	for outKey, fr := range bound {
		if fr.Err == "" {
			continue
		}
		rule := idx[outKey]
		label := rule.Label
		if label == "" {
			label = rule.InputKey
		}
		code, meta := splitErr(fr.Err)

		// 커스텀 템플릿 우선
		if rule.ErrMsg != nil {
			if tpl, ok := rule.ErrMsg[code]; ok && tpl != "" {
				msgs[outKey] = fillTemplate(tpl, label, meta)
				continue
			}
		}
		// 기본 메시지
		msgs[outKey] = defaultMessage(code, label, meta)
	}
	return msgs, len(msgs) > 0
}

func defaultMessage(code, label, n string) string {
	switch code {
	case "required":
		return fmt.Sprintf("%s은(는) 필수값입니다.", label)
	case "min_len":
		return fmt.Sprintf("%s은(는) 최소 %s자 이상이어야 합니다.", label, n)
	case "max_len":
		return fmt.Sprintf("%s은(는) 최대 %s자 이하여야 합니다.", label, n)
	case "min_val":
		return fmt.Sprintf("%s은(는) 최소 %s 이상이어야 합니다.", label, n)
	case "max_val":
		return fmt.Sprintf("%s은(는) 최대 %s 이하여야 합니다.", label, n)
	default:
		return fmt.Sprintf("%s 값이 유효하지 않습니다.", label)
	}
}

func splitErr(err string) (code string, meta string) {
	if i := strings.IndexByte(err, ':'); i >= 0 {
		return err[:i], err[i+1:]
	}
	return err, ""
}

func fillTemplate(tpl, label, n string) string {
	return strings.NewReplacer("{label}", label, "{n}", n).Replace(tpl)
}
