package user

import (
	"gin_starter/internal/config"
	"time"
)

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

type deps struct {
	repo   Repository
	config *config.Config
}

func NewDeps(repo Repository, cfg *config.Config) *deps {
	return &deps{repo: repo, config: cfg}
}

// Handler handles user HTTP endpoints.
type Handler struct {
	deps *deps
}

func NewHandler(deps *deps) *Handler {
	return &Handler{
		deps: deps,
	}
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
