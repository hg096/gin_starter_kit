package middleware

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"gin_starter/internal/config"
	"gin_starter/pkg/errors"
	"gin_starter/pkg/response"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
)

// Claims JWT 클레임 구조
type Claims struct {
	UserID string `json:"user_id"`
	jwt.RegisteredClaims
}

// EncryptedClaims 암호화된 클레임
type EncryptedClaims struct {
	Data string `json:"data"`
	jwt.RegisteredClaims
}

// AuthMiddleware JWT 인증 미들웨어
func AuthMiddleware(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			response.Unauthorized(c, "인증 토큰이 필요합니다")
			c.Abort()
			return
		}

		// Bearer 토큰 파싱
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			response.Unauthorized(c, "잘못된 토큰 형식입니다")
			c.Abort()
			return
		}

		tokenString := parts[1]

		// 토큰 검증
		claims, err := ValidateToken(tokenString, cfg.JWT.AccessSecret, cfg.JWT.TokenSecret)
		if err != nil {
			if errors.Is(err, errors.ErrExpiredToken) {
				response.TokenExpired(c)
			} else {
				response.TokenInvalid(c)
			}
			c.Abort()
			return
		}

		// 컨텍스트에 사용자 정보 저장
		c.Set("user_id", claims.UserID)
		c.Next()
	}
}

// RequireUserType 특정 사용자 타입 요구 미들웨어
func RequireUserType(userType string) gin.HandlerFunc {
	return func(c *gin.Context) {
		requestUserType, exists := c.Get("user_type")
		if !exists {
			response.Forbidden(c, "사용자 타입을 확인할 수 없습니다")
			c.Abort()
			return
		}

		if requestUserType != userType {
			response.Forbidden(c, "접근 권한이 없습니다")
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireAuthLevel 최소 권한 레벨 요구 미들웨어
func RequireAuthLevel(minLevel int) gin.HandlerFunc {
	return func(c *gin.Context) {
		userLevel, exists := c.Get("user_level")
		if !exists {
			response.Forbidden(c, "권한 레벨을 확인할 수 없습니다")
			c.Abort()
			return
		}

		level, ok := userLevel.(int)
		if !ok || level < minLevel {
			response.Forbidden(c, "접근 권한이 부족합니다")
			c.Abort()
			return
		}

		c.Next()
	}
}

// GenerateToken JWT 토큰 생성
func GenerateToken(userID string, expireMinutes int, signingKey, encryptionKey []byte, serviceName string) (string, error) {
	now := time.Now()

	// 페이로드 생성
	payload := struct {
		UserID string `json:"user_id"`
	}{
		UserID: userID,
	}

	raw, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	// AES-GCM 암호화
	cipherBytes, err := encryptAESGCM(encryptionKey, raw)
	if err != nil {
		return "", err
	}

	// Base64 인코딩
	dataB64 := base64.RawURLEncoding.EncodeToString(cipherBytes)

	// JWT 생성
	claims := EncryptedClaims{
		Data: dataB64,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(expireMinutes) * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(now),
			Subject:   serviceName,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(signingKey)
}

// ValidateToken JWT 토큰 검증
func ValidateToken(tokenStr string, signingKey, encryptionKey []byte) (*Claims, error) {
	// JWT 파싱 및 서명 검증
	token, err := jwt.ParseWithClaims(tokenStr, &EncryptedClaims{}, func(t *jwt.Token) (interface{}, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return signingKey, nil
	})

	if err != nil {
		return nil, errors.Wrap(err, "INVALID_TOKEN", "토큰 파싱 실패")
	}

	encClaims, ok := token.Claims.(*EncryptedClaims)
	if !ok || !token.Valid {
		return nil, errors.ErrInvalidToken
	}

	// 만료 체크
	if encClaims.ExpiresAt.Before(time.Now()) {
		return nil, errors.ErrExpiredToken
	}

	// Base64 디코딩
	cipherBytes, err := base64.RawURLEncoding.DecodeString(encClaims.Data)
	if err != nil {
		return nil, errors.Wrap(err, "INVALID_TOKEN", "Base64 디코딩 실패")
	}

	// AES-GCM 복호화
	raw, err := decryptAESGCM(encryptionKey, cipherBytes)
	if err != nil {
		return nil, errors.Wrap(err, "INVALID_TOKEN", "복호화 실패")
	}

	// JSON 언마샬
	var payload struct {
		UserID string `json:"user_id"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, errors.Wrap(err, "INVALID_TOKEN", "페이로드 파싱 실패")
	}

	return &Claims{
		UserID:           payload.UserID,
		RegisteredClaims: encClaims.RegisteredClaims,
	}, nil
}

// encryptAESGCM AES-GCM 암호화
func encryptAESGCM(key, plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, aead.NonceSize())
	// 실제 운영환경에서는 crypto/rand 사용 필요
	cipherText := aead.Seal(nil, nonce, plaintext, nil)

	return append(nonce, cipherText...), nil
}

// decryptAESGCM AES-GCM 복호화
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
		return nil, fmt.Errorf("cipherData가 너무 짧습니다")
	}

	nonce, cipherText := cipherData[:nonceSize], cipherData[nonceSize:]
	plaintext, err := aead.Open(nil, nonce, cipherText, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}