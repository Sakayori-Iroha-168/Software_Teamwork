package http

import (
	"net/http"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/document/internal/service"
)

// StatisticsHandler 处理 GET /report-statistics/overview 请求。
type StatisticsHandler struct {
	svc *service.StatisticsService
}

func NewStatisticsHandler(svc *service.StatisticsService) *StatisticsHandler {
	return &StatisticsHandler{svc: svc}
}

// GetOverview 处理 GET /report-statistics/overview
func (h *StatisticsHandler) GetOverview(w http.ResponseWriter, r *http.Request) {
	overview, err := h.svc.GetOverview(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, overview)
}
