package service

import "context"

type attachmentScopeKey struct{}

type AttachmentScope struct {
	UserID        string
	SessionID     string
	AttachmentIDs []string
}

func WithAttachmentScope(ctx context.Context, scope AttachmentScope) context.Context {
	return context.WithValue(ctx, attachmentScopeKey{}, scope)
}

func AttachmentScopeFromContext(ctx context.Context) (AttachmentScope, bool) {
	scope, ok := ctx.Value(attachmentScopeKey{}).(AttachmentScope)
	return scope, ok
}
