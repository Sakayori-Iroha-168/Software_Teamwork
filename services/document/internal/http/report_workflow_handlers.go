package httpapi

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/document/internal/service"
)

func (s *Server) registerReportWorkflowRoutes() {
	s.mux.HandleFunc("GET /reports/{reportId}/jobs", s.handleListReportJobs)
	s.mux.HandleFunc("POST /reports/{reportId}/jobs", s.handleCreateReportJob)
	s.mux.HandleFunc("GET /report-jobs/{jobId}", s.handleGetReportJob)
	s.mux.HandleFunc("GET /report-jobs/{jobId}/attempts", s.handleListReportJobAttempts)
	s.mux.HandleFunc("POST /report-jobs/{jobId}/attempts", s.handleCreateReportJobAttempt)
	s.mux.HandleFunc("GET /reports/{reportId}/events", s.handleListReportEvents)
	s.mux.HandleFunc("GET /report-files", s.handleListReportFiles)
	s.mux.HandleFunc("POST /report-files", s.handleCreateReportFile)
	s.mux.HandleFunc("GET /report-files/{reportFileId}", s.handleGetReportFile)
	s.mux.HandleFunc("GET /report-files/{reportFileId}/content", s.handleGetReportFileContent)
	s.mux.HandleFunc("GET /report-statistics/overview", s.handleGetReportStatisticsOverview)
	s.mux.HandleFunc("GET /report-statistics/daily", s.handleListDailyReportStatistics)
	s.mux.HandleFunc("GET /report-operation-logs", s.handleListReportOperationLogs)
	s.mux.HandleFunc("GET /report-settings", s.handleGetReportSettings)
	s.mux.HandleFunc("PATCH /report-settings", s.handleUpdateReportSettings)
}

type createReportJobRequest struct {
	JobType string `json:"jobType"`
	Target  struct {
		Scope     string `json:"scope,omitempty"`
		SectionID string `json:"sectionId,omitempty"`
	} `json:"target,omitempty"`
	Requirements string         `json:"requirements,omitempty"`
	MaterialIDs  []string       `json:"materialIds,omitempty"`
	Options      map[string]any `json:"options,omitempty"`
}

type reportJobDTO struct {
	ID            string         `json:"id"`
	ReportID      string         `json:"reportId"`
	TemplateID    string         `json:"templateId,omitempty"`
	JobType       string         `json:"jobType"`
	TargetType    string         `json:"targetType,omitempty"`
	TargetID      string         `json:"targetId,omitempty"`
	Status        string         `json:"status"`
	Progress      map[string]any `json:"progress,omitempty"`
	ResultSummary string         `json:"resultSummary,omitempty"`
	Error         *jobErrorDTO   `json:"error,omitempty"`
	StartedAt     *string        `json:"startedAt,omitempty"`
	FinishedAt    *string        `json:"finishedAt,omitempty"`
	CreatedAt     string         `json:"createdAt"`
}

