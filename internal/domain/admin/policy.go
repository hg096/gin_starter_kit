package admin

import (
	"gin_starter/internal/domain/user"
	"gin_starter/pkg/authz"
	"gin_starter/pkg/errors"
)

// IsSuperAdmin returns true for TA and legacy A + level 10.
func IsSuperAdmin(userType string, userLevel int) bool {
	return authz.IsSuperAdmin(userType, userLevel)
}

// CanManageTarget enforces hierarchy for mutable admin actions.
func CanManageTarget(actor Actor, target *user.User) bool {
	if target == nil {
		return false
	}
	if !authz.IsAdminType(actor.UserType) {
		return false
	}
	if actor.IsSuperAdmin || IsSuperAdmin(actor.UserType, actor.UserLevel) {
		return true
	}

	targetType := authz.NormalizeAuthType(target.AuthType)
	if authz.IsTopAdminType(targetType) {
		return false
	}

	if authz.IsAdminType(targetType) {
		if !actor.LevelPolicyEnabled {
			return true
		}
		return target.AuthLevel < actor.UserLevel
	}

	return true
}

// ValidateDelegation checks all permissions are in delegable allow-list.
func ValidateDelegation(permissionCodes []string, delegable map[string]struct{}) error {
	for _, code := range permissionCodes {
		if _, ok := delegable[code]; !ok {
			return errors.New("FORBIDDEN", "permission delegation is not allowed")
		}
	}
	return nil
}
