package utils

import (
	"net"
	"strings"

	"github.com/gin-gonic/gin"
)

func NormalizeHostName(rawHost string) string {
	host := strings.TrimSpace(rawHost)
	if host == "" {
		return ""
	}
	if strings.Contains(host, ",") {
		host = strings.TrimSpace(strings.Split(host, ",")[0])
	}
	if parsedHost, _, err := net.SplitHostPort(host); err == nil {
		host = parsedHost
	} else {
		host = strings.Trim(host, "[]")
		if strings.Count(host, ":") == 1 {
			host = strings.Split(host, ":")[0]
		}
	}
	return strings.ToLower(strings.Trim(host, "[]"))
}

func IsLocalhostHost(rawHost string) bool {
	switch NormalizeHostName(rawHost) {
	case "localhost", "127.0.0.1", "::1":
		return true
	default:
		return false
	}
}

func IsLocalDebugRequest(c *gin.Context) bool {
	if c == nil || c.Request == nil {
		return false
	}
	if !IsLocalhostHost(c.Request.Host) {
		return false
	}
	clientIP := net.ParseIP(strings.TrimSpace(c.ClientIP()))
	return clientIP != nil && clientIP.IsLoopback()
}