type jobErrorDTO struct {
	Code    string `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

func toReportJobDTO(job service.ReportJob) reportJobDTO {
	dto := reportJobDTO{
		ID:         job.ID,
		ReportID:   job.ReportID,
		TemplateID: job.TemplateID,
		JobType:    string(job.JobType),
		TargetType: job.TargetType,
		TargetID:   job.TargetID,
		Status:     string(job.Status),
		Progress: map[string]any{
			"percent": progressPercent(job.Status),
		},
		ResultSummary: resultSummary(job),
		StartedAt:     formatTimePtr(job.StartedAt),
		FinishedAt:    formatTimePtr(job.FinishedAt),
		CreatedAt:     job.CreatedAt.UTC().Format(time.RFC3339),
	}
	if job.ErrorCode != "" || job.ErrorMessage != "" {
		dto.Error = &jobErrorDTO{Code: job.ErrorCode, Message: job.ErrorMessage}
	}
	return dto
}

type createReportJobAttemptRequest struct {
	Reason  string         `json:"reason,omitempty"`
	Options map[string]any `json:"options,omitempty"`
}

type reportJobAttemptDTO struct {
	ID            string       `json:"id"`
	JobID         string       `json:"jobId"`
	AttemptNumber int          `json:"attemptNumber,omitempty"`
	Status        string       `json:"status"`
	Error         *jobErrorDTO `json:"error,omitempty"`
	StartedAt     *string      `json:"startedAt,omitempty"`
	FinishedAt    *string      `json:"finishedAt,omitempty"`
	CreatedAt     string       `json:"createdAt"`
}

func toReportJobAttemptDTO(attempt service.ReportJobAttempt) reportJobAttemptDTO {
	dto := reportJobAttemptDTO{
		ID:            attempt.ID,
		JobID:         attempt.JobID,
		AttemptNumber: attempt.AttemptNumber,
		Status:        string(attempt.Status),
		StartedAt:     formatTimePtr(attempt.StartedAt),
		FinishedAt:    formatTimePtr(attempt.FinishedAt),
		CreatedAt:     attempt.CreatedAt.UTC().Format(time.RFC3339),
	}
	if attempt.ErrorCode != "" || attempt.ErrorMessage != "" {
		dto.Error = &jobErrorDTO{Code: attempt.ErrorCode, Message: attempt.ErrorMessage}
	}
	return dto
}

type reportEventDTO struct {
	ID        string         `json:"id"`
	ReportID  string         `json:"reportId"`
	JobID     string         `json:"jobId,omitempty"`
	EventType string         `json:"eventType"`
	Message   string         `json:"message,omitempty"`
	Payload   map[string]any `json:"payload,omitempty"`
	CreatedAt string         `json:"createdAt"`
}

func toReportEventDTO(event service.ReportEvent) reportEventDTO {
	return reportEventDTO{
		ID:        event.ID,
		ReportID:  event.ReportID,
		JobID:     event.JobID,
		EventType: event.EventType,
		Message:   event.Message,
		Payload:   event.Payload,
		CreatedAt: event.CreatedAt.UTC().Format(time.RFC3339),
	}
}

type createReportFileRequest struct {
	ReportID     string         `json:"reportId"`
	Format       string         `json:"format"`
	TemplateID   string         `json:"templateId,omitempty"`
	StyleOptions map[string]any `json:"styleOptions,omitempty"`
}

type reportFileDTO struct {
	ID          string `json:"id"`
	ReportID    string `json:"reportId"`
	JobID       string `json:"jobId,omitempty"`
	Filename    string `json:"filename,omitempty"`
	Format      string `json:"format"`
	FileSize    int64  `json:"fileSize,omitempty"`
	Status      string `json:"status"`
	ContentPath string `json:"contentPath,omitempty"`
	CreatedBy   string `json:"createdBy,omitempty"`
	CreatedAt   string `json:"createdAt"`
}

func toReportFileDTO(file service.ReportFile) reportFileDTO {
	contentPath := file.ContentPath
	if contentPath == "" {
		contentPath = "/api/v1/report-files/" + file.ID + "/content"
	}
	return reportFileDTO{
		ID:          file.ID,
		ReportID:    file.ReportID,
		JobID:       file.JobID,
		Filename:    file.Filename,
		Format:      file.Format,
		FileSize:    file.FileSize,
		Status:      string(file.Status),
		ContentPath: contentPath,
		CreatedBy:   file.CreatedBy,
		CreatedAt:   file.CreatedAt.UTC().Format(time.RFC3339),
	}
}

type reportStatisticsOverviewDTO struct {
	ReportCount     int            `json:"reportCount"`
	TemplateCount   int            `json:"templateCount"`
	MaterialCount   int            `json:"materialCount"`
	JobStatusCounts map[string]int `json:"jobStatusCounts,omitempty"`
	RecentDays      int            `json:"recentDays,omitempty"`
}

type reportDailyStatisticDTO struct {
	Date           string `json:"date"`
	ReportType     string `json:"reportType,omitempty"`
	CreatedCount   int    `json:"createdCount"`
	GeneratedCount int    `json:"generatedCount"`
	FailedCount    int    `json:"failedCount"`
	ExportedCount  int    `json:"exportedCount"`
}

type reportOperationLogDTO struct {
	ID               string         `json:"id"`
	OperatorID       string         `json:"operatorId,omitempty"`
	OperatorName     string         `json:"operatorName,omitempty"`
	OperationType    string         `json:"operationType"`
	TargetType       string         `json:"targetType"`
	TargetID         string         `json:"targetId"`
	RequestID        string         `json:"requestId,omitempty"`
	RequestSource    string         `json:"requestSource,omitempty"`
	ToolName         string         `json:"toolName,omitempty"`
	ParameterSummary map[string]any `json:"parameterSummary,omitempty"`
	OperationResult  string         `json:"operationResult"`
	ErrorMessage     string         `json:"errorMessage,omitempty"`
	Metadata         map[string]any `json:"metadata,omitempty"`
	CreatedAt        string         `json:"createdAt"`
}

func toReportOperationLogDTO(value service.ReportOperationLog) reportOperationLogDTO {
	return reportOperationLogDTO{
		ID:               value.ID,
		OperatorID:       value.OperatorID,
		OperatorName:     value.OperatorName,
		OperationType:    value.OperationType,
		TargetType:       value.TargetType,
		TargetID:         value.TargetID,
		RequestID:        value.RequestID,
		RequestSource:    value.RequestSource,
		ToolName:         value.ToolName,
		ParameterSummary: value.ParameterSummary,
		OperationResult:  value.OperationResult,
		ErrorMessage:     value.ErrorMessage,
		Metadata:         value.Metadata,
		CreatedAt:        value.CreatedAt.UTC().Format(time.RFC3339),
	}
}

type reportSettingsDTO struct {
	LLM              map[string]any    `json:"llm,omitempty"`
	DefaultTemplates map[string]string `json:"defaultTemplates,omitempty"`
	File             map[string]any    `json:"file,omitempty"`
}

type reportSettingsUpdateDTO struct {
	UpdatedAt string `json:"updatedAt"`
}

func (s *Server) handleListReportJobs(w http.ResponseWriter, r *http.Request) {
	if !s.requireReportService(w, r) {
		return
	}
	jobs, err := s.reportService.ListReportJobs(r.Context(), s.requestContext(r), r.PathValue("reportId"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	data := make([]reportJobDTO, len(jobs))
	for i, job := range jobs {
		data[i] = toReportJobDTO(job)
	}
	writeData(w, r, http.StatusOK, data)
}

func (s *Server) handleCreateReportJob(w http.ResponseWriter, r *http.Request) {
	if !s.requireReportService(w, r) {
		return
	}
	var body createReportJobRequest
	if !decodeJSON(w, r, &body) {
		return
	}
	job, err := s.reportService.CreateReportJob(r.Context(), s.requestContext(r), r.PathValue("reportId"), service.CreateReportJobInput{
		JobType:      service.JobType(body.JobType),
		TargetScope:  body.Target.Scope,
		SectionID:    body.Target.SectionID,
		Requirements: body.Requirements,
		MaterialIDs:  body.MaterialIDs,
		Options:      body.Options,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeData(w, r, http.StatusAccepted, toReportJobDTO(job))
}

func (s *Server) handleGetReportJob(w http.ResponseWriter, r *http.Request) {
	if !s.requireReportService(w, r) {
		return
	}
	job, err := s.reportService.GetReportJob(r.Context(), s.requestContext(r), r.PathValue("jobId"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeData(w, r, http.StatusOK, toReportJobDTO(job))
}

func (s *Server) handleListReportJobAttempts(w http.ResponseWriter, r *http.Request) {
	if !s.requireReportService(w, r) {
		return
	}
	attempts, err := s.reportService.ListReportJobAttempts(r.Context(), s.requestContext(r), r.PathValue("jobId"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	data := make([]reportJobAttemptDTO, len(attempts))
	for i, attempt := range attempts {
		data[i] = toReportJobAttemptDTO(attempt)
	}
	writeData(w, r, http.StatusOK, data)
}

func (s *Server) handleCreateReportJobAttempt(w http.ResponseWriter, r *http.Request) {
	if !s.requireReportService(w, r) {
		return
	}
	var body createReportJobAttemptRequest
	if !decodeJSON(w, r, &body) {
		return
	}
	attempt, err := s.reportService.CreateReportJobAttempt(r.Context(), s.requestContext(r), r.PathValue("jobId"), service.CreateReportJobAttemptInput{
		Reason:  body.Reason,
		Options: body.Options,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeData(w, r, http.StatusAccepted, toReportJobAttemptDTO(attempt))
}

func (s *Server) handleListReportEvents(w http.ResponseWriter, r *http.Request) {
	if !s.requireReportService(w, r) {
		return
	}
	events, err := s.reportService.ListReportEvents(r.Context(), s.requestContext(r), r.PathValue("reportId"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	data := make([]reportEventDTO, len(events))
	for i, event := range events {
		data[i] = toReportEventDTO(event)
	}
	writeData(w, r, http.StatusOK, data)
}

func (s *Server) handleCreateReportFile(w http.ResponseWriter, r *http.Request) {
	if !s.requireReportService(w, r) {
		return
	}
	var body createReportFileRequest
	if !decodeJSON(w, r, &body) {
		return
	}
	file, err := s.reportService.CreateReportFile(r.Context(), s.requestContext(r), service.CreateReportFileInput{
		ReportID:     body.ReportID,
		Format:       body.Format,
		TemplateID:   body.TemplateID,
		StyleOptions: body.StyleOptions,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeData(w, r, http.StatusAccepted, toReportFileDTO(file))
}

func (s *Server) handleListReportFiles(w http.ResponseWriter, r *http.Request) {
	if !s.requireReportService(w, r) {
		return
	}
	page, pageSize, err := parsePage(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	result, err := s.reportService.ListReportFiles(r.Context(), s.requestContext(r), service.ReportFileListFilter{
		Page:     page,
		PageSize: pageSize,
		ReportID: r.URL.Query().Get("reportId"),
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	data := make([]reportFileDTO, len(result.Items))
	for i, file := range result.Items {
		data[i] = toReportFileDTO(file)
	}
	writePage(w, r, http.StatusOK, data, result.Page)
}

func (s *Server) handleGetReportFile(w http.ResponseWriter, r *http.Request) {
	if !s.requireReportService(w, r) {
		return
	}
	file, err := s.reportService.GetReportFile(r.Context(), s.requestContext(r), r.PathValue("reportFileId"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeData(w, r, http.StatusOK, toReportFileDTO(file))
}

func (s *Server) handleGetReportFileContent(w http.ResponseWriter, r *http.Request) {
	if !s.requireReportService(w, r) {
		return
	}
	file, content, err := s.reportService.BuildReportFileContent(r.Context(), s.requestContext(r), r.PathValue("reportFileId"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.wordprocessingml.document")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, strings.ReplaceAll(file.Filename, `"`, "")))
	w.Header().Set("Content-Length", strconv.Itoa(len(content)))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(content)
}

