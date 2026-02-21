package admin

import (
	"fmt"
	"regexp"
	"strings"
)

const (
	PermissionPageManage = "admin.page.manage"

	PageActionRead   = "read"
	PageActionCreate = "create"
	PageActionUpdate = "update"
	PageActionDelete = "delete"
)

var (
	pageKeyPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{1,47}$`)
	pageActions    = []string{PageActionRead, PageActionCreate, PageActionUpdate, PageActionDelete}
)

func NormalizePageKey(raw string) (string, bool) {
	key := strings.ToLower(strings.TrimSpace(raw))
	if !pageKeyPattern.MatchString(key) {
		return "", false
	}
	return key, true
}

func IsValidPageAction(action string) bool {
	switch strings.ToLower(strings.TrimSpace(action)) {
	case PageActionRead, PageActionCreate, PageActionUpdate, PageActionDelete:
		return true
	default:
		return false
	}
}

func BuildPagePermissionCode(pageKey, action string) string {
	return fmt.Sprintf("admin.page.%s.%s", pageKey, action)
}

func BuildPagePermissionCodes(pageKey string) []string {
	codes := make([]string, 0, len(pageActions))
	for _, action := range pageActions {
		codes = append(codes, BuildPagePermissionCode(pageKey, action))
	}
	return codes
}
