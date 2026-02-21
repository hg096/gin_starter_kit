package admin

import (
	"gin_starter/internal/domain/blog"
	"gin_starter/internal/domain/user"
	"time"
)

// Actor contains authenticated admin context.
type Actor struct {
	ID                 string
	UserType           string
	UserLevel          int
	IsSuperAdmin       bool
	LevelPolicyEnabled bool
	IP                 string
}

// AdminUserListRequest is user list request payload.
type AdminUserListRequest struct {
	Page     int    `json:"page"`
	Limit    int    `json:"limit"`
	UserType string `json:"user_type"`
}

// AdminUserListResponse is user list response payload.
type AdminUserListResponse struct {
	Users []user.User `json:"users"`
	Total int64       `json:"total"`
	Page  int         `json:"page"`
	Limit int         `json:"limit"`
}

// AdminUpdateUserAuthRequest updates user auth type/level.
type AdminUpdateUserAuthRequest struct {
	AuthType  string `json:"auth_type"`
	AuthLevel int    `json:"auth_level"`
}

// AdminUpdateUserProfileRequest updates basic profile fields.
type AdminUpdateUserProfileRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

// AdminUpdateUserStatusRequest updates account status.
type AdminUpdateUserStatusRequest struct {
	Status string `json:"status"`
}

// AdminResetUserPasswordRequest sets a new password.
type AdminResetUserPasswordRequest struct {
	NewPassword string `json:"new_password"`
}

// AdminUpdateUserPermissionsRequest replaces assigned permissions.
type AdminUpdateUserPermissionsRequest struct {
	Permissions []string `json:"permissions"`
}

// AdminUpdateDelegablePermissionsRequest replaces delegable allow-list.
type AdminUpdateDelegablePermissionsRequest struct {
	Permissions []string `json:"permissions"`
}

// AdminUpdateLevelPolicyRequest toggles global auth-level policy.
type AdminUpdateLevelPolicyRequest struct {
	Enabled *bool `json:"enabled"`
}

// AdminLevelPolicyResponse is global auth-level policy state.
type AdminLevelPolicyResponse struct {
	Enabled bool `json:"enabled"`
}

// Permission describes a manageable permission code.
type Permission struct {
	Code        string `json:"code"`
	Description string `json:"description"`
}

// AdminStatsResponse is admin dashboard stats.
type AdminStatsResponse struct {
	TotalUsers  int64 `json:"total_users"`
	AdminUsers  int64 `json:"admin_users"`
	NormalUsers int64 `json:"normal_users"`
	TotalBlogs  int64 `json:"total_blogs"`
}

// AdminAuditLogListRequest is audit-log query payload.
type AdminAuditLogListRequest struct {
	Page         int    `json:"page"`
	Limit        int    `json:"limit"`
	Action       string `json:"action"`
	ActorID      string `json:"actor_id"`
	TargetUserID string `json:"target_user_id"`
	DateFrom     string `json:"date_from"`
	DateTo       string `json:"date_to"`
}

// AdminAuditLogItem is a single audit-log row.
type AdminAuditLogItem struct {
	ID           int64                  `json:"id"`
	ActorID      string                 `json:"actor_id"`
	TargetUserID string                 `json:"target_user_id"`
	Action       string                 `json:"action"`
	Status       string                 `json:"status"`
	Message      string                 `json:"message"`
	IPAddress    string                 `json:"ip_addr"`
	BeforeData   map[string]interface{} `json:"before_data"`
	AfterData    map[string]interface{} `json:"after_data"`
	CreatedAt    time.Time              `json:"created_at"`
}

// AdminAuditLogListResponse is paginated audit-log response.
type AdminAuditLogListResponse struct {
	Logs  []AdminAuditLogItem `json:"logs"`
	Total int64               `json:"total"`
	Page  int                 `json:"page"`
	Limit int                 `json:"limit"`
}

