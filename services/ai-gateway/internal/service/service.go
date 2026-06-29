package service

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"slices"
	"strings"
	"time"
)

const (
	DefaultTimeoutMs = 60000

	credentialStorageModeEncrypted = "encrypted_column"
	credentialStatusActive         = "active"
	localCredentialKeyVersion      = "local-development-key"
)

type ProfileRepository interface {
	CreateProfile(ctx context.Context, profile ModelProfile, credential ProviderCredential, revision ModelProfileRevision) (ModelProfile, error)
	ListProfiles(ctx context.Context, filter ListFilter) ([]ModelProfile, error)
	GetProfile(ctx context.Context, id string) (ModelProfile, error)
	UpdateProfile(ctx context.Context, profile ModelProfile, credential *ProviderCredential, revision ModelProfileRevision) (ModelProfile, error)
	DeleteProfile(ctx context.Context, id string, deletedAt time.Time, revision ModelProfileRevision) error
	Ping(ctx context.Context) error
}

type Service struct {
	repo                 ProfileRepository
	now                  func() time.Time
	newID                func(prefix string) (string, error)
	encryptionKeyVersion string
	defaultTimeoutMs     int
}

type Option func(*Service)

func New(repo ProfileRepository, opts ...Option) *Service {
	s := &Service{
		repo:             repo,
		now:              func() time.Time { return time.Now().UTC() },
		newID:            newPublicID,
		defaultTimeoutMs: DefaultTimeoutMs,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

func WithClock(now func() time.Time) Option {
	return func(s *Service) {
		if now != nil {
			s.now = now
		}
	}
}

func WithIDGenerator(newID func(prefix string) (string, error)) Option {
	return func(s *Service) {
		if newID != nil {
			s.newID = newID
		}
	}
}

func WithEncryptionKeyVersion(version string) Option {
	return func(s *Service) {
		s.encryptionKeyVersion = strings.TrimSpace(version)
	}
}

func WithDefaultTimeoutMs(timeoutMs int) Option {
	return func(s *Service) {
		if timeoutMs >= 1000 {
			s.defaultTimeoutMs = timeoutMs
		}
	}
}

func (s *Service) CreateProfile(ctx context.Context, reqCtx RequestContext, input CreateModelProfileInput) (ModelProfile, error) {
	if err := validateInternalCaller(reqCtx); err != nil {
		return ModelProfile{}, err
	}
	now := s.now()
	profile, fields := s.profileFromCreate(input, now)
	if strings.TrimSpace(input.APIKey) == "" {
		fields["apiKey"] = "is required"
	}
	if len(fields) > 0 {
		return ModelProfile{}, ValidationError("request validation failed", fields)
	}

	profileID, err := s.newID("mp")
	if err != nil {
		return ModelProfile{}, DependencyError("profile id generation failed", err)
	}
	profile.ID = profileID
	profile.CreatedByUserID = strings.TrimSpace(reqCtx.UserID)
	profile.UpdatedByUserID = strings.TrimSpace(reqCtx.UserID)

	credential, err := s.credentialFromAPIKey(profile.ID, input.APIKey, reqCtx.UserID, now)
	if err != nil {
		return ModelProfile{}, err
	}
	revision, err := s.revision(profile, ModelProfile{}, "created", []string{"created"}, reqCtx, now)
	if err != nil {
		return ModelProfile{}, err
	}

	created, err := s.repo.CreateProfile(ctx, profile, credential, revision)
	if err != nil {
		return ModelProfile{}, mapRepositoryError(err, "model profile already exists")
	}
	return created, nil
}

func (s *Service) ListProfiles(ctx context.Context, reqCtx RequestContext, filter ListFilter) ([]ModelProfile, error) {
	if err := validateInternalCaller(reqCtx); err != nil {
		return nil, err
	}
	profiles, err := s.repo.ListProfiles(ctx, filter)
	if err != nil {
		return nil, DependencyError("model profile store read failed", err)
	}
	return profiles, nil
}

func (s *Service) GetProfile(ctx context.Context, reqCtx RequestContext, id string) (ModelProfile, error) {
	if err := validateInternalCaller(reqCtx); err != nil {
		return ModelProfile{}, err
	}
	profile, err := s.repo.GetProfile(ctx, strings.TrimSpace(id))
	if err != nil {
		return ModelProfile{}, mapRepositoryError(err, "model profile not found")
	}
	return profile, nil
}

func (s *Service) UpdateProfile(ctx context.Context, reqCtx RequestContext, input UpdateModelProfileInput) (ModelProfile, error) {
	if err := validateInternalCaller(reqCtx); err != nil {
		return ModelProfile{}, err
	}
	id := strings.TrimSpace(input.ID)
	if id == "" {
		return ModelProfile{}, ValidationError("request validation failed", map[string]string{"profileId": "is required"})
	}
	current, err := s.repo.GetProfile(ctx, id)
	if err != nil {
		return ModelProfile{}, mapRepositoryError(err, "model profile not found")
	}

	next := cloneProfile(current)
	changed := make([]string, 0)
	applyString(input.Name, &next.Name, "name", &changed)
	if input.Provider != nil {
		next.Provider = *input.Provider
		changed = append(changed, "provider")
	}
	applyString(input.BaseURL, &next.BaseURL, "baseUrl", &changed)
	applyString(input.Model, &next.Model, "model", &changed)
	applyBool(input.Enabled, &next.Enabled, "enabled", &changed)
	applyBool(input.IsDefault, &next.IsDefault, "isDefault", &changed)
	if input.TimeoutMs != nil {
		next.TimeoutMs = *input.TimeoutMs
		changed = append(changed, "timeoutMs")
	}
	if input.SupportsStreaming != nil {
		next.SupportsStreaming = *input.SupportsStreaming
		changed = append(changed, "supportsStreaming")
	}
	if input.Dimensions != nil {
		next.Dimensions = cloneIntPtr(input.Dimensions)
		changed = append(changed, "dimensions")
	}
	if input.TopN != nil {
		next.TopN = cloneIntPtr(input.TopN)
		changed = append(changed, "topN")
	}
	if input.DefaultParameters != nil {
		next.DefaultParameters = cloneRaw(*input.DefaultParameters)
		changed = append(changed, "defaultParameters")
	}

	now := s.now()
	next.UpdatedAt = now
	next.UpdatedByUserID = strings.TrimSpace(reqCtx.UserID)

	fields := validateProfile(next)
	if len(fields) > 0 {
		return ModelProfile{}, ValidationError("request validation failed", fields)
	}

	var credential *ProviderCredential
	if input.APIKey != nil && strings.TrimSpace(*input.APIKey) != "" {
		newCredential, err := s.credentialFromAPIKey(next.ID, *input.APIKey, reqCtx.UserID, now)
		if err != nil {
			return ModelProfile{}, err
		}
		credential = &newCredential
		next.APIKeyConfigured = true
		changed = append(changed, "apiKeyConfigured")
	}
	if len(changed) == 0 {
		changed = append(changed, "updated")
	}
	changeType := "updated"
	if credential != nil {
		changeType = "credential_rotated"
	}
	if next.IsDefault && next.Enabled && (!current.IsDefault || !current.Enabled) {
		changeType = "default_changed"
	}
	revision, err := s.revision(next, current, changeType, changed, reqCtx, now)
	if err != nil {
		return ModelProfile{}, err
	}

	updated, err := s.repo.UpdateProfile(ctx, next, credential, revision)
	if err != nil {
		return ModelProfile{}, mapRepositoryError(err, "model profile update conflict")
	}
	return updated, nil
}

func (s *Service) DeleteProfile(ctx context.Context, reqCtx RequestContext, id string) error {
	if err := validateInternalCaller(reqCtx); err != nil {
		return err
	}
	profile, err := s.repo.GetProfile(ctx, strings.TrimSpace(id))
	if err != nil {
		return mapRepositoryError(err, "model profile not found")
	}
	now := s.now()
	deleted := cloneProfile(profile)
	deleted.DeletedAt = &now
	deleted.Enabled = false
	deleted.IsDefault = false
	deleted.UpdatedAt = now
	revision, err := s.revision(deleted, profile, "deleted", []string{"deletedAt", "enabled", "isDefault"}, reqCtx, now)
	if err != nil {
		return err
	}
	if err := s.repo.DeleteProfile(ctx, profile.ID, now, revision); err != nil {
		return mapRepositoryError(err, "model profile not found")
	}
	return nil
}

func (s *Service) Readiness(ctx context.Context) Readiness {
	checks := []ReadinessCheck{}
	status := "ok"
	if err := s.repo.Ping(ctx); err != nil {
		return Readiness{
			Status: "unavailable",
			Checks: []ReadinessCheck{{
				Name:    "config_store",
				Status:  "failed",
				Message: "profile store is unavailable",
			}},
		}
	}
	checks = append(checks, ReadinessCheck{Name: "config_store", Status: "ok"})
	profiles, err := s.repo.ListProfiles(ctx, ListFilter{})
	if err != nil {
		return Readiness{
			Status: "unavailable",
			Checks: []ReadinessCheck{{
				Name:    "config_store",
				Status:  "failed",
				Message: "profile store is unavailable",
			}},
		}
	}
	for _, purpose := range []ModelPurpose{PurposeChat, PurposeEmbedding, PurposeRerank} {
		check := readinessForPurpose(purpose, profiles)
		if check.Status != "ok" && status == "ok" {
			status = "degraded"
		}
		checks = append(checks, check)
	}
	return Readiness{Status: status, Checks: checks}
}

func (s *Service) profileFromCreate(input CreateModelProfileInput, now time.Time) (ModelProfile, map[string]string) {
	enabled := true
	if input.Enabled != nil {
		enabled = *input.Enabled
	}
	isDefault := false
	if input.IsDefault != nil {
		isDefault = *input.IsDefault
	}
	timeoutMs := s.defaultTimeoutMs
	if input.TimeoutMs != nil {
		timeoutMs = *input.TimeoutMs
	}
	supportsStreaming := false
	if input.SupportsStreaming != nil {
		supportsStreaming = *input.SupportsStreaming
	}
	profile := ModelProfile{
		Name:              strings.TrimSpace(input.Name),
		Purpose:           input.Purpose,
		Provider:          input.Provider,
		BaseURL:           strings.TrimSpace(input.BaseURL),
		Model:             strings.TrimSpace(input.Model),
		Enabled:           enabled,
		IsDefault:         isDefault,
		TimeoutMs:         timeoutMs,
		SupportsStreaming: supportsStreaming,
		Dimensions:        cloneIntPtr(input.Dimensions),
		TopN:              cloneIntPtr(input.TopN),
		DefaultParameters: cloneRaw(input.DefaultParameters),
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	if len(profile.DefaultParameters) == 0 {
		profile.DefaultParameters = json.RawMessage(`{}`)
	}
	return profile, validateProfile(profile)
}

func (s *Service) credentialFromAPIKey(profileID string, apiKey string, userID string, now time.Time) (ProviderCredential, error) {
	trimmed := strings.TrimSpace(apiKey)
	if trimmed == "" {
		return ProviderCredential{}, ValidationError("request validation failed", map[string]string{"apiKey": "must not be empty"})
	}
	id, err := s.newID("cred")
	if err != nil {
		return ProviderCredential{}, DependencyError("credential id generation failed", err)
	}
	ciphertext, err := s.encryptCredential(trimmed)
	if err != nil {
		return ProviderCredential{}, DependencyError("credential encryption failed", err)
	}
	fingerprint := sha256.Sum256([]byte(trimmed))
	last4 := trimmed
	if len(last4) > 4 {
		last4 = last4[len(last4)-4:]
	}
	return ProviderCredential{
		ID:                   id,
		ProfileID:            profileID,
		StorageMode:          credentialStorageModeEncrypted,
		Ciphertext:           ciphertext,
		EncryptionKeyVersion: s.credentialKeyVersion(),
		FingerprintSHA256:    hex.EncodeToString(fingerprint[:]),
		KeyLast4:             last4,
		Status:               credentialStatusActive,
		CreatedByUserID:      strings.TrimSpace(userID),
		CreatedAt:            now,
	}, nil
}

func (s *Service) encryptCredential(value string) ([]byte, error) {
	keyMaterial := sha256.Sum256([]byte(s.credentialKeyVersion()))
	block, err := aes.NewCipher(keyMaterial[:])
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	out := make([]byte, 0, len(nonce)+len(value)+gcm.Overhead())
	out = append(out, nonce...)
	out = gcm.Seal(out, nonce, []byte(value), nil)
	return out, nil
}

func (s *Service) credentialKeyVersion() string {
	if s.encryptionKeyVersion == "" {
		return localCredentialKeyVersion
	}
	return s.encryptionKeyVersion
}

func (s *Service) revision(after ModelProfile, before ModelProfile, changeType string, fields []string, reqCtx RequestContext, now time.Time) (ModelProfileRevision, error) {
	id, err := s.newID("mpr")
	if err != nil {
		return ModelProfileRevision{}, DependencyError("revision id generation failed", err)
	}
	fieldBytes, err := json.Marshal(slices.Compact(fields))
	if err != nil {
		return ModelProfileRevision{}, DependencyError("revision encode failed", err)
	}
	afterSnapshot, err := json.Marshal(after)
	if err != nil {
		return ModelProfileRevision{}, DependencyError("revision encode failed", err)
	}
	var beforeSnapshot json.RawMessage
	if before.ID != "" {
		beforeSnapshot, err = json.Marshal(before)
		if err != nil {
			return ModelProfileRevision{}, DependencyError("revision encode failed", err)
		}
	}
	return ModelProfileRevision{
		ID:                 id,
		ProfileID:          after.ID,
		RevisionNo:         0,
		ChangeType:         changeType,
		ChangedFieldsJSON:  fieldBytes,
		BeforeSnapshotJSON: beforeSnapshot,
		AfterSnapshotJSON:  afterSnapshot,
		ChangedByUserID:    strings.TrimSpace(reqCtx.UserID),
		CallerService:      strings.TrimSpace(reqCtx.CallerService),
		RequestID:          strings.TrimSpace(reqCtx.RequestID),
		CreatedAt:          now,
	}, nil
}

func validateInternalCaller(reqCtx RequestContext) error {
	if strings.TrimSpace(reqCtx.ServiceToken) == "" || strings.TrimSpace(reqCtx.CallerService) == "" {
		return UnauthorizedError()
	}
	return nil
}

func validateProfile(profile ModelProfile) map[string]string {
	fields := map[string]string{}
	if strings.TrimSpace(profile.Name) == "" {
		fields["name"] = "is required"
	}
	if !slices.Contains([]ModelPurpose{PurposeChat, PurposeEmbedding, PurposeRerank}, profile.Purpose) {
		fields["purpose"] = "must be chat, embedding, or rerank"
	}
	if !slices.Contains([]ModelProvider{ProviderOpenAICompatible, ProviderSiliconFlow, ProviderLocalCompatible}, profile.Provider) {
		fields["provider"] = "must be openai_compatible, siliconflow, or local_compatible"
	}
	if strings.TrimSpace(profile.Model) == "" {
		fields["model"] = "is required"
	}
	if err := validateBaseURL(profile.BaseURL); err != nil {
		fields["baseUrl"] = err.Error()
	}
	if profile.TimeoutMs < 1000 {
		fields["timeoutMs"] = "must be at least 1000"
	}
	if profile.Purpose != PurposeChat && profile.SupportsStreaming {
		fields["supportsStreaming"] = "is only valid for chat profiles"
	}
	if profile.Dimensions != nil && *profile.Dimensions <= 0 {
		fields["dimensions"] = "must be positive"
	}
	if profile.TopN != nil && *profile.TopN <= 0 {
		fields["topN"] = "must be positive"
	}
	if len(profile.DefaultParameters) > 0 {
		if err := validateSafeJSONObject(profile.DefaultParameters); err != nil {
			fields["defaultParameters"] = err.Error()
		}
	}
	return fields
}

func validateBaseURL(raw string) error {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return fmt.Errorf("is required")
	}
	parsed, err := url.Parse(trimmed)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return fmt.Errorf("must be an absolute URL")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("must use http or https")
	}
	for key := range parsed.Query() {
		if isSensitiveKey(key) {
			return fmt.Errorf("must not contain sensitive query parameters")
		}
	}
	return nil
}

func validateSafeJSONObject(raw json.RawMessage) error {
	if len(raw) == 0 {
		return nil
	}
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return fmt.Errorf("must be a valid JSON object")
	}
	obj, ok := value.(map[string]any)
	if !ok {
		return fmt.Errorf("must be a JSON object")
	}
	return walkSafeJSON(obj)
}

