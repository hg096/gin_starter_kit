// auth.go
package auth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"gin_starter/model/dbCore"
	"gin_starter/util/utilCore"
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
	"github.com/joho/godotenv"
)

// Claims는 액세스 토큰과 리프레시 토큰에 공통적으로 담을 클레임
type Claims struct {
	JWTUserID string `json:"user_id"`
	jwt.RegisteredClaims
}

type EncryptedClaims struct {
	Data string `json:"data"` // Base64(AES‑GCM( JSON(Claims) ))
	jwt.RegisteredClaims
}

var (
	AccessSecret  []byte
	RefreshSecret []byte
	TokenSecret   []byte
)

func init() {
	// .env 파일 로드 (이미 main 에서도 로드하시면 중복 호출해도 무해)
	if err := godotenv.Load(); err != nil {
		log.Fatal("[종료] Error loading .env file in auth package")
		// log.Println("⚠️ .env 파일 로드:", err)
	}

	// 이제 환경변수를 읽어서 패키지 변수에 할당
	AccessSecret = []byte(os.Getenv("JWT_SECRET"))
	RefreshSecret = []byte(os.Getenv("JWT_REFRESH_SECRET"))
	TokenSecret = []byte(os.Getenv("JWT_TOKEN_SECRET"))

	if len(AccessSecret) != 32 || len(RefreshSecret) != 32 || len(TokenSecret) != 32 {
		log.Fatal("[종료] JWT_SECRET, JWT_REFRESH_SECRET, JWT_TOKEN_SECRET 는 32자여야 합니다")
	}
	if len(AccessSecret) == 0 || len(RefreshSecret) == 0 || len(TokenSecret) == 0 {
		log.Fatal("[종료] JWT_SECRET, JWT_REFRESH_SECRET 모두 설정 필요")
	}
}

// 토큰 생성 함수
func NewEncryptedToken(userID string, expMin int, signingKey []byte, encryptionKey []byte,
) (string, error) {
	now := time.Now()

	serviceName := os.Getenv("SERVICE_NAME")
	if utilCore.EmptyString(serviceName) {
		serviceName = "ginStart"
	}

	payload := struct {
		UserID string `json:"user_id"`
	}{
		UserID: userID,
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	// AES‑GCM 암호화
	cipherBytes, err := EncryptAESGCM(encryptionKey, raw)
	if err != nil {
		return "", err
	}
	// Base64-URL 인코딩
	dataB64 := base64.RawURLEncoding.EncodeToString(cipherBytes)

	// EncryptedClaims에 담아서 JWT 서명
	enc := EncryptedClaims{
		Data: dataB64,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(expMin) * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(now),
			Subject:   serviceName,
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, enc)
	return token.SignedString(signingKey)
}

// 토큰 검증 함수
func ValidateToken(tokenStr string, signingKey []byte, encryptionKey []byte,
) (*Claims, error) {
	// 서명 검증 & EncryptedClaims 채우기
	token, err := jwt.ParseWithClaims(tokenStr, &EncryptedClaims{}, func(t *jwt.Token) (interface{}, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return signingKey, nil
	})
	if err != nil {
		return nil, err
	}
	enc, ok := token.Claims.(*EncryptedClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	// Base64 → 바이트
	cipherBytes, err := base64.RawURLEncoding.DecodeString(enc.Data)
	if err != nil {
		return nil, err
	}

	// AES‑GCM 복호화 → 원본 JSON
	raw, err := DecryptAESGCM(encryptionKey, cipherBytes)
	if err != nil {
		return nil, err
	}

	var p struct {
		JWTUserID string `json:"user_id"`
	}
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, err
	}

	return &Claims{
		JWTUserID:        p.JWTUserID,
		RegisteredClaims: enc.RegisteredClaims,
	}, nil
}

func EncryptAESGCM(key, plaintext []byte) ([]byte, error) {
	// AES 블록 생성
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("AES NewCipher: %w", err)
	}
	// GCM 모드 생성
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("cipher NewGCM: %w", err)
	}
	// 랜덤 nonce 생성
	nonce := make([]byte, aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("nonce 생성 실패: %w", err)
	}
	// Seal: nonce || cipherText||tag
	cipherText := aead.Seal(nil, nonce, plaintext, nil)
	// 결과에 nonce를 앞에 붙여 반환
	return append(nonce, cipherText...), nil
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
	// nonce와 cipherText 분리
	nonce, cipherText := cipherData[:nonceSize], cipherData[nonceSize:]
	// Open: 복호화 및 인증 태그 검증
	plaintext, err := aead.Open(nil, nonce, cipherText, nil)
	if err != nil {
		return nil, fmt.Errorf("AES-GCM 복호화 실패: %w", err)
	}
	return plaintext, nil
}