// AdminBlogListResponse is paginated blog list for admin console.
type AdminBlogListResponse struct {
	Blogs []blog.Blog `json:"blogs"`
	Total int64       `json:"total"`
	Page  int         `json:"page"`
	Limit int         `json:"limit"`
}

// AdminCreateBlogRequest creates blog post from admin console.
type AdminCreateBlogRequest struct {
	Title    string `json:"title"`
	Content  string `json:"content"`
	AuthorID string `json:"author_id"`
}

// AdminUpdateBlogRequest updates blog post from admin console.
type AdminUpdateBlogRequest struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}

// AdminBootstrapStatusResponse is public bootstrap availability response.
type AdminBootstrapStatusResponse struct {
	CanBootstrap    bool   `json:"can_bootstrap"`
	Reason          string `json:"reason,omitempty"`
	AdminCount      int64  `json:"admin_count"`
	SuperAdminCount int64  `json:"super_admin_count"`
}

// AdminBootstrapSuperAdminRequest creates the first/recovery top-admin.
type AdminBootstrapSuperAdminRequest struct {
	ID       string `json:"user_id"`
	Password string `json:"user_pass"`
	Name     string `json:"user_name"`
	Email    string `json:"user_email"`
}

// AdminPagePermissionCodes contains generated permission codes for one page.
type AdminPagePermissionCodes struct {
	Read   string `json:"read"`
	Create string `json:"create"`
	Update string `json:"update"`
	Delete string `json:"delete"`
}

// AdminPage represents a dynamic admin page registry item.
type AdminPage struct {
	Key             string                   `json:"page_key"`
	Title           string                   `json:"title"`
	Path            string                   `json:"path"`
	Description     string                   `json:"description"`
	GroupKey        string                   `json:"group_key"`
	GroupLabel      string                   `json:"group_label"`
	GroupOrder      int                      `json:"group_order"`
	VisibleRoles    []string                 `json:"visible_roles,omitempty"`
	Icon            string                   `json:"icon"`
	SortOrder       int                      `json:"sort_order"`
	Enabled         bool                     `json:"enabled"`
	Builtin         bool                     `json:"builtin"`
	PermissionCodes AdminPagePermissionCodes `json:"permission_codes"`
	CreatedBy       string                   `json:"created_by,omitempty"`
	CreatedAt       time.Time                `json:"created_at"`
	UpdatedAt       time.Time                `json:"updated_at"`
}

// AdminPageRouteSpec defines a route-driven page registration spec.
type AdminPageRouteSpec struct {
	PageKey      string   `json:"page_key"`
	Path         string   `json:"path"`
	Title        string   `json:"title"`
	Description  string   `json:"description"`
	GroupKey     string   `json:"group_key"`
	GroupLabel   string   `json:"group_label"`
	GroupOrder   int      `json:"group_order"`
	VisibleRoles []string `json:"visible_roles"`
	Icon         string   `json:"icon"`
	SortOrder    int      `json:"sort_order"`
}

// AdminCreatePageRequest creates a dynamic admin page.
type AdminCreatePageRequest struct {
	PageKey      string   `json:"page_key"`
	Title        string   `json:"title"`
	Description  string   `json:"description"`
	GroupKey     string   `json:"group_key"`
	GroupLabel   string   `json:"group_label"`
	GroupOrder   int      `json:"group_order"`
	VisibleRoles []string `json:"visible_roles"`
	Icon         string   `json:"icon"`
	SortOrder    int      `json:"sort_order"`
	Enabled      *bool    `json:"enabled"`
}

// AdminUpdatePageRequest updates dynamic admin page metadata.
type AdminUpdatePageRequest struct {
	Title        *string   `json:"title"`
	Description  *string   `json:"description"`
	GroupKey     *string   `json:"group_key"`
	GroupLabel   *string   `json:"group_label"`
	GroupOrder   *int      `json:"group_order"`
	VisibleRoles *[]string `json:"visible_roles"`
	Icon         *string   `json:"icon"`
	SortOrder    *int      `json:"sort_order"`
	Enabled      *bool     `json:"enabled"`
}