func walkSafeJSON(value any) error {
	switch typed := value.(type) {
	case map[string]any:
		for key, nested := range typed {
			if isSensitiveKey(key) {
				return fmt.Errorf("must not contain sensitive fields")
			}
			if err := walkSafeJSON(nested); err != nil {
				return err
			}
		}
	case []any:
		for _, nested := range typed {
			if err := walkSafeJSON(nested); err != nil {
				return err
			}
		}
	}
	return nil
}

func isSensitiveKey(key string) bool {
	normalized := strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(key, "-", "_"), " ", "_"))
	for _, part := range []string{"api_key", "apikey", "authorization", "bearer", "token", "secret", "password", "credential", "connection_string", "database_url", "object_key", "prompt", "document_text", "provider_response"} {
		if strings.Contains(normalized, part) {
			return true
		}
	}
	return false
}

func readinessForPurpose(purpose ModelPurpose, profiles []ModelProfile) ReadinessCheck {
	for _, profile := range profiles {
		if profile.Purpose == purpose && profile.Enabled && profile.APIKeyConfigured {
			return ReadinessCheck{Name: string(purpose) + "_profile", Status: "ok"}
		}
	}
	return ReadinessCheck{Name: string(purpose) + "_profile", Status: "missing", Message: "enabled profile with configured API key is missing"}
}

