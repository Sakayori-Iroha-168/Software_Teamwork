package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	AttachmentStatusUploaded = "uploaded"
	AttachmentStatusParsing  = "parsing"
	AttachmentStatusReady    = "ready"
	AttachmentStatusFailed   = "failed"
	AttachmentStatusDeleting = "deleting"
	AttachmentStatusDeleted  = "deleted"

	defaultAttachmentTTLHours = 24
	defaultAttachmentMaxBytes = 25 << 20
	defaultMaxPerSession      = 10
)

type SessionAttachment struct {
	ID           string     `json:"id"`
	SessionID    string     `json:"sessionId"`
	Filename     string     `json:"filename"`
	ContentType  string     `json:"contentType"`
	SizeBytes    int64      `json:"sizeBytes"`
	Status       string     `json:"status"`
	ErrorCode    string     `json:"errorCode,omitempty"`
	ErrorSummary string     `json:"errorSummary,omitempty"`
	PageCount    int        `json:"pageCount"`
	ChunkCount   int        `json:"chunkCount"`
	ExpiresAt    time.Time  `json:"expiresAt"`
	DeletedAt    *time.Time `json:"deletedAt,omitempty"`
	CreatedAt    time.Time  `json:"createdAt"`
	UpdatedAt    time.Time  `json:"updatedAt"`

	OwnerUserID string `json:"-"`
	FileRef     string `json:"-"`
}

type AttachmentChunk struct {
	ID           string    `json:"id"`
	AttachmentID string    `json:"attachmentId"`
	ChunkOrder   int       `json:"chunkOrder"`
	PageNumber   *int      `json:"pageNumber,omitempty"`
	SectionPath  string    `json:"sectionPath,omitempty"`
	Body         string    `json:"-"`
	Preview      string    `json:"preview"`
	TokenCount   int       `json:"tokenCount"`
	CharCount    int       `json:"charCount"`
	Filename     string    `json:"filename,omitempty"`
	CreatedAt    time.Time `json:"createdAt"`
}

type UploadAttachmentInput struct {
	Filename    string
	ContentType string
	SizeBytes   int64
	Body        io.Reader
}

type AttachmentConfig struct {
	TTL                 time.Duration
	MaxBytes            int64
	MaxPerSession       int
	ProcessTimeout      time.Duration
	AutoProcessUploaded bool
}

type AttachmentRepository interface {
	CreateAttachment(context.Context, SessionAttachment) (SessionAttachment, error)
	CountLiveAttachments(context.Context, string, string) (int, error)
	ListAttachments(context.Context, string, string) ([]SessionAttachment, error)
	GetAttachment(context.Context, string, string, string) (SessionAttachment, error)
	MarkAttachmentParsing(context.Context, string) (SessionAttachment, error)
	MarkAttachmentFailed(context.Context, string, string, string) error
	ReplaceAttachmentChunks(context.Context, string, []AttachmentChunk, int) error
	SoftDeleteAttachment(context.Context, string, string, string) (SessionAttachment, error)
	ValidateReadyAttachments(context.Context, string, string, []string) ([]SessionAttachment, error)
	SearchSessionAttachmentChunks(context.Context, string, string, []string, string, int) ([]AttachmentChunk, error)
	CleanupExpiredAttachments(context.Context, time.Time, int) ([]SessionAttachment, error)
	ListAttachmentsPendingFileDeleteRetry(context.Context, int) ([]SessionAttachment, error)
	CleanupSessionAttachments(context.Context, string, string) ([]SessionAttachment, error)
	MarkAttachmentFileDeleteRequested(context.Context, string, string) error
}

type FileClient interface {
	Upload(context.Context, FileUploadInput) (FileObject, error)
	Read(context.Context, string) ([]byte, error)
	Delete(context.Context, string) error
}

type FileUploadInput struct {
	Filename    string
	ContentType string
	SizeBytes   int64
	Body        io.Reader
}

type FileObject struct {
	ID          string
	Filename    string
	ContentType string
	SizeBytes   int64
}

