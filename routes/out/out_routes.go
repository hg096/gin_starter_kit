package out

import (
	"gin_starter/util"
	"net/http"

	"github.com/gin-gonic/gin"
)

func SetupOutRoutes(rg *gin.RouterGroup) {
	outGroup := rg.Group("/out")
	{
		outGroup.GET("/info", func(c *gin.Context) {
			util.EndResponse(c, http.StatusOK, gin.H{"message": "Public out info"}, "rest /out/info")
		})

	}
}
