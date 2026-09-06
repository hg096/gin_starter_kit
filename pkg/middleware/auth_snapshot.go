package middleware

import (
	"database/sql"
	"gin_starter/pkg/authz"
	"gin_starter/pkg/db/database"
	appErrors "gin_starter/pkg/errors"
	"gin_starter/pkg/utils"
	"strings"
	"time"
)

type authSnapshot struct {
	UserType           string
	UserLevel          int
	Status             string
	TokenValidAfter    *time.Time
	Permissions        []string
	LevelPolicyEnabled bool
}

var authSnapshotLoader = loadAuthSnapshotFromDB

func setAuthSnapshotLoaderForTest(loader func(string) (*authSnapshot, error)) func() {
	prev := authSnapshotLoader
	authSnapshotLoader = loader
	return func() {
		authSnapshotLoader = prev
	}
}

func validateSnapshotAgainstClaims(snapshot *authSnapshot, claims *Claims, levelPolicyEnabled bool) error {
	if snapshot == nil {
		return nil
	}

	status := utils.TrimLower(snapshot.Status)
	if status == "" {
		status = "active"
	}
	if status != "active" {
		return appErrors.ErrAccountLocked
	}

	if authz.NormalizeAuthType(snapshot.UserType) != authz.NormalizeAuthType(claims.UserType) {
		return appErrors.ErrTokenStale
	}

	if levelPolicyEnabled && snapshot.UserLevel != claims.UserLevel {
		return appErrors.ErrTokenStale
	}

	if snapshot.TokenValidAfter != nil {
		if claims.IssuedAt == nil || !claims.IssuedAt.Time.After(*snapshot.TokenValidAfter) {
			return appErrors.ErrTokenStale
		}
	}

	return nil
}

func loadAuthSnapshotFromDB(userID string) (*authSnapshot, error) {
	db := database.GetDB()
	if db == nil || db.DB == nil {
		return nil, nil
	}

	row := db.QueryRow(`SELECT u_auth_type, u_auth_level, COALESCE(u_status, 'active'), u_token_valid_after FROM _user WHERE u_id = ?`, userID)
	var snapshot authSnapshot
	var tokenValidAfter sql.NullTime
	if err := row.Scan(&snapshot.UserType, &snapshot.UserLevel, &snapshot.Status, &tokenValidAfter); err != nil {
		if err == sql.ErrNoRows {
			return nil, appErrors.ErrUnauthorized
		}
		return nil, err
	}
	if tokenValidAfter.Valid {
		t := tokenValidAfter.Time
		snapshot.TokenValidAfter = &t
	}
	snapshot.UserType = authz.NormalizeAuthType(snapshot.UserType)

	levelPolicyEnabled, err := loadLevelPolicyEnabledFromDB(db)
	if err != nil {
		return nil, err
	}
	snapshot.LevelPolicyEnabled = levelPolicyEnabled

	rows, err := db.Query(`SELECT permission_code FROM _a_user_permissions WHERE u_id = ?`, userID)
	if err != nil {
		if isMissingTableError(err) {
			return &snapshot, nil
		}
		return nil, err
	}
	defer rows.Close()

	permissions := make([]string, 0, 8)
	for rows.Next() {
		var permission string
		if scanErr := rows.Scan(&permission); scanErr != nil {
			return nil, scanErr
		}
		permissions = append(permissions, permission)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	snapshot.Permissions = permissions
	return &snapshot, nil
}

func loadLevelPolicyEnabledFromDB(db *database.DB) (bool, error) {
	if db == nil || db.DB == nil {
		return true, nil
	}

	var raw sql.NullString
	err := db.QueryRow(`SELECT setting_value FROM _a_system_settings WHERE setting_key = 'level_policy_enabled' LIMIT 1`).Scan(&raw)
	if err != nil {
		if err == sql.ErrNoRows || isMissingTableError(err) {
			return true, nil
		}
		return false, err
	}

	value := utils.TrimLower(raw.String)
	switch value {
	case "", "1", "true", "on", "yes", "y":
		return true, nil
	case "0", "false", "off", "no", "n":
		return false, nil
	default:
		return true, nil
	}
}

func isMissingTableError(err error) bool {
	if err == nil {
		return false
	}
	msg := utils.TrimLower(err.Error())
	return strings.Contains(msg, "doesn't exist") || strings.Contains(msg, "no such table")
}
