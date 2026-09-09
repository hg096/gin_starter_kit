package blog

import (
	"database/sql"
	"errors"
	"github.com/gin-gonic/gin"
	"net/http/httptest"
	"testing"
)

type lookupFailureRepository struct {
	Repository
	err error
}

func (r lookupFailureRepository) FindByID(int64) (*Blog, error) { return nil, r.err }

func TestLookupErrorStatus(t *testing.T) {
	for _, action := range []string{"get", "update", "delete"} {
		for _, tc := range []struct {
			name   string
			err    error
			status int
		}{
			{"missing", sql.ErrNoRows, 404},
			{"database failure", errors.New("connection failed"), 500},
		} {
			t.Run(action+"/"+tc.name, func(t *testing.T) {
				h := NewHandler(NewDeps(lookupFailureRepository{err: tc.err}))
				w := httptest.NewRecorder()
				c, _ := gin.CreateTestContext(w)
				c.Request = httptest.NewRequest("GET", "/1", nil)
				c.Params = gin.Params{{Key: "id", Value: "1"}}
				c.Set("user_id", "tester")
				switch action {
				case "get":
					h.Get(c)
				case "update":
					h.Update(c)
				case "delete":
					h.Delete(c)
				}
				if w.Code != tc.status {
					t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
				}
			})
		}
	}
}
