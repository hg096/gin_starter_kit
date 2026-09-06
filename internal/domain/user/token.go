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
	req, ok := readRefreshTokenRequest(c)
	if !ok {
		return
	}

	tokens, err := h.service.RefreshToken(req)
	if err != nil {
		response.FromError(c, err)
		return
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

	if err := h.service.Logout(userID); err != nil {
		response.FromError(c, err)
		return
	}

	h.clearAuthCookies(c)
	response.Success(c, gin.H{"message": "로그아웃되었습니다"})
}

func readRefreshTokenRequest(c *gin.Context) (*RefreshTokenRequest, bool) {
	var req RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil && !stderrors.Is(err, io.EOF) {
		response.BadRequest(c, "잘못된 요청 형식입니다")
		return nil, false
	}

	if utils.EmptyString(strings.TrimSpace(req.RefreshToken)) {
		if cookieToken, err := c.Cookie(middleware.RefreshTokenCookieName); err == nil {
			req.RefreshToken = cookieToken
		}
	}

	if utils.EmptyString(strings.TrimSpace(req.RefreshToken)) {
		response.BadRequest(c, "리프레시 토큰이 필요합니다")
		return nil, false
	}

	return &req, true
}

func (s *service) RefreshToken(req *RefreshTokenRequest) (*RefreshTokenResponse, error) {
	claims, err := middleware.ValidateToken(
		req.RefreshToken,
		s.config.JWT.RefreshSecret,
		s.config.JWT.TokenSecret,
		s.config.JWT.Issuer,
		s.config.JWT.Audience,
		s.config.JWT.Subject,
	)
	if err != nil {
		return nil, err
	}

	user, err := s.repo.FindByID(claims.UserID)
	if err != nil {
		return nil, err
	}

	if utils.TrimLower(user.Status) == UserStatusLocked {
		return nil, errors.ErrAccountLocked
	}

	if user.TokenValidAfter != nil {
		if claims.IssuedAt == nil || !claims.IssuedAt.Time.After(*user.TokenValidAfter) {
			return nil, errors.ErrTokenStale
		}
	}

	levelPolicyEnabled, err := s.repo.GetLevelPolicyEnabled()
	if err != nil {
		return nil, err
	}

	if authz.NormalizeAuthType(user.AuthType) != authz.NormalizeAuthType(claims.UserType) {
		return nil, errors.ErrTokenStale
	}
	if levelPolicyEnabled && user.AuthLevel != claims.UserLevel {
		return nil, errors.ErrTokenStale
	}

	matched, legacyToken := isRefreshTokenMatch(user.RefreshToken, req.RefreshToken)
	if !matched {
		logger.Warn("invalid refresh token: %s", claims.UserID)
		return nil, errors.ErrInvalidToken
	}

	if legacyToken {
		if err := s.repo.UpdateRefreshToken(user.ID, hashRefreshToken(req.RefreshToken)); err != nil {
			return nil, err
		}
	}

	accessToken, err := middleware.GenerateToken(
		user.ID,
		user.AuthType,
		user.AuthLevel,
		s.config.JWT.AccessExpireMin,
		s.config.JWT.AccessSecret,
		s.config.JWT.TokenSecret,
		s.config.JWT.Issuer,
		s.config.JWT.Audience,
		s.config.JWT.Subject,
	)
	if err != nil {
		return nil, errors.Wrap(err, "TOKEN_GENERATION_FAILED", "failed to generate token")
	}

	newRefreshToken := req.RefreshToken
	if claims.ExpiresAt != nil && time.Until(claims.ExpiresAt.Time) < time.Duration(s.config.JWT.RefreshReuseHours)*time.Hour {
		newRefreshToken, err = middleware.GenerateToken(
			user.ID,
			user.AuthType,
			user.AuthLevel,
			s.config.JWT.RefreshExpireDays*24*60,
			s.config.JWT.RefreshSecret,
			s.config.JWT.TokenSecret,
			s.config.JWT.Issuer,
			s.config.JWT.Audience,
			s.config.JWT.Subject,
		)
		if err != nil {
			return nil, errors.Wrap(err, "TOKEN_GENERATION_FAILED", "failed to generate token")
		}

		if err := s.repo.UpdateRefreshToken(user.ID, hashRefreshToken(newRefreshToken)); err != nil {
			return nil, err
		}
	}

	logger.Info("token refreshed: %s", user.ID)

	return &RefreshTokenResponse{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
	}, nil
}

func (s *service) Logout(userID string) error {
	updates := map[string]interface{}{
		"u_re_token":          "",
		"u_token_valid_after": time.Now(),
	}
	if err := s.repo.Update(userID, updates); err != nil {
		return err
	}

	logger.Info("logout completed: %s", userID)
	return nil
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
