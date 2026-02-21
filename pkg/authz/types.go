package authz

import "strings"

const (
	AuthTypeTopAdmin    = "TA"
	AuthTypeAdmin       = "A"
	AuthTypeManager     = "M"
	AuthTypeGuest       = "G"
	AuthTypeGuestLegacy = "AG"
	AuthTypeUser        = "U"
)

// NormalizeAuthType converts input user type to canonical code.
func NormalizeAuthType(raw string) string {
	normalized := strings.ToUpper(strings.TrimSpace(raw))
	switch normalized {
	case AuthTypeGuestLegacy:
		return AuthTypeGuest
	default:
		return normalized
	}
}

// IsAdminType returns true for all admin-console account types.
func IsAdminType(userType string) bool {
	switch NormalizeAuthType(userType) {
	case AuthTypeTopAdmin, AuthTypeAdmin, AuthTypeManager, AuthTypeGuest:
		return true
	default:
		return false
	}
}

// IsTopAdminType returns true for the dedicated top-admin role.
func IsTopAdminType(userType string) bool {
	return NormalizeAuthType(userType) == AuthTypeTopAdmin
}

// IsLegacySuperAdmin returns true for old A + level 10 format.
func IsLegacySuperAdmin(userType string, userLevel int) bool {
	return NormalizeAuthType(userType) == AuthTypeAdmin && userLevel == 10
}

// IsSuperAdmin returns true for TA and legacy A+10.
func IsSuperAdmin(userType string, userLevel int) bool {
	return IsTopAdminType(userType) || IsLegacySuperAdmin(userType, userLevel)
}

// IsAssignableAdminType returns true for types that can receive admin permissions.
func IsAssignableAdminType(userType string) bool {
	switch NormalizeAuthType(userType) {
	case AuthTypeTopAdmin, AuthTypeAdmin, AuthTypeManager, AuthTypeGuest:
		return true
	default:
		return false
	}
}
