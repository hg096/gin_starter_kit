package errors

import (
	"fmt"
)

// AppError 애플리케이션 에러
type AppError struct {
	Code    string                 // 에러 코드
	Message string                 // 에러 메시지
	Err     error                  // 원본 에러
	Meta    map[string]interface{} // 추가 메타데이터
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

// New 새로운 AppError 생성
func New(code, message string) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
	}
}

// Wrap 기존 에러를 AppError로 래핑
func Wrap(err error, code, message string) *AppError {
	if err == nil {
		return nil
	}
	return &AppError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// WithMeta 메타데이터 추가
func (e *AppError) WithMeta(key string, value interface{}) *AppError {
	if e.Meta == nil {
		e.Meta = make(map[string]interface{})
	}
	e.Meta[key] = value
	return e
}

// 미리 정의된 에러들
var (
	// 일반 에러
	ErrInternal        = New("INTERNAL_ERROR", "내부 서버 오류가 발생했습니다")
	ErrBadRequest      = New("BAD_REQUEST", "잘못된 요청입니다")
	ErrNotFound        = New("NOT_FOUND", "요청한 리소스를 찾을 수 없습니다")
	ErrUnauthorized    = New("UNAUTHORIZED", "인증이 필요합니다")
	ErrForbidden       = New("FORBIDDEN", "접근 권한이 없습니다")
	ErrConflict        = New("CONFLICT", "리소스 충돌이 발생했습니다")
	ErrValidation      = New("VALIDATION_ERROR", "입력값 검증에 실패했습니다")

	// 데이터베이스 에러
	ErrDatabase        = New("DATABASE_ERROR", "데이터베이스 오류가 발생했습니다")
	ErrDuplicateEntry  = New("DUPLICATE_ENTRY", "이미 존재하는 데이터입니다")
	ErrRecordNotFound  = New("RECORD_NOT_FOUND", "데이터를 찾을 수 없습니다")

	// 인증 에러
	ErrInvalidToken    = New("INVALID_TOKEN", "유효하지 않은 토큰입니다")
	ErrExpiredToken    = New("EXPIRED_TOKEN", "만료된 토큰입니다")
	ErrInvalidPassword = New("INVALID_PASSWORD", "비밀번호가 일치하지 않습니다")

	// 사용자 에러
	ErrUserNotFound    = New("USER_NOT_FOUND", "사용자를 찾을 수 없습니다")
	ErrUserExists      = New("USER_EXISTS", "이미 존재하는 사용자입니다")
	ErrInvalidCredentials = New("INVALID_CREDENTIALS", "아이디 또는 비밀번호가 잘못되었습니다")
)

// Is 에러 타입 확인
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