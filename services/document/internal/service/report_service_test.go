package service

import (
	"bytes"
	"context"
	"testing"
	"time"
)

// fakeReportRepository is an in-memory ReportRepository used to unit test
// ReportService business rules without standing up PostgreSQL.
type fakeReportRepository struct {
	reports        map[string]Report
	outlines       map[string]ReportOutline
	sections       map[string]ReportSection
	sectionVersion map[string][]ReportSectionVersion
	jobs           map[string]ReportJob
	attempts       map[string][]ReportJobAttempt
	events         map[string][]ReportEvent
	files          map[string]ReportFile
	fileContents   map[string][]byte
	settings       ReportSettings
}

func newFakeReportRepository() *fakeReportRepository {
	return &fakeReportRepository{
		reports:        map[string]Report{},
		outlines:       map[string]ReportOutline{},
		sections:       map[string]ReportSection{},
		sectionVersion: map[string][]ReportSectionVersion{},
		jobs:           map[string]ReportJob{},
		attempts:       map[string][]ReportJobAttempt{},
		events:         map[string][]ReportEvent{},
		files:          map[string]ReportFile{},
		fileContents:   map[string][]byte{},
	}
}

func (f *fakeReportRepository) CreateReport(_ context.Context, value Report) (Report, error) {
	f.reports[value.ID] = value
	return value, nil
}

func (f *fakeReportRepository) GetReportByID(_ context.Context, id string) (Report, error) {
	report, ok := f.reports[id]
	if !ok {
		return Report{}, NewError(CodeNotFound, "report not found", nil)
	}
	return report, nil
}

func (f *fakeReportRepository) ListReports(_ context.Context, filter ReportListFilter) ([]Report, int, error) {
	var result []Report
	for _, report := range f.reports {
		if filter.CreatorID != "" && report.CreatorID != filter.CreatorID {
			continue
		}
		result = append(result, report)
	}
	return result, len(result), nil
}

func (f *fakeReportRepository) UpdateReport(_ context.Context, value Report) (Report, error) {
	if _, ok := f.reports[value.ID]; !ok {
		return Report{}, NewError(CodeNotFound, "report not found", nil)
	}
	f.reports[value.ID] = value
	return value, nil
}

func (f *fakeReportRepository) UpdateReportWorkflowState(_ context.Context, value Report) (Report, error) {
	report, ok := f.reports[value.ID]
	if !ok {
		return Report{}, NewError(CodeNotFound, "report not found", nil)
	}
	report.Status = value.Status
	report.LatestJobID = value.LatestJobID
	report.LatestReportFileID = value.LatestReportFileID
	report.GeneratedAt = value.GeneratedAt
	report.ExportedAt = value.ExportedAt
	report.UpdatedAt = value.UpdatedAt
	f.reports[value.ID] = report
	return report, nil
}

func (f *fakeReportRepository) SoftDeleteReport(_ context.Context, id string, deletedAt time.Time) (Report, error) {
	report, ok := f.reports[id]
	if !ok {
		return Report{}, NewError(CodeNotFound, "report not found", nil)
	}
	report.Status = ReportStatusDeleted
	report.DeletedAt = &deletedAt
	f.reports[id] = report
	return report, nil
}

func (f *fakeReportRepository) CreateReportOutline(_ context.Context, value ReportOutline) (ReportOutline, error) {
	if value.IsCurrent {
		for id, outline := range f.outlines {
			if outline.ReportID == value.ReportID {
				outline.IsCurrent = false
				f.outlines[id] = outline
			}
		}
	}
	f.outlines[value.ID] = value
	return value, nil
}

func (f *fakeReportRepository) ListReportOutlines(_ context.Context, reportID string) ([]ReportOutline, error) {
	var result []ReportOutline
	for _, outline := range f.outlines {
		if outline.ReportID == reportID {
			result = append(result, outline)
		}
	}
	return result, nil
}

func (f *fakeReportRepository) GetReportOutlineByID(_ context.Context, id string) (ReportOutline, error) {
	outline, ok := f.outlines[id]
	if !ok {
		return ReportOutline{}, NewError(CodeNotFound, "report outline not found", nil)
	}
	return outline, nil
}

func (f *fakeReportRepository) UpdateReportOutline(_ context.Context, value ReportOutline) (ReportOutline, error) {
	if _, ok := f.outlines[value.ID]; !ok {
		return ReportOutline{}, NewError(CodeNotFound, "report outline not found", nil)
	}
	f.outlines[value.ID] = value
	return value, nil
}

func (f *fakeReportRepository) CreateReportSection(_ context.Context, value ReportSection) (ReportSection, error) {
	f.sections[value.ID] = value
	return value, nil
}

func (f *fakeReportRepository) ListReportSections(_ context.Context, reportID string) ([]ReportSection, error) {
	var result []ReportSection
	for _, section := range f.sections {
		if section.ReportID == reportID {
			result = append(result, section)
		}
	}
	return result, nil
}

func (f *fakeReportRepository) GetReportSectionByID(_ context.Context, id string) (ReportSection, error) {
	section, ok := f.sections[id]
	if !ok {
		return ReportSection{}, NewError(CodeNotFound, "report section not found", nil)
	}
	return section, nil
}

