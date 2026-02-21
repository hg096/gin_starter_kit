package middleware

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"gin_starter/internal/config"
	"gin_starter/internal/infrastructure/database"
	"gin_starter/pkg/authz"
	appErrors "gin_starter/pkg/errors"
	"gin_starter/pkg/response"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
)

const (
	// AccessTokenCookieName is the cookie name used for access tokens.
	AccessTokenCookieName = "access_token"
	// RefreshTokenCookieName is the cookie name used for refresh tokens.
	RefreshTokenCookieName = "refresh_token"
)

// Claims represents decrypted JWT claims.
type Claims struct {
	UserID    string `json:"user_id"`
	UserType  string `json:"user_type"`
	UserLevel int    `json:"user_level"`
	jwt.RegisteredClaims
}

// EncryptedClaims stores encrypted payload data in JWT claims.
type EncryptedClaims struct {
	Data string `json:"data"`
	jwt.RegisteredClaims
}

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

// AuthMiddleware validates access tokens from Authorization header first, then cookie.
func AuthMiddleware(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString, err := extractAccessToken(c)
		if err != nil {
			response.Unauthorized(c, "authentication token is required")
			c.Abort()
			return
		}

		claims, err := ValidateToken(
			tokenString,
			cfg.JWT.AccessSecret,
			cfg.JWT.TokenSecret,
			cfg.JWT.Issuer,
			cfg.JWT.Audience,
			cfg.JWT.Subject,
		)
		if err != nil {
			handleTokenError(c, err)
			return
		}

		snapshot, err := authSnapshotLoader(claims.UserID)
		if err != nil {
			response.TokenInvalid(c)
			c.Abort()
			return
		}

		levelPolicyEnabled := true
		if snapshot != nil {
			levelPolicyEnabled = snapshot.LevelPolicyEnabled
			if err := validateSnapshotAgainstClaims(snapshot, claims, levelPolicyEnabled); err != nil {
				handleTokenError(c, err)
				return
			}

			setAuthContext(c, claims.UserID, snapshot.UserType, snapshot.UserLevel, snapshot.Permissions, levelPolicyEnabled)
		} else {
			setAuthContext(c, claims.UserID, claims.UserType, claims.UserLevel, nil, levelPolicyEnabled)
		}

		c.Next()
	}
}

// AdminPageAuthMiddleware protects admin page routes.
func AdminPageAuthMiddleware(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, err := validateAccessTokenFromRequest(c, cfg)
		if err != nil {
			claims, err = reissueAccessTokenFromRefreshCookie(c, cfg)
			if err != nil {
				c.Redirect(http.StatusFound, "/admin/login")
				c.Abort()
				return
			}
		}

		snapshot, err := authSnapshotLoader(claims.UserID)
		if err != nil {
			c.Redirect(http.StatusFound, "/admin/login")
			c.Abort()
			return
		}

		userType := claims.UserType
		userLevel := claims.UserLevel
		permissions := []string(nil)
		levelPolicyEnabled := true

		if snapshot != nil {
			levelPolicyEnabled = snapshot.LevelPolicyEnabled
			if err := validateSnapshotAgainstClaims(snapshot, claims, levelPolicyEnabled); err != nil {
				if appErrors.Is(err, appErrors.ErrTokenStale) || appErrors.Is(err, appErrors.ErrExpiredToken) || appErrors.Is(err, appErrors.ErrInvalidToken) {
					c.Redirect(http.StatusFound, "/admin/login")
				} else {
					c.String(http.StatusForbidden, "forbidden")
				}
				c.Abort()
				return
			}

			userType = snapshot.UserType
			userLevel = snapshot.UserLevel
			permissions = snapshot.Permissions
		}

		if !authz.IsAdminType(userType) {
			c.String(http.StatusForbidden, "forbidden")
			c.Abort()
			return
		}

		setAuthContext(c, claims.UserID, userType, userLevel, permissions, levelPolicyEnabled)
		c.Next()
	}
}

func validateAccessTokenFromRequest(c *gin.Context, cfg *config.Config) (*Claims, error) {
	tokenString, err := extractAccessToken(c)
	if err != nil {
		return nil, err
	}

	return ValidateToken(
		tokenString,
		cfg.JWT.AccessSecret,
		cfg.JWT.TokenSecret,
		cfg.JWT.Issuer,
		cfg.JWT.Audience,
		cfg.JWT.Subject,
	)
}