func (s *Server) handleGetReportStatisticsOverview(w http.ResponseWriter, r *http.Request) {
	if !s.requireReportService(w, r) {
		return
	}
	overview, err := s.reportService.GetReportStatisticsOverview(r.Context(), s.requestContext(r))
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeData(w, r, http.StatusOK, reportStatisticsOverviewDTO(overview))
}

func (s *Server) handleListDailyReportStatistics(w http.ResponseWriter, r *http.Request) {
	if !s.requireReportService(w, r) {
		return
	}
	days := 30
	rawDays := strings.TrimSpace(r.URL.Query().Get("days"))
	if rawDays != "" {
		value, err := strconv.Atoi(rawDays)
		if err != nil || value <= 0 {
			writeError(w, r, service.ValidationError(map[string]string{"days": "must be a positive integer"}))
			return
		}
		days = value
	}
	items, err := s.reportService.ListDailyReportStatistics(r.Context(), s.requestContext(r), days)
	if err != nil {
		writeError(w, r, err)
		return
	}
	data := make([]reportDailyStatisticDTO, len(items))
	for i, item := range items {
		data[i] = reportDailyStatisticDTO(item)
	}
	writeData(w, r, http.StatusOK, data)
}

func (s *Server) handleListReportOperationLogs(w http.ResponseWriter, r *http.Request) {
	if !s.requireReportService(w, r) {
		return
	}
	page, pageSize, err := parsePage(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	result, err := s.reportService.ListReportOperationLogs(r.Context(), s.requestContext(r), service.ReportOperationLogFilter{
		Page:          page,
		PageSize:      pageSize,
		TargetType:    r.URL.Query().Get("targetType"),
		TargetID:      r.URL.Query().Get("targetId"),
		OperationType: r.URL.Query().Get("operationType"),
		RequestID:     r.URL.Query().Get("requestId"),
		RequestSource: r.URL.Query().Get("requestSource"),
		ToolName:      r.URL.Query().Get("toolName"),
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	data := make([]reportOperationLogDTO, len(result.Items))
	for i, item := range result.Items {
		data[i] = toReportOperationLogDTO(item)
	}
	writePage(w, r, http.StatusOK, data, result.Page)
}

func (s *Server) handleGetReportSettings(w http.ResponseWriter, r *http.Request) {
	if !s.requireReportService(w, r) {
		return
	}
	settings, err := s.reportService.GetReportSettings(r.Context(), s.requestContext(r))
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeData(w, r, http.StatusOK, reportSettingsDTO(settings))
}

func (s *Server) handleUpdateReportSettings(w http.ResponseWriter, r *http.Request) {
	if !s.requireReportService(w, r) {
		return
	}
	var body reportSettingsDTO
	if !decodeJSON(w, r, &body) {
		return
	}
	result, err := s.reportService.UpdateReportSettings(r.Context(), s.requestContext(r), service.ReportSettings(body))
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeData(w, r, http.StatusOK, reportSettingsUpdateDTO{UpdatedAt: result.UpdatedAt.UTC().Format(time.RFC3339)})
}

func progressPercent(status service.JobStatus) int {
	switch status {
	case service.JobStatusPending:
		return 10
	case service.JobStatusRunning:
		return 50
	case service.JobStatusSucceeded, service.JobStatusPartialSucceeded:
		return 100
	case service.JobStatusFailed, service.JobStatusCanceled:
		return 0
	default:
		return 0
	}
}

func resultSummary(job service.ReportJob) string {
	if job.Status == service.JobStatusSucceeded {
		return "Report workflow step completed and persisted."
	}
	if job.ErrorMessage != "" {
		return job.ErrorMessage
	}
	return ""
}

var _ ReportService = (*service.ReportService)(nil)