func (f *fakeReportRepository) UpdateReportSection(_ context.Context, value ReportSection) (ReportSection, error) {
	if _, ok := f.sections[value.ID]; !ok {
		return ReportSection{}, NewError(CodeNotFound, "report section not found", nil)
	}
	f.sections[value.ID] = value
	return value, nil
}

func (f *fakeReportRepository) WithinTx(ctx context.Context, fn func(ReportRepository) error) error {
	return fn(f)
}

func (f *fakeReportRepository) CreateReportSectionVersion(_ context.Context, value ReportSectionVersion) (ReportSectionVersion, error) {
	f.sectionVersion[value.SectionID] = append(f.sectionVersion[value.SectionID], value)
	return value, nil
}

func (f *fakeReportRepository) ListReportSectionVersions(_ context.Context, sectionID string) ([]ReportSectionVersion, error) {
	return f.sectionVersion[sectionID], nil
}

func (f *fakeReportRepository) CreateReportJob(_ context.Context, value ReportJob) (ReportJob, error) {
	f.jobs[value.ID] = value
	return value, nil
}

func (f *fakeReportRepository) FindReportJobByID(_ context.Context, id string) (ReportJob, error) {
	job, ok := f.jobs[id]
	if !ok {
		return ReportJob{}, NewError(CodeNotFound, "report job not found", nil)
	}
	return job, nil
}

func (f *fakeReportRepository) ListReportJobs(_ context.Context, reportID string) ([]ReportJob, error) {
	result := []ReportJob{}
	for _, job := range f.jobs {
		if job.ReportID == reportID {
			result = append(result, job)
		}
	}
	return result, nil
}

func (f *fakeReportRepository) UpdateReportJobRetryState(_ context.Context, id string, retryCount int, status JobStatus, updatedAt time.Time) (ReportJob, error) {
	job, ok := f.jobs[id]
	if !ok {
		return ReportJob{}, NewError(CodeNotFound, "report job not found", nil)
	}
	job.RetryCount = retryCount
	job.Status = status
	job.ErrorCode = ""
	job.ErrorMessage = ""
	if job.StartedAt == nil {
		job.StartedAt = &updatedAt
	}
	job.FinishedAt = &updatedAt
	f.jobs[id] = job
	return job, nil
}

func (f *fakeReportRepository) CreateReportJobAttempt(_ context.Context, value ReportJobAttempt) (ReportJobAttempt, error) {
	f.attempts[value.JobID] = append(f.attempts[value.JobID], value)
	return value, nil
}

func (f *fakeReportRepository) ListReportJobAttempts(_ context.Context, jobID string) ([]ReportJobAttempt, error) {
	return f.attempts[jobID], nil
}

func (f *fakeReportRepository) CreateReportEvent(_ context.Context, value ReportEvent) (ReportEvent, error) {
	f.events[value.ReportID] = append(f.events[value.ReportID], value)
	return value, nil
}

func (f *fakeReportRepository) ListReportEvents(_ context.Context, reportID string) ([]ReportEvent, error) {
	return f.events[reportID], nil
}

func (f *fakeReportRepository) CreateReportFile(_ context.Context, value ReportFile) (ReportFile, error) {
	f.files[value.ID] = value
	return value, nil
}

func (f *fakeReportRepository) SaveReportFileContent(_ context.Context, reportFileID string, content []byte, _ time.Time) error {
	f.fileContents[reportFileID] = append([]byte(nil), content...)
	return nil
}

func (f *fakeReportRepository) GetReportFileByID(_ context.Context, id string) (ReportFile, error) {
	file, ok := f.files[id]
	if !ok {
		return ReportFile{}, NewError(CodeNotFound, "report file not found", nil)
	}
	return file, nil
}

func (f *fakeReportRepository) GetReportFileContent(_ context.Context, reportFileID string) ([]byte, error) {
	content, ok := f.fileContents[reportFileID]
	if !ok {
		return nil, NewError(CodeNotFound, "report file content not found", nil)
	}
	return append([]byte(nil), content...), nil
}

func (f *fakeReportRepository) ListReportFiles(_ context.Context, filter ReportFileListFilter) ([]ReportFile, int, error) {
	result := []ReportFile{}
	for _, file := range f.files {
		if filter.ReportID != "" && file.ReportID != filter.ReportID {
			continue
		}
		if filter.CreatorID != "" {
			report, ok := f.reports[file.ReportID]
			if !ok || report.CreatorID != filter.CreatorID || report.DeletedAt != nil {
				continue
			}
		}
		result = append(result, file)
	}
	return result, len(result), nil
}

func (f *fakeReportRepository) GetReportStatisticsOverview(context.Context, int) (ReportStatisticsOverview, error) {
	return ReportStatisticsOverview{
		ReportCount:     len(f.reports),
		TemplateCount:   0,
		MaterialCount:   0,
		JobStatusCounts: map[string]int{},
		RecentDays:      30,
	}, nil
}

func (f *fakeReportRepository) ListDailyReportStatistics(context.Context, int) ([]ReportDailyStatistic, error) {
	return nil, nil
}

func (f *fakeReportRepository) ListReportOperationLogs(context.Context, ReportOperationLogFilter) ([]ReportOperationLog, int, error) {
	return nil, 0, nil
}

