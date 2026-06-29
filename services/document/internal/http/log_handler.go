package http

import (
	"net/http"
	"strconv"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/document/internal/service"
)

// LogHandler 处理 GET /report-operation-logs 请求。
type LogHandler struct {
	svc *service.AuditService
}

func NewLogHandler(svc *service.AuditService) *LogHandler {
	return &LogHandler{svc: svc}
}

// ListLogs 处理 GET /report-operation-logs
// 支持查询参数：operationType, targetId, requestSource, page, pageSize
func (h *LogHandler) ListLogs(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	input := service.ListLogsInput{
		Page:     1,
		PageSize: 20,
	}

	if v := q.Get("operationType"); v != "" {
		input.OperationType = &v
	}
	if v := q.Get("targetId"); v != "" {
		input.TargetID = &v
	}
	if v := q.Get("requestSource"); v != "" {
		input.RequestSource = &v
	}
	if v := q.Get("page"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			input.Page = n
		}
	}
	if v := q.Get("pageSize"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 100 {
			input.PageSize = n
		}
	}

	result, err := h.svc.ListLogs(r.Context(), input)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}