type ParserClient interface {
	Parse(context.Context, ParseDocumentInput) (ParsedDocument, error)
}

type ParseDocumentInput struct {
	DocumentName string
	ContentType  string
	SizeBytes    int64
	Data         []byte
}

type ParsedDocument struct {
	Content string
	Title   string
	Backend string
	Pages   []ParsedPage
}

type ParsedPage struct {
	PageNumber int
	Content    string
	Blocks     []ParsedBlock
}

type ParsedBlock struct {
	Text string
	Type string
}

type AttachmentService struct {
	repository AttachmentRepository
	file       FileClient
	parser     ParserClient
	cfg        AttachmentConfig
	now        func() time.Time
}

func NewAttachmentService(repository AttachmentRepository, file FileClient, parser ParserClient, cfg AttachmentConfig) (*AttachmentService, error) {
	if repository == nil || file == nil || parser == nil {
		return nil, errors.New("attachment repository, file client and parser client are required")
	}
	if cfg.TTL <= 0 {
		cfg.TTL = defaultAttachmentTTLHours * time.Hour
	}
	if cfg.MaxBytes <= 0 {
		cfg.MaxBytes = defaultAttachmentMaxBytes
	}
	if cfg.MaxPerSession <= 0 {
		cfg.MaxPerSession = defaultMaxPerSession
	}
	if cfg.ProcessTimeout <= 0 {
		cfg.ProcessTimeout = 120 * time.Second
	}
	return &AttachmentService{repository: repository, file: file, parser: parser, cfg: cfg, now: time.Now}, nil
}

func (s *AttachmentService) Upload(ctx context.Context, userID, sessionID string, input UploadAttachmentInput) (SessionAttachment, error) {
	if strings.TrimSpace(userID) == "" {
		return SessionAttachment{}, NewError(CodeUnauthorized, "authentication required", nil)
	}
	filename := sanitizeFilename(input.Filename)
	contentType := normalizeAttachmentContentType(input.ContentType, filename)
	fields := map[string]string{}
	if filename == "" {
		fields["filename"] = "is required"
	}
	if !isAllowedAttachmentContentType(contentType) {
		fields["contentType"] = "must be PDF or image"
	}
	if input.SizeBytes <= 0 || input.SizeBytes > s.cfg.MaxBytes {
		fields["sizeBytes"] = fmt.Sprintf("must be between 1 and %d", s.cfg.MaxBytes)
	}
	if len(fields) > 0 {
		return SessionAttachment{}, ValidationError(fields)
	}
	count, err := s.repository.CountLiveAttachments(ctx, userID, sessionID)
	if err != nil {
		return SessionAttachment{}, err
	}
	if count >= s.cfg.MaxPerSession {
		return SessionAttachment{}, ValidationError(map[string]string{"attachments": "session attachment limit exceeded"})
	}
	file, err := s.file.Upload(ctx, FileUploadInput{Filename: filename, ContentType: contentType, SizeBytes: input.SizeBytes, Body: input.Body})
	if err != nil {
		return SessionAttachment{}, NewError(CodeDependency, "file upload failed", err)
	}
	now := s.now().UTC()
	attachment, err := s.repository.CreateAttachment(ctx, SessionAttachment{
		ID: newUUID(), SessionID: sessionID, OwnerUserID: userID, FileRef: file.ID,
		Filename: filename, ContentType: contentType, SizeBytes: input.SizeBytes,
		Status: AttachmentStatusUploaded, ExpiresAt: now.Add(s.cfg.TTL), CreatedAt: now, UpdatedAt: now,
	})
	if err != nil {
		_ = s.file.Delete(context.WithoutCancel(ctx), file.ID)
		return SessionAttachment{}, err
	}
	if s.cfg.AutoProcessUploaded {
		go s.processDetached(attachment.ID)
	}
	return attachment, nil
}

func (s *AttachmentService) List(ctx context.Context, userID, sessionID string) ([]SessionAttachment, error) {
	return s.repository.ListAttachments(ctx, userID, sessionID)
}

