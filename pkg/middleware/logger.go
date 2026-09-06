package middleware

import (
	"gin_starter/pkg/logger"
	"time"

	"github.com/gin-gonic/gin"
)

// LoggerMiddleware 요청/응답 로깅 미들웨어
func LoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		// 요청 처리
		c.Next()

		// 응답 로깅
		latency := time.Since(start)
		statusCode := c.Writer.Status()
		clientIP := c.ClientIP()
		method := c.Request.Method

		if raw != "" {
			path = path + "?" + raw
		}

		logger.WithFields(map[string]interface{}{
			"status":  statusCode,
			"latency": latency,
			"ip":      clientIP,
			"method":  method,
			"path":    path,
		}).Info("Request processed")
	}
}