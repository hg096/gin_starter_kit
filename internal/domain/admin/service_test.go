package admin

import (
	"database/sql"
	"gin_starter/internal/domain/user"
	appErrors "gin_starter/pkg/errors"
	"strings"
	"testing"
)

func TestBuildUserListQueries_UsesRegiDateColumn(t *testing.T) {
	query, countQuery, args := buildUserListQueries("")

	if !strings.Contains(query, "u_regi_date") {
		t.Fatalf("expected query to use u_regi_date, got: %s", query)
	}
	if strings.Contains(query, "u_reg_date") {
		t.Fatalf("query should not include deprecated column u_reg_date: %s", query)
	}
	if countQuery != "SELECT COUNT(*) FROM _user" {
		t.Fatalf("unexpected count query: %s", countQuery)
	}
	if len(args) != 0 {
		t.Fatalf("expected no args for empty filter, got %d", len(args))
	}
}

func TestBuildUserListQueries_WithUserTypeFilter(t *testing.T) {
	query, countQuery, args := buildUserListQueries("A")

	if !strings.Contains(query, "WHERE u_auth_type = ?") {
		t.Fatalf("expected user type filter in query: %s", query)
	}
	if !strings.Contains(countQuery, "WHERE u_auth_type = ?") {
		t.Fatalf("expected user type filter in count query: %s", countQuery)
	}
	if len(args) != 1 || args[0] != "A" {
		t.Fatalf("unexpected args: %#v", args)
	}
}

func TestBuildUserListQueries_GuestFilterIncludesLegacyCode(t *testing.T) {
	query, countQuery, args := buildUserListQueries("G")

	if !strings.Contains(query, "u_auth_type IN (?, ?)") {
		t.Fatalf("expected guest filter to include IN clause, got: %s", query)
	}
	if !strings.Contains(countQuery, "u_auth_type IN (?, ?)") {
		t.Fatalf("expected guest filter in count query, got: %s", countQuery)
	}
	if len(args) != 2 || args[0] != "G" || args[1] != "AG" {
		t.Fatalf("unexpected args for guest filter: %#v", args)
	}
}

func TestCanManageTarget_HierarchyRules(t *testing.T) {
	super := Actor{ID: "sa", UserType: "TA", UserLevel: 0, IsSuperAdmin: true, LevelPolicyEnabled: true}
	normalAdmin := Actor{ID: "admin5", UserType: "A", UserLevel: 5, IsSuperAdmin: false, LevelPolicyEnabled: true}
	levelDisabledAdmin := Actor{ID: "admin5", UserType: "A", UserLevel: 5, IsSuperAdmin: false, LevelPolicyEnabled: false}

	lowerAdmin := &user.User{ID: "admin3", AuthType: "A", AuthLevel: 3}
	sameLevelAdmin := &user.User{ID: "admin5b", AuthType: "A", AuthLevel: 5}
	higherAdmin := &user.User{ID: "admin7", AuthType: "A", AuthLevel: 7}
	topAdmin := &user.User{ID: "top", AuthType: "TA", AuthLevel: 0}
	regularUser := &user.User{ID: "user1", AuthType: "U", AuthLevel: 1}

	if !CanManageTarget(super, lowerAdmin) || !CanManageTarget(super, sameLevelAdmin) || !CanManageTarget(super, higherAdmin) || !CanManageTarget(super, topAdmin) {
		t.Fatalf("super-admin should manage all targets")
	}

	if !CanManageTarget(normalAdmin, lowerAdmin) {
		t.Fatalf("normal admin should manage lower-level admin")
	}
	if CanManageTarget(normalAdmin, sameLevelAdmin) {
		t.Fatalf("normal admin should not manage same-level admin")
	}
	if CanManageTarget(normalAdmin, higherAdmin) {
		t.Fatalf("normal admin should not manage higher-level admin")
	}
	if CanManageTarget(normalAdmin, topAdmin) {
		t.Fatalf("normal admin should not manage top-admin")
	}
	if !CanManageTarget(normalAdmin, regularUser) {
		t.Fatalf("normal admin should manage regular users")
	}
	if !CanManageTarget(levelDisabledAdmin, higherAdmin) {
		t.Fatalf("level-disabled admin should manage higher-level admin when policy is disabled")
	}
}

