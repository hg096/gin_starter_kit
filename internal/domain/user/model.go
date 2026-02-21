package user

import "time"

const (
	UserStatusActive = "active"
	UserStatusLocked = "locked"
)

// User is the domain user model.
type User struct {
	ID              string     `json:"id" db:"u_id"`
	Password        string     `json:"-" db:"u_pass"`
	Name            string     `json:"name" db:"u_name"`
	Email           string     `json:"email" db:"u_email"`
	AuthType        string     `json:"auth_type" db:"u_auth_type"`
	AuthLevel       int        `json:"auth_level" db:"u_auth_level"`
	Status          string     `json:"status" db:"u_status"`
	TokenValidAfter *time.Time `json:"-" db:"u_token_valid_after"`
	RefreshToken    string     `json:"-" db:"u_re_token"`
	CreatedAt       time.Time  `json:"created_at" db:"u_regi_date"`
}

// AuthSnapshot represents security-sensitive fields loaded per request.
type AuthSnapshot struct {
	ID              string
	AuthType        string
	AuthLevel       int
	Status          string
	TokenValidAfter *time.Time
}

// CreateUserRequest is register payload.
type CreateUserRequest struct {
	ID       string `json:"user_id" binding:"required"`
	Password string `json:"user_pass" binding:"required"`
	Name     string `json:"user_name" binding:"required"`
	Email    string `json:"user_email" binding:"required,email"`
}

// UpdateUserRequest is profile update payload.
type UpdateUserRequest struct {
	Password string `json:"user_pass,omitempty"`
	Name     string `json:"user_name,omitempty"`
	Email    string `json:"user_email,omitempty"`
}

// LoginRequest is login payload.
type LoginRequest struct {
	ID       string `json:"user_id" binding:"required"`
	Password string `json:"user_pass" binding:"required"`
}

// LoginResponse is login response payload.
type LoginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	User         *User  `json:"user"`
}

// RefreshTokenRequest is refresh payload.
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// RefreshTokenResponse is refresh response payload.
type RefreshTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// ToPublic strips sensitive values.
func (u *User) ToPublic() *User {
	return &User{
		ID:              u.ID,
		Name:            u.Name,
		Email:           u.Email,
		AuthType:        u.AuthType,
		AuthLevel:       u.AuthLevel,
		Status:          u.Status,
		TokenValidAfter: u.TokenValidAfter,
		CreatedAt:       u.CreatedAt,
	}
}
