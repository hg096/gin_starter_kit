package middleware

import (
	"gin_starter/internal/config"
	"gin_starter/pkg/authz"
	appErrors "gin_starter/pkg/errors"
	"gin_starter/pkg/response"
	"gin_starter/pkg/utils"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	// AccessTokenCookieName is the cookie name used for access tokens.
	AccessTokenCookieName = "access_token"
	// RefreshTokenCookieName is the cookie name used for refresh tokens.
	RefreshTokenCookieName = "refresh_token"
)

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
	if err != nil || !utils.HasText(refreshToken) {
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
		requestUserTypeStr, exists := contextString(c, ContextUserType)
		if !exists {
			response.Forbidden(c, "failed to resolve user type")
			c.Abort()
			return
		}

		if authz.NormalizeAuthType(requestUserTypeStr) != authz.NormalizeAuthType(userType) {
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
		requestUserTypeStr, exists := contextString(c, ContextUserType)
		if !exists {
			response.Forbidden(c, "failed to resolve user type")
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
		if levelPolicyEnabled, ok := contextBool(c, ContextLevelPolicyEnabled); ok && !levelPolicyEnabled {
			c.Next()
			return
		}

		level, exists := contextInt(c, ContextUserLevel)
		if !exists {
			response.Forbidden(c, "failed to resolve auth level")
			c.Abort()
			return
		}

		if level < minLevel {
			response.Forbidden(c, "insufficient auth level")
			c.Abort()
			return
		}

		c.Next()
	}
}

func extractAccessToken(c *gin.Context) (string, error) {
	authHeader := strings.TrimSpace(c.GetHeader("Authorization"))
	if authHeader != "" {
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" || !utils.HasText(parts[1]) {
			return "", appErrors.New("INVALID_TOKEN", "invalid authorization header")
		}
		return strings.TrimSpace(parts[1]), nil
	}

	cookieToken, err := c.Cookie(AccessTokenCookieName)
	if err == nil && utils.HasText(cookieToken) {
		return strings.TrimSpace(cookieToken), nil
	}

	return "", appErrors.ErrUnauthorized
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