func TestValidateDelegation_AllowList(t *testing.T) {
	allow := map[string]struct{}{
		"admin.account.status.update":     {},
		"admin.account.password.reset":    {},
		"admin.account.permission.manage": {},
	}

	if err := ValidateDelegation([]string{"admin.account.status.update"}, allow); err != nil {
		t.Fatalf("expected allowed delegation, got %v", err)
	}

	if err := ValidateDelegation([]string{"admin.account.delete"}, allow); err == nil {
		t.Fatalf("expected disallowed delegation to fail")
	}
}

type auditRepoStub struct {
	lastReq *AdminAuditLogListRequest
	resp    *AdminAuditLogListResponse
	err     error
}

func (s *auditRepoStub) ListPermissions() ([]Permission, error) { return nil, nil }
func (s *auditRepoStub) ListAuditLogs(req *AdminAuditLogListRequest) (*AdminAuditLogListResponse, error) {
	s.lastReq = req
	if s.resp != nil {
		return s.resp, s.err
	}
	return &AdminAuditLogListResponse{}, s.err
}
func (s *auditRepoStub) GetLevelPolicyEnabled() (bool, error) { return true, nil }
func (s *auditRepoStub) SetLevelPolicyEnabledTx(tx *sql.Tx, enabled bool, updatedBy string) error {
	return nil
}
func (s *auditRepoStub) ListUserPermissions(userID string) ([]string, error) { return nil, nil }
func (s *auditRepoStub) ReplaceUserPermissionsTx(tx *sql.Tx, userID string, permissionCodes []string) error {
	return nil
}
func (s *auditRepoStub) ListDelegablePermissions() ([]string, error) { return nil, nil }
func (s *auditRepoStub) DelegableSet() (map[string]struct{}, error) {
	return map[string]struct{}{}, nil
}
func (s *auditRepoStub) ReplaceDelegablePermissionsTx(tx *sql.Tx, permissionCodes []string) error {
	return nil
}
func (s *auditRepoStub) PermissionCodesExist(codes []string) (bool, error) { return true, nil }
func (s *auditRepoStub) GrantPermissionFromExistingTx(tx *sql.Tx, targetCode, sourceCode string) error {
	return nil
}
func (s *auditRepoStub) WriteAuditLogTx(tx *sql.Tx, entry *AuditLogEntry) error { return nil }
func (s *auditRepoStub) WriteAuditLog(entry *AuditLogEntry) error               { return nil }
func (s *auditRepoStub) DeleteUserPermissionsTx(tx *sql.Tx, userID string) error {
	return nil
}

func TestGetAuditLogs_ValidatesDateRange(t *testing.T) {
	repo := &auditRepoStub{}
	svc := &service{permissionRepo: repo}

	_, err := svc.GetAuditLogs(1, 20, "", "", "", "2026-13-01", "")
	if !appErrors.Is(err, appErrors.ErrBadRequest) {
		t.Fatalf("expected BAD_REQUEST for invalid date_from, got %v", err)
	}

	_, err = svc.GetAuditLogs(1, 20, "", "", "", "2026-02-21", "2026-02-20")
	if !appErrors.Is(err, appErrors.ErrBadRequest) {
		t.Fatalf("expected BAD_REQUEST for reversed date range, got %v", err)
	}
}

func TestGetAuditLogs_NormalizesDateToExclusiveUpperBound(t *testing.T) {
	repo := &auditRepoStub{}
	svc := &service{permissionRepo: repo}

	_, err := svc.GetAuditLogs(2, 10, " admin.account.permission.manage ", " actor1 ", " target1 ", "2026-02-01", "2026-02-07")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if repo.lastReq == nil {
		t.Fatalf("expected repository to receive request")
	}
	if repo.lastReq.Action != "admin.account.permission.manage" {
		t.Fatalf("unexpected action normalization: %q", repo.lastReq.Action)
	}
	if repo.lastReq.ActorID != "actor1" || repo.lastReq.TargetUserID != "target1" {
		t.Fatalf("unexpected actor/target normalization: actor=%q target=%q", repo.lastReq.ActorID, repo.lastReq.TargetUserID)
	}
	if repo.lastReq.DateFrom != "2026-02-01" {
		t.Fatalf("unexpected date_from: %q", repo.lastReq.DateFrom)
	}
	if repo.lastReq.DateTo != "2026-02-08" {
		t.Fatalf("expected exclusive date_to next day, got %q", repo.lastReq.DateTo)
	}
}