func (s *AttachmentService) Get(ctx context.Context, userID, sessionID, attachmentID string) (SessionAttachment, error) {
	return s.repository.GetAttachment(ctx, userID, sessionID, attachmentID)
}

func (s *AttachmentService) Delete(ctx context.Context, userID, sessionID, attachmentID string) error {
	attachment, err := s.repository.SoftDeleteAttachment(ctx, userID, sessionID, attachmentID)
	if err != nil {
		return err
	}
	return s.requestFileDelete(ctx, attachment)
}

func (s *AttachmentService) DeleteSession(ctx context.Context, userID, sessionID string) error {
	attachments, err := s.repository.CleanupSessionAttachments(ctx, userID, sessionID)
	if err != nil {
		return err
	}
	var firstErr error
	for _, attachment := range attachments {
		if err := s.requestFileDelete(ctx, attachment); err != nil {
			if firstErr == nil {
				firstErr = err
			}
		}
	}
	return firstErr
}

func (s *AttachmentService) Process(ctx context.Context, attachmentID string) error {
	attachment, err := s.repository.MarkAttachmentParsing(ctx, attachmentID)
	if err != nil {
		return err
	}
	data, err := s.file.Read(ctx, attachment.FileRef)
	if err != nil {
		_ = s.repository.MarkAttachmentFailed(context.WithoutCancel(ctx), attachmentID, "file_read_failed", "file content could not be read")
		return NewError(CodeDependency, "file content could not be read", err)
	}
	parsed, err := s.parser.Parse(ctx, ParseDocumentInput{DocumentName: attachment.Filename, ContentType: attachment.ContentType, SizeBytes: attachment.SizeBytes, Data: data})
	if err != nil {
		_ = s.repository.MarkAttachmentFailed(context.WithoutCancel(ctx), attachmentID, "parser_failed", "attachment parsing failed")
		return NewError(CodeDependency, "attachment parsing failed", err)
	}
	chunks := chunksFromParsedDocument(attachment.ID, parsed)
	if len(chunks) == 0 {
		_ = s.repository.MarkAttachmentFailed(context.WithoutCancel(ctx), attachmentID, "empty_parsed_content", "attachment did not contain readable text")
		return NewError(CodeValidation, "attachment did not contain readable text", nil)
	}
	pageCount := len(parsed.Pages)
	if pageCount == 0 {
		pageCount = maxPageNumber(chunks)
	}
	return s.repository.ReplaceAttachmentChunks(ctx, attachmentID, chunks, pageCount)
}

func (s *AttachmentService) Search(ctx context.Context, userID, sessionID string, attachmentIDs []string, query string, limit int) ([]AttachmentChunk, error) {
	ids := normalizeAttachmentIDs(attachmentIDs)
	if len(ids) == 0 {
		return nil, ValidationError(map[string]string{"attachmentIds": "at least one ready attachment must be bound to the message"})
	}
	if limit <= 0 || limit > 20 {
		limit = 5
	}
	return s.repository.SearchSessionAttachmentChunks(ctx, userID, sessionID, ids, strings.TrimSpace(query), limit)
}

func (s *AttachmentService) CleanupExpired(ctx context.Context, limit int) ([]SessionAttachment, error) {
	if limit <= 0 {
		limit = 100
	}
	items, err := s.repository.CleanupExpiredAttachments(ctx, s.now().UTC(), limit)
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		_ = s.requestFileDelete(ctx, item)
	}
	retryItems, err := s.repository.ListAttachmentsPendingFileDeleteRetry(ctx, limit)
	if err != nil {
		return nil, err
	}
	for _, item := range retryItems {
		_ = s.requestFileDelete(ctx, item)
	}
	return items, nil
}

