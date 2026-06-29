package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/document/internal/service"
)

type mockStatisticsRepo struct {
	overview *service.StatOverview
	trend    []service.DailyTrend
	err      error
}

func (m *mockStatisticsRepo) GetOverview(_ context.Context) (*service.StatOverview, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.overview, nil
}

func (m *mockStatisticsRepo) GetDailyTrend(_ context.Context) ([]service.DailyTrend, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.trend, nil
}

func TestStatisticsService_GetOverview_MergesTrend(t *testing.T) {
	repo := &mockStatisticsRepo{
		overview: &service.StatOverview{
			TemplateCount:  2,
			ReportCount:    10,
			GeneratedCount: 8,
			FailedCount:    2,
		},
		trend: []service.DailyTrend{
			{Date: time.Now(), GeneratedCount: 3},
			{Date: time.Now().AddDate(0, 0, -1), GeneratedCount: 5},
		},
	}

	svc := service.NewStatisticsService(repo)
	got, err := svc.GetOverview(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.TemplateCount != 2 {
		t.Errorf("TemplateCount: got %d, want 2", got.TemplateCount)
	}
	if len(got.Trend30d) != 2 {
		t.Errorf("Trend30d length: got %d, want 2", len(got.Trend30d))
	}
}

func TestStatisticsService_GetOverview_OverviewError(t *testing.T) {
	repo := &mockStatisticsRepo{err: errors.New("db error")}
	svc := service.NewStatisticsService(repo)

	_, err := svc.GetOverview(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestStatisticsService_GetOverview_EmptyTrend(t *testing.T) {
	repo := &mockStatisticsRepo{
		overview: &service.StatOverview{ReportCount: 0},
		trend:    []service.DailyTrend{},
	}
	svc := service.NewStatisticsService(repo)

	got, err := svc.GetOverview(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got.Trend30d) != 0 {
		t.Errorf("expected empty trend, got %d items", len(got.Trend30d))
	}
}
