package service

import (
	"context"
	"fmt"
	"time"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/qa/internal/domain"
	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/qa/internal/repository"
)

type StepEmitter interface {
	EmitThinkingStep(ctx context.Context, step domain.ThinkingStep) error
}

type ProcessStepTracker struct {
	repo          *repository.ProcessStepRepository
	responseRunID string
	emitter       StepEmitter
}

func NewProcessStepTracker(
	repo *repository.ProcessStepRepository,
	responseRunID string,
	emitter StepEmitter,
) *ProcessStepTracker {
	return &ProcessStepTracker{
		repo:          repo,
		responseRunID: responseRunID,
		emitter:       emitter,
	}
}

func (t *ProcessStepTracker) StartStep(
	ctx context.Context,
	stepType domain.StepType,
	label string,
) (domain.ThinkingStep, error) {
	startedAt, exists, err := t.repo.GetStartedAt(ctx, t.responseRunID, stepType)
	if err != nil {
		return domain.ThinkingStep{}, err
	}
	if !exists {
		startedAt = time.Now().UTC()
	}

	record := domain.ProcessStepRecord{
		ResponseRunID: t.responseRunID,
		StepOrder:     domain.StepOrderFor(stepType),
		StepType:      stepType,
		Label:         label,
		Status:        domain.StepStatusRunning,
		StartedAt:     startedAt,
	}

	step, err := t.repo.UpsertStep(ctx, t.responseRunID, record)
	if err != nil {
		return domain.ThinkingStep{}, err
	}
	if err := t.emitter.EmitThinkingStep(ctx, step); err != nil {
		return domain.ThinkingStep{}, fmt.Errorf("emit thinking step: %w", err)
	}
	return step, nil
}

func (t *ProcessStepTracker) CompleteStep(
	ctx context.Context,
	stepType domain.StepType,
	label string,
	rawDetail string,
) (domain.ThinkingStep, error) {
	detail := domain.SanitizeStepDetail(rawDetail)

	startedAt, exists, err := t.repo.GetStartedAt(ctx, t.responseRunID, stepType)
	if err != nil {
		return domain.ThinkingStep{}, err
	}
	if !exists {
		startedAt = time.Now().UTC()
	}

	now := time.Now().UTC()
	record := domain.ProcessStepRecord{
		ResponseRunID: t.responseRunID,
		StepOrder:     domain.StepOrderFor(stepType),
		StepType:      stepType,
		Label:         label,
		Detail:        detail,
		Status:        domain.StepStatusDone,
		StartedAt:     startedAt,
		FinishedAt:    &now,
	}

	step, err := t.repo.UpsertStep(ctx, t.responseRunID, record)
	if err != nil {
		return domain.ThinkingStep{}, err
	}
	if err := t.emitter.EmitThinkingStep(ctx, step); err != nil {
		return domain.ThinkingStep{}, fmt.Errorf("emit thinking step: %w", err)
	}
	return step, nil
}

func (t *ProcessStepTracker) MarkRunningAsFailed(
	ctx context.Context,
	finalStatus domain.StepStatus,
) error {
	if finalStatus != domain.StepStatusFailed && finalStatus != domain.StepStatusStopped {
		return fmt.Errorf("invalid final status for running steps: %s", finalStatus)
	}
	return t.repo.MarkRunningAsFailed(ctx, t.responseRunID, finalStatus)
}
