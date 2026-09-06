package admin

import (
	"gin_starter/pkg/errors"
	"gin_starter/pkg/response"
	"gin_starter/pkg/utils"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

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

// GetAuditLogs returns admin audit logs.
// @Summary      List admin audit logs
// @Tags         admin
// @Produce      json
// @Param        page query int false "page" default(1)
// @Param        limit query int false "limit (max 100)" default(20)
// @Param        action query string false "action filter"
// @Param        actor_id query string false "actor id filter"
// @Param        target_user_id query string false "target user id filter"
// @Param        date_from query string false "start date (YYYY-MM-DD)"
// @Param        date_to query string false "end date (YYYY-MM-DD)"
// @Success      200 {object} response.Response{data=AdminAuditLogListResponse}
// @Failure      400 {object} response.Response
// @Failure      401 {object} response.Response
// @Failure      403 {object} response.Response
// @Security     BearerAuth
// @Router       /api/admin/audit-logs [get]
func (h *Handler) GetAuditLogs(c *gin.Context) {
	pagination := utils.PaginationFromQuery(c, 20, 100)
	trimQuery := func(key string) string {
		return strings.TrimSpace(c.Query(key))
	}

	result, err := h.service.GetAuditLogs(
		pagination.Page,
		pagination.Limit,
		trimQuery("action"),
		trimQuery("actor_id"),
		trimQuery("target_user_id"),
		trimQuery("date_from"),
		trimQuery("date_to"),
	)
	if err != nil {
		response.FromError(c, err)
		return
	}

	response.Success(c, result)
}

func (s *service) GetAuditLogs(page, limit int, action, actorID, targetID, dateFrom, dateTo string) (*AdminAuditLogListResponse, error) {
	req := &AdminAuditLogListRequest{
		Page:         page,
		Limit:        limit,
		Action:       strings.TrimSpace(action),
		ActorID:      strings.TrimSpace(actorID),
		TargetUserID: strings.TrimSpace(targetID),
		DateFrom:     strings.TrimSpace(dateFrom),
		DateTo:       strings.TrimSpace(dateTo),
	}

	var parsedFrom time.Time
	var parsedTo time.Time
	var hasFrom bool
	var hasTo bool

	if utils.HasText(req.DateFrom) {
		t, err := time.Parse("2006-01-02", req.DateFrom)
		if err != nil {
			return nil, errors.New("BAD_REQUEST", "date_from must be YYYY-MM-DD")
		}
		parsedFrom = t
		hasFrom = true
		req.DateFrom = t.Format("2006-01-02")
	}

	if utils.HasText(req.DateTo) {
		t, err := time.Parse("2006-01-02", req.DateTo)
		if err != nil {
			return nil, errors.New("BAD_REQUEST", "date_to must be YYYY-MM-DD")
		}
		parsedTo = t
		hasTo = true
		// Repository uses an exclusive upper bound, so pass next day.
		req.DateTo = t.AddDate(0, 0, 1).Format("2006-01-02")
	}

	if hasFrom && hasTo && parsedFrom.After(parsedTo) {
		return nil, errors.New("BAD_REQUEST", "date_from must be before or equal to date_to")
	}

	return s.permissionRepo.ListAuditLogs(req)
}

func (s *service) denyWithAudit(actor Actor, targetID, action, message string) error {
	err := s.permissionRepo.WriteAuditLog(&AuditLogEntry{
		ActorID:  actor.ID,
		TargetID: targetID,
		Action:   action,
		Status:   "denied",
		Message:  message,
		IP:       actor.IP,
	})
	if err != nil {
		return errors.Wrap(err, "AUDIT_LOG_FAILED", "failed to write audit log")
	}
	return errors.New("FORBIDDEN", message)
}

func coalesceString(value interface{}, fallback string) string {
	str, ok := value.(string)
	if !ok {
		return fallback
	}
	if !utils.HasText(str) {
		return fallback
	}
	return strings.TrimSpace(str)
}