// reissueAccessTokenFromRefreshCookie refreshes admin page session without forcing a login redirect.
func reissueAccessTokenFromRefreshCookie(c *gin.Context, cfg *config.Config) (*Claims, error) {
	refreshToken, err := c.Cookie(RefreshTokenCookieName)
	if err != nil || strings.TrimSpace(refreshToken) == "" {
		return nil, appErrors.ErrUnauthorized
	}

	refreshClaims, err := ValidateToken(
		refreshToken,
		cfg.JWT.RefreshSecret,
		cfg.JWT.TokenSecret,
		cfg.JWT.Issuer,
		cfg.JWT.Audience,
		cfg.JWT.Subject,
	)
	if err != nil {
		return nil, err
	}

	newAccessToken, err := GenerateToken(
		refreshClaims.UserID,
		refreshClaims.UserType,
		refreshClaims.UserLevel,
		cfg.JWT.AccessExpireMin,
		cfg.JWT.AccessSecret,
		cfg.JWT.TokenSecret,
		cfg.JWT.Issuer,
		cfg.JWT.Audience,
		cfg.JWT.Subject,
	)
	if err != nil {
		return nil, err
	}

	secure := cfg != nil && cfg.IsProduction()
	maxAge := 30 * 60
	if cfg != nil && cfg.JWT.AccessExpireMin > 0 {
		maxAge = cfg.JWT.AccessExpireMin * 60
	}
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(AccessTokenCookieName, newAccessToken, maxAge, "/", "", secure, true)

	return ValidateToken(
		newAccessToken,
		cfg.JWT.AccessSecret,
		cfg.JWT.TokenSecret,
		cfg.JWT.Issuer,
		cfg.JWT.Audience,
		cfg.JWT.Subject,
	)
}

