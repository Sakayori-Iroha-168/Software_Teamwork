package httpx

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/qa/internal/service"
)

type ConversationHandler struct {
	conversations *service.ConversationService
}

func NewConversationHandler(conversations *service.ConversationService) *ConversationHandler {
	return &ConversationHandler{conversations: conversations}
}

func (h *ConversationHandler) Create(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		WriteError(w, http.StatusMethodNotAllowed, 40000, "method not allowed")
		return
	}

	var body struct {
		ExternalUserID string `json:"external_user_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		WriteError(w, http.StatusBadRequest, 40000, "invalid request body")
		return
	}

	result, err := h.conversations.Create(r.Context(), service.CreateConversationRequest{
		ExternalUserID: body.ExternalUserID,
	})
	if err != nil {
		WriteError(w, http.StatusInternalServerError, 50000, err.Error())
		return
	}

	response := map[string]any{
		"id":              result.ID,
		"external_user_id": result.ExternalUserID,
		"status":          result.Status,
		"created_at":      result.CreatedAt,
		"updated_at":      result.UpdatedAt,
	}
	WriteSuccess(w, response)
}

func (h *ConversationHandler) Get(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		WriteError(w, http.StatusMethodNotAllowed, 40000, "method not allowed")
		return
	}

	var conversationID string
	if strings.HasPrefix(r.URL.Path, "/api/v1/qa-sessions/") {
		parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/v1/qa-sessions/"), "/")
		if len(parts) > 0 && parts[0] != "" && !strings.Contains(parts[0], "/") {
			conversationID = parts[0]
		}
	} else if strings.HasPrefix(r.URL.Path, "/api/conversations/") {
		conversationID = strings.TrimPrefix(r.URL.Path, "/api/conversations/")
	}

	if conversationID != "" {
		h.getByID(w, r, conversationID)
		return
	}

	h.list(w, r)
}

func (h *ConversationHandler) getByID(w http.ResponseWriter, r *http.Request, id string) {
	result, err := h.conversations.GetByID(r.Context(), id)
	if err != nil {
		WriteError(w, http.StatusNotFound, 40400, err.Error())
		return
	}

	response := map[string]any{
		"id":               result.ID,
		"external_user_id": result.ExternalUserID,
		"status":           result.Status,
		"created_at":       result.CreatedAt,
		"updated_at":       result.UpdatedAt,
		"messages":         result.Messages,
	}
	WriteSuccess(w, response)
}

func (h *ConversationHandler) list(w http.ResponseWriter, r *http.Request) {
	pageStr := r.URL.Query().Get("page")
	pageSizeStr := r.URL.Query().Get("page_size")

	page := 1
	if pageStr != "" {
		var err error
		page, err = strconv.Atoi(pageStr)
		if err != nil || page < 1 {
			page = 1
		}
	}

	pageSize := 20
	if pageSizeStr != "" {
		var err error
		pageSize, err = strconv.Atoi(pageSizeStr)
		if err != nil || pageSize < 1 || pageSize > 100 {
			pageSize = 20
		}
	}

	externalUserID := r.URL.Query().Get("external_user_id")

	result, err := h.conversations.List(r.Context(), service.ListConversationsRequest{
		ExternalUserID: externalUserID,
		Page:           page,
		PageSize:       pageSize,
	})
	if err != nil {
		WriteError(w, http.StatusInternalServerError, 50000, err.Error())
		return
	}

	response := map[string]any{
		"data": result.Data,
		"pagination": map[string]any{
			"page":       result.Page,
			"page_size":  result.PageSize,
			"total":      result.Total,
			"total_pages": (result.Total + int64(result.PageSize) - 1) / int64(result.PageSize),
		},
	}
	WriteSuccess(w, response)
}
