package service

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"sort"
	"strings"
	"time"
)

const (
	reportTargetTypeReport  = "report"
	reportTargetTypeOutline = "outline"
	reportTargetTypeSection = "section"
	reportTargetTypeFile    = "file"
)

func (s *ReportService) ListReportJobs(ctx context.Context, reqCtx RequestContext, reportID string) ([]ReportJob, error) {
	if _, err := s.GetReport(ctx, reqCtx, reportID); err != nil {
		return nil, err
	}
	jobs, err := s.repo.ListReportJobs(ctx, reportID)
	if err != nil {
		return nil, dependencyError("list report jobs", err)
	}
	return jobs, nil
}

func (s *ReportService) CreateReportJob(ctx context.Context, reqCtx RequestContext, reportID string, input CreateReportJobInput) (ReportJob, error) {
	report, err := s.GetReport(ctx, reqCtx, reportID)
	if err != nil {
		return ReportJob{}, err
	}
	if report.Status == ReportStatusDeleted || report.DeletedAt != nil {
		return ReportJob{}, NewError(CodeConflict, "report has been deleted", nil)
	}
	if err := validateReportJobInput(input); err != nil {
		return ReportJob{}, err
	}

	var created ReportJob
	err = s.repo.WithinTx(ctx, func(txRepo ReportRepository) error {
		now := s.now()
		targetType, targetID := normalizeJobTarget(reportID, input)
		job := ReportJob{
			ID:          newID(),
			RequestID:   reqCtx.RequestID,
			Source:      "api",
			JobType:     input.JobType,
			TargetType:  targetType,
			TargetID:    targetID,
			QueueName:   "document",
			ReportID:    reportID,
			TemplateID:  report.TemplateID,
			Status:      JobStatusSucceeded,
			RetryCount:  0,
			MaxAttempts: 3,
			StartedAt:   &now,
			FinishedAt:  &now,
			CreatedAt:   now,
		}
		persistedJob, err := txRepo.CreateReportJob(ctx, job)
		if err != nil {
			return mapRepositoryReadError(err, "create report job failed")
		}
		created = persistedJob

		if _, err := txRepo.CreateReportJobAttempt(ctx, ReportJobAttempt{
			ID:            newID(),
			JobID:         persistedJob.ID,
			AttemptNumber: 1,
			TriggerSource: "api",
			Reason:        input.Requirements,
			Status:        JobStatusSucceeded,
			StartedAt:     &now,
			FinishedAt:    &now,
			CreatedAt:     now,
		}); err != nil {
			return mapRepositoryReadError(err, "create report job attempt failed")
		}
		if err := createReportEvent(ctx, txRepo, reportID, persistedJob.ID, "job.created", "report job created", now); err != nil {
			return err
		}

		switch input.JobType {
		case JobTypeOutlineGeneration, JobTypeOutlineRegeneration:
			if err := s.applyOutlineGeneration(ctx, txRepo, report, persistedJob, now); err != nil {
				return err
			}
		case JobTypeContentGeneration, JobTypeContentRegeneration:
			if err := s.applyContentGeneration(ctx, txRepo, report, persistedJob, now); err != nil {
				return err
			}
		case JobTypeSectionRegeneration:
			if strings.TrimSpace(input.SectionID) == "" {
				return ValidationError(map[string]string{"target.sectionId": "sectionId is required for section_regeneration"})
			}
			if err := s.applySectionRegeneration(ctx, txRepo, report, persistedJob, input.SectionID, now); err != nil {
				return err
			}
		case JobTypeReportFileCreation:
			if _, err := s.createReportFileInTx(ctx, txRepo, reqCtx, report, persistedJob.ID, now); err != nil {
				return err
			}
		}

		if err := createReportEvent(ctx, txRepo, reportID, persistedJob.ID, "job.completed", "report job completed", now); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return ReportJob{}, err
	}
	return created, nil
}

func (s *ReportService) GetReportJob(ctx context.Context, reqCtx RequestContext, jobID string) (ReportJob, error) {
	if err := requireGatewayContext(reqCtx); err != nil {
		return ReportJob{}, err
	}
	job, err := s.repo.FindReportJobByID(ctx, jobID)
	if err != nil {
		return ReportJob{}, mapRepositoryReadError(err, "report job not found")
	}
	if _, err := s.GetReport(ctx, reqCtx, job.ReportID); err != nil {
		return ReportJob{}, err
	}
	return job, nil
}

func (s *ReportService) ListReportJobAttempts(ctx context.Context, reqCtx RequestContext, jobID string) ([]ReportJobAttempt, error) {
	job, err := s.GetReportJob(ctx, reqCtx, jobID)
	if err != nil {
		return nil, err
	}
	attempts, err := s.repo.ListReportJobAttempts(ctx, job.ID)
	if err != nil {
		return nil, dependencyError("list report job attempts", err)
	}
	return attempts, nil
}

func (s *ReportService) CreateReportJobAttempt(ctx context.Context, reqCtx RequestContext, jobID string, input CreateReportJobAttemptInput) (ReportJobAttempt, error) {
	job, err := s.GetReportJob(ctx, reqCtx, jobID)
	if err != nil {
		return ReportJobAttempt{}, err
	}
	if job.RetryCount+1 >= job.MaxAttempts {
		return ReportJobAttempt{}, NewError(CodeConflict, "report job retry limit reached", nil)
	}
	if !canRetryReportJob(job.Status) {
		return ReportJobAttempt{}, NewError(CodeConflict, "report job is not retryable", nil)
	}
	existing, err := s.repo.ListReportJobAttempts(ctx, jobID)
	if err != nil {
		return ReportJobAttempt{}, dependencyError("list report job attempts", err)
	}
	nextAttempt := 1
	for _, attempt := range existing {
		if attempt.AttemptNumber >= nextAttempt {
			nextAttempt = attempt.AttemptNumber + 1
		}
	}
	now := s.now()
	var attempt ReportJobAttempt
	err = s.repo.WithinTx(ctx, func(txRepo ReportRepository) error {
		created, err := txRepo.CreateReportJobAttempt(ctx, ReportJobAttempt{
			ID:            newID(),
			JobID:         jobID,
			AttemptNumber: nextAttempt,
			TriggerSource: "api",
			Reason:        strings.TrimSpace(input.Reason),
			Status:        JobStatusSucceeded,
			StartedAt:     &now,
			FinishedAt:    &now,
			CreatedAt:     now,
		})
		if err != nil {
			return mapRepositoryReadError(err, "create report job attempt failed")
		}
		attempt = created
		if _, err := txRepo.UpdateReportJobRetryState(ctx, job.ID, job.RetryCount+1, JobStatusSucceeded, now); err != nil {
			return mapRepositoryReadError(err, "update report job retry state failed")
		}
		if err := createReportEvent(ctx, txRepo, job.ReportID, job.ID, "job.retry_succeeded", "report job retry recorded", now); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return ReportJobAttempt{}, err
	}
	return attempt, nil
}

func (s *ReportService) ListReportEvents(ctx context.Context, reqCtx RequestContext, reportID string) ([]ReportEvent, error) {
	if _, err := s.GetReport(ctx, reqCtx, reportID); err != nil {
		return nil, err
	}
	events, err := s.repo.ListReportEvents(ctx, reportID)
	if err != nil {
		return nil, dependencyError("list report events", err)
	}
	return events, nil
}

func (s *ReportService) CreateReportFile(ctx context.Context, reqCtx RequestContext, input CreateReportFileInput) (ReportFile, error) {
	report, err := s.GetReport(ctx, reqCtx, input.ReportID)
	if err != nil {
		return ReportFile{}, err
	}
	if strings.TrimSpace(input.Format) != "docx" {
		return ReportFile{}, ValidationError(map[string]string{"format": "only docx is supported"})
	}
	now := s.now()
	var file ReportFile
	err = s.repo.WithinTx(ctx, func(txRepo ReportRepository) error {
		job := ReportJob{
			ID:          newID(),
			RequestID:   reqCtx.RequestID,
			Source:      "api",
			JobType:     JobTypeReportFileCreation,
			TargetType:  reportTargetTypeFile,
			TargetID:    report.ID,
			QueueName:   "document",
			ReportID:    report.ID,
			TemplateID:  firstNonEmpty(input.TemplateID, report.TemplateID),
			Status:      JobStatusSucceeded,
			RetryCount:  0,
			MaxAttempts: 1,
			StartedAt:   &now,
			FinishedAt:  &now,
			CreatedAt:   now,
		}
		createdJob, err := txRepo.CreateReportJob(ctx, job)
		if err != nil {
			return mapRepositoryReadError(err, "create report file job failed")
		}
		createdFile, err := s.createReportFileInTx(ctx, txRepo, reqCtx, report, createdJob.ID, now)
		if err != nil {
			return err
		}
		file = createdFile
		if err := createReportEvent(ctx, txRepo, report.ID, createdJob.ID, "file.created", "report DOCX file resource created", now); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return ReportFile{}, err
	}
	return file, nil
}

func (s *ReportService) ListReportFiles(ctx context.Context, reqCtx RequestContext, filter ReportFileListFilter) (ReportFileListResult, error) {
	if err := requireGatewayContext(reqCtx); err != nil {
		return ReportFileListResult{}, err
	}
	filter.Page, filter.PageSize = normalizePage(filter.Page, filter.PageSize)
	if strings.TrimSpace(filter.ReportID) != "" {
		if _, err := s.GetReport(ctx, reqCtx, filter.ReportID); err != nil {
			return ReportFileListResult{}, err
		}
	} else if !reqCtx.IsAdmin() {
		filter.CreatorID = reqCtx.UserID
	}
	files, total, err := s.repo.ListReportFiles(ctx, filter)
	if err != nil {
		return ReportFileListResult{}, dependencyError("list report files", err)
	}
	return ReportFileListResult{Items: files, Page: PageMeta{Page: filter.Page, PageSize: filter.PageSize, Total: total}}, nil
}

func (s *ReportService) GetReportFile(ctx context.Context, reqCtx RequestContext, fileID string) (ReportFile, error) {
	if err := requireGatewayContext(reqCtx); err != nil {
		return ReportFile{}, err
	}
	file, err := s.repo.GetReportFileByID(ctx, fileID)
	if err != nil {
		return ReportFile{}, mapRepositoryReadError(err, "report file not found")
	}
	if _, err := s.GetReport(ctx, reqCtx, file.ReportID); err != nil {
		return ReportFile{}, err
	}
	return file, nil
}

func (s *ReportService) BuildReportFileContent(ctx context.Context, reqCtx RequestContext, fileID string) (ReportFile, []byte, error) {
	file, err := s.GetReportFile(ctx, reqCtx, fileID)
	if err != nil {
		return ReportFile{}, nil, err
	}
	content, err := s.repo.GetReportFileContent(ctx, file.ID)
	if err != nil {
		return ReportFile{}, nil, mapRepositoryReadError(err, "report file content not found")
	}
	return file, content, nil
}

func (s *ReportService) GetReportStatisticsOverview(ctx context.Context, reqCtx RequestContext) (ReportStatisticsOverview, error) {
	if err := requireAdmin(reqCtx); err != nil {
		return ReportStatisticsOverview{}, err
	}
	overview, err := s.repo.GetReportStatisticsOverview(ctx, 30)
	if err != nil {
		return ReportStatisticsOverview{}, dependencyError("get report statistics overview", err)
	}
	return overview, nil
}

func (s *ReportService) ListDailyReportStatistics(ctx context.Context, reqCtx RequestContext, days int) ([]ReportDailyStatistic, error) {
	if err := requireAdmin(reqCtx); err != nil {
		return nil, err
	}
	if days <= 0 {
		days = 30
	}
	if days > 366 {
		return nil, ValidationError(map[string]string{"days": "must be less than or equal to 366"})
	}
	items, err := s.repo.ListDailyReportStatistics(ctx, days)
	if err != nil {
		return nil, dependencyError("list daily report statistics", err)
	}
	return items, nil
}

func (s *ReportService) ListReportOperationLogs(ctx context.Context, reqCtx RequestContext, filter ReportOperationLogFilter) (ReportOperationLogListResult, error) {
	if err := requireAdmin(reqCtx); err != nil {
		return ReportOperationLogListResult{}, err
	}
	filter.Page, filter.PageSize = normalizePage(filter.Page, filter.PageSize)
	items, total, err := s.repo.ListReportOperationLogs(ctx, filter)
	if err != nil {
		return ReportOperationLogListResult{}, dependencyError("list report operation logs", err)
	}
	return ReportOperationLogListResult{Items: items, Page: PageMeta{Page: filter.Page, PageSize: filter.PageSize, Total: total}}, nil
}

func (s *ReportService) GetReportSettings(ctx context.Context, reqCtx RequestContext) (ReportSettings, error) {
	if err := requireAdmin(reqCtx); err != nil {
		return ReportSettings{}, err
	}
	settings, err := s.repo.GetReportSettings(ctx)
	if err != nil {
		return ReportSettings{}, dependencyError("get report settings", err)
	}
	return settings, nil
}

func (s *ReportService) UpdateReportSettings(ctx context.Context, reqCtx RequestContext, settings ReportSettings) (UpdateReportSettingsResult, error) {
	if err := requireAdmin(reqCtx); err != nil {
		return UpdateReportSettingsResult{}, err
	}
	result, err := s.repo.UpdateReportSettings(ctx, settings, reqCtx.UserID, s.now())
	if err != nil {
		return UpdateReportSettingsResult{}, dependencyError("update report settings", err)
	}
	return result, nil
}

func validateReportJobInput(input CreateReportJobInput) error {
	switch input.JobType {
	case JobTypeOutlineGeneration, JobTypeOutlineRegeneration, JobTypeContentGeneration, JobTypeContentRegeneration, JobTypeSectionRegeneration, JobTypeReportFileCreation:
		return nil
	default:
		return ValidationError(map[string]string{"jobType": "unsupported report job type"})
	}
}

func canRetryReportJob(status JobStatus) bool {
	return status == JobStatusFailed || status == JobStatusCanceled
}

func normalizeJobTarget(reportID string, input CreateReportJobInput) (string, string) {
	if input.JobType == JobTypeSectionRegeneration {
		return reportTargetTypeSection, strings.TrimSpace(input.SectionID)
	}
	scope := strings.TrimSpace(input.TargetScope)
	if scope == "" {
		scope = reportTargetTypeReport
	}
	targetID := strings.TrimSpace(input.TargetID)
	if targetID == "" {
		targetID = reportID
	}
	return scope, targetID
}

func (s *ReportService) applyOutlineGeneration(ctx context.Context, repo ReportRepository, report Report, job ReportJob, now time.Time) error {
	outline := ReportOutline{
		ID:           newID(),
		ReportID:     report.ID,
		Sections:     defaultOutlineForReport(report),
		Version:      1,
		Source:       OutlineSourceAI,
		SourceJobID:  job.ID,
		IsCurrent:    true,
		ManualEdited: false,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	existing, err := repo.ListReportOutlines(ctx, report.ID)
	if err != nil {
		return dependencyError("list report outlines", err)
	}
	for _, item := range existing {
		if item.Version >= outline.Version {
			outline.Version = item.Version + 1
		}
	}
	if _, err := repo.CreateReportOutline(ctx, outline); err != nil {
		return mapRepositoryReadError(err, "create generated outline failed")
	}
	report.Status = ReportStatusOutlineGenerated
	report.LatestJobID = job.ID
	report.UpdatedAt = now
	if _, err := repo.UpdateReportWorkflowState(ctx, report); err != nil {
		return mapRepositoryReadError(err, "update report workflow state failed")
	}
	return nil
}

func (s *ReportService) applyContentGeneration(ctx context.Context, repo ReportRepository, report Report, job ReportJob, now time.Time) error {
	outline, err := currentOutline(ctx, repo, report.ID)
	if err != nil {
		return err
	}
	existingSections, err := repo.ListReportSections(ctx, report.ID)
	if err != nil {
		return dependencyError("list report sections", err)
	}
	byOutlineNode := map[string]ReportSection{}
	for _, section := range existingSections {
		if section.OutlineNodeID != "" {
			byOutlineNode[section.OutlineNodeID] = section
		}
	}
	flat := flattenOutline(outline.Sections)
	for index, node := range flat {
		content := generatedSectionContent(report, node)
		section, exists := byOutlineNode[node.ID]
		if exists {
			if !section.ManualEdited {
				section.Title = node.Title
				section.Content = content
				section.GenerationStatus = JobStatusSucceeded
				section.ContentSource = ContentSourceAI
				section.GeneratedAt = &now
				section.LastJobID = job.ID
				section.UpdatedAt = now
				if _, err := repo.UpdateReportSection(ctx, section); err != nil {
					return mapRepositoryReadError(err, "update generated section failed")
				}
			}
			continue
		}
		section = ReportSection{
			ID:               newID(),
			ReportID:         report.ID,
			OutlineID:        outline.ID,
			OutlineNodeID:    node.ID,
			SectionPath:      node.ID,
			Title:            node.Title,
			Level:            node.Level,
			SortOrder:        index,
			Numbering:        node.Numbering,
			SectionType:      SectionTypeText,
			Content:          content,
			Tables:           []map[string]any{},
			GenerationStatus: JobStatusSucceeded,
			ContentSource:    ContentSourceAI,
			Version:          1,
			LastJobID:        job.ID,
			GeneratedAt:      &now,
			CreatedAt:        now,
			UpdatedAt:        now,
		}
		if _, err := repo.CreateReportSection(ctx, section); err != nil {
			return mapRepositoryReadError(err, "create generated section failed")
		}
	}
	report.Status = ReportStatusGenerated
	report.LatestJobID = job.ID
	report.GeneratedAt = &now
	report.UpdatedAt = now
	if _, err := repo.UpdateReportWorkflowState(ctx, report); err != nil {
		return mapRepositoryReadError(err, "update report workflow state failed")
	}
	return nil
}

func (s *ReportService) applySectionRegeneration(ctx context.Context, repo ReportRepository, report Report, job ReportJob, sectionID string, now time.Time) error {
	section, err := repo.GetReportSectionByID(ctx, sectionID)
	if err != nil {
		return mapRepositoryReadError(err, "report section not found")
	}
	if section.ReportID != report.ID {
		return NewError(CodeNotFound, "report section not found", nil)
	}
	section.Content = fmt.Sprintf("%s\n\n%s", section.Title, "This section was regenerated from the current report context and can be edited before export.")
	section.GenerationStatus = JobStatusSucceeded
	section.ContentSource = ContentSourceAI
	section.ManualEdited = false
	section.Version++
	section.LastJobID = job.ID
	section.GeneratedAt = &now
	section.UpdatedAt = now
	if _, err := repo.UpdateReportSection(ctx, section); err != nil {
		return mapRepositoryReadError(err, "update regenerated section failed")
	}
	return nil
}

func (s *ReportService) createReportFileInTx(ctx context.Context, repo ReportRepository, reqCtx RequestContext, report Report, jobID string, now time.Time) (ReportFile, error) {
	sections, err := repo.ListReportSections(ctx, report.ID)
	if err != nil {
		return ReportFile{}, dependencyError("list report sections", err)
	}
	content, err := BuildMinimalDOCX(report, sections)
	if err != nil {
		return ReportFile{}, NewError(CodeInternal, "build report file failed", err)
	}
	filename := safeReportFilename(report)
	fileID := newID()
	file := ReportFile{
		ID:          fileID,
		ReportID:    report.ID,
		JobID:       jobID,
		Filename:    filename,
		Format:      "docx",
		FileSize:    int64(len(content)),
		Status:      JobStatusSucceeded,
		ContentPath: "/api/v1/report-files/" + fileID + "/content",
		CreatedBy:   reqCtx.UserID,
		CreatedAt:   now,
	}
	created, err := repo.CreateReportFile(ctx, file)
	if err != nil {
		return ReportFile{}, mapRepositoryReadError(err, "create report file failed")
	}
	if err := repo.SaveReportFileContent(ctx, created.ID, content, now); err != nil {
		return ReportFile{}, mapRepositoryReadError(err, "save report file content failed")
	}
	report.Status = ReportStatusExported
	report.LatestJobID = jobID
	report.LatestReportFileID = created.ID
	report.ExportedAt = &now
	report.UpdatedAt = now
	if _, err := repo.UpdateReportWorkflowState(ctx, report); err != nil {
		return ReportFile{}, mapRepositoryReadError(err, "update report workflow state failed")
	}
	return created, nil
}

func requireAdmin(reqCtx RequestContext) error {
	if err := requireGatewayContext(reqCtx); err != nil {
		return err
	}
	if !reqCtx.IsAdmin() {
		return NewError(CodeForbidden, "admin permission is required", nil)
	}
	return nil
}

func createReportEvent(ctx context.Context, repo ReportRepository, reportID, jobID, eventType, message string, now time.Time) error {
	_, err := repo.CreateReportEvent(ctx, ReportEvent{
		ID:        newID(),
		ReportID:  reportID,
		JobID:     jobID,
		EventType: eventType,
		Message:   message,
		CreatedAt: now,
	})
	if err != nil {
		return mapRepositoryReadError(err, "create report event failed")
	}
	return nil
}

func currentOutline(ctx context.Context, repo ReportRepository, reportID string) (ReportOutline, error) {
	outlines, err := repo.ListReportOutlines(ctx, reportID)
	if err != nil {
		return ReportOutline{}, dependencyError("list report outlines", err)
	}
	for _, outline := range outlines {
		if outline.IsCurrent {
			return outline, nil
		}
	}
	if len(outlines) == 0 {
		return ReportOutline{}, NewError(CodeConflict, "report outline is required before content generation", nil)
	}
	sort.SliceStable(outlines, func(i, j int) bool { return outlines[i].Version > outlines[j].Version })
	return outlines[0], nil
}

func defaultOutlineForReport(report Report) []ReportOutlineNode {
	if report.ReportType == "coal_inventory_audit" {
		return RenumberOutline([]ReportOutlineNode{
			{ID: newID(), Title: "Audit Scope and Basis"},
			{ID: newID(), Title: "Inventory Count Findings"},
			{ID: newID(), Title: "Variance Analysis"},
			{ID: newID(), Title: "Remediation Recommendations"},
		})
	}
	return RenumberOutline([]ReportOutlineNode{
		{ID: newID(), Title: "Inspection Overview"},
		{ID: newID(), Title: "Equipment Operation Status"},
		{ID: newID(), Title: "Issue Analysis"},
		{ID: newID(), Title: "Remediation Recommendations"},
	})
}

func flattenOutline(nodes []ReportOutlineNode) []ReportOutlineNode {
	result := make([]ReportOutlineNode, 0)
	var walk func([]ReportOutlineNode)
	walk = func(items []ReportOutlineNode) {
		for _, item := range items {
			result = append(result, item)
			walk(item.Children)
		}
	}
	walk(nodes)
	return result
}

func generatedSectionContent(report Report, node ReportOutlineNode) string {
	numbering := strings.TrimSpace(node.Numbering)
	if numbering != "" {
		numbering += " "
	}
	return fmt.Sprintf("%s%s\n\nThis draft section summarizes %q for the selected report type, business object, and reporting year. The content is persisted and can be edited before DOCX export.", numbering, node.Title, report.Topic)
}

func safeReportFilename(report Report) string {
	name := strings.TrimSpace(report.Name)
	if name == "" {
		name = "report"
	}
	replacer := strings.NewReplacer("/", "-", "\\", "-", ":", "-", "*", "-", "?", "-", "\"", "", "<", "", ">", "", "|", "-")
	return replacer.Replace(name) + ".docx"
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func BuildMinimalDOCX(report Report, sections []ReportSection) ([]byte, error) {
	var buffer bytes.Buffer
	zipWriter := zip.NewWriter(&buffer)
	files := map[string]string{
		"[Content_Types].xml": `<?xml version="1.0" encoding="UTF-8"?><Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/><Default Extension="xml" ContentType="application/xml"/><Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/></Types>`,
		"_rels/.rels":         `<?xml version="1.0" encoding="UTF-8"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/></Relationships>`,
		"word/document.xml":   buildDocumentXML(report, sections),
	}
	for name, content := range files {
		writer, err := zipWriter.Create(name)
		if err != nil {
			return nil, err
		}
		if _, err := writer.Write([]byte(content)); err != nil {
			return nil, err
		}
	}
	if err := zipWriter.Close(); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func buildDocumentXML(report Report, sections []ReportSection) string {
	var builder strings.Builder
	builder.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?><w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body>`)
	writeParagraph(&builder, report.Name)
	writeParagraph(&builder, report.Topic)
	sort.SliceStable(sections, func(i, j int) bool {
		if sections[i].SortOrder == sections[j].SortOrder {
			return sections[i].CreatedAt.Before(sections[j].CreatedAt)
		}
		return sections[i].SortOrder < sections[j].SortOrder
	})
	for _, section := range sections {
		title := strings.TrimSpace(section.Numbering + " " + section.Title)
		writeParagraph(&builder, title)
		for _, paragraph := range strings.Split(section.Content, "\n") {
			if strings.TrimSpace(paragraph) != "" {
				writeParagraph(&builder, paragraph)
			}
		}
	}
	builder.WriteString(`<w:sectPr><w:pgSz w:w="11906" w:h="16838"/><w:pgMar w:top="1440" w:right="1440" w:bottom="1440" w:left="1440"/></w:sectPr></w:body></w:document>`)
	return builder.String()
}

func writeParagraph(builder *strings.Builder, text string) {
	builder.WriteString("<w:p><w:r><w:t>")
	_ = xml.EscapeText(builder, []byte(text))
	builder.WriteString("</w:t></w:r></w:p>")
}
