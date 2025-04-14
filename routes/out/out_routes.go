package out

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// SetupOutRoutes는 /out 경로에 해당하는 라우트를 등록합니다.
func SetupOutRoutes(rg *gin.RouterGroup) {
	// 예제: /out/info 라우트
	rg.GET("/info", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "Public out info"})
	})
}
