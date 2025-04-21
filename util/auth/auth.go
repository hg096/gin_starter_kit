// auth.go
package auth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"gin_starter/model/core"
	"io"
	"log"
	"strings"

	"encoding/base64"
	"encoding/json"
	"os"
	"strconv"
	"time"

	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
)

// Claims는 액세스 토큰과 리프레시 토큰에 공통적으로 담을 클레임입니다.
type Claims struct {
	UserID string `json:"user_id"`
	jwt.RegisteredClaims
}

type EncryptedClaims struct {
	Data string `json:"data"` // Base64(AES‑GCM( JSON(Claims) ))
	jwt.RegisteredClaims
}

var (
	AccessSecret  = []byte(os.Getenv("JWT_SECRET"))
	RefreshSecret = []byte(os.Getenv("JWT_REFRESH_SECRET"))
	TokenSecret   = []byte(os.Getenv("JWT_TOKEN_SECRET"))
)

func init() {
	if len(AccessSecret) != 32 || len(RefreshSecret) != 32 || len(TokenSecret) != 32 {
		log.Fatal("JWT_SECRET, JWT_REFRESH_SECRET, JWT_TOKEN_SECRET 는 32자여야 합니다")
	}
	if len(AccessSecret) == 0 || len(RefreshSecret) == 0 || len(TokenSecret) == 0 {
		log.Fatal("JWT_SECRET, JWT_REFRESH_SECRET 모두 설정 필요")
	}
}

// 토큰 생성 함수
func NewEncryptedToken(userID string, expMin int, signingKey []byte, encryptionKey []byte,
) (string, error) {
	now := time.Now()

	// A) 원본 Claims JSON
	orig := Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(expMin) * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(now),
			Subject:   userID,
		},
	}
	raw, err := json.Marshal(orig)
	if err != nil {
		return "", err
	}

	// B) AES‑GCM 암호화
	cipherBytes, err := EncryptAESGCM(encryptionKey, raw)
	if err != nil {
		return "", err
	}
	// Base64-URL 인코딩
	dataB64 := base64.RawURLEncoding.EncodeToString(cipherBytes)

	// C) EncryptedClaims에 담아서 JWT 서명
	enc := EncryptedClaims{
		Data: dataB64,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(expMin) * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(now),
			Subject:   userID,
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, enc)
	return token.SignedString(signingKey)
}

// 토큰 검증 함수
func ValidateToken(tokenStr string, signingKey []byte, encryptionKey []byte,
) (*Claims, error) {
	// A) 서명 검증 & EncryptedClaims 채우기
	token, err := jwt.ParseWithClaims(tokenStr, &EncryptedClaims{}, func(t *jwt.Token) (interface{}, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return signingKey, nil
	})
	if err != nil {
		return nil, err
	}
	encClaims, ok := token.Claims.(*EncryptedClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	// B) Base64 → 바이트
	cipherBytes, err := base64.RawURLEncoding.DecodeString(encClaims.Data)
	if err != nil {
		return nil, err
	}

	// C) AES‑GCM 복호화 → 원본 JSON
	raw, err := DecryptAESGCM(encryptionKey, cipherBytes)
	if err != nil {
		return nil, err
	}

	// D) JSON → Claims 구조체
	var orig Claims
	if err := json.Unmarshal(raw, &orig); err != nil {
		return nil, err
	}

	return &orig, nil
}
func EncryptAESGCM(key, plaintext []byte) ([]byte, error) {
	// 1) AES 블록 생성
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("AES NewCipher: %w", err)
	}
	// 2) GCM 모드 생성
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("cipher NewGCM: %w", err)
	}
	// 3) 랜덤 nonce 생성
	nonce := make([]byte, aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("nonce 생성 실패: %w", err)
	}
	// 4) Seal: nonce || ciphertext||tag
	ciphertext := aead.Seal(nil, nonce, plaintext, nil)
	// 5) 결과에 nonce를 앞에 붙여 반환
	return append(nonce, ciphertext...), nil
}
func DecryptAESGCM(key, cipherData []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("AES NewCipher: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("cipher NewGCM: %w", err)
	}
	nonceSize := aead.NonceSize()
	if len(cipherData) < nonceSize {
		return nil, fmt.Errorf("cipherData 길이가 nonceSize(%d)보다 작음", nonceSize)
	}
	// nonce와 ciphertext 분리
	nonce, ciphertext := cipherData[:nonceSize], cipherData[nonceSize:]
	// Open: 복호화 및 인증 태그 검증
	plaintext, err := aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("AES-GCM 복호화 실패: %w", err)
	}
	return plaintext, nil
}

// GenerateTokens은 userID로 액세스/리프레시 토큰을 생성해 반환합니다.
func GenerateTokens(userID string) (accessToken, refreshToken string, err error) {
	// 액세스 토큰 만료(분)
	accessExpMin := 60
	if v := os.Getenv("JWT_EXPIRES_IN"); v != "" {
		if m, err := strconv.Atoi(v); err == nil {
			accessExpMin = m
		}
	}
	accessToken, err = NewEncryptedToken(userID, accessExpMin, AccessSecret, TokenSecret)
	// accessToken, err = at.SignedString(AccessSecret)
	if err != nil {
		return "", "", err
	}

	// 리프레시 토큰 만료(일)
	refreshExpMin := 60 * 24 * 7 // 기본 1주일
	if v := os.Getenv("JWT_EXPIRES_RE"); v != "" {
		if d, err := strconv.Atoi(v); err == nil {
			refreshExpMin = 60 * 24 * d
		}
	}
	refreshToken, err = NewEncryptedToken(userID, refreshExpMin, RefreshSecret, TokenSecret)
	// refreshToken, err = rt.SignedString(RefreshSecret)
	if err != nil {
		return "", "", err
	}

	return accessToken, refreshToken, nil
}

// RefreshHandler은 POST /refresh 에 매핑할 수 있는 Gin 핸들러로,
// JSON 바디로 받은 { "refresh_token": "..." } 를 검사해 새 토큰을 발급합니다.
func RefreshHandler(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "refresh_token 필수"})
		return
	}

	// 1) 리프레시 토큰 검증
	claims, err := ValidateToken(req.RefreshToken, RefreshSecret, TokenSecret)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "invalid refresh token"})
		return
	}

	// (선택) DB나 캐시에서 이 리프레시 토큰이 유효한지 재검증 → 탈취 방지

	// 2) 새 토큰 생성
	newAT, newRT, err := GenerateTokens(claims.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "토큰 생성 실패"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token":  newAT,
		"refresh_token": newRT,
	})
}

// 미들웨어 엑세스 토큰 검증
func JWTAuthMiddleware(lv int) gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.GetHeader("Authorization")
		parts := strings.SplitN(h, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "invalid token", "TOKEN": "N"})
			return
		}

		claims, err := ValidateToken(parts[1], AccessSecret, TokenSecret)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "invalid token", "TOKEN": "R"})
			return
		}

		if lv > 0 {
			result, err := core.BuildSelectQuery(c, nil, "select u_auth_type, u_auth_level from _user where u_id = ? ", []string{claims.UserID}, "JWTAuthMiddleware.err")
			if err != nil {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "invalid user"})
				return
			}
			u_auth_level, _ := strconv.Atoi(result[0]["u_auth_level"])
			if lv > u_auth_level {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "invalid user"})
				return
			}
		}

		c.Set("user_id", claims.UserID)
		c.Next()
	}
}
