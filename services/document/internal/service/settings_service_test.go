package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/document/internal/service"
)

// mockSettingsRepo 实现 SettingsRepository 接口，供测试使用。
type mockSettingsRepo struct {
	settings *service.ReportSettings
	err      error
}

func (m *mockSettingsRepo) GetSettings(_ context.Context) (*service.ReportSettings, error) {
	return m.settings, m.err
}

func (m *mockSettingsRepo) UpdateSettings(_ context.Context, _ service.UpdateReportSettingsInput) (*service.ReportSettings, error) {
	return m.settings, m.err
}

func newTestSettings() *service.ReportSettings {
	id := "test-id"
	return &service.ReportSettings{
		ID:                   id,
		DefaultFileFormat:    "docx",
		DefaultNumberingMode: "global",
		UpdatedAt:            time.Now(),
		CreatedAt:            time.Now(),
	}
}

func TestSettingsService_GetSettings_OK(t *testing.T) {
	want := newTestSettings()
	svc := service.NewSettingsService(&mockSettingsRepo{settings: want})

	got, err := svc.GetSettings(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != want.ID {
		t.Errorf("ID: got %q, want %q", got.ID, want.ID)
	}
}

func TestSettingsService_GetSettings_RepoError(t *testing.T) {
	svc := service.NewSettingsService(&mockSettingsRepo{err: errors.New("db error")})

	_, err := svc.GetSettings(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestSettingsService_UpdateSettings_OK(t *testing.T) {
	want := newTestSettings()
	format := "pdf"
	want.DefaultFileFormat = format

	svc := service.NewSettingsService(&mockSettingsRepo{settings: want})

	got, err := svc.UpdateSettings(context.Background(), service.UpdateReportSettingsInput{
		DefaultFileFormat: &format,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.DefaultFileFormat != format {
		t.Errorf("DefaultFileFormat: got %q, want %q", got.DefaultFileFormat, format)
	}
}

func TestSettingsService_UpdateSettings_RepoError(t *testing.T) {
	svc := service.NewSettingsService(&mockSettingsRepo{err: errors.New("db error")})

	_, err := svc.UpdateSettings(context.Background(), service.UpdateReportSettingsInput{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
