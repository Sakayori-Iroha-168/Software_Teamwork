package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/qa/internal/repository/sqlc"
	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/qa/internal/service"
)

type Postgres struct {
	pool    *pgxpool.Pool
	queries *sqlc.Queries
}

func NewPostgres(ctx context.Context, databaseURL string) (*Postgres, error) {
	if strings.TrimSpace(databaseURL) == "" {
		return nil, errors.New("QA_DATABASE_URL is required")
	}
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, errors.New("QA_DATABASE_URL is invalid")
	}
	config.MaxConns = 10
	config.MinConns = 1
	config.MaxConnLifetime = 30 * time.Minute
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("create postgres pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("connect postgres: %w", err)
	}
	return &Postgres{pool: pool, queries: sqlc.New(pool)}, nil
}

func (r *Postgres) Close() { r.pool.Close() }

func (r *Postgres) Ping(ctx context.Context) error {
	_, err := r.queries.Ping(ctx)
	return err
}

func (r *Postgres) CreateConversation(ctx context.Context, conversation service.Conversation) (service.Conversation, error) {
	if err := r.queries.InsertConversation(ctx, sqlc.InsertConversationParams{
		ID:             conversation.ID,
		ExternalUserID: conversation.OwnerUserID,
		Title:          conversation.Title,
		Status:         conversation.Status,
		CreatedAt:      conversation.CreatedAt,
		UpdatedAt:      conversation.UpdatedAt,
	}); err != nil {
		return service.Conversation{}, fmt.Errorf("insert conversation: %w", err)
	}
	return conversation, nil
}

func (r *Postgres) ListConversations(ctx context.Context, userID string, options service.ConversationListOptions) (service.Page[service.Conversation], error) {
	total, err := r.queries.CountConversationsByStatus(ctx, userID, options.Status)
	if err != nil {
		return service.Page[service.Conversation]{}, fmt.Errorf("count conversations: %w", err)
	}
	params := listConversationsParams(userID, options)
	rows, err := r.listConversationRows(ctx, options.Sort, params)
	if err != nil {
		return service.Page[service.Conversation]{}, fmt.Errorf("list conversations: %w", err)
	}
	items := make([]service.Conversation, 0, len(rows))
	for _, row := range rows {
		items = append(items, conversationFromRow(row))
	}
	return service.Page[service.Conversation]{Items: items, Page: options.Page, PageSize: options.PageSize, Total: int(total)}, nil
}

func (r *Postgres) listConversationRows(ctx context.Context, sort string, params sqlc.ListConversationsParams) ([]sqlc.ConversationSummaryRow, error) {
	switch sort {
	case "updatedAt":
		return r.queries.ListConversationsUpdatedAsc(ctx, params)
	case "-createdAt":
		return r.queries.ListConversationsCreatedDesc(ctx, params)
	case "createdAt":
		return r.queries.ListConversationsCreatedAsc(ctx, params)
	default:
		return r.queries.ListConversationsUpdatedDesc(ctx, params)
	}
}

func (r *Postgres) GetConversation(ctx context.Context, userID, id string) (service.Conversation, error) {
	row, err := r.queries.GetConversationForUser(ctx, id, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return service.Conversation{}, r.conversationAccessError(ctx, userID, id)
	}
	if err != nil {
		return service.Conversation{}, fmt.Errorf("get conversation: %w", err)
	}
	return conversationFromRow(row), nil
}

func (r *Postgres) UpdateConversation(ctx context.Context, userID string, conversation service.Conversation) (service.Conversation, error) {
	rowsAffected, err := r.queries.UpdateConversation(ctx, sqlc.UpdateConversationParams{
		Title:          conversation.Title,
		Status:         conversation.Status,
		UpdatedAt:      conversation.UpdatedAt,
		ID:             conversation.ID,
		ExternalUserID: userID,
	})
	if err != nil {
		return service.Conversation{}, fmt.Errorf("update conversation: %w", err)
	}
	if rowsAffected == 0 {
		return service.Conversation{}, r.conversationAccessError(ctx, userID, conversation.ID)
	}
	return conversation, nil
}