func (f *fakeReportRepository) GetReportSettings(context.Context) (ReportSettings, error) {
	if f.settings.File == nil {
		return ReportSettings{LLM: map[string]any{"provider": "ai-gateway"}, DefaultTemplates: map[string]string{}, File: map[string]any{"defaultFormat": "docx"}}, nil
	}
	return f.settings, nil
}

func (f *fakeReportRepository) UpdateReportSettings(_ context.Context, value ReportSettings, _ string, updatedAt time.Time) (UpdateReportSettingsResult, error) {
	f.settings = value
	return UpdateReportSettingsResult{UpdatedAt: updatedAt}, nil
}

func newTestService() (*ReportService, *fakeReportRepository) {
	repo := newFakeReportRepository()
	svc := NewReportService(repo)
	svc.clock = func() time.Time { return time.Date(2026, 6, 29, 0, 0, 0, 0, time.UTC) }
	return svc, repo
}

func mustCreateReport(t *testing.T, svc *ReportService, owner string) Report {
	t.Helper()
	report, err := svc.CreateReport(context.Background(), RequestContext{UserID: owner}, CreateReportInput{
		Name:       "June report",
		ReportType: "summer_peak_inspection",
		TemplateID: "tpl-1",
		Topic:      "summer peak",
	})
	if err != nil {
		t.Fatalf("CreateReport() error = %v", err)
	}
	return report
}

func TestCreateReportValidatesRequiredFields(t *testing.T) {
	svc, _ := newTestService()
	_, err := svc.CreateReport(context.Background(), RequestContext{UserID: "u1"}, CreateReportInput{})
	appErr, ok := Classify(err)
	if !ok || appErr.Code != CodeValidation {
		t.Fatalf("expected validation error, got %v", err)
	}
}

func TestStandardUserCannotAccessOthersReport(t *testing.T) {
	svc, _ := newTestService()
	report := mustCreateReport(t, svc, "owner-1")

	_, err := svc.GetReport(context.Background(), RequestContext{UserID: "intruder"}, report.ID)
	appErr, ok := Classify(err)
	if !ok || appErr.Code != CodeForbidden {
		t.Fatalf("expected forbidden error, got %v", err)
	}
}

func TestAdminCanAccessOthersReport(t *testing.T) {
	svc, _ := newTestService()
	report := mustCreateReport(t, svc, "owner-1")

	got, err := svc.GetReport(context.Background(), RequestContext{UserID: "admin-1", Roles: []string{"admin"}}, report.ID)
	if err != nil {
		t.Fatalf("admin GetReport() error = %v", err)
	}
	if got.ID != report.ID {
		t.Fatalf("got report %q, want %q", got.ID, report.ID)
	}
}

func TestListReportsScopedToOwnerForStandardUser(t *testing.T) {
	svc, _ := newTestService()
	mustCreateReport(t, svc, "owner-1")
	mustCreateReport(t, svc, "owner-2")

	result, err := svc.ListReports(context.Background(), RequestContext{UserID: "owner-1"}, ReportListFilter{})
	if err != nil {
		t.Fatalf("ListReports() error = %v", err)
	}
	if result.Page.Total != 1 || len(result.Items) != 1 || result.Items[0].CreatorID != "owner-1" {
		t.Fatalf("expected only owner-1's report, got %+v", result)
	}
}

func TestListReportFilesScopedToOwnerForStandardUser(t *testing.T) {
	svc, repo := newTestService()
	ownerReport := mustCreateReport(t, svc, "owner-1")
	otherReport := mustCreateReport(t, svc, "owner-2")
	now := time.Date(2026, 6, 29, 12, 0, 0, 0, time.UTC)
	repo.files["file-owner"] = ReportFile{ID: "file-owner", ReportID: ownerReport.ID, Filename: "owner.docx", CreatedAt: now}
	repo.files["file-other"] = ReportFile{ID: "file-other", ReportID: otherReport.ID, Filename: "other.docx", CreatedAt: now}

	result, err := svc.ListReportFiles(context.Background(), RequestContext{UserID: "owner-1"}, ReportFileListFilter{})
	if err != nil {
		t.Fatalf("ListReportFiles() error = %v", err)
	}
	if result.Page.Total != 1 || len(result.Items) != 1 || result.Items[0].ReportID != ownerReport.ID {
		t.Fatalf("expected only owner-1's file, got %+v", result)
	}
}

func TestAdminCanListAllReportFiles(t *testing.T) {
	svc, repo := newTestService()
	ownerReport := mustCreateReport(t, svc, "owner-1")
	otherReport := mustCreateReport(t, svc, "owner-2")
	now := time.Date(2026, 6, 29, 12, 0, 0, 0, time.UTC)
	repo.files["file-owner"] = ReportFile{ID: "file-owner", ReportID: ownerReport.ID, Filename: "owner.docx", CreatedAt: now}
	repo.files["file-other"] = ReportFile{ID: "file-other", ReportID: otherReport.ID, Filename: "other.docx", CreatedAt: now}

	result, err := svc.ListReportFiles(context.Background(), RequestContext{UserID: "admin-1", Roles: []string{"admin"}}, ReportFileListFilter{})
	if err != nil {
		t.Fatalf("admin ListReportFiles() error = %v", err)
	}
	if result.Page.Total != 2 || len(result.Items) != 2 {
		t.Fatalf("expected admin to see all files, got %+v", result)
	}
}

