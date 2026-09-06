package middleware

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"gin_starter/pkg/authz"
	appErrors "gin_starter/pkg/errors"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v4"
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