// RequireUserType ensures a specific user type.
func RequireUserType(userType string) gin.HandlerFunc {
	return func(c *gin.Context) {
		requestUserType, exists := c.Get("user_type")
		if !exists {
			response.Forbidden(c, "failed to resolve user type")
			c.Abort()
			return
		}

		requestUserTypeStr, ok := requestUserType.(string)
		if !ok || authz.NormalizeAuthType(requestUserTypeStr) != authz.NormalizeAuthType(userType) {
			response.Forbidden(c, "forbidden")
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireUserTypes ensures user type matches one of allowed values.
func RequireUserTypes(userTypes ...string) gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(userTypes))
	for _, userType := range userTypes {
		normalized := authz.NormalizeAuthType(userType)
		if normalized == "" {
			continue
		}
		allowed[normalized] = struct{}{}
	}

	return func(c *gin.Context) {
		requestUserType, exists := c.Get("user_type")
		if !exists {
			response.Forbidden(c, "failed to resolve user type")
			c.Abort()
			return
		}

		requestUserTypeStr, ok := requestUserType.(string)
		if !ok {
			response.Forbidden(c, "forbidden")
			c.Abort()
			return
		}

		if _, exists := allowed[authz.NormalizeAuthType(requestUserTypeStr)]; !exists {
			response.Forbidden(c, "forbidden")
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireAuthLevel ensures the user has minimum auth level.
func RequireAuthLevel(minLevel int) gin.HandlerFunc {
	return func(c *gin.Context) {
		if enabled, ok := c.Get("level_policy_enabled"); ok {
			if levelPolicyEnabled, castOK := enabled.(bool); castOK && !levelPolicyEnabled {
				c.Next()
				return
			}
		}

		userLevel, exists := c.Get("user_level")
		if !exists {
			response.Forbidden(c, "failed to resolve auth level")
			c.Abort()
			return
		}

		level, ok := userLevel.(int)
		if !ok || level < minLevel {
			response.Forbidden(c, "insufficient auth level")
			c.Abort()
			return
		}

		c.Next()
	}
}

// GenerateToken creates signed JWT with AES-GCM encrypted payload.
func GenerateToken(
	userID,
	userType string,
	userLevel,
	expireMinutes int,
	signingKey,
	encryptionKey []byte,
	issuer,
	audience,
	subject string,
) (string, error) {
	now := time.Now()
	userType = authz.NormalizeAuthType(userType)

	payload := struct {
		UserID    string `json:"user_id"`
		UserType  string `json:"user_type"`
		UserLevel int    `json:"user_level"`
	}{
		UserID:    userID,
		UserType:  userType,
		UserLevel: userLevel,
	}

	raw, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	cipherBytes, err := encryptAESGCM(encryptionKey, raw)
	if err != nil {
		return "", err
	}

	dataB64 := base64.RawURLEncoding.EncodeToString(cipherBytes)

	claims := EncryptedClaims{
		Data: dataB64,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    issuer,
			Subject:   subject,
			Audience:  jwt.ClaimStrings{audience},
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(expireMinutes) * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(signingKey)
}

// ValidateToken validates JWT signature, registered claims, and decrypts payload.
func ValidateToken(
	tokenStr string,
	signingKey,
	encryptionKey []byte,
	issuer,
	audience,
	subject string,
) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &EncryptedClaims{}, func(t *jwt.Token) (interface{}, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return signingKey, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))

	if err != nil {
		if ve, ok := err.(*jwt.ValidationError); ok {
			if ve.Errors&jwt.ValidationErrorExpired != 0 {
				return nil, appErrors.ErrExpiredToken
			}
		}
		return nil, appErrors.Wrap(err, "INVALID_TOKEN", "token parsing failed")
	}

	encClaims, ok := token.Claims.(*EncryptedClaims)
	if !ok || !token.Valid {
		return nil, appErrors.ErrInvalidToken
	}

	now := time.Now()
	if !encClaims.VerifyExpiresAt(now, true) {
		return nil, appErrors.ErrExpiredToken
	}
	if !encClaims.VerifyIssuedAt(now.Add(time.Minute), true) {
		return nil, appErrors.ErrInvalidToken
	}
	if issuer != "" && !encClaims.VerifyIssuer(issuer, true) {
		return nil, appErrors.ErrInvalidToken
	}
	if audience != "" && !encClaims.VerifyAudience(audience, true) {
		return nil, appErrors.ErrInvalidToken
	}
	if subject != "" && strings.TrimSpace(encClaims.Subject) != subject {
		return nil, appErrors.ErrInvalidToken
	}

	cipherBytes, err := base64.RawURLEncoding.DecodeString(encClaims.Data)
	if err != nil {
		return nil, appErrors.Wrap(err, "INVALID_TOKEN", "base64 decoding failed")
	}

	raw, err := decryptAESGCM(encryptionKey, cipherBytes)
	if err != nil {
		return nil, appErrors.Wrap(err, "INVALID_TOKEN", "payload decryption failed")
	}

	var payload struct {
		UserID    string `json:"user_id"`
		UserType  string `json:"user_type"`
		UserLevel int    `json:"user_level"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, appErrors.Wrap(err, "INVALID_TOKEN", "payload parsing failed")
	}

	return &Claims{
		UserID:           payload.UserID,
		UserType:         authz.NormalizeAuthType(payload.UserType),
		UserLevel:        payload.UserLevel,
		RegisteredClaims: encClaims.RegisteredClaims,
	}, nil
}

func extractAccessToken(c *gin.Context) (string, error) {
	authHeader := strings.TrimSpace(c.GetHeader("Authorization"))
	if authHeader != "" {
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" || strings.TrimSpace(parts[1]) == "" {
			return "", appErrors.New("INVALID_TOKEN", "invalid authorization header")
		}
		return strings.TrimSpace(parts[1]), nil
	}

	cookieToken, err := c.Cookie(AccessTokenCookieName)
	if err == nil && strings.TrimSpace(cookieToken) != "" {
		return strings.TrimSpace(cookieToken), nil
	}

	return "", appErrors.ErrUnauthorized
}

func encryptAESGCM(key, plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := aead.NonceSize()
	if nonceSize <= 0 {
		return nil, fmt.Errorf("invalid nonce size: %d", nonceSize)
	}

	nonce := make([]byte, nonceSize)
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}

	cipherText := aead.Seal(nil, nonce, plaintext, nil)
	return append(nonce, cipherText...), nil
}

func decryptAESGCM(key, cipherData []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := aead.NonceSize()
	if len(cipherData) < nonceSize {
		return nil, fmt.Errorf("cipher data is too short")
	}

	nonce, cipherText := cipherData[:nonceSize], cipherData[nonceSize:]
	plaintext, err := aead.Open(nil, nonce, cipherText, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}

func handleTokenError(c *gin.Context, err error) {
	if appErrors.Is(err, appErrors.ErrExpiredToken) {
		response.TokenExpired(c)
		c.Abort()
		return
	}
	if appErrors.Is(err, appErrors.ErrAccountLocked) {
		response.Forbidden(c, "account is locked")
		c.Abort()
		return
	}
	if appErrors.Is(err, appErrors.ErrTokenStale) {
		response.Error(c, http.StatusUnauthorized, "TOKEN_STALE", "token has been invalidated")
		c.Abort()
		return
	}

	response.TokenInvalid(c)
	c.Abort()
}

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

	c.Set("user_id", userID)
	c.Set("user_type", normalizedUserType)
	c.Set("user_level", userLevel)
	c.Set("is_super_admin", authz.IsSuperAdmin(normalizedUserType, userLevel))
	c.Set("user_permissions", permSet)
	c.Set("user_permissions_list", permissions)
	c.Set("level_policy_enabled", levelPolicyEnabled)
}

func validateSnapshotAgainstClaims(snapshot *authSnapshot, claims *Claims, levelPolicyEnabled bool) error {
	if snapshot == nil {
		return nil
	}

	status := strings.ToLower(strings.TrimSpace(snapshot.Status))
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

	value := strings.ToLower(strings.TrimSpace(raw.String))
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
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "doesn't exist") || strings.Contains(msg, "no such table")
}