func TestGlobalReportOperationsRequireAdmin(t *testing.T) {
	svc, _ := newTestService()
	actor := RequestContext{UserID: "owner-1"}

	cases := []struct {
		name string
		run  func() error
	}{
		{
			name: "statistics overview",
			run: func() error {
				_, err := svc.GetReportStatisticsOverview(context.Background(), actor)
				return err
			},
		},
		{
			name: "daily statistics",
			run: func() error {
				_, err := svc.ListDailyReportStatistics(context.Background(), actor, 30)
				return err
			},
		},
		{
			name: "operation logs",
			run: func() error {
				_, err := svc.ListReportOperationLogs(context.Background(), actor, ReportOperationLogFilter{})
				return err
			},
		},
		{
			name: "get settings",
			run: func() error {
				_, err := svc.GetReportSettings(context.Background(), actor)
				return err
			},
		},
		{
			name: "update settings",
			run: func() error {
				_, err := svc.UpdateReportSettings(context.Background(), actor, ReportSettings{
					LLM:              map[string]any{"provider": "ai-gateway"},
					DefaultTemplates: map[string]string{},
					File:             map[string]any{"defaultFormat": "docx"},
				})
				return err
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.run()
			appErr, ok := Classify(err)
			if !ok || appErr.Code != CodeForbidden {
				t.Fatalf("expected forbidden error, got %v", err)
			}
		})
	}
}

func TestSoftDeleteReportIsIdempotentAndConflicts(t *testing.T) {
	svc, _ := newTestService()
	report := mustCreateReport(t, svc, "owner-1")
	actor := RequestContext{UserID: "owner-1"}

	if err := svc.SoftDeleteReport(context.Background(), actor, report.ID); err != nil {
		t.Fatalf("first SoftDeleteReport() error = %v", err)
	}

	err := svc.SoftDeleteReport(context.Background(), actor, report.ID)
	appErr, ok := Classify(err)
	if !ok || appErr.Code != CodeConflict {
		t.Fatalf("expected conflict on second delete, got %v", err)
	}
}

func TestUpdateReportRejectsDeletedReport(t *testing.T) {
	svc, _ := newTestService()
	report := mustCreateReport(t, svc, "owner-1")
	actor := RequestContext{UserID: "owner-1"}
	if err := svc.SoftDeleteReport(context.Background(), actor, report.ID); err != nil {
		t.Fatalf("SoftDeleteReport() error = %v", err)
	}

	newTopic := "updated topic"
	_, err := svc.UpdateReport(context.Background(), actor, report.ID, UpdateReportInput{Topic: &newTopic})
	appErr, ok := Classify(err)
	if !ok || appErr.Code != CodeConflict {
		t.Fatalf("expected conflict updating deleted report, got %v", err)
	}
}

func TestCreateOutlineRenumbersAndVersions(t *testing.T) {
	svc, _ := newTestService()
	report := mustCreateReport(t, svc, "owner-1")
	actor := RequestContext{UserID: "owner-1"}

	outline, err := svc.CreateOutline(context.Background(), actor, report.ID, CreateOutlineInput{
		Source: OutlineSourceManual,
		Sections: []ReportOutlineNode{
			{Title: "Intro"},
			{Title: "Body", Children: []ReportOutlineNode{{Title: "Detail"}}},
		},
	})
	if err != nil {
		t.Fatalf("CreateOutline() error = %v", err)
	}
	if outline.Version != 1 || !outline.IsCurrent {
		t.Fatalf("unexpected outline version/current: %+v", outline)
	}
	if outline.Sections[1].Children[0].Numbering != "2.1" {
		t.Fatalf("expected renumbered child 2.1, got %q", outline.Sections[1].Children[0].Numbering)
	}

	second, err := svc.CreateOutline(context.Background(), actor, report.ID, CreateOutlineInput{
		Source:   OutlineSourceAI,
		Sections: []ReportOutlineNode{{Title: "Regenerated"}},
	})
	if err != nil {
		t.Fatalf("second CreateOutline() error = %v", err)
	}
	if second.Version != 2 {
		t.Fatalf("expected version 2, got %d", second.Version)
	}
}

func TestDeleteOutlineSectionRenumbersRemaining(t *testing.T) {
	svc, _ := newTestService()
	report := mustCreateReport(t, svc, "owner-1")
	actor := RequestContext{UserID: "owner-1"}

	outline, err := svc.CreateOutline(context.Background(), actor, report.ID, CreateOutlineInput{
		Source: OutlineSourceManual,
		Sections: []ReportOutlineNode{
			{Title: "Intro"},
			{Title: "Body"},
			{Title: "Conclusion"},
		},
	})
	if err != nil {
		t.Fatalf("CreateOutline() error = %v", err)
	}
	bodyID := outline.Sections[1].ID

	updated, err := svc.DeleteOutlineSection(context.Background(), actor, report.ID, outline.ID, bodyID)
	if err != nil {
		t.Fatalf("DeleteOutlineSection() error = %v", err)
	}
	if len(updated.Sections) != 2 {
		t.Fatalf("expected 2 remaining sections, got %d", len(updated.Sections))
	}
	if updated.Sections[1].Numbering != "2" {
		t.Fatalf("expected conclusion renumbered to 2, got %q", updated.Sections[1].Numbering)
	}
	if !updated.ManualEdited {
		t.Fatalf("expected manualEdited = true after delete")
	}
}