func (r *Postgres) DeleteConversation(ctx context.Context, userID, id string) error {
	rowsAffected, err := r.queries.SoftDeleteConversation(ctx, id, userID)
	if err != nil {
		return fmt.Errorf("delete conversation: %w", err)
	}
	if rowsAffected == 0 {
		return r.conversationAccessError(ctx, userID, id)
	}
	return nil
}

func (r *Postgres) ListMessages(ctx context.Context, userID, conversationID string, options service.MessageListOptions) (service.Page[service.Message], error) {
	total, err := r.queries.CountMessagesForConversation(ctx, conversationID, userID)
	if err != nil {
		return service.Page[service.Message]{}, fmt.Errorf("count messages: %w", err)
	}
	rows, err := r.queries.ListMessagesForConversation(ctx, sqlc.ListMessagesForConversationParams{
		ConversationID: conversationID,
		ExternalUserID: userID,
		PageSize:       int32(options.PageSize),
		PageOffset:     int32((options.Page - 1) * options.PageSize),
	})
	if err != nil {
		return service.Page[service.Message]{}, fmt.Errorf("list messages: %w", err)
	}
	items := make([]service.Message, 0, len(rows))
	for _, row := range rows {
		items = append(items, messageFromRow(row))
	}
	if total == 0 {
		if _, err := r.GetConversation(ctx, userID, conversationID); err != nil {
			return service.Page[service.Message]{}, err
		}
	}
	if err := r.enrichMessages(ctx, userID, conversationID, items, options); err != nil {
		return service.Page[service.Message]{}, err
	}
	return service.Page[service.Message]{Items: items, Page: options.Page, PageSize: options.PageSize, Total: int(total)}, nil
}

