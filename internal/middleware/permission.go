package middleware

import (
	"gin_starter/pkg/response"
	"net/http"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
)

var adminPageKeyPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{1,47}$`)

// RequirePermission enforces a single permission code.
func RequirePermission(code string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if isSuperAdmin(c) {
			c.Next()
			return
		}

		permissionSet, ok := c.Get("user_permissions")
		if !ok {
			response.Forbidden(c, "permission denied")
			c.Abort()
			return
		}

		perms, ok := permissionSet.(map[string]struct{})
		if !ok {
			response.Forbidden(c, "permission denied")
			c.Abort()
			return
		}

		if _, exists := perms[code]; !exists {
			response.Forbidden(c, "permission denied")
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireSuperAdmin allows only top-admin (TA) and legacy A + level 10.
func RequireSuperAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !isSuperAdmin(c) {
			response.Forbidden(c, "top-admin permission required")
			c.Abort()
			return
		}
		c.Next()
	}
}

// AdminPagePermissionMiddleware returns 403 for authenticated but unauthorized page requests.
func AdminPagePermissionMiddleware(code string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if isSuperAdmin(c) {
			c.Next()
			return
		}

		permissionSet, ok := c.Get("user_permissions")
		if !ok {
			c.String(http.StatusForbidden, "forbidden")
			c.Abort()
			return
		}

		perms, ok := permissionSet.(map[string]struct{})
		if !ok {
			c.String(http.StatusForbidden, "forbidden")
			c.Abort()
			return
		}

		if _, exists := perms[code]; !exists {
			c.String(http.StatusForbidden, "forbidden")
			c.Abort()
			return
		}

		c.Next()
	}
}

// AdminPageDynamicPermissionMiddleware enforces page-scoped read permission for /admin/p/:page_key.
func AdminPageDynamicPermissionMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if isSuperAdmin(c) {
			c.Next()
			return
		}

		pageKey := strings.ToLower(strings.TrimSpace(c.Param("page_key")))
		if !adminPageKeyPattern.MatchString(pageKey) {
			c.String(http.StatusBadRequest, "invalid page key")
			c.Abort()
			return
		}

		permissionCode := "admin.page." + pageKey + ".read"
		permissionSet, ok := c.Get("user_permissions")
		if !ok {
			c.String(http.StatusForbidden, "forbidden")
			c.Abort()
			return
		}

		perms, ok := permissionSet.(map[string]struct{})
		if !ok {
			c.String(http.StatusForbidden, "forbidden")
			c.Abort()
			return
		}

		if _, exists := perms[permissionCode]; !exists {
			c.String(http.StatusForbidden, "forbidden")
			c.Abort()
			return
		}

		c.Next()
	}
}

func isSuperAdmin(c *gin.Context) bool {
	value, ok := c.Get("is_super_admin")
	if !ok {
		return false
	}
	super, ok := value.(bool)
	return ok && super
}