func TestDeleteOutlineSectionNotFound(t *testing.T) {
	svc, _ := newTestService()
	report := mustCreateReport(t, svc, "owner-1")
	actor := RequestContext{UserID: "owner-1"}
	outline, err := svc.CreateOutline(context.Background(), actor, report.ID, CreateOutlineInput{
		Source:   OutlineSourceManual,
		Sections: []ReportOutlineNode{{Title: "Intro"}},
	})
	if err != nil {
		t.Fatalf("CreateOutline() error = %v", err)
	}

	_, err = svc.DeleteOutlineSection(context.Background(), actor, report.ID, outline.ID, "missing-node")
	appErr, ok := Classify(err)
	if !ok || appErr.Code != CodeNotFound {
		t.Fatalf("expected not_found error, got %v", err)
	}
}

func TestUpdateSectionMarksManualEditedAndBumpsVersion(t *testing.T) {
	svc, _ := newTestService()
	report := mustCreateReport(t, svc, "owner-1")
	actor := RequestContext{UserID: "owner-1"}

	section, err := svc.CreateSection(context.Background(), actor, report.ID, CreateSectionInput{Title: "Intro"})
	if err != nil {
		t.Fatalf("CreateSection() error = %v", err)
	}
	if section.Version != 1 {
		t.Fatalf("expected initial version 1, got %d", section.Version)
	}

	newContent := "edited body"
	updated, err := svc.UpdateSection(context.Background(), actor, report.ID, section.ID, UpdateSectionInput{Content: &newContent})
	if err != nil {
		t.Fatalf("UpdateSection() error = %v", err)
	}
	if updated.Version != 2 {
		t.Fatalf("expected version bumped to 2, got %d", updated.Version)
	}
	if !updated.ManualEdited {
		t.Fatalf("expected manualEdited = true")
	}
	if updated.ContentSource != ContentSourceManual {
		t.Fatalf("expected contentSource manual, got %q", updated.ContentSource)
	}
}

func TestUpdateSectionContentEditCannotBeUnmarkedAsManual(t *testing.T) {
	svc, _ := newTestService()
	report := mustCreateReport(t, svc, "owner-1")
	actor := RequestContext{UserID: "owner-1"}

	section, err := svc.CreateSection(context.Background(), actor, report.ID, CreateSectionInput{Title: "Intro"})
	if err != nil {
		t.Fatalf("CreateSection() error = %v", err)
	}

	newContent := "edited body"
	manualEdited := false
	updated, err := svc.UpdateSection(context.Background(), actor, report.ID, section.ID, UpdateSectionInput{
		Content:      &newContent,
		ManualEdited: &manualEdited,
	})
	if err != nil {
		t.Fatalf("UpdateSection() error = %v", err)
	}
	if !updated.ManualEdited {
		t.Fatalf("expected manualEdited to stay true even though the request set manualEdited:false alongside a content change")
	}
}

func TestSaveSectionsUpdatesExistingAndCreatesNewSections(t *testing.T) {
	svc, repo := newTestService()
	report := mustCreateReport(t, svc, "owner-1")
	actor := RequestContext{UserID: "owner-1"}

	existing, err := svc.CreateSection(context.Background(), actor, report.ID, CreateSectionInput{
		Title:   "Intro",
		Content: "original body",
		Tables:  []map[string]any{{"name": "old"}},
	})
	if err != nil {
		t.Fatalf("CreateSection() error = %v", err)
	}

	newTitle := "Updated intro"
	newContent := "edited body"
	newTables := []map[string]any{{"name": "updated"}}
	createdTitle := "New section"
	createdContent := "new body"
	sections, err := svc.SaveSections(context.Background(), actor, report.ID, SaveSectionsInput{
		Sections: []SaveSectionInput{
			{
				ID:      existing.ID,
				Title:   &newTitle,
				Content: &newContent,
				Tables:  &newTables,
			},
			{
				Title:   &createdTitle,
				Content: &createdContent,
			},
		},
	})
	if err != nil {
		t.Fatalf("SaveSections() error = %v", err)
	}
	if len(sections) != 2 {
		t.Fatalf("SaveSections() len = %d, want 2", len(sections))
	}

	updated := sections[0]
	if updated.ID != existing.ID {
		t.Fatalf("first section ID = %q, want %q", updated.ID, existing.ID)
	}
	if updated.Title != newTitle || updated.Content != newContent {
		t.Fatalf("updated section did not preserve requested fields: %+v", updated)
	}
	if updated.Version != existing.Version+1 {
		t.Fatalf("updated version = %d, want %d", updated.Version, existing.Version+1)
	}
	if !updated.ManualEdited {
		t.Fatalf("expected updated section to be marked manual edited")
	}

	created := sections[1]
	if created.ID == "" || created.ID == existing.ID {
		t.Fatalf("new section ID was not generated: %+v", created)
	}
	if created.Title != createdTitle || created.Content != createdContent {
		t.Fatalf("created section did not preserve requested fields: %+v", created)
	}
	if created.ManualEdited != true || created.Version != 1 {
		t.Fatalf("unexpected created manual/version fields: %+v", created)
	}
	if repo.sections[existing.ID].Content != newContent {
		t.Fatalf("repository did not persist updated section: %+v", repo.sections[existing.ID])
	}
	if _, ok := repo.sections[created.ID]; !ok {
		t.Fatalf("repository did not persist created section %q", created.ID)
	}
}

