package middleware

import (
	"gin_starter/pkg/response"
	"gin_starter/pkg/utils"
	"net/http"
	"regexp"

	"github.com/gin-gonic/gin"
)

var adminPageKeyPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{1,47}$`)

// RequirePermission enforces a single permission code.
func RequirePermission(code string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !hasPermission(c, code) {
			abortAPIForbidden(c, "permission denied")
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
			abortAPIForbidden(c, "top-admin permission required")
			c.Abort()
			return
		}
		c.Next()
	}
}

// AdminPagePermissionMiddleware returns 403 for authenticated but unauthorized page requests.
func AdminPagePermissionMiddleware(code string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !hasPermission(c, code) {
			abortPageForbidden(c)
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

		pageKey := utils.TrimLower(c.Param("page_key"))
		if !adminPageKeyPattern.MatchString(pageKey) {
			c.String(http.StatusBadRequest, "invalid page key")
			c.Abort()
			return
		}

		permissionCode := "admin.page." + pageKey + ".read"
		if !hasPermission(c, permissionCode) {
			abortPageForbidden(c)
			c.Abort()
			return
		}

		c.Next()
	}
}

func isSuperAdmin(c *gin.Context) bool {
	super, ok := contextBool(c, ContextIsSuperAdmin)
	return ok && super
}

func hasPermission(c *gin.Context, code string) bool {
	if isSuperAdmin(c) {
		return true
	}

	permissionSet, ok := c.Get(ContextUserPermissions)
	if !ok {
		return false
	}
	perms, ok := permissionSet.(map[string]struct{})
	if !ok {
		return false
	}
	_, exists := perms[code]
	return exists
}

func abortAPIForbidden(c *gin.Context, message string) {
	response.Forbidden(c, message)
}

func abortPageForbidden(c *gin.Context) {
	c.String(http.StatusForbidden, "forbidden")
}
