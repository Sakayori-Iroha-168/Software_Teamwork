package service_test

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/ai-gateway/internal/repository"
	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/ai-gateway/internal/service"
)

func TestCreateProfileKeepsOneEnabledDefaultPerPurpose(t *testing.T) {
	repo := repository.NewMemoryRepository()
	svc := service.New(repo, service.WithClock(fixedClock()), service.WithIDGenerator(sequenceIDs()))
	ctx := service.RequestContext{RequestID: "req_test", CallerService: "gateway", ServiceToken: "token"}
	enabled := true
	isDefault := true
	streaming := true

	first, err := svc.CreateProfile(context.Background(), ctx, service.CreateModelProfileInput{
		Name:              "chat-a",
		Purpose:           service.PurposeChat,
		Provider:          service.ProviderOpenAICompatible,
		BaseURL:           "https://api.example.com/v1",
		Model:             "gpt-a",
		APIKey:            "sk_first",
		Enabled:           &enabled,
		IsDefault:         &isDefault,
		SupportsStreaming: &streaming,
	})
	if err != nil {
		t.Fatalf("CreateProfile(first) error = %v", err)
	}
	second, err := svc.CreateProfile(context.Background(), ctx, service.CreateModelProfileInput{
		Name:              "chat-b",
		Purpose:           service.PurposeChat,
		Provider:          service.ProviderOpenAICompatible,
		BaseURL:           "https://api.example.com/v1",
		Model:             "gpt-b",
		APIKey:            "sk_second",
		Enabled:           &enabled,
		IsDefault:         &isDefault,
		SupportsStreaming: &streaming,
	})
	if err != nil {
		t.Fatalf("CreateProfile(second) error = %v", err)
	}
	profiles, err := svc.ListProfiles(context.Background(), ctx, service.ListFilter{Purpose: ptr(service.PurposeChat)})
	if err != nil {
		t.Fatalf("ListProfiles() error = %v", err)
	}
	defaults := 0
	for _, profile := range profiles {
		if profile.IsDefault && profile.Enabled {
			defaults++
			if profile.ID != second.ID {
				t.Fatalf("default profile = %s, want %s", profile.ID, second.ID)
			}
		}
	}
	if defaults != 1 {
		t.Fatalf("enabled defaults = %d, profiles = %+v", defaults, profiles)
	}
	refreshedFirst, err := svc.GetProfile(context.Background(), ctx, first.ID)
	if err != nil {
		t.Fatalf("GetProfile(first) error = %v", err)
	}
	if refreshedFirst.IsDefault {
		t.Fatal("first profile remained default")
	}
}

func TestRejectsSensitiveDefaultParameters(t *testing.T) {
	repo := repository.NewMemoryRepository()
	svc := service.New(repo, service.WithClock(fixedClock()), service.WithIDGenerator(sequenceIDs()))
	params := json.RawMessage(`{"temperature":0.2,"api_key":"sk_secret"}`)
	_, err := svc.CreateProfile(context.Background(), service.RequestContext{CallerService: "gateway", ServiceToken: "token"}, service.CreateModelProfileInput{
		Name:              "chat",
		Purpose:           service.PurposeChat,
		Provider:          service.ProviderOpenAICompatible,
		BaseURL:           "https://api.example.com/v1",
		Model:             "gpt",
		APIKey:            "sk_secret",
		SupportsStreaming: ptr(true),
		DefaultParameters: params,
	})
	if err == nil {
		t.Fatal("CreateProfile() error = nil")
	}
	appErr, ok := service.Classify(err)
	if !ok || appErr.Code != service.CodeValidation {
		t.Fatalf("error = %#v", err)
	}
	if got := fmt.Sprint(appErr.Fields); strings.Contains(got, "sk_secret") {
		t.Fatalf("error leaked secret: %s", got)
	}
}

func TestCreateProfileUsesConfiguredDefaultTimeout(t *testing.T) {
	repo := repository.NewMemoryRepository()
	svc := service.New(repo,
		service.WithClock(fixedClock()),
		service.WithIDGenerator(sequenceIDs()),
		service.WithDefaultTimeoutMs(45000),
	)
	created, err := svc.CreateProfile(context.Background(), service.RequestContext{CallerService: "gateway", ServiceToken: "token"}, service.CreateModelProfileInput{
		Name:              "chat",
		Purpose:           service.PurposeChat,
		Provider:          service.ProviderOpenAICompatible,
		BaseURL:           "https://api.example.com/v1",
		Model:             "gpt",
		APIKey:            "sk_secret",
		SupportsStreaming: ptr(true),
	})
	if err != nil {
		t.Fatalf("CreateProfile() error = %v", err)
	}
	if created.TimeoutMs != 45000 {
		t.Fatalf("TimeoutMs = %d, want 45000", created.TimeoutMs)
	}
}

func fixedClock() func() time.Time {
	return func() time.Time { return time.Date(2026, 6, 29, 10, 0, 0, 0, time.UTC) }
}

func sequenceIDs() func(string) (string, error) {
	next := 0
	return func(prefix string) (string, error) {
		next++
		sum := sha256.Sum256([]byte(fmt.Sprintf("%s-%d", prefix, next)))
		return fmt.Sprintf("%s_%x", prefix, sum[:4]), nil
	}
}

func ptr[T any](value T) *T {
	return &value
}