func TestSaveSectionsUpdatesMetadataWithoutBumpingVersion(t *testing.T) {
	svc, _ := newTestService()
	report := mustCreateReport(t, svc, "owner-1")
	actor := RequestContext{UserID: "owner-1"}

	existing, err := svc.CreateSection(context.Background(), actor, report.ID, CreateSectionInput{
		Title:         "Intro",
		Level:         1,
		Numbering:     "1",
		OutlineNodeID: "outline-1",
	})
	if err != nil {
		t.Fatalf("CreateSection() error = %v", err)
	}
	parent, err := svc.CreateSection(context.Background(), actor, report.ID, CreateSectionInput{Title: "Parent"})
	if err != nil {
		t.Fatalf("CreateSection(parent) error = %v", err)
	}

	parentID := parent.ID
	outlineNodeID := "outline-2"
	title := "Updated intro"
	level := 2
	numbering := "1.1"
	manualEdited := false
	sections, err := svc.SaveSections(context.Background(), actor, report.ID, SaveSectionsInput{
		Sections: []SaveSectionInput{{
			ID:            existing.ID,
			ParentID:      &parentID,
			OutlineNodeID: &outlineNodeID,
			Title:         &title,
			Level:         &level,
			Numbering:     &numbering,
			ManualEdited:  &manualEdited,
		}},
	})
	if err != nil {
		t.Fatalf("SaveSections() error = %v", err)
	}
	if len(sections) != 1 {
		t.Fatalf("SaveSections() len = %d, want 1", len(sections))
	}

	updated := sections[0]
	if updated.ParentID != parentID || updated.OutlineNodeID != outlineNodeID || updated.Level != level || updated.Numbering != numbering {
		t.Fatalf("metadata fields were not saved: %+v", updated)
	}
	if updated.Title != title {
		t.Fatalf("Title = %q, want %q", updated.Title, title)
	}
	if updated.Version != existing.Version {
		t.Fatalf("metadata-only save bumped version to %d, want %d", updated.Version, existing.Version)
	}
	if updated.ManualEdited {
		t.Fatalf("metadata-only save should respect manualEdited=false when content is unchanged")
	}
}

func TestCreateSectionRejectsParentFromAnotherReport(t *testing.T) {
	svc, _ := newTestService()
	report := mustCreateReport(t, svc, "owner-1")
	otherReport := mustCreateReport(t, svc, "owner-1")
	actor := RequestContext{UserID: "owner-1"}

	otherParent, err := svc.CreateSection(context.Background(), actor, otherReport.ID, CreateSectionInput{Title: "Other parent"})
	if err != nil {
		t.Fatalf("CreateSection(other parent) error = %v", err)
	}

	_, err = svc.CreateSection(context.Background(), actor, report.ID, CreateSectionInput{Title: "Child", ParentID: otherParent.ID})
	appErr, ok := Classify(err)
	if !ok || appErr.Code != CodeValidation || appErr.Fields["parentId"] == "" {
		t.Fatalf("expected parentId validation error, got %v", err)
	}
}

func TestSaveSectionsRejectsParentCycle(t *testing.T) {
	svc, _ := newTestService()
	report := mustCreateReport(t, svc, "owner-1")
	actor := RequestContext{UserID: "owner-1"}

	first, err := svc.CreateSection(context.Background(), actor, report.ID, CreateSectionInput{Title: "First"})
	if err != nil {
		t.Fatalf("CreateSection(first) error = %v", err)
	}
	second, err := svc.CreateSection(context.Background(), actor, report.ID, CreateSectionInput{Title: "Second"})
	if err != nil {
		t.Fatalf("CreateSection(second) error = %v", err)
	}

	firstParent := second.ID
	secondParent := first.ID
	_, err = svc.SaveSections(context.Background(), actor, report.ID, SaveSectionsInput{
		Sections: []SaveSectionInput{
			{ID: first.ID, ParentID: &firstParent},
			{ID: second.ID, ParentID: &secondParent},
		},
	})
	appErr, ok := Classify(err)
	if !ok || appErr.Code != CodeValidation || appErr.Fields["parentId"] == "" {
		t.Fatalf("expected parentId cycle validation error, got %v", err)
	}
}

func TestSaveSectionsPersistsExplicitSortOrder(t *testing.T) {
	svc, repo := newTestService()
	report := mustCreateReport(t, svc, "owner-1")
	actor := RequestContext{UserID: "owner-1"}

	first, err := svc.CreateSection(context.Background(), actor, report.ID, CreateSectionInput{Title: "First"})
	if err != nil {
		t.Fatalf("CreateSection(first) error = %v", err)
	}
	second, err := svc.CreateSection(context.Background(), actor, report.ID, CreateSectionInput{Title: "Second"})
	if err != nil {
		t.Fatalf("CreateSection(second) error = %v", err)
	}

	firstSortOrder := 1
	secondSortOrder := 0
	_, err = svc.SaveSections(context.Background(), actor, report.ID, SaveSectionsInput{
		Sections: []SaveSectionInput{
			{ID: second.ID, SortOrder: &secondSortOrder},
			{ID: first.ID, SortOrder: &firstSortOrder},
		},
	})
	if err != nil {
		t.Fatalf("SaveSections() error = %v", err)
	}
	if repo.sections[first.ID].SortOrder != firstSortOrder || repo.sections[second.ID].SortOrder != secondSortOrder {
		t.Fatalf("sortOrder was not persisted: first=%d second=%d", repo.sections[first.ID].SortOrder, repo.sections[second.ID].SortOrder)
	}
}

