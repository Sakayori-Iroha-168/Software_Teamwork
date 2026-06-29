package httpapi

import (
	"net/http"
	"time"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/document/internal/service"
)

// ── GET /report-settings ─────────────────────────────────────────────────────

// reportSettingsLLM is the public llm sub-object (no provider credentials).
type reportSettingsLLM struct {
	Provider  string  `json:"provider"`
	ProfileID *string `json:"profileId,omitempty"`
}

// reportSettingsFile is the public file-defaults sub-object.
type reportSettingsFile struct {
	DefaultFormat        string `json:"defaultFormat"`
	DefaultNumberingMode string `json:"defaultNumberingMode"`
}

// reportSettingsResponse matches the document OpenAPI ReportSettings schema.
type reportSettingsResponse struct {
	LLM              *reportSettingsLLM `json:"llm,omitempty"`
	DefaultTemplates map[string]string  `json:"defaultTemplates,omitempty"`
	File             reportSettingsFile `json:"file"`
}

func reportSettingsFromDomain(s service.ReportSettings) reportSettingsResponse {
	resp := reportSettingsResponse{
		File: reportSettingsFile{
			DefaultFormat:        s.DefaultFileFormat,
			DefaultNumberingMode: s.DefaultNumberingMode,
		},
	}
	if s.LLMProfileID != nil {
		resp.LLM = &reportSettingsLLM{
			Provider:  "ai-gateway",
			ProfileID: s.LLMProfileID,
		}
	}
	if len(s.DefaultTemplates) > 0 {
		resp.DefaultTemplates = s.DefaultTemplates
	}
	return resp
}

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

// ── PATCH /report-settings ───────────────────────────────────────────────────

type updateReportSettingsLLMRequest struct {
	ProfileID *string `json:"profileId"`
}

type updateReportSettingsFileRequest struct {
	DefaultFormat        *string `json:"defaultFormat"`
	DefaultNumberingMode *string `json:"defaultNumberingMode"`
}

type updateReportSettingsRequest struct {
	LLM              *updateReportSettingsLLMRequest `json:"llm"`
	DefaultTemplates map[string]string               `json:"defaultTemplates"`
	File             *updateReportSettingsFileRequest `json:"file"`
}

// updatedAtResponse matches the document OpenAPI UpdatedAt schema.
type updatedAtResponse struct {
	UpdatedAt string `json:"updatedAt"`
}

func (s *Server) handleUpdateReportSettings(w http.ResponseWriter, r *http.Request) {
	if !s.requireDocumentService(w, r) {
		return
	}
	var req updateReportSettingsRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	input := service.UpdateReportSettingsInput{}
	if req.LLM != nil {
		input.LLMProfileID = req.LLM.ProfileID
	}
	if req.DefaultTemplates != nil {
		input.DefaultTemplates = req.DefaultTemplates
	}
	if req.File != nil {
		input.DefaultFileFormat = req.File.DefaultFormat
		input.DefaultNumberingMode = req.File.DefaultNumberingMode
	}

	updated, err := s.documents.UpdateReportSettings(r.Context(), s.requestContext(r), input)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeData(w, r, http.StatusOK, updatedAtResponse{
		UpdatedAt: updated.UpdatedAt.Format(time.RFC3339),
	})
}

// ── GET /report-statistics/overview ─────────────────────────────────────────

// dailyTrendResponse matches the trend30d item in ReportStats.
type dailyTrendResponse struct {
	Date           string `json:"date"`
	GeneratedCount int    `json:"generatedCount"`
}

// statisticsOverviewResponse matches the document OpenAPI ReportStats schema.
type statisticsOverviewResponse struct {
	TemplateCount int                  `json:"templateCount"`
	ReportCount   int                  `json:"reportCount"`
	Trend30d      []dailyTrendResponse `json:"trend30d"`
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
		TemplateCount: o.TemplateCount,
		ReportCount:   o.ReportCount,
		Trend30d:      trend,
	}
}

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

// ── GET /report-operation-logs ───────────────────────────────────────────────

// operationLogResponse matches the document OpenAPI ReportOperationLog schema.
// Only the sanitized summary stored in parameter_summary_json is forwarded;
// raw provider keys and internal paths are never included.
type operationLogResponse struct {
	ID               string         `json:"id"`
	OperatorID       *string        `json:"operatorId,omitempty"`
	OperatorName     *string        `json:"operatorName,omitempty"`
	OperationType    string         `json:"operationType"`
	TargetType       string         `json:"targetType"`
	TargetID         string         `json:"targetId"`
	RequestID        *string        `json:"requestId,omitempty"`
	RequestSource    *string        `json:"requestSource,omitempty"`
	ToolName         *string        `json:"toolName,omitempty"`
	ParameterSummary map[string]any `json:"parameterSummary,omitempty"`
	OperationResult  string         `json:"operationResult"`
	ErrorMessage     *string        `json:"errorMessage,omitempty"`
	Metadata         map[string]any `json:"metadata,omitempty"`
	CreatedAt        string         `json:"createdAt"`
}

func operationLogFromDomain(l service.OperationLog) operationLogResponse {
	return operationLogResponse{
		ID:               l.ID,
		OperatorID:       l.OperatorID,
		OperatorName:     l.OperatorName,
		OperationType:    l.OperationType,
		TargetType:       l.TargetType,
		TargetID:         l.TargetID,
		RequestID:        l.RequestID,
		RequestSource:    l.RequestSource,
		ToolName:         l.ToolName,
		ParameterSummary: l.ParameterSummary,
		OperationResult:  l.OperationResult,
		ErrorMessage:     l.ErrorMessage,
		Metadata:         l.Metadata,
		CreatedAt:        l.CreatedAt.Format(time.RFC3339),
	}
}

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
		TargetType:    q.Get("targetType"),
		TargetID:      q.Get("targetId"),
		RequestID:     q.Get("requestId"),
		RequestSource: q.Get("requestSource"),
		ToolName:      q.Get("toolName"),
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
