package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/document/internal/service"
)

type SettingsHandler struct {
	svc *service.SettingsService
}

func NewSettingsHandler(svc *service.SettingsService) *SettingsHandler {
	return &SettingsHandler{svc: svc}
}

func (h *SettingsHandler) handleGetSettings(w http.ResponseWriter, r *http.Request) {
	settings, err := h.svc.GetSettings(r.Context())
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeData(w, r, http.StatusOK, settings)
}

func (h *SettingsHandler) handleUpdateSettings(w http.ResponseWriter, r *http.Request) {
	var input service.UpdateReportSettingsInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, r, service.NewError(service.CodeValidation, "invalid request body", err))
		return
	}
	settings, err := h.svc.UpdateSettings(r.Context(), input)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeData(w, r, http.StatusOK, settings)
}
