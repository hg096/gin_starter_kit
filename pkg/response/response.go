package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Response 표준 API 응답 구조
type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *ErrorInfo  `json:"error,omitempty"`
	Meta    *Meta       `json:"meta,omitempty"`
}

// ErrorInfo 에러 상세 정보
type ErrorInfo struct {
	Code    string                 `json:"code"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// Meta 페이지네이션 등 메타 정보
type Meta struct {
	Page       int `json:"page,omitempty"`
	PerPage    int `json:"per_page,omitempty"`
	Total      int `json:"total,omitempty"`
	TotalPages int `json:"total_pages,omitempty"`
}

// Success 성공 응답
func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    data,
	})
}

// SuccessWithMeta 메타 정보를 포함한 성공 응답
func SuccessWithMeta(c *gin.Context, data interface{}, meta *Meta) {
	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    data,
		Meta:    meta,
	})
}

// Created 리소스 생성 성공 응답
func Created(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, Response{
		Success: true,
		Data:    data,
	})
}

// NoContent 내용 없는 성공 응답 (삭제 등)
func NoContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
}

// Error 에러 응답
func Error(c *gin.Context, statusCode int, code string, message string, details ...map[string]interface{}) {
	errorInfo := &ErrorInfo{
		Code:    code,
		Message: message,
	}

	if len(details) > 0 {
		errorInfo.Details = details[0]
	}

	c.JSON(statusCode, Response{
		Success: false,
		Error:   errorInfo,
	})
}

// BadRequest 400 에러
func BadRequest(c *gin.Context, message string, details ...map[string]interface{}) {
	Error(c, http.StatusBadRequest, "BAD_REQUEST", message, details...)
}

// Unauthorized 401 에러
func Unauthorized(c *gin.Context, message string) {
	Error(c, http.StatusUnauthorized, "UNAUTHORIZED", message)
}

// Forbidden 403 에러
func Forbidden(c *gin.Context, message string) {
	Error(c, http.StatusForbidden, "FORBIDDEN", message)
}

// NotFound 404 에러
func NotFound(c *gin.Context, message string) {
	Error(c, http.StatusNotFound, "NOT_FOUND", message)
}

// Conflict 409 에러
func Conflict(c *gin.Context, message string) {
	Error(c, http.StatusConflict, "CONFLICT", message)
}

// ValidationError 유효성 검증 실패
func ValidationError(c *gin.Context, details map[string]interface{}) {
	Error(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "입력값 검증에 실패했습니다", details)
}

// InternalError 500 에러
func InternalError(c *gin.Context, message string) {
	Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", message)
}

// TokenExpired 토큰 만료
func TokenExpired(c *gin.Context) {
	Error(c, http.StatusUnauthorized, "TOKEN_EXPIRED", "토큰이 만료되었습니다")
}

// TokenInvalid 토큰 무효
func TokenInvalid(c *gin.Context) {
	Error(c, http.StatusUnauthorized, "TOKEN_INVALID", "유효하지 않은 토큰입니다")
}