func (s *AttachmentService) requestFileDelete(ctx context.Context, attachment SessionAttachment) error {
	if attachment.FileRef == "" {
		return nil
	}
	if err := s.file.Delete(ctx, attachment.FileRef); err != nil {
		_ = s.repository.MarkAttachmentFileDeleteRequested(context.WithoutCancel(ctx), attachment.ID, "file delete request failed")
		return NewError(CodeDependency, "file delete request failed", err)
	}
	return s.repository.MarkAttachmentFileDeleteRequested(context.WithoutCancel(ctx), attachment.ID, "")
}

func (s *AttachmentService) processDetached(attachmentID string) {
	ctx, cancel := context.WithTimeout(context.Background(), s.cfg.ProcessTimeout)
	defer cancel()
	_ = s.Process(ctx, attachmentID)
}

func ValidateAttachmentIDs(ctx context.Context, repository AttachmentRepository, userID, sessionID string, ids []string) ([]SessionAttachment, error) {
	ids = normalizeAttachmentIDs(ids)
	if len(ids) == 0 {
		return nil, nil
	}
	if len(ids) > 20 {
		return nil, ValidationError(map[string]string{"attachmentIds": "must not contain more than 20 items"})
	}
	return repository.ValidateReadyAttachments(ctx, userID, sessionID, ids)
}

func normalizeAttachmentIDs(ids []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

func sanitizeFilename(value string) string {
	value = strings.TrimSpace(filepath.Base(strings.ReplaceAll(value, "\\", "/")))
	value = strings.Trim(value, ". ")
	if utf8.RuneCountInString(value) > 255 {
		value = string([]rune(value)[:255])
	}
	return value
}

func normalizeAttachmentContentType(contentType, filename string) string {
	contentType = strings.TrimSpace(strings.Split(contentType, ";")[0])
	if contentType == "" {
		contentType = mime.TypeByExtension(strings.ToLower(filepath.Ext(filename)))
	}
	if contentType == "" {
		return "application/octet-stream"
	}
	return strings.ToLower(contentType)
}

func isAllowedAttachmentContentType(value string) bool {
	return value == "application/pdf" || strings.HasPrefix(value, "image/")
}

func chunksFromParsedDocument(attachmentID string, parsed ParsedDocument) []AttachmentChunk {
	chunks := []AttachmentChunk{}
	for _, page := range parsed.Pages {
		text := strings.TrimSpace(page.Content)
		if text == "" {
			parts := make([]string, 0, len(page.Blocks))
			for _, block := range page.Blocks {
				if trimmed := strings.TrimSpace(block.Text); trimmed != "" {
					parts = append(parts, trimmed)
				}
			}
			text = strings.Join(parts, "\n")
		}
		if text == "" {
			continue
		}
		pageNo := page.PageNumber
		chunks = appendTextChunks(chunks, attachmentID, text, &pageNo, "")
	}
	if len(chunks) == 0 {
		chunks = appendTextChunks(chunks, attachmentID, parsed.Content, nil, "")
	}
	for i := range chunks {
		chunks[i].ChunkOrder = i + 1
	}
	return chunks
}

func appendTextChunks(chunks []AttachmentChunk, attachmentID, text string, page *int, section string) []AttachmentChunk {
	const maxRunes = 1800
	runes := []rune(strings.TrimSpace(text))
	for len(runes) > 0 {
		n := len(runes)
		if n > maxRunes {
			n = maxRunes
		}
		body := strings.TrimSpace(string(runes[:n]))
		if body != "" {
			chunks = append(chunks, AttachmentChunk{ID: newUUID(), AttachmentID: attachmentID, PageNumber: page, SectionPath: section, Body: body, Preview: truncateRunes(body, 360), TokenCount: approxTokenCount(body), CharCount: utf8.RuneCountInString(body), CreatedAt: time.Now().UTC()})
		}
		runes = runes[n:]
	}
	return chunks
}

func approxTokenCount(value string) int {
	return len(strings.Fields(value))
}

func maxPageNumber(chunks []AttachmentChunk) int {
	maxPage := 0
	for _, chunk := range chunks {
		if chunk.PageNumber != nil && *chunk.PageNumber > maxPage {
			maxPage = *chunk.PageNumber
		}
	}
	return maxPage
}