func mapRepositoryError(err error, conflictMessage string) error {
	if errors.Is(err, ErrNotFound) {
		return NotFoundError("model profile not found")
	}
	if errors.Is(err, ErrConflict) {
		return ConflictError(conflictMessage, err)
	}
	return DependencyError("model profile store access failed", err)
}

func applyString(input *string, target *string, field string, changed *[]string) {
	if input == nil {
		return
	}
	*target = strings.TrimSpace(*input)
	*changed = append(*changed, field)
}

func applyBool(input *bool, target *bool, field string, changed *[]string) {
	if input == nil {
		return
	}
	*target = *input
	*changed = append(*changed, field)
}

func cloneProfile(profile ModelProfile) ModelProfile {
	profile.Dimensions = cloneIntPtr(profile.Dimensions)
	profile.TopN = cloneIntPtr(profile.TopN)
	profile.DefaultParameters = cloneRaw(profile.DefaultParameters)
	if profile.CredentialID != nil {
		value := *profile.CredentialID
		profile.CredentialID = &value
	}
	if profile.DeletedAt != nil {
		value := *profile.DeletedAt
		profile.DeletedAt = &value
	}
	return profile
}

func cloneIntPtr(value *int) *int {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func cloneRaw(value []byte) []byte {
	if value == nil {
		return nil
	}
	return append([]byte(nil), value...)
}

func newPublicID(prefix string) (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return prefix + "_" + hex.EncodeToString(bytes), nil
}
