package service

import (
	"context"
	"strings"
)

var validFileFormats = map[string]bool{"docx": true}
var validNumberingModes = map[string]bool{"global": true, "by_chapter": true}

// GetReportSettings returns the singleton report settings row.
func (s *Service) GetReportSettings(ctx context.Context, reqCtx RequestContext) (ReportSettings, error) {
	if err := requireGatewayContext(reqCtx); err != nil {
		return ReportSettings{}, err
	}
	settings, err := s.repo.GetReportSettings(ctx)
	if err != nil {
		return ReportSettings{}, dependencyError("get report settings", err)
	}
	return settings, nil
}

// UpdateReportSettings patches the singleton report settings row.
// Provider baseUrl and apiKey must never be stored here; only profileId references.
func (s *Service) UpdateReportSettings(ctx context.Context, reqCtx RequestContext, input UpdateReportSettingsInput) (ReportSettings, error) {
	if err := requireGatewayContext(reqCtx); err != nil {
		return ReportSettings{}, err
	}
	if err := validateUpdateReportSettingsInput(input); err != nil {
		return ReportSettings{}, err
	}
	settings, err := s.repo.UpdateReportSettings(ctx, input)
	if err != nil {
		return ReportSettings{}, dependencyError("update report settings", err)
	}
	return settings, nil
}

func validateUpdateReportSettingsInput(input UpdateReportSettingsInput) error {
	fields := map[string]string{}
	if input.DefaultFileFormat != nil {
		v := strings.TrimSpace(*input.DefaultFileFormat)
		if !validFileFormats[v] {
			fields["defaultFormat"] = "must be one of: docx"
		}
	}
	if input.DefaultNumberingMode != nil {
		v := strings.TrimSpace(*input.DefaultNumberingMode)
		if !validNumberingModes[v] {
			fields["defaultNumberingMode"] = "must be one of: global, by_chapter"
		}
	}
	if len(fields) > 0 {
		return ValidationError(fields)
	}
	return nil
}

// GetReportStatisticsOverview returns aggregate counts and 30-day trend.
func (s *Service) GetReportStatisticsOverview(ctx context.Context, reqCtx RequestContext) (ReportStatisticsOverview, error) {
	if err := requireGatewayContext(reqCtx); err != nil {
		return ReportStatisticsOverview{}, err
	}
	overview, err := s.repo.GetReportStatisticsOverview(ctx)
	if err != nil {
		return ReportStatisticsOverview{}, dependencyError("get report statistics overview", err)
	}
	return overview, nil
}

// ListOperationLogs returns a paginated list of report operation logs.
func (s *Service) ListOperationLogs(ctx context.Context, reqCtx RequestContext, filter OperationLogListFilter) (OperationLogListResult, error) {
	if err := requireGatewayContext(reqCtx); err != nil {
		return OperationLogListResult{}, err
	}
	filter.Page, filter.PageSize = normalizePage(filter.Page, filter.PageSize)
	result, err := s.repo.ListOperationLogs(ctx, filter)
	if err != nil {
		return OperationLogListResult{}, dependencyError("list operation logs", err)
	}
	return result, nil
}
