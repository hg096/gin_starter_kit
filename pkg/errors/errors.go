package errors

import "fmt"

// AppError is the standard application error type.
type AppError struct {
	Code    string
	Message string
	Err     error
	Meta    map[string]interface{}
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

func (e *AppError) Unwrap() error {
	return e.Err
}

// New creates a new AppError.
func New(code, message string) *AppError {
	return &AppError{Code: code, Message: message}
}

// Wrap wraps an existing error as AppError.
func Wrap(err error, code, message string) *AppError {
	if err == nil {
		return nil
	}
	return &AppError{Code: code, Message: message, Err: err}
}

// WithMeta appends metadata.
func (e *AppError) WithMeta(key string, value interface{}) *AppError {
	if e.Meta == nil {
		e.Meta = make(map[string]interface{})
	}
	e.Meta[key] = value
	return e
}

var (
	ErrInternal           = New("INTERNAL_ERROR", "internal server error")
	ErrBadRequest         = New("BAD_REQUEST", "invalid request")
	ErrNotFound           = New("NOT_FOUND", "resource not found")
	ErrUnauthorized       = New("UNAUTHORIZED", "authentication is required")
	ErrForbidden          = New("FORBIDDEN", "forbidden")
	ErrConflict           = New("CONFLICT", "resource conflict")
	ErrValidation         = New("VALIDATION_ERROR", "validation failed")
	ErrDatabase           = New("DATABASE_ERROR", "database error")
	ErrDuplicateEntry     = New("DUPLICATE_ENTRY", "duplicate entry")
	ErrRecordNotFound     = New("RECORD_NOT_FOUND", "record not found")
	ErrInvalidToken       = New("INVALID_TOKEN", "invalid token")
	ErrExpiredToken       = New("EXPIRED_TOKEN", "expired token")
	ErrTokenStale         = New("TOKEN_STALE", "token was invalidated by policy changes")
	ErrAccountLocked      = New("ACCOUNT_LOCKED", "account is locked")
	ErrInvalidPassword    = New("INVALID_PASSWORD", "invalid password")
	ErrUserNotFound       = New("USER_NOT_FOUND", "user not found")
	ErrUserExists         = New("USER_EXISTS", "user already exists")
	ErrInvalidCredentials = New("INVALID_CREDENTIALS", "invalid credentials")
)

// Is checks whether err has the same code as target.
func Is(err error, target *AppError) bool {
	if err == nil || target == nil {
		return false
	}

	appErr, ok := err.(*AppError)
	if !ok {
		return false
	}

	return appErr.Code == target.Code
}
