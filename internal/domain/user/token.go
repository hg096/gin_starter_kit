package user

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	stderrors "errors"
	"gin_starter/pkg/authz"
	"gin_starter/pkg/errors"
	"gin_starter/pkg/logger"
	"gin_starter/pkg/middleware"
	"gin_starter/pkg/response"
	"gin_starter/pkg/utils"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const refreshTokenHashPrefix = "h1:"

// RefreshTokenRequest is refresh payload.
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// RefreshTokenResponse is refresh response payload.
type RefreshTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// RefreshToken refresh endpoint.
// @Summary 토큰 갱신
// @Tags User
// @Accept json
// @Produce json
// @Param body body RefreshTokenRequest false "리프레시 토큰"
// @Success 200 {object} response.Response
// @Router /api/user/refresh [post]
func (h *Handler) RefreshToken(c *gin.Context) {
	var req RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil && !stderrors.Is(err, io.EOF) {
		response.BadRequest(c, "잘못된 요청 형식입니다")
		return
	}

	if utils.EmptyString(strings.TrimSpace(req.RefreshToken)) {
		if cookieToken, err := c.Cookie(middleware.RefreshTokenCookieName); err == nil {
			req.RefreshToken = cookieToken
		}
	}

	if utils.EmptyString(strings.TrimSpace(req.RefreshToken)) {
		response.BadRequest(c, "리프레시 토큰이 필요합니다")
		return
	}

	claims, err := middleware.ValidateToken(
		req.RefreshToken,
		h.deps.config.JWT.RefreshSecret,
		h.deps.config.JWT.TokenSecret,
		h.deps.config.JWT.Issuer,
		h.deps.config.JWT.Audience,
		h.deps.config.JWT.Subject,
	)
	if err != nil {
		response.FromError(c, err)
		return
	}

	user, err := h.deps.repo.FindByID(claims.UserID)
	if err != nil {
		response.FromError(c, err)
		return
	}

	if utils.TrimLower(user.Status) == UserStatusLocked {
		response.FromError(c, errors.ErrAccountLocked)
		return
	}

	if user.TokenValidAfter != nil {
		if claims.IssuedAt == nil || !claims.IssuedAt.Time.After(*user.TokenValidAfter) {
			response.FromError(c, errors.ErrTokenStale)
			return
		}
	}

	levelPolicyEnabled, err := h.deps.repo.GetLevelPolicyEnabled()
	if err != nil {
		response.FromError(c, err)
		return
	}

	if authz.NormalizeAuthType(user.AuthType) != authz.NormalizeAuthType(claims.UserType) {
		response.FromError(c, errors.ErrTokenStale)
		return
	}
	if levelPolicyEnabled && user.AuthLevel != claims.UserLevel {
		response.FromError(c, errors.ErrTokenStale)
		return
	}

	matched, legacyToken := isRefreshTokenMatch(user.RefreshToken, req.RefreshToken)
	if !matched {
		logger.Warn("invalid refresh token: %s", claims.UserID)
		response.FromError(c, errors.ErrInvalidToken)
		return
	}

	if legacyToken {
		if err := h.deps.repo.UpdateRefreshToken(user.ID, hashRefreshToken(req.RefreshToken)); err != nil {
			response.FromError(c, err)
			return
		}
	}

	accessToken, err := middleware.GenerateToken(
		user.ID,
		user.AuthType,
		user.AuthLevel,
		h.deps.config.JWT.AccessExpireMin,
		h.deps.config.JWT.AccessSecret,
		h.deps.config.JWT.TokenSecret,
		h.deps.config.JWT.Issuer,
		h.deps.config.JWT.Audience,
		h.deps.config.JWT.Subject,
	)
	if err != nil {
		response.FromError(c, errors.Wrap(err, "TOKEN_GENERATION_FAILED", "failed to generate token"))
		return
	}

	newRefreshToken := req.RefreshToken
	if claims.ExpiresAt != nil && time.Until(claims.ExpiresAt.Time) < time.Duration(h.deps.config.JWT.RefreshReuseHours)*time.Hour {
		newRefreshToken, err = middleware.GenerateToken(
			user.ID,
			user.AuthType,
			user.AuthLevel,
			h.deps.config.JWT.RefreshExpireDays*24*60,
			h.deps.config.JWT.RefreshSecret,
			h.deps.config.JWT.TokenSecret,
			h.deps.config.JWT.Issuer,
			h.deps.config.JWT.Audience,
			h.deps.config.JWT.Subject,
		)
		if err != nil {
			response.FromError(c, errors.Wrap(err, "TOKEN_GENERATION_FAILED", "failed to generate token"))
			return
		}

		if err := h.deps.repo.UpdateRefreshToken(user.ID, hashRefreshToken(newRefreshToken)); err != nil {
			response.FromError(c, err)
			return
		}
	}

	logger.Info("token refreshed: %s", user.ID)

	tokens := &RefreshTokenResponse{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
	}
	h.setAuthCookies(c, tokens.AccessToken, tokens.RefreshToken)
	response.Success(c, tokens)
}

// Logout user logout endpoint.
// @Summary 로그아웃
// @Tags User
// @Security BearerAuth
// @Produce json
// @Success 200 {object} response.Response
// @Router /api/user/logout [post]
func (h *Handler) Logout(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	updates := map[string]interface{}{
		"u_re_token":          "",
		"u_token_valid_after": time.Now(),
	}
	if err := h.deps.repo.Update(userID, updates); err != nil {
		response.FromError(c, err)
		return
	}

	logger.Info("logout completed: %s", userID)
	h.clearAuthCookies(c)
	response.Success(c, gin.H{"message": "로그아웃되었습니다"})
}

func (h *Handler) setAuthCookies(c *gin.Context, accessToken, refreshToken string) {
	secure := h.cfg != nil && h.cfg.IsProduction()
	accessMaxAge := 30 * 60
	refreshMaxAge := 7 * 24 * 60 * 60

	if h.cfg != nil {
		if h.cfg.JWT.AccessExpireMin > 0 {
			accessMaxAge = h.cfg.JWT.AccessExpireMin * 60
		}
		if h.cfg.JWT.RefreshExpireDays > 0 {
			refreshMaxAge = h.cfg.JWT.RefreshExpireDays * 24 * 60 * 60
		}
	}

	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(middleware.AccessTokenCookieName, accessToken, accessMaxAge, "/", "", secure, true)
	c.SetCookie(middleware.RefreshTokenCookieName, refreshToken, refreshMaxAge, "/", "", secure, true)
}

func (h *Handler) clearAuthCookies(c *gin.Context) {
	secure := h.cfg != nil && h.cfg.IsProduction()

	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(middleware.AccessTokenCookieName, "", -1, "/", "", secure, true)
	c.SetCookie(middleware.RefreshTokenCookieName, "", -1, "/", "", secure, true)
}

func hashRefreshToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return refreshTokenHashPrefix + hex.EncodeToString(sum[:])
}

func isRefreshTokenMatch(storedToken, incomingToken string) (matched bool, legacyToken bool) {
	if utils.EmptyString(strings.TrimSpace(storedToken)) || utils.EmptyString(strings.TrimSpace(incomingToken)) {
		return false, false
	}

	if strings.HasPrefix(storedToken, refreshTokenHashPrefix) {
		hashedIncoming := hashRefreshToken(incomingToken)
		return subtle.ConstantTimeCompare([]byte(storedToken), []byte(hashedIncoming)) == 1, false
	}

	return subtle.ConstantTimeCompare([]byte(storedToken), []byte(incomingToken)) == 1, true
}
