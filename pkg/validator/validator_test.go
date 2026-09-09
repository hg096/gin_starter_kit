package validator

import (
	"github.com/gin-gonic/gin"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestValidateJSONAndText(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		rule       Rule
		valid      bool
	}{
		{"null required", `{"title":null}`, Rule{Field: "title", Required: true}, false},
		{"number", `{"title":123}`, Rule{Field: "title"}, false},
		{"object", `{"title":{}}`, Rule{Field: "title"}, false},
		{"malformed", `{"title":`, Rule{Field: "title"}, false},
		{"one Korean character", `{"title":"가"}`, Rule{Field: "title", MinLen: 2}, false},
		{"two Korean characters", `{"title":"가나"}`, Rule{Field: "title", MinLen: 2, MaxLen: 2}, true},
		{"byte limit", `{"title":"가나"}`, Rule{Field: "title", MaxBytes: 5}, false},
		{"optional null", `{"title":null}`, Rule{Field: "title"}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest("POST", "/", strings.NewReader(tc.body))
			c.Request.Header.Set("Content-Type", "application/json")
			if got := Validate(c, []Rule{tc.rule}); got.Valid != tc.valid {
				t.Fatalf("valid=%v errors=%v", got.Valid, got.Errors)
			}
		})
	}
}
