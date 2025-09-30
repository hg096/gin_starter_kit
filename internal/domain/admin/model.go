package admin

import "gin_starter/internal/domain/user"

// AdminUserListRequest 관리자 사용자 목록 조회 요청
type AdminUserListRequest struct {
	Page     int    `json:"page"`
	Limit    int    `json:"limit"`
	UserType string `json:"user_type"` // U, A, 전체는 빈 문자열
}

// AdminUserListResponse 관리자 사용자 목록 응답
type AdminUserListResponse struct {
	Users []user.User `json:"users"`
	Total int64       `json:"total"`
	Page  int         `json:"page"`
	Limit int         `json:"limit"`
}

// AdminUpdateUserAuthRequest 사용자 권한 수정 요청
type AdminUpdateUserAuthRequest struct {
	AuthType  string `json:"auth_type"`  // U, A
	AuthLevel int    `json:"auth_level"` // 1-10
}

// AdminStatsResponse 관리자 통계 응답
type AdminStatsResponse struct {
	TotalUsers  int64 `json:"total_users"`
	AdminUsers  int64 `json:"admin_users"`
	NormalUsers int64 `json:"normal_users"`
	TotalBlogs  int64 `json:"total_blogs"`
}