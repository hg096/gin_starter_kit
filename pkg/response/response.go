package response

import (
	stderrors "errors"
	appErrors "gin_starter/pkg/errors"
	"gin_starter/pkg/logger"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// Response is the standard API response shape.
type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *ErrorInfo  `json:"error,omitempty"`
	Meta    *Meta       `json:"meta,omitempty"`
}

// ErrorInfo carries error response details.
type ErrorInfo struct {
	Code    string                 `json:"code"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// Meta contains pagination metadata.
type Meta struct {
	Page       int `json:"page,omitempty"`
	PerPage    int `json:"per_page,omitempty"`
	Total      int `json:"total,omitempty"`
	TotalPages int `json:"total_pages,omitempty"`
}

func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    data,
	})
}

func SuccessWithMeta(c *gin.Context, data interface{}, meta *Meta) {
	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    data,
		Meta:    meta,
	})
}

func Created(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, Response{
		Success: true,
		Data:    data,
	})
}

func NoContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
}

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

func BadRequest(c *gin.Context, message string, details ...map[string]interface{}) {
	Error(c, http.StatusBadRequest, "BAD_REQUEST", message, details...)
}

func Unauthorized(c *gin.Context, message string) {
	Error(c, http.StatusUnauthorized, "UNAUTHORIZED", message)
}

func Forbidden(c *gin.Context, message string) {
	Error(c, http.StatusForbidden, "FORBIDDEN", message)
}

func NotFound(c *gin.Context, message string) {
	Error(c, http.StatusNotFound, "NOT_FOUND", message)
}

func Conflict(c *gin.Context, message string) {
	Error(c, http.StatusConflict, "CONFLICT", message)
}

func ValidationError(c *gin.Context, details map[string]interface{}) {
	Error(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "입력값 검증에 실패했습니다", details)
}

func InternalError(c *gin.Context, message string) {
	Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", message)
}

func TokenExpired(c *gin.Context) {
	Error(c, http.StatusUnauthorized, "TOKEN_EXPIRED", "토큰이 만료되었습니다")
}

func TokenInvalid(c *gin.Context) {
	Error(c, http.StatusUnauthorized, "TOKEN_INVALID", "유효하지 않은 토큰입니다")
}

// FromError maps AppError to HTTP response while masking internal messages.
func FromError(c *gin.Context, err error) {
	if err == nil {
		logger.Error("response.FromError called with nil error")
		InternalError(c, "내부 서버 오류가 발생했습니다")
		return
	}

	var appErr *appErrors.AppError
	if !stderrors.As(err, &appErr) {
		logger.Error("unhandled non-app error: %v", err)
		InternalError(c, "내부 서버 오류가 발생했습니다")
		return
	}

	statusCode := mapErrorStatus(appErr.Code)
	message := appErr.Message

	if statusCode >= 500 || shouldMaskCode(appErr.Code) {
		logger.Error("request failed with masked internal error (code=%s): %v", appErr.Code, err)
		statusCode = http.StatusInternalServerError
		message = "내부 서버 오류가 발생했습니다"
	}

	Error(c, statusCode, appErr.Code, message)
}

func mapErrorStatus(code string) int {
	switch code {
	case "UNAUTHORIZED", "INVALID_CREDENTIALS", "INVALID_TOKEN", "EXPIRED_TOKEN", "TOKEN_INVALID", "TOKEN_EXPIRED", "TOKEN_STALE":
		return http.StatusUnauthorized
	case "FORBIDDEN", "ACCOUNT_LOCKED":
		return http.StatusForbidden
	case "NOT_FOUND", "USER_NOT_FOUND", "RECORD_NOT_FOUND", "BLOG_NOT_FOUND":
		return http.StatusNotFound
	case "CONFLICT", "USER_EXISTS", "DUPLICATE_ENTRY":
		return http.StatusConflict
	case "VALIDATION_ERROR":
		return http.StatusUnprocessableEntity
	case "BAD_REQUEST", "NO_UPDATES", "NO_UPDATE_DATA", "INVALID_AUTH_TYPE", "INVALID_AUTH_LEVEL", "TITLE_REQUIRED", "TITLE_LENGTH", "CONTENT_REQUIRED", "CONTENT_LENGTH":
		return http.StatusBadRequest
	default:
		if shouldMaskCode(code) {
			return http.StatusInternalServerError
		}
		return http.StatusBadRequest
	}
}

func shouldMaskCode(code string) bool {
	if code == "INTERNAL_ERROR" || code == "DATABASE_ERROR" {
		return true
	}

	return strings.HasSuffix(code, "_FAILED")
}