// GenerateTokens은 userID로 액세스/리프레시 토큰을 생성해 반환, 24시간 이내면 리프레시 토큰 재사용
func GenerateTokens(userID string, refreshTokenPrev string) (accessToken string, refreshToken string, err error) {
	// 액세스 토큰 만료(분)
	accessExpMin := 30
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

	// 24시간 이내면 리프레시 토큰 재사용
	if refreshTokenPrev != "" {
		if claims, err := ValidateToken(refreshTokenPrev, RefreshSecret, TokenSecret); err == nil {
			if time.Until(claims.ExpiresAt.Time) >= 24*time.Hour {
				return accessToken, refreshTokenPrev, nil
			}
		}
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

// RefreshHandler은 POST /refresh 에 매핑할 수 있는 Gin 핸들러 JSON 바디로 받은 { "refresh_token": "..." } 를 검사해 새 토큰을 발급
func RefreshHandler(c *gin.Context, postData map[string]string) (accessToken string, refreshToken string, errMsg string) {

	if postData["refresh_token"] == "" {
		return "", "", "fn auth/RefreshHandler-missingToken"
	}

	// fmt.Println("RefreshHandler postData")
	// fmt.Println(postData["refresh_token"])

	// 리프레시 토큰 검증
	claims, err := ValidateToken(postData["refresh_token"], RefreshSecret, TokenSecret)
	if err != nil {
		return "", "", "fn auth/RefreshHandler-missingToken"
	}

	// 디비 검증
	resultUser, err := dbCore.BuildSelectQuery(c, nil, "select u_re_token from _user where u_re_token = ? ", []string{postData["refresh_token"]}, "RefreshHandler.err")
	if err != nil || utilCore.EmptyString(resultUser[0]["u_re_token"]) {
		return "", "", "fn auth/RefreshHandler-BuildSelectQuery"
	}

	// 새 토큰 생성
	newAT, newRT, err := GenerateTokens(claims.JWTUserID, postData["refresh_token"])
	if err != nil {
		return "", "", "fn auth/RefreshHandler-GenerateTokens"
	}

	_, err = dbCore.BuildUpdateQuery(c, nil, "_user", map[string]string{"u_re_token": newRT}, "u_re_token = ?", []string{postData["refresh_token"]}, "fn auth/RefreshHandler-BuildUpdateQuery")
	if err != nil {
		return "", "", "fn auth/RefreshHandler-BuildUpdateQuery"
	}

	return newAT, newRT, ""
}

// 미들웨어 엑세스 토큰 검증 - 사용자 타입, 레벨
func ApiCheckLogin(userType string, lv int8) gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.GetHeader("Authorization")
		parts := strings.SplitN(h, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			utilCore.EndResponse(c, http.StatusBadRequest, gin.H{"TOKEN": "N"}, "fn auth/ApiCheckLogin")
			return
		}

		claims, err := ValidateToken(parts[1], AccessSecret, TokenSecret)
		if err != nil {
			utilCore.EndResponse(c, http.StatusBadRequest, gin.H{"TOKEN": "R"}, "fn auth/ApiCheckLogin-ValidateToken")
			return
		}

		result, err := dbCore.BuildSelectQuery(c, nil, "select u_auth_type, u_auth_level from _user where u_id = ? ", []string{claims.JWTUserID}, "ApiCheckLogin.err")
		if err != nil {
			utilCore.EndResponse(c, http.StatusBadRequest, gin.H{}, "fn auth/ApiCheckLogin-BuildSelectQuery")
			return
		}

		// 사용자 타입 찾기
		if result[0]["u_auth_type"] != userType {
			// 만약에 타입이 두가지 이상 들어가야할때
			index := strings.Index(userType, result[0]["u_auth_type"])
			if index < 0 {
				utilCore.EndResponse(c, http.StatusBadRequest, gin.H{}, "fn auth/ApiCheckLogin-type")
			}
			return
		}

		// 등급 레벨 조건이 맞는지 확인
		if lv > 0 {
			u_auth_level, _ := utilCore.StringToNumeric[int8](result[0]["u_auth_level"])
			if lv > u_auth_level {
				utilCore.EndResponse(c, http.StatusBadRequest, gin.H{}, "fn auth/ApiCheckLogin-level")
				return
			}
		}

		// 기본설정이 더 필요할때 여기서 추가
		c.Set("user_id", claims.JWTUserID)
		c.Set("user_type", result[0]["u_auth_type"])
		c.Set("user_level", result[0]["u_auth_level"])

		c.Next()
	}
}