func TestCreateSectionPersistsExplicitSortOrder(t *testing.T) {
	svc, _ := newTestService()
	report := mustCreateReport(t, svc, "owner-1")
	actor := RequestContext{UserID: "owner-1"}

	sortOrder := 5
	section, err := svc.CreateSection(context.Background(), actor, report.ID, CreateSectionInput{
		Title:     "Sorted section",
		SortOrder: &sortOrder,
	})
	if err != nil {
		t.Fatalf("CreateSection() error = %v", err)
	}
	if section.SortOrder != sortOrder {
		t.Fatalf("SortOrder = %d, want %d", section.SortOrder, sortOrder)
	}
}

func TestCreateSectionWithoutContentDefaultsToManualSource(t *testing.T) {
	svc, _ := newTestService()
	report := mustCreateReport(t, svc, "owner-1")
	actor := RequestContext{UserID: "owner-1"}

	section, err := svc.CreateSection(context.Background(), actor, report.ID, CreateSectionInput{Title: "Intro"})
	if err != nil {
		t.Fatalf("CreateSection() error = %v", err)
	}
	if section.ContentSource != ContentSourceManual {
		t.Fatalf("expected contentSource manual for a content-less section, got %q", section.ContentSource)
	}
	if section.ManualEdited {
		t.Fatalf("expected manualEdited = false for a section created without content")
	}
}

func TestUpdateSectionConflictsWhileGenerationRunning(t *testing.T) {
	svc, repo := newTestService()
	report := mustCreateReport(t, svc, "owner-1")
	actor := RequestContext{UserID: "owner-1"}

	section, err := svc.CreateSection(context.Background(), actor, report.ID, CreateSectionInput{Title: "Intro"})
	if err != nil {
		t.Fatalf("CreateSection() error = %v", err)
	}
	section.GenerationStatus = JobStatusRunning
	repo.sections[section.ID] = section

	newContent := "should not apply"
	_, err = svc.UpdateSection(context.Background(), actor, report.ID, section.ID, UpdateSectionInput{Content: &newContent})
	appErr, ok := Classify(err)
	if !ok || appErr.Code != CodeConflict {
		t.Fatalf("expected conflict while generation running, got %v", err)
	}
}

func TestCreateSectionVersionDoesNotRequireRegeneration(t *testing.T) {
	svc, _ := newTestService()
	report := mustCreateReport(t, svc, "owner-1")
	actor := RequestContext{UserID: "owner-1"}

	section, err := svc.CreateSection(context.Background(), actor, report.ID, CreateSectionInput{Title: "Intro", Content: "v1"})
	if err != nil {
		t.Fatalf("CreateSection() error = %v", err)
	}

	version, err := svc.CreateSectionVersion(context.Background(), actor, report.ID, section.ID, CreateSectionVersionInput{Source: ContentSourceManual})
	if err != nil {
		t.Fatalf("CreateSectionVersion() error = %v", err)
	}
	if version.Version != 1 || version.Content != "v1" {
		t.Fatalf("unexpected first version: %+v", version)
	}

	second, err := svc.CreateSectionVersion(context.Background(), actor, report.ID, section.ID, CreateSectionVersionInput{Source: ContentSourceManual})
	if err != nil {
		t.Fatalf("CreateSectionVersion() error = %v", err)
	}
	if second.Version != 2 {
		t.Fatalf("expected version 2, got %d", second.Version)
	}
}

func TestCreateReportJobsGenerateOutlineAndContent(t *testing.T) {
	svc, repo := newTestService()
	report := mustCreateReport(t, svc, "owner-1")
	actor := RequestContext{UserID: "owner-1", RequestID: "req-test"}

	outlineJob, err := svc.CreateReportJob(context.Background(), actor, report.ID, CreateReportJobInput{
		JobType:     JobTypeOutlineGeneration,
		TargetScope: "outline",
	})
	if err != nil {
		t.Fatalf("CreateReportJob(outline) error = %v", err)
	}
	if outlineJob.Status != JobStatusSucceeded {
		t.Fatalf("outline job status = %q", outlineJob.Status)
	}
	outlines, err := svc.ListOutlines(context.Background(), actor, report.ID)
	if err != nil {
		t.Fatalf("ListOutlines() error = %v", err)
	}
	if len(outlines) != 1 || len(outlines[0].Sections) == 0 {
		t.Fatalf("expected generated outline, got %+v", outlines)
	}

	contentJob, err := svc.CreateReportJob(context.Background(), actor, report.ID, CreateReportJobInput{
		JobType:     JobTypeContentGeneration,
		TargetScope: "report",
	})
	if err != nil {
		t.Fatalf("CreateReportJob(content) error = %v", err)
	}
	if contentJob.Status != JobStatusSucceeded {
		t.Fatalf("content job status = %q", contentJob.Status)
	}
	sections, err := svc.ListSections(context.Background(), actor, report.ID)
	if err != nil {
		t.Fatalf("ListSections() error = %v", err)
	}
	if len(sections) != len(outlines[0].Sections) {
		t.Fatalf("sections len = %d, want %d", len(sections), len(outlines[0].Sections))
	}
	for _, section := range sections {
		if section.GenerationStatus != JobStatusSucceeded || section.Content == "" || section.LastJobID != contentJob.ID {
			t.Fatalf("unexpected generated section: %+v", section)
		}
	}
	events := repo.events[report.ID]
	if len(events) < 4 {
		t.Fatalf("expected job lifecycle events, got %+v", events)
	}
}

