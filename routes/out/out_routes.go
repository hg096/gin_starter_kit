package out

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func SetupOutRoutes(rg *gin.RouterGroup) {

	rg.GET("/info", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "Public out info"})
	})
}
