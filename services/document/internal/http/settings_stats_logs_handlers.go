package httpapi

import (
	"net/http"
	"time"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/document/internal/service"
)

// reportSettingsResponse is the public shape of ReportSettings.
// provider baseUrl and apiKey are never included.
type reportSettingsResponse struct {
	ID                   string  `json:"id"`
	LLMProfileID         *string `json:"llmProfileId,omitempty"`
	DefaultTemplateID    *string `json:"defaultTemplateId,omitempty"`
	DefaultFileFormat    string  `json:"defaultFileFormat"`
	DefaultNumberingMode string  `json:"defaultNumberingMode"`
	UpdatedAt            string  `json:"updatedAt"`
	CreatedAt            string  `json:"createdAt"`
}

func reportSettingsFromDomain(s service.ReportSettings) reportSettingsResponse {
	return reportSettingsResponse{
		ID:                   s.ID,
		LLMProfileID:         s.LLMProfileID,
		DefaultTemplateID:    s.DefaultTemplateID,
		DefaultFileFormat:    s.DefaultFileFormat,
		DefaultNumberingMode: s.DefaultNumberingMode,
		UpdatedAt:            s.UpdatedAt.Format(time.RFC3339),
		CreatedAt:            s.CreatedAt.Format(time.RFC3339),
	}
}

// handleGetReportSettings handles GET /report-settings.
func (s *Server) handleGetReportSettings(w http.ResponseWriter, r *http.Request) {
	if !s.requireDocumentService(w, r) {
		return
	}
	settings, err := s.documents.GetReportSettings(r.Context(), s.requestContext(r))
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeData(w, r, http.StatusOK, reportSettingsFromDomain(settings))
}

type updateReportSettingsRequest struct {
	LLMProfileID         *string `json:"llmProfileId"`
	DefaultTemplateID    *string `json:"defaultTemplateId"`
	DefaultFileFormat    *string `json:"defaultFileFormat"`
	DefaultNumberingMode *string `json:"defaultNumberingMode"`
}

// handleUpdateReportSettings handles PATCH /report-settings.
func (s *Server) handleUpdateReportSettings(w http.ResponseWriter, r *http.Request) {
	if !s.requireDocumentService(w, r) {
		return
	}
	var req updateReportSettingsRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	settings, err := s.documents.UpdateReportSettings(r.Context(), s.requestContext(r), service.UpdateReportSettingsInput{
		LLMProfileID:         req.LLMProfileID,
		DefaultTemplateID:    req.DefaultTemplateID,
		DefaultFileFormat:    req.DefaultFileFormat,
		DefaultNumberingMode: req.DefaultNumberingMode,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeData(w, r, http.StatusOK, reportSettingsFromDomain(settings))
}

// statisticsOverviewResponse is the public shape for GET /report-statistics/overview.
type statisticsOverviewResponse struct {
	TemplateCount  int            `json:"templateCount"`
	ReportCount    int            `json:"reportCount"`
	GeneratedCount int            `json:"generatedCount"`
	FailedCount    int            `json:"failedCount"`
	Trend30d       []dailyTrendResponse `json:"trend30d"`
}

type dailyTrendResponse struct {
	Date           string `json:"date"`
	GeneratedCount int    `json:"generatedCount"`
}

func statisticsOverviewFromDomain(o service.ReportStatisticsOverview) statisticsOverviewResponse {
	trend := make([]dailyTrendResponse, len(o.Trend30d))
	for i, t := range o.Trend30d {
		trend[i] = dailyTrendResponse{
			Date:           t.Date.Format("2006-01-02"),
			GeneratedCount: t.GeneratedCount,
		}
	}
	return statisticsOverviewResponse{
		TemplateCount:  o.TemplateCount,
		ReportCount:    o.ReportCount,
		GeneratedCount: o.GeneratedCount,
		FailedCount:    o.FailedCount,
		Trend30d:       trend,
	}
}

// handleGetReportStatisticsOverview handles GET /report-statistics/overview.
func (s *Server) handleGetReportStatisticsOverview(w http.ResponseWriter, r *http.Request) {
	if !s.requireDocumentService(w, r) {
		return
	}
	overview, err := s.documents.GetReportStatisticsOverview(r.Context(), s.requestContext(r))
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeData(w, r, http.StatusOK, statisticsOverviewFromDomain(overview))
}

// operationLogResponse is the public shape of OperationLog.
// Sensitive fields (full parameter payloads, internal paths) are never included.
type operationLogResponse struct {
	ID              string         `json:"id"`
	OperatorID      *string        `json:"operatorId,omitempty"`
	OperatorName    *string        `json:"operatorName,omitempty"`
	OperationType   string         `json:"operationType"`
	TargetType      string         `json:"targetType"`
	TargetID        string         `json:"targetId"`
	RequestID       *string        `json:"requestId,omitempty"`
	RequestSource   *string        `json:"requestSource,omitempty"`
	ToolName        *string        `json:"toolName,omitempty"`
	OperationResult string         `json:"operationResult"`
	ErrorMessage    *string        `json:"errorMessage,omitempty"`
	CreatedAt       string         `json:"createdAt"`
}

func operationLogFromDomain(l service.OperationLog) operationLogResponse {
	return operationLogResponse{
		ID:              l.ID,
		OperatorID:      l.OperatorID,
		OperatorName:    l.OperatorName,
		OperationType:   l.OperationType,
		TargetType:      l.TargetType,
		TargetID:        l.TargetID,
		RequestID:       l.RequestID,
		RequestSource:   l.RequestSource,
		ToolName:        l.ToolName,
		OperationResult: l.OperationResult,
		ErrorMessage:    l.ErrorMessage,
		CreatedAt:       l.CreatedAt.Format(time.RFC3339),
	}
}

// handleListOperationLogs handles GET /report-operation-logs.
func (s *Server) handleListOperationLogs(w http.ResponseWriter, r *http.Request) {
	if !s.requireDocumentService(w, r) {
		return
	}
	page, pageSize, err := parsePage(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	q := r.URL.Query()
	result, err := s.documents.ListOperationLogs(r.Context(), s.requestContext(r), service.OperationLogListFilter{
		Page:          page,
		PageSize:      pageSize,
		OperationType: q.Get("operationType"),
		TargetID:      q.Get("targetId"),
		RequestSource: q.Get("requestSource"),
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	items := make([]operationLogResponse, len(result.Items))
	for i, l := range result.Items {
		items[i] = operationLogFromDomain(l)
	}
	writePage(w, r, http.StatusOK, items, result.Page)
}
