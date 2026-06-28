package service_test

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/qa/internal/domain"
	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/qa/internal/service"
)

func TestSSEWriterThinkingStepFormat(t *testing.T) {
	rec := httptest.NewRecorder()
	writer := service.NewSSEWriter(rec)

	err := writer.EmitStep(context.Background(), domain.ThinkingStep{
		Type:   domain.StepTypeGeneration,
		Label:  "生成回答",
		Status: domain.StepStatusRunning,
	})
	if err != nil {
		t.Fatal(err)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "event: thinking_step") {
		t.Fatalf("missing event type: %s", body)
	}
	if !strings.Contains(body, `"type":"generation"`) {
		t.Fatalf("missing step type: %s", body)
	}
}

func TestSameStepTypeRunningThenDone(t *testing.T) {
	steps := map[domain.StepType]domain.ThinkingStep{}

	upsert := func(step domain.ThinkingStep) domain.ThinkingStep {
		if existing, ok := steps[step.Type]; ok && step.Detail == "" {
			step.Detail = existing.Detail
		}
		steps[step.Type] = step
		return step
	}

	running := upsert(domain.ThinkingStep{
		Type: domain.StepTypeRetrieval, Label: "检索知识库", Status: domain.StepStatusRunning,
	})
	if running.Status != domain.StepStatusRunning {
		t.Fatalf("expected running")
	}

	done := upsert(domain.ThinkingStep{
		Type: domain.StepTypeRetrieval, Label: "检索知识库", Status: domain.StepStatusDone, Detail: "命中 5 条结果",
	})
	if done.Status != domain.StepStatusDone {
		t.Fatalf("expected done")
	}
	if len(steps) != 1 {
		t.Fatalf("same step_type must update one record, got %d", len(steps))
	}
}
