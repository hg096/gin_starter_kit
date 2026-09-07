package websocket

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"gin_starter/pkg/utils"
	"sort"
	"strings"
	"time"
)

func normalizeMemberIDs(actorID string, memberIDs []string) []string {
	set := make(map[string]struct{}, len(memberIDs)+1)
	set[strings.TrimSpace(actorID)] = struct{}{}

	for _, memberID := range memberIDs {
		trimmed := strings.TrimSpace(memberID)
		if trimmed == "" {
			continue
		}
		set[trimmed] = struct{}{}
	}

	members := make([]string, 0, len(set))
	for memberID := range set {
		if memberID == "" {
			continue
		}
		members = append(members, memberID)
	}
	sort.Strings(members)
	return members
}

func buildGroupRoomKey() (string, error) {
	randomBytes := make([]byte, 4)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", err
	}
	suffix := hex.EncodeToString(randomBytes)
	return fmt.Sprintf("grp_%d_%s", time.Now().Unix(), suffix), nil
}

func buildDirectRoomKey(userA, userB string) string {
	a := normalizeRoomKeyPart(userA)
	b := normalizeRoomKeyPart(userB)
	if a > b {
		a, b = b, a
	}
	return "dm_" + a + "_" + b
}

func normalizeRoomKeyPart(value string) string {
	value = utils.TrimLower(value)
	if value == "" {
		return "unknown"
	}

	builder := strings.Builder{}
	builder.Grow(len(value))
	for _, ch := range value {
		if ch >= 'a' && ch <= 'z' {
			builder.WriteRune(ch)
			continue
		}
		if ch >= '0' && ch <= '9' {
			builder.WriteRune(ch)
			continue
		}
		builder.WriteRune('_')
	}

	normalized := strings.Trim(builder.String(), "_")
	if normalized == "" {
		return "unknown"
	}
	return normalized
}