func TestEvaluateBootstrapState(t *testing.T) {
	can, reason := evaluateBootstrapState(&bootstrapState{AdminCount: 0, SuperAdminCount: 0})
	if !can || reason != "no_admin_account" {
		t.Fatalf("unexpected state for no admin: can=%v reason=%q", can, reason)
	}

	can, reason = evaluateBootstrapState(&bootstrapState{AdminCount: 3, SuperAdminCount: 0})
	if !can || reason != "no_super_admin_account" {
		t.Fatalf("unexpected state for no super-admin: can=%v reason=%q", can, reason)
	}

	can, reason = evaluateBootstrapState(&bootstrapState{AdminCount: 3, SuperAdminCount: 1})
	if can || reason != "" {
		t.Fatalf("unexpected state for healthy admin hierarchy: can=%v reason=%q", can, reason)
	}
}

func TestNormalizeBootstrapRequest(t *testing.T) {
	req := &AdminBootstrapSuperAdminRequest{
		ID:       " root01 ",
		Password: " secret123 ",
		Name:     " Root Admin ",
		Email:    " root@example.com ",
	}
	if err := normalizeBootstrapRequest(req); err != nil {
		t.Fatalf("expected valid bootstrap request, got %v", err)
	}
	if req.ID != "root01" || req.Password != "secret123" || req.Name != "Root Admin" || req.Email != "root@example.com" {
		t.Fatalf("expected normalization to trim fields, got %+v", req)
	}
}

func TestNormalizeAdminPageRouteSpecs(t *testing.T) {
	specs := []AdminPageRouteSpec{
		{
			PageKey:     " dashboard ",
			Path:        "/admin/",
			Title:       "Dashboard",
			Description: "Main dashboard",
			Icon:        "D",
			SortOrder:   10,
		},
		{
			PageKey:   "reports",
			Path:      "/admin/reports/",
			Title:     "",
			SortOrder: 0,
		},
		{
			PageKey: "bad",
			Path:    "/console/bad",
		},
		{
			PageKey: "reports",
			Path:    "/admin/reports2",
		},
	}

	normalized := normalizeAdminPageRouteSpecs(specs)
	if len(normalized) != 2 {
		t.Fatalf("expected 2 valid route specs, got %d", len(normalized))
	}

	if normalized[0].PageKey != "dashboard" || normalized[0].Path != "/admin" {
		t.Fatalf("unexpected normalized dashboard spec: %+v", normalized[0])
	}
	if normalized[1].PageKey != "reports" || normalized[1].Path != "/admin/reports" {
		t.Fatalf("unexpected normalized reports spec: %+v", normalized[1])
	}
	if normalized[1].Title != "reports" {
		t.Fatalf("expected fallback title to use page key, got %q", normalized[1].Title)
	}
	if normalized[1].SortOrder != 100 {
		t.Fatalf("expected default sort order 100, got %d", normalized[1].SortOrder)
	}
}

func TestKnownPermissionDescriptionsForCodes_RecognizesCoreAndBuiltinPage(t *testing.T) {
	svc := &service{}

	got := svc.knownPermissionDescriptionsForCodes([]string{
		"admin.audit.read",
		"admin.page.admin_chat.read",
		"unknown.permission.code",
	})

	if _, exists := got["admin.audit.read"]; !exists {
		t.Fatalf("expected admin.audit.read to be recognized")
	}
	if _, exists := got["admin.page.admin_chat.read"]; !exists {
		t.Fatalf("expected admin.page.admin_chat.read to be recognized")
	}
	if _, exists := got["unknown.permission.code"]; exists {
		t.Fatalf("did not expect unknown permission to be recognized")
	}
}
