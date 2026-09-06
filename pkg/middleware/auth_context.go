package middleware

import (
	"gin_starter/pkg/authz"
	"gin_starter/pkg/utils"
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	ContextUserID              = "user_id"
	ContextUserType            = "user_type"
	ContextUserLevel           = "user_level"
	ContextIsSuperAdmin        = "is_super_admin"
	ContextUserPermissions     = "user_permissions"
	ContextUserPermissionsList = "user_permissions_list"
	ContextLevelPolicyEnabled  = "level_policy_enabled"
)

func setAuthContext(c *gin.Context, userID, userType string, userLevel int, permissions []string, levelPolicyEnabled bool) {
	permSet := make(map[string]struct{}, len(permissions))
	for _, permission := range permissions {
		trimmed := strings.TrimSpace(permission)
		if trimmed == "" {
			continue
		}
		permSet[trimmed] = struct{}{}
	}

	normalizedUserType := authz.NormalizeAuthType(userType)

	c.Set(ContextUserID, userID)
	c.Set(ContextUserType, normalizedUserType)
	c.Set(ContextUserLevel, userLevel)
	c.Set(ContextIsSuperAdmin, authz.IsSuperAdmin(normalizedUserType, userLevel))
	c.Set(ContextUserPermissions, permSet)
	c.Set(ContextUserPermissionsList, permissions)
	c.Set(ContextLevelPolicyEnabled, levelPolicyEnabled)
}

func contextString(c *gin.Context, key string) (string, bool) {
	value, ok := utils.GetContextVal(c, key)
	return value, ok
}

func contextBool(c *gin.Context, key string) (bool, bool) {
	value, ok := c.Get(key)
	if !ok {
		return false, false
	}
	casted, ok := value.(bool)
	return casted, ok
}

func contextInt(c *gin.Context, key string) (int, bool) {
	value, ok := c.Get(key)
	if !ok {
		return 0, false
	}
	casted, ok := value.(int)
	return casted, ok
}