func TestCreateReportJobAttemptUpdatesRetryState(t *testing.T) {
	svc, repo := newTestService()
	report := mustCreateReport(t, svc, "owner-1")
	actor := RequestContext{UserID: "owner-1"}
	job, err := svc.CreateReportJob(context.Background(), actor, report.ID, CreateReportJobInput{
		JobType: JobTypeOutlineGeneration,
	})
	if err != nil {
		t.Fatalf("CreateReportJob() error = %v", err)
	}
	failedJob := repo.jobs[job.ID]
	failedJob.Status = JobStatusFailed
	repo.jobs[job.ID] = failedJob

	attempt, err := svc.CreateReportJobAttempt(context.Background(), actor, job.ID, CreateReportJobAttemptInput{
		Reason: "retry after frontend error",
	})
	if err != nil {
		t.Fatalf("CreateReportJobAttempt() error = %v", err)
	}
	if attempt.AttemptNumber != 2 || attempt.Status != JobStatusSucceeded {
		t.Fatalf("unexpected retry attempt: %+v", attempt)
	}
	updated := repo.jobs[job.ID]
	if updated.RetryCount != 1 || updated.Status != JobStatusSucceeded {
		t.Fatalf("job retry state = retry_count:%d status:%s", updated.RetryCount, updated.Status)
	}
}

func TestCreateReportJobAttemptRejectsSucceededJob(t *testing.T) {
	svc, _ := newTestService()
	report := mustCreateReport(t, svc, "owner-1")
	actor := RequestContext{UserID: "owner-1"}
	job, err := svc.CreateReportJob(context.Background(), actor, report.ID, CreateReportJobInput{
		JobType: JobTypeOutlineGeneration,
	})
	if err != nil {
		t.Fatalf("CreateReportJob() error = %v", err)
	}

	_, err = svc.CreateReportJobAttempt(context.Background(), actor, job.ID, CreateReportJobAttemptInput{
		Reason: "retry should be rejected",
	})
	appErr, ok := Classify(err)
	if !ok || appErr.Code != CodeConflict {
		t.Fatalf("expected conflict retrying succeeded job, got %v", err)
	}
}

func TestCreateReportFileAndBuildContent(t *testing.T) {
	svc, repo := newTestService()
	report := mustCreateReport(t, svc, "owner-1")
	actor := RequestContext{UserID: "owner-1"}
	if _, err := svc.CreateReportJob(context.Background(), actor, report.ID, CreateReportJobInput{JobType: JobTypeOutlineGeneration}); err != nil {
		t.Fatalf("CreateReportJob(outline) error = %v", err)
	}
	if _, err := svc.CreateReportJob(context.Background(), actor, report.ID, CreateReportJobInput{JobType: JobTypeContentGeneration}); err != nil {
		t.Fatalf("CreateReportJob(content) error = %v", err)
	}

	file, err := svc.CreateReportFile(context.Background(), actor, CreateReportFileInput{
		ReportID: report.ID,
		Format:   "docx",
	})
	if err != nil {
		t.Fatalf("CreateReportFile() error = %v", err)
	}
	if file.Status != JobStatusSucceeded || file.FileSize == 0 || file.ContentPath == "" {
		t.Fatalf("unexpected report file: %+v", file)
	}

	_, content, err := svc.BuildReportFileContent(context.Background(), actor, file.ID)
	if err != nil {
		t.Fatalf("BuildReportFileContent() error = %v", err)
	}
	if len(content) < 100 || string(content[:2]) != "PK" {
		t.Fatalf("expected docx zip content, len=%d prefix=%q", len(content), content[:2])
	}
	if file.FileSize != int64(len(content)) {
		t.Fatalf("file size = %d, content len = %d", file.FileSize, len(content))
	}

	var sectionID string
	for id, section := range repo.sections {
		if section.ReportID == report.ID {
			sectionID = id
			break
		}
	}
	if sectionID == "" {
		t.Fatal("expected generated section to update")
	}
	updatedContent := "content edited after export"
	if _, err := svc.UpdateSection(context.Background(), actor, report.ID, sectionID, UpdateSectionInput{Content: &updatedContent}); err != nil {
		t.Fatalf("UpdateSection() error = %v", err)
	}

	_, contentAfterEdit, err := svc.BuildReportFileContent(context.Background(), actor, file.ID)
	if err != nil {
		t.Fatalf("BuildReportFileContent(after edit) error = %v", err)
	}
	if !bytes.Equal(contentAfterEdit, content) {
		t.Fatal("expected old report file id to return the original exported snapshot")
	}
}
