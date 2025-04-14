package routes

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"gin_starter/db" // 실제 모듈 경로로 수정하세요.
	"gin_starter/routes/api"
	"gin_starter/routes/out"
)

// SetupRoutes 함수는 전달받은 gin.Engine에 모든 라우트를 등록합니다.
func SetupRoutes(r *gin.Engine) {
	// 기본 홈 라우트
	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "home"})
	})

	// Ping 테스트 라우트
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "pong"})
	})

	// /api 그룹 생성
	apiGroup := r.Group("/api")
	{
		// /api/user 라우트 등록
		api.SetupUserRoutes(apiGroup)
		// /api/blog 라우트 등록
		api.SetupBlogRoutes(apiGroup)
	}

	// /out 그룹 생성
	outGroup := r.Group("/out")
	{
		// /out 관련 라우트 등록
		out.SetupOutRoutes(outGroup)
	}

	// MySQL 테이블 _user의 데이터를 동적으로 반환하는 예제 라우트
	r.GET("/mysql", func(c *gin.Context) {
		rows, err := db.Conn.Query("SELECT * FROM _user LIMIT 10")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer rows.Close()

		columns, err := rows.Columns()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		var results []map[string]interface{}
		for rows.Next() {
			values := make([]interface{}, len(columns))
			valuePtrs := make([]interface{}, len(columns))
			for i := range values {
				valuePtrs[i] = &values[i]
			}

			if err := rows.Scan(valuePtrs...); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			rowMap := make(map[string]interface{})
			for i, col := range columns {
				var v interface{}
				if b, ok := values[i].([]byte); ok {
					v = string(b)
				} else {
					v = values[i]
				}
				rowMap[col] = v
			}
			results = append(results, rowMap)
		}

		c.JSON(http.StatusOK, results)
	})

}
