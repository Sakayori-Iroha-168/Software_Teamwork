package repository

import (
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/qa/internal/repository/sqlc"
	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/qa/internal/service"
)

func conversationFromRow(row sqlc.ConversationSummaryRow) service.Conversation {
	conversation := service.Conversation{
		ID:                 row.ID,
		Title:              row.Title,
		OwnerUserID:        row.ExternalUserID,
		Status:             row.Status,
		CreatedAt:          row.CreatedAt,
		UpdatedAt:          row.UpdatedAt,
		MessageCount:       int(row.MessageCount),
		LastMessagePreview: row.LastMessagePreview,
	}
	if row.LastMessageAt.Valid {
		conversation.LastMessageAt = &row.LastMessageAt.Time
	}
	return conversation
}

func listConversationsParams(userID string, options service.ConversationListOptions) sqlc.ListConversationsParams {
	return sqlc.ListConversationsParams{
		ExternalUserID: userID,
		Status:         options.Status,
		PageSize:       int32(options.PageSize),
		PageOffset:     int32((options.Page - 1) * options.PageSize),
	}
}

func nullableText(value string) pgtype.Text {
	if value == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: value, Valid: true}
}

func nullableInt4(value int) pgtype.Int4 {
	if value == 0 {
		return pgtype.Int4{}
	}
	return pgtype.Int4{Int32: int32(value), Valid: true}
}

func nullableInt8(value int64) pgtype.Int8 {
	if value == 0 {
		return pgtype.Int8{}
	}
	return pgtype.Int8{Int64: value, Valid: true}
}

func messageFromRow(row sqlc.MessageRow) service.Message {
	message := service.Message{
		ID:             row.ID,
		ConversationID: row.ConversationID,
		SequenceNo:     int(row.SequenceNo),
		Role:           row.Role,
		Content:        row.Content,
		Intent:         row.Intent,
		Status:         row.Status,
		CitationCount:  int(row.CitationCount),
		CreatedAt:      row.CreatedAt,
	}
	if row.CompletedAt.Valid {
		message.CompletedAt = &row.CompletedAt.Time
	}
	if message.Status == "generating" {
		message.Status = "streaming"
	}
	return message
}

func responseRunFromRow(row sqlc.ResponseRunRow) service.ResponseRun {
	value := service.ResponseRun{
		ID:                 row.ID,
		SessionID:          row.ConversationID,
		UserMessageID:      row.UserMessageID,
		AssistantMessageID: row.AssistantMessageID,
		Status:             row.Status,
		CurrentIteration:   int(row.CurrentIteration),
		MaxIterations:      int(row.MaxIterations),
		TotalTokens:        int(row.TotalTokens),
		LatencyMS:          row.LatencyMs,
		CreatedAt:          row.StartedAt,
	}
	if row.CompletedAt.Valid {
		value.CompletedAt = &row.CompletedAt.Time
	}
	if row.StopReason.Valid {
		reason := row.StopReason.String
		if reason == "failed" {
			reason = "model_error"
		}
		value.TerminationReason = &reason
	}
	return value
}