func (r *Postgres) AppendMessages(ctx context.Context, userID, conversationID string, start service.ResponseRunStart, messages ...service.Message) (service.ResponseRun, error) {
	if len(messages) == 0 {
		return service.ResponseRun{}, nil
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return service.ResponseRun{}, fmt.Errorf("begin append messages: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := r.queries.WithTx(tx)
	if _, err := q.LockConversationForUser(ctx, conversationID, userID); errors.Is(err, pgx.ErrNoRows) {
		return service.ResponseRun{}, r.conversationAccessError(ctx, userID, conversationID)
	} else if err != nil {
		return service.ResponseRun{}, fmt.Errorf("lock conversation: %w", err)
	}
	sequence, err := q.GetMaxMessageSequence(ctx, conversationID)
	if err != nil {
		return service.ResponseRun{}, fmt.Errorf("get message sequence: %w", err)
	}
	var userMessageID, assistantMessageID, intent string
	for _, message := range messages {
		sequence++
		if err := q.InsertMessage(ctx, sqlc.InsertMessageParams{
			ID: message.ID, ConversationID: conversationID, Role: message.Role,
			SequenceNo: sequence, Intent: message.Intent, Status: message.Status, CreatedAt: message.CreatedAt,
		}); err != nil {
			return service.ResponseRun{}, fmt.Errorf("insert message: %w", err)
		}
		if err := q.InsertMessageContentBlock(ctx, sqlc.InsertMessageContentBlockParams{
			MessageID: message.ID, Content: message.Content, Status: blockStatus(message.Status), CreatedAt: message.CreatedAt,
		}); err != nil {
			return service.ResponseRun{}, fmt.Errorf("insert message content: %w", err)
		}
		if message.Role == "user" {
			userMessageID = message.ID
		}
		if message.Role == "assistant" {
			assistantMessageID, intent = message.ID, message.Intent
		}
	}
	lastAt := messages[len(messages)-1].CreatedAt
	if err := q.TouchConversationActivity(ctx, sqlc.TouchConversationActivityParams{
		UpdatedAt: lastAt, LastMessageAt: lastAt, ID: conversationID,
	}); err != nil {
		return service.ResponseRun{}, fmt.Errorf("touch conversation: %w", err)
	}
	var run service.ResponseRun
	if userMessageID != "" && assistantMessageID != "" {
		requestID := start.RequestID
		if requestID == "" {
			requestID = service.RequestIDFromContext(ctx)
		}
		inserted, err := q.InsertResponseRun(ctx, sqlc.InsertResponseRunParams{
			ConversationID: conversationID, UserMessageID: userMessageID,
			AssistantMessageID: assistantMessageID, QaConfigVersionID: start.QAConfigVersionID,
			LlmConfigVersionID: start.LLMConfigVersionID, RequestID: requestID,
			IntentType: intent, MaxIterations: int32(start.MaxIterations),
		})
		if err != nil {
			return service.ResponseRun{}, fmt.Errorf("insert response run: %w", err)
		}
		run = service.ResponseRun{
			ID: inserted.ID, SessionID: inserted.ConversationID, UserMessageID: inserted.UserMessageID,
			AssistantMessageID: inserted.AssistantMessageID, Status: inserted.Status, CreatedAt: inserted.StartedAt,
			CurrentIteration: int(inserted.CurrentIteration), MaxIterations: int(inserted.MaxIterations),
		}
		payload, err := json.Marshal(map[string]any{
			"responseRunId": run.ID, "userMessageId": userMessageID,
			"assistantMessageId": assistantMessageID, "status": "running",
		})
		if err != nil {
			return service.ResponseRun{}, fmt.Errorf("encode initial stream event: %w", err)
		}
		if err := q.InsertStreamEvent(ctx, sqlc.InsertStreamEventParams{
			ResponseRunID: run.ID, EventSeq: 1, EventType: "message.created",
			Payload: payload, CreatedAt: inserted.StartedAt,
		}); err != nil {
			return service.ResponseRun{}, fmt.Errorf("insert initial stream event: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return service.ResponseRun{}, fmt.Errorf("commit append messages: %w", err)
	}
	return run, nil
}

func (r *Postgres) UpdateMessage(ctx context.Context, userID string, message service.Message) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin update message: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := r.queries.WithTx(tx)
	rowsAffected, err := q.UpdateMessageStatus(ctx, sqlc.UpdateMessageStatusParams{
		Status: message.Status, Intent: message.Intent, ID: message.ID, ExternalUserID: userID,
	})
	if err != nil {
		return fmt.Errorf("update message: %w", err)
	}
	if rowsAffected == 0 {
		return service.NewError(service.CodeNotFound, "message not found", nil)
	}
	if err := q.UpdateMessageContentBlock(ctx, message.Content, blockStatus(message.Status), message.ID); err != nil {
		return fmt.Errorf("update message content: %w", err)
	}
	if err := q.UpdateResponseRunByAssistantMessage(ctx, runStatus(message.Status), message.ID); err != nil {
		return fmt.Errorf("update response run: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit update message: %w", err)
	}
	return nil
}

func (r *Postgres) FinalizeResponseRun(ctx context.Context, userID string, final service.ResponseRunFinalization) (service.ResponseRun, error) {
	if final.CompletedAt.IsZero() {
		final.CompletedAt = time.Now().UTC()
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return service.ResponseRun{}, fmt.Errorf("begin finalize response run: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := r.queries.WithTx(tx)
	row, err := q.FinalizeResponseRun(ctx, sqlc.FinalizeResponseRunParams{
		Status: final.Status, TerminationReason: final.TerminationReason,
		CurrentIteration: int32(final.CurrentIteration),
		PromptTokens:     int32(final.PromptTokens), CompletionTokens: int32(final.CompletionTokens),
		ReasoningTokens: int32(final.ReasoningTokens), CompletedAt: final.CompletedAt,
		ID: final.RunID, ExternalUserID: userID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		existing, loadErr := q.GetResponseRunForUser(ctx, final.RunID, userID)
		if errors.Is(loadErr, pgx.ErrNoRows) {
			return service.ResponseRun{}, service.NewError(service.CodeNotFound, "response run not found", err)
		}
		if loadErr != nil {
			return service.ResponseRun{}, fmt.Errorf("load response run finalization state: %w", loadErr)
		}
		return responseRunFromRow(existing), service.NewError(service.CodeConflict, "response run already finalized", err)
	}
	if err != nil {
		return service.ResponseRun{}, fmt.Errorf("finalize response run: %w", err)
	}
	rowsAffected, err := q.UpdateMessageStatus(ctx, sqlc.UpdateMessageStatusParams{
		Status: final.AssistantMessage.Status, Intent: final.AssistantMessage.Intent,
		ID: final.AssistantMessage.ID, ExternalUserID: userID,
	})
	if err != nil {
		return service.ResponseRun{}, fmt.Errorf("update assistant message: %w", err)
	}
	if rowsAffected == 0 {
		return service.ResponseRun{}, service.NewError(service.CodeNotFound, "message not found", nil)
	}
	if err := q.UpdateMessageContentBlock(ctx, final.AssistantMessage.Content, blockStatus(final.AssistantMessage.Status), final.AssistantMessage.ID); err != nil {
		return service.ResponseRun{}, fmt.Errorf("update assistant content: %w", err)
	}
	if err := replaceReasoningSteps(ctx, q, final.RunID, final.ReasoningSteps); err != nil {
		return service.ResponseRun{}, err
	}
	if err := replaceStreamEvents(ctx, tx, q, final.RunID, final.StreamEvents); err != nil {
		return service.ResponseRun{}, err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM citations WHERE message_id = $1::uuid`, final.AssistantMessage.ID); err != nil {
		return service.ResponseRun{}, fmt.Errorf("delete existing citations: %w", err)
	}
	if len(final.Citations) > 0 {
		for _, citation := range final.Citations {
			metadata, _ := json.Marshal(citation.Metadata)
			if metadata == nil {
				metadata = []byte("{}")
			}
			var pageNum *int32
			if citation.PageNumber != nil {
				pn := int32(*citation.PageNumber)
				pageNum = &pn
			}
			var score *float64
			if citation.Score != nil {
				score = citation.Score
			}
			var rerankScore *float64
			if citation.RerankScore != nil {
				rerankScore = citation.RerankScore
			}
			_, err = tx.Exec(ctx, `INSERT INTO citations(id,message_id,citation_no,char_start,char_end,external_kb_id,external_doc_id,external_chunk_id,doc_name,section_path,quote_text,context,page_number,score,rerank_score,chunk_type,metadata) VALUES($1::uuid,$2::uuid,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17)`,
				citation.ID, final.AssistantMessage.ID, citation.CitationNo,
				nil, nil,
				citation.KnowledgeBaseID, citation.DocumentID, citation.ChunkID,
				citation.DocumentName, citation.SectionPath, citation.Text,
				citation.Context, pageNum, score,
				rerankScore, citation.ChunkType, metadata,
			)
			if err != nil {
				return service.ResponseRun{}, fmt.Errorf("insert citation: %w", err)
			}
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return service.ResponseRun{}, fmt.Errorf("commit finalize response run: %w", err)
	}
	return responseRunFromRow(row), nil
}

func (r *Postgres) SaveReasoningSteps(ctx context.Context, userID, assistantMessageID string, steps []service.ReasoningStep) error {
	if len(steps) == 0 {
		return nil
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin save reasoning steps: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := r.queries.WithTx(tx)
	runID, err := q.GetResponseRunIDByAssistantMessage(ctx, assistantMessageID, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return service.NewError(service.CodeNotFound, "response run not found", err)
	}
	if err != nil {
		return fmt.Errorf("find response run: %w", err)
	}
	if err := replaceReasoningSteps(ctx, q, runID, steps); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit reasoning steps: %w", err)
	}
	return nil
}

func (r *Postgres) SaveStreamEvents(ctx context.Context, userID, runID string, events []service.StreamEvent) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin save stream events: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := r.queries.WithTx(tx)
	if _, err := q.AuthorizeResponseRunForUser(ctx, runID, userID); errors.Is(err, pgx.ErrNoRows) {
		return service.NewError(service.CodeNotFound, "response run not found", err)
	} else if err != nil {
		return fmt.Errorf("authorize stream events: %w", err)
	}
	if err := replaceStreamEvents(ctx, tx, q, runID, events); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit stream events: %w", err)
	}
	return nil
}

func replaceReasoningSteps(ctx context.Context, q *sqlc.Queries, runID string, steps []service.ReasoningStep) error {
	if err := q.DeleteProcessStepsByRun(ctx, runID); err != nil {
		return fmt.Errorf("replace reasoning steps: %w", err)
	}
	for index, step := range steps {
		if err := q.InsertProcessStep(ctx, sqlc.InsertProcessStepParams{
			ID: step.ID, ResponseRunID: runID, StepOrder: int32(index + 1),
			StepType: step.Type, Label: step.Title, Detail: step.Summary, Status: step.Status, CreatedAt: step.CreatedAt,
		}); err != nil {
			return fmt.Errorf("insert reasoning step: %w", err)
		}
	}
	return nil
}

func replaceStreamEvents(ctx context.Context, tx pgx.Tx, q *sqlc.Queries, runID string, events []service.StreamEvent) error {
	if err := q.DeleteStreamEventsByRun(ctx, runID); err != nil {
		return fmt.Errorf("replace stream events: %w", err)
	}
	if err := q.DeleteToolCallsByRun(ctx, runID); err != nil {
		return fmt.Errorf("replace tool call summaries: %w", err)
	}
	if len(events) == 0 {
		return nil
	}
	for _, event := range events {
		payload, err := json.Marshal(event.Payload)
		if err != nil {
			return fmt.Errorf("encode stream event: %w", err)
		}
		if err := q.InsertStreamEvent(ctx, sqlc.InsertStreamEventParams{
			ResponseRunID: runID, EventSeq: int32(event.EventSeq), EventType: event.EventType,
			Payload: payload, CreatedAt: event.CreatedAt,
		}); err != nil {
			return fmt.Errorf("insert stream event: %w", err)
		}
		iteration, _ := event.Payload["iterationNo"].(int)
		if event.EventType == "agent.iteration.started" && iteration > 0 {
			if err := q.UpdateResponseRunIteration(ctx, int32(iteration), runID); err != nil {
				return fmt.Errorf("update response run iteration: %w", err)
			}
		}
		if event.EventType == "tool.started" || event.EventType == "tool.completed" || event.EventType == "tool.failed" {
			toolCallID, _ := event.Payload["toolCallId"].(string)
			toolName, _ := event.Payload["tool"].(string)
			argumentsSummary, _ := event.Payload["arguments"].(map[string]any)
			resultSummary, _ := event.Payload["result"].(map[string]any)
			if toolCallID == "" {
				continue
			}
			status := "running"
			if event.EventType == "tool.completed" {
				status = "completed"
			}
			if event.EventType == "tool.failed" {
				status = "failed"
			}
			argsJSON, _ := json.Marshal(argumentsSummary)
			resultJSON, _ := json.Marshal(resultSummary)
			if err := upsertAgentToolCall(ctx, tx, runID, int32(iteration), toolCallID, toolName, status, argsJSON, resultJSON, event.CreatedAt); err != nil {
				return fmt.Errorf("save tool call summary: %w", err)
			}
		}
	}
	return nil
}

func upsertAgentToolCall(ctx context.Context, tx pgx.Tx, runID string, iteration int32, toolCallID, toolName, status string, argumentsSummary, resultSummary []byte, startedAt time.Time) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO agent_tool_calls (
			response_run_id,
			iteration_no,
			tool_call_id,
			tool_name,
			status,
			arguments_summary,
			result_summary,
			started_at,
			finished_at
		) VALUES (
			$1::uuid,
			GREATEST($2, 1),
			$3,
			$4,
			$5,
			$6::jsonb,
			$7::jsonb,
			$8::timestamptz,
			CASE
				WHEN $5 = 'running' THEN NULL
				ELSE $8::timestamptz
			END
		)
		ON CONFLICT (response_run_id, tool_call_id) DO UPDATE SET
			status = EXCLUDED.status,
			arguments_summary = EXCLUDED.arguments_summary,
			result_summary = EXCLUDED.result_summary,
			finished_at = CASE
				WHEN EXCLUDED.status = 'running' THEN agent_tool_calls.finished_at
				ELSE EXCLUDED.finished_at
			END,
			latency_ms = CASE
				WHEN EXCLUDED.status = 'running' THEN agent_tool_calls.latency_ms
				ELSE EXTRACT(EPOCH FROM (EXCLUDED.finished_at - agent_tool_calls.started_at)) * 1000
			END`,
		runID,
		iteration,
		toolCallID,
		toolName,
		status,
		argumentsSummary,
		resultSummary,
		startedAt,
	)
	return err
}

func (r *Postgres) SaveModelInvocation(ctx context.Context, userID string, invocation service.ModelInvocation) (string, error) {
	if _, err := r.queries.AuthorizeResponseRunForUser(ctx, invocation.ResponseRunID, userID); errors.Is(err, pgx.ErrNoRows) {
		return "", service.NewError(service.CodeNotFound, "response run not found", err)
	} else if err != nil {
		return "", fmt.Errorf("authorize model invocation: %w", err)
	}
	finishReason := nullableText(invocation.FinishReason)
	errorCode := nullableText(invocation.ErrorCode)
	errorMessage := nullableText(invocation.ErrorMessage)
	finishedAt := pgtype.Timestamptz{}
	if invocation.FinishedAt != nil {
		finishedAt = pgtype.Timestamptz{Time: *invocation.FinishedAt, Valid: true}
	}
	id, err := r.queries.InsertModelInvocation(ctx, sqlc.InsertModelInvocationParams{
		ResponseRunID:    invocation.ResponseRunID,
		IterationNo:      int32(invocation.IterationNo),
		Provider:         invocation.Provider,
		ProfileID:        invocation.ProfileID,
		ModelName:        invocation.ModelName,
		FinishReason:     finishReason,
		Status:           invocation.Status,
		PromptTokens:     nullableInt4(invocation.PromptTokens),
		CompletionTokens: nullableInt4(invocation.CompletionTokens),
		ReasoningTokens:  nullableInt4(invocation.ReasoningTokens),
		TotalTokens:      nullableInt4(invocation.TotalTokens),
		LatencyMs:        nullableInt8(invocation.LatencyMS),
		ErrorCode:        errorCode,
		ErrorMessage:     errorMessage,
		StartedAt:        invocation.StartedAt,
		FinishedAt:       finishedAt,
	})
	if err != nil {
		return "", fmt.Errorf("insert model invocation: %w", err)
	}
	return id, nil
}

func (r *Postgres) GetResponseRun(ctx context.Context, userID, runID string) (service.ResponseRun, error) {
	row, err := r.queries.GetResponseRunForUser(ctx, runID, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return service.ResponseRun{}, service.NewError(service.CodeNotFound, "response run not found", err)
	}
	if err != nil {
		return service.ResponseRun{}, fmt.Errorf("get response run: %w", err)
	}
	return responseRunFromRow(row), nil
}

func (r *Postgres) SaveCitations(ctx context.Context, userID, messageID string, citations []service.Citation) error {
	if len(citations) == 0 {
		return nil
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin save citations: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var authorized bool
	err = tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM messages m
			JOIN conversations c ON c.id = m.conversation_id
			WHERE m.id::text = $1
				AND c.external_user_id = $2
				AND c.deleted_at IS NULL
		)`, messageID, userID).Scan(&authorized)
	if err != nil {
		return fmt.Errorf("authorize message access: %w", err)
	}
	if !authorized {
		return service.NewError(service.CodeNotFound, "message not found", nil)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM citations WHERE message_id = $1::uuid`, messageID); err != nil {
		return fmt.Errorf("delete existing citations: %w", err)
	}
	for _, citation := range citations {
		metadata, _ := json.Marshal(citation.Metadata)
		if metadata == nil {
			metadata = []byte("{}")
		}
		var pageNum *int32
		if citation.PageNumber != nil {
			pn := int32(*citation.PageNumber)
			pageNum = &pn
		}
		var score *float64
		if citation.Score != nil {
			score = citation.Score
		}
		var rerankScore *float64
		if citation.RerankScore != nil {
			rerankScore = citation.RerankScore
		}
		_, err = tx.Exec(ctx, `INSERT INTO citations(id,message_id,citation_no,char_start,char_end,external_kb_id,external_doc_id,external_chunk_id,doc_name,section_path,quote_text,context,page_number,score,rerank_score,chunk_type,metadata) VALUES($1::uuid,$2::uuid,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17)`,
			citation.ID, messageID, citation.CitationNo,
			nil, nil,
			citation.KnowledgeBaseID, citation.DocumentID, citation.ChunkID,
			citation.DocumentName, citation.SectionPath, citation.Text,
			citation.Context, pageNum, score,
			rerankScore, citation.ChunkType, metadata,
		)
		if err != nil {
			return fmt.Errorf("insert citation: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit citations: %w", err)
	}
	return nil
}

func (r *Postgres) CancelResponseRun(ctx context.Context, userID, runID string) (service.ResponseRun, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return service.ResponseRun{}, fmt.Errorf("begin cancel response run: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := r.queries.WithTx(tx)
	assistantID, err := q.CancelResponseRun(ctx, runID, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		if _, accessErr := q.GetResponseRunForUser(ctx, runID, userID); errors.Is(accessErr, pgx.ErrNoRows) {
			return service.ResponseRun{}, service.NewError(service.CodeNotFound, "response run not found", accessErr)
		} else if accessErr != nil {
			return service.ResponseRun{}, fmt.Errorf("authorize response run cancellation: %w", accessErr)
		}
		return service.ResponseRun{}, service.NewError(service.CodeConflict, "response run cannot be cancelled", err)
	}
	if err != nil {
		return service.ResponseRun{}, fmt.Errorf("cancel response run: %w", err)
	}
	if err := q.CancelAssistantMessage(ctx, assistantID); err != nil {
		return service.ResponseRun{}, fmt.Errorf("cancel assistant message: %w", err)
	}
	if err := q.CancelAssistantMessageContent(ctx, assistantID); err != nil {
		return service.ResponseRun{}, fmt.Errorf("cancel assistant content: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return service.ResponseRun{}, fmt.Errorf("commit response run cancellation: %w", err)
	}
	return r.GetResponseRun(ctx, userID, runID)
}

func (r *Postgres) conversationAccessError(ctx context.Context, userID, id string) error {
	row, err := r.queries.GetConversationAccess(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return service.NewError(service.CodeNotFound, "conversation not found", err)
	}
	if err != nil {
		return fmt.Errorf("authorize conversation access: %w", err)
	}
	if row.DeletedAt.Valid {
		return service.NewError(service.CodeNotFound, "conversation not found", nil)
	}
	if row.ExternalUserID != userID {
		return service.NewError(service.CodeForbidden, "conversation access denied", nil)
	}
	return service.NewError(service.CodeNotFound, "conversation not found", nil)
}

func blockStatus(messageStatus string) string {
	switch messageStatus {
	case "queued":
		return "queued"
	case "generating", "streaming":
		return "streaming"
	case "failed":
		return "failed"
	case "stopped", "cancelled":
		return messageStatus
	default:
		return "completed"
	}
}

func runStatus(messageStatus string) string {
	switch messageStatus {
	case "generating", "queued", "streaming":
		return "running"
	case "stopped", "cancelled":
		return "cancelled"
	case "failed":
		return "failed"
	default:
		return "completed"
	}
}
