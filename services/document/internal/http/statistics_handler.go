package httpapi

import (
	"net/http"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/document/internal/service"
)

type StatisticsHandler struct {
	svc *service.StatisticsService
}

func NewStatisticsHandler(svc *service.StatisticsService) *StatisticsHandler {
	return &StatisticsHandler{svc: svc}
}

func (h *StatisticsHandler) handleGetOverview(w http.ResponseWriter, r *http.Request) {
	overview, err := h.svc.GetOverview(r.Context())
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeData(w, r, http.StatusOK, overview)
}
