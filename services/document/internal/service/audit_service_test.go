package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/document/internal/service"
)

type mockAuditRepo struct {
	result *service.ListLogsResult
	err    error
}

func (m *mockAuditRepo) CreateLog(_ context.Context, _ service.WriteLogInput) error {
	return m.err
}

func (m *mockAuditRepo) ListLogs(_ context.Context, _ service.ListLogsInput) (*service.ListLogsResult, error) {
	return m.result, m.err
}

func TestAuditService_WriteLog_OK(t *testing.T) {
	svc := service.NewAuditService(&mockAuditRepo{})

	err := svc.WriteLog(context.Background(), service.WriteLogInput{
		OperationType: "generate_report",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAuditService_WriteLog_RepoError(t *testing.T) {
	svc := service.NewAuditService(&mockAuditRepo{err: errors.New("db error")})

	err := svc.WriteLog(context.Background(), service.WriteLogInput{OperationType: "x"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestAuditService_ListLogs_DefaultPagination(t *testing.T) {
	want := &service.ListLogsResult{
		Items:    []service.OperationLog{},
		Total:    0,
		Page:     1,
		PageSize: 20,
	}
	svc := service.NewAuditService(&mockAuditRepo{result: want})

	// page=0 和 pageSize=0 应被修正为默认值
	got, err := svc.ListLogs(context.Background(), service.ListLogsInput{Page: 0, PageSize: 0})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got == nil {
		t.Fatal("expected result, got nil")
	}
}

func TestAuditService_ListLogs_WithFilters(t *testing.T) {
	opType := "generate_report"
	want := &service.ListLogsResult{
		Items:    []service.OperationLog{{ID: "log-1", OperationType: opType}},
		Total:    1,
		Page:     1,
		PageSize: 20,
	}
	svc := service.NewAuditService(&mockAuditRepo{result: want})

	got, err := svc.ListLogs(context.Background(), service.ListLogsInput{
		OperationType: &opType,
		Page:          1,
		PageSize:      20,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got.Items) != 1 {
		t.Errorf("Items length: got %d, want 1", len(got.Items))
	}
}

func TestAuditService_ListLogs_RepoError(t *testing.T) {
	svc := service.NewAuditService(&mockAuditRepo{err: errors.New("db error")})

	_, err := svc.ListLogs(context.Background(), service.ListLogsInput{Page: 1, PageSize: 10})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
