package httpapi

import (
	"net/http"
	"strconv"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/document/internal/service"
)

type LogHandler struct {
	svc *service.AuditService
}

func NewLogHandler(svc *service.AuditService) *LogHandler {
	return &LogHandler{svc: svc}
}

func (h *LogHandler) handleListLogs(w http.ResponseWriter, r *http.Request) {
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
		writeError(w, r, err)
		return
	}
	writeData(w, r, http.StatusOK, result)
}
