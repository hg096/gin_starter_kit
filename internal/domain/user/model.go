package user

import (
	"time"
)

// User 사용자 모델
type User struct {
	ID           string    `json:"id" db:"u_id"`
	Password     string    `json:"-" db:"u_pass"` // JSON 응답에서 제외
	Name         string    `json:"name" db:"u_name"`
	Email        string    `json:"email" db:"u_email"`
	AuthType     string    `json:"auth_type" db:"u_auth_type"`
	AuthLevel    int       `json:"auth_level" db:"u_auth_level"`
	RefreshToken string    `json:"-" db:"u_re_token"` // JSON 응답에서 제외
	CreatedAt    time.Time `json:"created_at" db:"u_regi_date"`
}

// CreateUserRequest 회원가입 요청
type CreateUserRequest struct {
	ID       string `json:"user_id" binding:"required"`
	Password string `json:"user_pass" binding:"required"`
	Name     string `json:"user_name" binding:"required"`
	Email    string `json:"user_email" binding:"required,email"`
}

// UpdateUserRequest 회원정보 수정 요청
type UpdateUserRequest struct {
	Password string `json:"user_pass,omitempty"`
	Name     string `json:"user_name,omitempty"`
	Email    string `json:"user_email,omitempty"`
}

// LoginRequest 로그인 요청
type LoginRequest struct {
	ID       string `json:"user_id" binding:"required"`
	Password string `json:"user_pass" binding:"required"`
}

// LoginResponse 로그인 응답
type LoginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	User         *User  `json:"user"`
}

// RefreshTokenRequest 토큰 갱신 요청
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// RefreshTokenResponse 토큰 갱신 응답
type RefreshTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// ToPublic 비밀번호와 토큰 제거 후 반환
func (u *User) ToPublic() *User {
	return &User{
		ID:        u.ID,
		Name:      u.Name,
		Email:     u.Email,
		AuthType:  u.AuthType,
		AuthLevel: u.AuthLevel,
		CreatedAt: u.CreatedAt,
	}
}